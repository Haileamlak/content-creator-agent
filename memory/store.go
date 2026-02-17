package memory

import (
	"content-creator-agent/models"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Store defines the interface for long-term persistence.
type Store interface {
	// Posts
	SavePost(post models.Post) error
	GetHistory(brandID string) ([]models.Post, error)
	GetAnalytics(brandID string) ([]models.Analytics, error)
	UpdateAnalytics(brandID string, postID string, analytics models.Analytics) error

	// Brands
	SaveBrand(brand models.BrandProfile, userID string) error
	GetBrand(id string) (models.BrandProfile, string, error) // Returns brand + userID
	ListBrands(userID string) ([]models.BrandProfile, error)
	ListAllBrands() ([]models.BrandProfile, error)
	DeleteBrand(id string) error

	// Calendar & Approval
	SaveScheduledPost(post models.ScheduledPost) error
	GetScheduledPosts(brandID string) ([]models.ScheduledPost, error)
	UpdateScheduledPostStatus(postID string, status models.PostStatus) error
	GetPendingScheduledPosts() ([]models.ScheduledPost, error) // For the scheduler to publish

	// User management
	CreateUser(email, passwordHash string) (string, error)
	GetUserByEmail(email string) (string, string, error) // Returns userID, passwordHash, error
}

// FileStore implements Store using JSON files on disk.
type FileStore struct {
	BaseDir string
	mu      sync.Mutex
}

func NewFileStore(baseDir string) *FileStore {
	if baseDir == "" {
		baseDir = "data"
	}
	return &FileStore{BaseDir: baseDir}
}

func (f *FileStore) brandPath(brandID string) string {
	return filepath.Join(f.BaseDir, brandID)
}

func (f *FileStore) SavePost(post models.Post) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	path := f.brandPath(post.BrandID)
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create brand dir: %w", err)
	}

	historyPath := filepath.Join(path, "history.json")
	var history []models.Post

	// Read existing history
	data, err := os.ReadFile(historyPath)
	if err == nil {
		json.Unmarshal(data, &history)
	}

	// Append new post
	history = append(history, post)

	// Save back
	updatedData, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(historyPath, updatedData, 0644)
}

func (f *FileStore) GetHistory(brandID string) ([]models.Post, error) {
	historyPath := filepath.Join(f.brandPath(brandID), "history.json")
	data, err := os.ReadFile(historyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []models.Post{}, nil
		}
		return nil, err
	}

	var history []models.Post
	if err := json.Unmarshal(data, &history); err != nil {
		return nil, err
	}
	return history, nil
}

// GetAnalytics is a placeholder for retrieving aggregated analytics.
func (f *FileStore) GetAnalytics(brandID string) ([]models.Analytics, error) {
	history, err := f.GetHistory(brandID)
	if err != nil {
		return nil, err
	}

	var analytics []models.Analytics
	for _, p := range history {
		analytics = append(analytics, p.Analytics)
	}
	return analytics, nil
}
func (f *FileStore) UpdateAnalytics(brandID string, postID string, analytics models.Analytics) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	historyPath := filepath.Join(f.brandPath(brandID), "history.json")
	data, err := os.ReadFile(historyPath)
	if err != nil {
		return err
	}

	var history []models.Post
	if err := json.Unmarshal(data, &history); err != nil {
		return err
	}

	found := false
	for i := range history {
		if history[i].ID == postID {
			history[i].Analytics = analytics
			history[i].UpdatedAt = time.Now()
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("post %s not found in history", postID)
	}

	updatedData, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(historyPath, updatedData, 0644)
}

// --- Brand Management (FileStore Impl) ---

func (f *FileStore) SaveBrand(brand models.BrandProfile, userID string) error {
	path := filepath.Join(f.BaseDir, brand.ID)
	if err := os.MkdirAll(path, 0755); err != nil {
		return err
	}
	configPath := filepath.Join(path, "config.json")
	data, _ := json.MarshalIndent(brand, "", "  ")
	return os.WriteFile(configPath, data, 0644)
}

func (f *FileStore) GetBrand(id string) (models.BrandProfile, string, error) {
	configPath := filepath.Join(f.BaseDir, id, "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return models.BrandProfile{}, "", err
	}
	var brand models.BrandProfile
	if err := json.Unmarshal(data, &brand); err != nil {
		return models.BrandProfile{}, "", err
	}
	return brand, "local-user", nil
}

func (f *FileStore) ListBrands(userID string) ([]models.BrandProfile, error) {
	return f.ListAllBrands()
}

func (f *FileStore) ListAllBrands() ([]models.BrandProfile, error) {
	entries, err := os.ReadDir(f.BaseDir)
	if err != nil {
		return nil, err
	}
	var brands []models.BrandProfile
	for _, entry := range entries {
		if entry.IsDir() {
			brand, _, err := f.GetBrand(entry.Name())
			if err == nil {
				brands = append(brands, brand)
			}
		}
	}
	return brands, nil
}

func (f *FileStore) DeleteBrand(id string) error {
	path := filepath.Join(f.BaseDir, id)
	return os.RemoveAll(path)
}

// Calendar Stubs for FileStore
func (f *FileStore) SaveScheduledPost(post models.ScheduledPost) error { return nil }
func (f *FileStore) GetScheduledPosts(brandID string) ([]models.ScheduledPost, error) {
	return nil, nil
}
func (f *FileStore) UpdateScheduledPostStatus(postID string, status models.PostStatus) error {
	return nil
}
func (f *FileStore) GetPendingScheduledPosts() ([]models.ScheduledPost, error) { return nil, nil }

// --- User Management (FileStore Impl) ---

type fileUser struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	PasswordHash string `json:"password_hash"`
}

func (f *FileStore) CreateUser(email, passwordHash string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	usersPath := filepath.Join(f.BaseDir, "users.json")
	var users []fileUser

	data, err := os.ReadFile(usersPath)
	if err == nil {
		json.Unmarshal(data, &users)
	}

	for _, u := range users {
		if u.Email == email {
			return "", fmt.Errorf("user already exists")
		}
	}

	userID := fmt.Sprintf("u-%d", time.Now().Unix())
	users = append(users, fileUser{
		ID:           userID,
		Email:        email,
		PasswordHash: passwordHash,
	})

	updatedData, _ := json.MarshalIndent(users, "", "  ")
	os.WriteFile(usersPath, updatedData, 0644)

	return userID, nil
}

func (f *FileStore) GetUserByEmail(email string) (string, string, error) {
	usersPath := filepath.Join(f.BaseDir, "users.json")
	data, err := os.ReadFile(usersPath)
	if err != nil {
		return "", "", err
	}

	var users []fileUser
	json.Unmarshal(data, &users)

	for _, u := range users {
		if u.Email == email {
			return u.ID, u.PasswordHash, nil
		}
	}

	return "", "", fmt.Errorf("user not found")
}
