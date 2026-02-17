package memory

import (
	"content-creator-agent/models"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore implements Store using a PostgreSQL database.
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore creates a new PostgresStore and initializes the connection pool.
func NewPostgresStore(connStr string) (*PostgresStore, error) {
	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("unable to parse connection string: %v", err)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %v", err)
	}

	return &PostgresStore{pool: pool}, nil
}

func (p *PostgresStore) Close() {
	p.pool.Close()
}

// --- Post Management ---

func (p *PostgresStore) SavePost(post models.Post) error {
	query := `
		INSERT INTO posts (id, social_id, brand_id, topic, content, platform, status, views, likes, shares, comments, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err := p.pool.Exec(context.Background(), query,
		post.ID, post.SocialID, post.BrandID, post.Topic, post.Content, post.Platform,
		string(post.Status), post.Analytics.Views, post.Analytics.Likes,
		post.Analytics.Shares, post.Analytics.Comments, post.CreatedAt, post.UpdatedAt,
	)
	return err
}

func (p *PostgresStore) GetHistory(brandID string) ([]models.Post, error) {
	query := `SELECT id, social_id, brand_id, topic, content, platform, status, views, likes, shares, comments, created_at, updated_at 
	          FROM posts WHERE brand_id = $1 ORDER BY created_at DESC`
	rows, err := p.pool.Query(context.Background(), query, brandID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []models.Post
	for rows.Next() {
		var post models.Post
		var status string
		var socialID sql.NullString
		err := rows.Scan(
			&post.ID, &socialID, &post.BrandID, &post.Topic, &post.Content,
			&post.Platform, &status, &post.Analytics.Views, &post.Analytics.Likes,
			&post.Analytics.Shares, &post.Analytics.Comments, &post.CreatedAt, &post.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		post.SocialID = socialID.String
		post.Status = models.PostStatus(status)
		posts = append(posts, post)
	}
	return posts, nil
}

func (p *PostgresStore) GetAnalytics(brandID string) ([]models.Analytics, error) {
	query := `SELECT views, likes, shares, comments FROM posts WHERE brand_id = $1`
	rows, err := p.pool.Query(context.Background(), query, brandID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.Analytics
	for rows.Next() {
		var a models.Analytics
		if err := rows.Scan(&a.Views, &a.Likes, &a.Shares, &a.Comments); err != nil {
			return nil, err
		}
		results = append(results, a)
	}
	return results, nil
}

func (p *PostgresStore) UpdateAnalytics(brandID string, postID string, a models.Analytics) error {
	query := `UPDATE posts SET views = $1, likes = $2, shares = $3, comments = $4, updated_at = $5 WHERE id = $6 AND brand_id = $7`
	_, err := p.pool.Exec(context.Background(), query, a.Views, a.Likes, a.Shares, a.Comments, time.Now(), postID, brandID)
	return err
}

// --- Brand Management ---

func (p *PostgresStore) SaveBrand(brand models.BrandProfile, userID string) error {
	query := `
		INSERT INTO brands (id, user_id, name, industry, voice, target_audience, topics, anti_topics, schedule_interval_hours)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			industry = EXCLUDED.industry,
			voice = EXCLUDED.voice,
			target_audience = EXCLUDED.target_audience,
			topics = EXCLUDED.topics,
			anti_topics = EXCLUDED.anti_topics,
			schedule_interval_hours = EXCLUDED.schedule_interval_hours
	`
	topicsJSON, _ := json.Marshal(brand.Topics)
	antiTopicsJSON, _ := json.Marshal(brand.AntiTopics)

	_, err := p.pool.Exec(context.Background(), query,
		brand.ID, userID, brand.Name, brand.Industry, brand.Voice, brand.TargetAudience, topicsJSON, antiTopicsJSON, brand.ScheduleIntervalHours,
	)
	return err
}

func (p *PostgresStore) GetBrand(id string) (models.BrandProfile, string, error) {
	query := `SELECT id, user_id, name, industry, voice, target_audience, topics, anti_topics, schedule_interval_hours FROM brands WHERE id = $1`
	var brand models.BrandProfile
	var userID string
	var topics, antiTopics []byte

	err := p.pool.QueryRow(context.Background(), query, id).Scan(
		&brand.ID, &userID, &brand.Name, &brand.Industry, &brand.Voice, &brand.TargetAudience, &topics, &antiTopics, &brand.ScheduleIntervalHours,
	)
	if err != nil {
		return brand, "", err
	}

	json.Unmarshal(topics, &brand.Topics)
	json.Unmarshal(antiTopics, &brand.AntiTopics)

	return brand, userID, nil
}

func (p *PostgresStore) ListBrands(userID string) ([]models.BrandProfile, error) {
	query := `SELECT id, name, industry, voice, target_audience, topics, anti_topics, schedule_interval_hours FROM brands WHERE user_id = $1`
	rows, err := p.pool.Query(context.Background(), query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var brands []models.BrandProfile
	for rows.Next() {
		var b models.BrandProfile
		var topics, antiTopics []byte
		err := rows.Scan(&b.ID, &b.Name, &b.Industry, &b.Voice, &b.TargetAudience, &topics, &antiTopics, &b.ScheduleIntervalHours)
		if err != nil {
			return nil, err
		}
		json.Unmarshal(topics, &b.Topics)
		json.Unmarshal(antiTopics, &b.AntiTopics)
		brands = append(brands, b)
	}
	return brands, nil
}

func (p *PostgresStore) ListAllBrands() ([]models.BrandProfile, error) {
	query := `SELECT id, name, industry, voice, target_audience, topics, anti_topics, schedule_interval_hours FROM brands`
	rows, err := p.pool.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var brands []models.BrandProfile
	for rows.Next() {
		var b models.BrandProfile
		var topics, antiTopics []byte
		err := rows.Scan(&b.ID, &b.Name, &b.Industry, &b.Voice, &b.TargetAudience, &topics, &antiTopics, &b.ScheduleIntervalHours)
		if err != nil {
			return nil, err
		}
		json.Unmarshal(topics, &b.Topics)
		json.Unmarshal(antiTopics, &b.AntiTopics)
		brands = append(brands, b)
	}
	return brands, nil
}

func (p *PostgresStore) DeleteBrand(id string) error {
	query := `DELETE FROM brands WHERE id = $1`
	_, err := p.pool.Exec(context.Background(), query, id)
	return err
}

// --- Calendar & Approval ---

func (p *PostgresStore) SaveScheduledPost(post models.ScheduledPost) error {
	query := `
		INSERT INTO scheduled_posts (id, brand_id, topic, content, platform, status, scheduled_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			content = EXCLUDED.content,
			scheduled_at = EXCLUDED.scheduled_at
	`
	_, err := p.pool.Exec(context.Background(), query,
		post.ID, post.BrandID, post.Topic, post.Content, post.Platform, string(post.Status), post.ScheduledAt, post.CreatedAt,
	)
	return err
}

func (p *PostgresStore) GetScheduledPosts(brandID string) ([]models.ScheduledPost, error) {
	query := `SELECT id, brand_id, topic, content, platform, status, scheduled_at, created_at FROM scheduled_posts WHERE brand_id = $1 ORDER BY scheduled_at ASC`
	rows, err := p.pool.Query(context.Background(), query, brandID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []models.ScheduledPost
	for rows.Next() {
		var post models.ScheduledPost
		var status string
		err := rows.Scan(&post.ID, &post.BrandID, &post.Topic, &post.Content, &post.Platform, &status, &post.ScheduledAt, &post.CreatedAt)
		if err != nil {
			return nil, err
		}
		post.Status = models.PostStatus(status)
		posts = append(posts, post)
	}
	return posts, nil
}

func (p *PostgresStore) UpdateScheduledPostStatus(postID string, status models.PostStatus) error {
	query := `UPDATE scheduled_posts SET status = $1 WHERE id = $2`
	_, err := p.pool.Exec(context.Background(), query, string(status), postID)
	return err
}

func (p *PostgresStore) GetPendingScheduledPosts() ([]models.ScheduledPost, error) {
	query := `SELECT id, brand_id, topic, content, platform, status, scheduled_at, created_at FROM scheduled_posts WHERE status = $1 AND scheduled_at <= $2`
	rows, err := p.pool.Query(context.Background(), query, string(models.StatusApproved), time.Now())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []models.ScheduledPost
	for rows.Next() {
		var post models.ScheduledPost
		var status string
		err := rows.Scan(&post.ID, &post.BrandID, &post.Topic, &post.Content, &post.Platform, &status, &post.ScheduledAt, &post.CreatedAt)
		if err != nil {
			return nil, err
		}
		post.Status = models.PostStatus(status)
		posts = append(posts, post)
	}
	return posts, nil
}

// --- User Management ---

func (p *PostgresStore) CreateUser(email, passwordHash string) (string, error) {
	query := `INSERT INTO users (id, email, password_hash) VALUES (gen_random_uuid(), $1, $2) RETURNING id`
	var userID string
	err := p.pool.QueryRow(context.Background(), query, email, passwordHash).Scan(&userID)
	return userID, err
}

func (p *PostgresStore) GetUserByEmail(email string) (string, string, error) {
	query := `SELECT id, password_hash FROM users WHERE email = $1`
	var id, hash string
	err := p.pool.QueryRow(context.Background(), query, email).Scan(&id, &hash)
	return id, hash, err
}
