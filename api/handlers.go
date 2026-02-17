package api

import (
	"content-creator-agent/memory"
	"content-creator-agent/models"
	"content-creator-agent/scheduler"
	"content-creator-agent/tools"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Handlers holds the dependencies for all HTTP handlers.
type Handlers struct {
	Store     memory.Store
	Queue     scheduler.Queue
	JWTSecret string

	// Tools needed to construct agents on-the-fly per brand
	Search    tools.SearchTool
	LLM       tools.LLMTool
	Social    tools.SocialClient
	Embedding tools.EmbeddingTool
	Analytics tools.AnalyticsFetcher
	DataDir   string
}

// --- Auth Handlers ---

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handlers) Register(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Email == "" || req.Password == "" {
		Error(w, http.StatusBadRequest, "email and password are required")
		return
	}

	hash := hashPassword(req.Password)
	userID, err := h.Store.CreateUser(req.Email, hash)
	if err != nil {
		Error(w, http.StatusConflict, "failed to create user: email might be taken")
		return
	}

	token, err := GenerateToken(userID, req.Email, h.JWTSecret)
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	JSON(w, http.StatusCreated, map[string]string{"token": token, "user_id": userID})
}

func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	userID, hash, err := h.Store.GetUserByEmail(req.Email)
	if err != nil {
		Error(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if hashPassword(req.Password) != hash {
		Error(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := GenerateToken(userID, req.Email, h.JWTSecret)
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	JSON(w, http.StatusOK, map[string]string{"token": token, "user_id": userID})
}

// --- Brand Handlers ---

func (h *Handlers) CreateBrand(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	var brand models.BrandProfile
	if err := json.NewDecoder(r.Body).Decode(&brand); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if brand.ID == "" || brand.Name == "" {
		Error(w, http.StatusBadRequest, "brand id and name are required")
		return
	}

	// Prefix brand ID with user ID for uniqueness in multi-tenant DB if needed,
	// but with P0 DB we just store user_id in the row.
	// For backward compatibility with the current system we prefix it.
	brand.ID = fmt.Sprintf("%s_%s", userID[:8], brand.ID)

	if err := h.Store.SaveBrand(brand, userID); err != nil {
		Error(w, http.StatusInternalServerError, "failed to save brand")
		return
	}

	JSON(w, http.StatusCreated, brand)
}

func (h *Handlers) ListBrands(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	brands, err := h.Store.ListBrands(userID)
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to load brands")
		return
	}
	JSON(w, http.StatusOK, brands)
}

func (h *Handlers) GetBrand(w http.ResponseWriter, r *http.Request) {
	brandID := chi.URLParam(r, "brandID")
	brand, _, err := h.Store.GetBrand(brandID)
	if err != nil {
		Error(w, http.StatusNotFound, "brand not found")
		return
	}
	JSON(w, http.StatusOK, brand)
}

func (h *Handlers) UpdateBrand(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	brandID := chi.URLParam(r, "brandID")
	var brand models.BrandProfile
	if err := json.NewDecoder(r.Body).Decode(&brand); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	brand.ID = brandID

	if err := h.Store.SaveBrand(brand, userID); err != nil {
		Error(w, http.StatusInternalServerError, "failed to update brand")
		return
	}

	JSON(w, http.StatusOK, brand)
}

func (h *Handlers) DeleteBrand(w http.ResponseWriter, r *http.Request) {
	brandID := chi.URLParam(r, "brandID")
	if err := h.Store.DeleteBrand(brandID); err != nil {
		Error(w, http.StatusNotFound, "brand not found")
		return
	}
	JSON(w, http.StatusOK, map[string]string{"deleted": brandID})
}

// --- Agent Action Handlers ---

func (h *Handlers) TriggerRun(w http.ResponseWriter, r *http.Request) {
	brandID := chi.URLParam(r, "brandID")
	if _, _, err := h.Store.GetBrand(brandID); err != nil {
		Error(w, http.StatusNotFound, "brand not found")
		return
	}

	if err := h.Queue.Enqueue(brandID, scheduler.JobTypeRun, 0, ""); err != nil {
		Error(w, http.StatusInternalServerError, "failed to enqueue job")
		return
	}

	JSON(w, http.StatusAccepted, map[string]string{
		"status":  "accepted",
		"brand":   brandID,
		"message": "Agent cycle started in background",
	})
}

func (h *Handlers) TriggerSync(w http.ResponseWriter, r *http.Request) {
	brandID := chi.URLParam(r, "brandID")
	if _, _, err := h.Store.GetBrand(brandID); err != nil {
		Error(w, http.StatusNotFound, "brand not found")
		return
	}

	if err := h.Queue.Enqueue(brandID, scheduler.JobTypeSync, 0, ""); err != nil {
		Error(w, http.StatusInternalServerError, "failed to enqueue sync job")
		return
	}

	JSON(w, http.StatusAccepted, map[string]string{
		"status":  "accepted",
		"brand":   brandID,
		"message": "Analytics sync started in background",
	})
}

// --- Post & Analytics Handlers ---

func (h *Handlers) ListPosts(w http.ResponseWriter, r *http.Request) {
	brandID := chi.URLParam(r, "brandID")
	posts, err := h.Store.GetHistory(brandID)
	if err != nil {
		JSON(w, http.StatusOK, []models.Post{})
		return
	}
	JSON(w, http.StatusOK, posts)
}

func (h *Handlers) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	brandID := chi.URLParam(r, "brandID")
	analytics, err := h.Store.GetAnalytics(brandID)
	if err != nil {
		JSON(w, http.StatusOK, []models.Analytics{})
		return
	}
	JSON(w, http.StatusOK, analytics)
}

func hashPassword(password string) string {
	h := sha256.Sum256([]byte(password))
	return hex.EncodeToString(h[:])
}
