package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server holds the HTTP server and all its dependencies.
type Server struct {
	Router    *chi.Mux
	Handlers  *Handlers
	JWTSecret string
	Port      string
}

// NewServer creates and configures the API server.
func NewServer(h *Handlers, jwtSecret, port string) *Server {
	s := &Server{
		Router:    chi.NewRouter(),
		Handlers:  h,
		JWTSecret: jwtSecret,
		Port:      port,
	}
	s.mountMiddleware()
	s.mountRoutes()
	return s
}

func (s *Server) mountMiddleware() {
	s.Router.Use(middleware.RequestID)
	s.Router.Use(middleware.RealIP)
	s.Router.Use(middleware.Logger)
	s.Router.Use(middleware.Recoverer)
	s.Router.Use(middleware.Timeout(60 * time.Second))
	s.Router.Use(corsMiddleware)
}

func (s *Server) mountRoutes() {
	r := s.Router

	// Public routes
	r.Post("/api/auth/register", s.Handlers.Register)
	r.Post("/api/auth/login", s.Handlers.Login)
	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(s.AuthMiddleware)

		// Brands
		r.Post("/api/brands", s.Handlers.CreateBrand)
		r.Get("/api/brands", s.Handlers.ListBrands)
		r.Get("/api/brands/{brandID}", s.Handlers.GetBrand)
		r.Put("/api/brands/{brandID}", s.Handlers.UpdateBrand)
		r.Delete("/api/brands/{brandID}", s.Handlers.DeleteBrand)

		// Agent Actions
		r.Post("/api/brands/{brandID}/run", s.Handlers.TriggerRun)
		r.Post("/api/brands/{brandID}/sync", s.Handlers.TriggerSync)

		// Posts & Analytics
		r.Get("/api/brands/{brandID}/posts", s.Handlers.ListPosts)
		r.Get("/api/brands/{brandID}/analytics", s.Handlers.GetAnalytics)
	})

	// Static files for Dashboard
	workDir, _ := os.Getwd()
	filesDir := http.Dir(filepath.Join(workDir, "web"))
	FileServer(r, "/", filesDir)
}

// FileServer conveniently sets up a http.FileServer handler to serve
// static files from a http.FileSystem.
func FileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit any URL parameters.")
	}

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", http.StatusMovedPermanently).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		fs.ServeHTTP(w, r)
	})
}

// Start begins listening on the configured port.
func (s *Server) Start() error {
	return http.ListenAndServe(":"+s.Port, s.Router)
}

// --- Helpers ---

// JSON writes a JSON response.
func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// Error writes a JSON error response.
func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, map[string]string{"error": message})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
