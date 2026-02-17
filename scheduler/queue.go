package scheduler

import (
	"database/sql"
	"time"

	_ "modernc.org/sqlite"
)

type JobType string

const (
	JobTypeRun     JobType = "run"
	JobTypeSync    JobType = "sync"
	JobTypePlan    JobType = "plan"    // Batch planning
	JobTypePublish JobType = "publish" // Specific post publication
)

type JobStatus string

const (
	StatusPending   JobStatus = "pending"
	StatusRunning   JobStatus = "running"
	StatusCompleted JobStatus = "completed"
	StatusFailed    JobStatus = "failed"
)

type Job struct {
	ID        int64
	BrandID   string
	Type      JobType
	Status    JobStatus
	Retries   int
	NextRunAt time.Time
	Payload   string // Additional data like ScheduledPostID
	Error     string
}

// Queue defines the job management interface
type Queue interface {
	Enqueue(brandID string, jobType JobType, delay time.Duration, payload string) error
	Dequeue() (*Job, error)
	Ack(jobID int64) error
	Fail(jobID int64, errMsg string, retry bool) error
	HasPendingJob(brandID string) (bool, error)
}

type SQLiteQueue struct {
	db *sql.DB
}

func NewSQLiteQueue(dbPath string) (*SQLiteQueue, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// Create jobs table
	schema := `
	CREATE TABLE IF NOT EXISTS jobs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		brand_id TEXT NOT NULL,
		type TEXT NOT NULL,
		status TEXT NOT NULL,
		retries INTEGER DEFAULT 0,
		next_run_at DATETIME NOT NULL,
		payload TEXT,
		error TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_jobs_status_next_run ON jobs(status, next_run_at);
	`
	if _, err := db.Exec(schema); err != nil {
		return nil, err
	}

	return &SQLiteQueue{db: db}, nil
}

func (q *SQLiteQueue) Enqueue(brandID string, jobType JobType, delay time.Duration, payload string) error {
	nextRun := time.Now().Add(delay)
	query := `INSERT INTO jobs (brand_id, type, status, next_run_at, payload) VALUES (?, ?, ?, ?, ?)`
	_, err := q.db.Exec(query, brandID, string(jobType), string(StatusPending), nextRun, payload)
	return err
}

func (q *SQLiteQueue) Dequeue() (*Job, error) {
	tx, err := q.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	query := `
		SELECT id, brand_id, type, status, retries, next_run_at, payload, error 
		FROM jobs 
		WHERE status = ? AND next_run_at <= ? 
		ORDER BY next_run_at ASC 
		LIMIT 1
	`
	var job Job
	var jobType, status string
	var payload, errStr sql.NullString
	err = tx.QueryRow(query, string(StatusPending), time.Now()).Scan(
		&job.ID, &job.BrandID, &jobType, &status, &job.Retries, &job.NextRunAt, &payload, &errStr,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	job.Type = JobType(jobType)
	job.Status = StatusRunning
	job.Payload = payload.String
	job.Error = errStr.String

	// Update status to running
	updateQuery := `UPDATE jobs SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	if _, err := tx.Exec(updateQuery, string(StatusRunning), job.ID); err != nil {
		return nil, err
	}

	return &job, tx.Commit()
}

func (q *SQLiteQueue) Ack(jobID int64) error {
	query := `DELETE FROM jobs WHERE id = ?`
	_, err := q.db.Exec(query, jobID)
	return err
}

func (q *SQLiteQueue) Fail(jobID int64, errMsg string, retry bool) error {
	if retry {
		// Backoff: 5m, 15m, 1h, 4h...
		delay := 5 * time.Minute
		query := `UPDATE jobs SET status = ?, retries = retries + 1, next_run_at = ?, error = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
		_, err := q.db.Exec(query, string(StatusPending), time.Now().Add(delay), errMsg, jobID)
		return err
	}

	query := `UPDATE jobs SET status = ?, error = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := q.db.Exec(query, string(StatusFailed), errMsg, jobID)
	return err
}

func (q *SQLiteQueue) HasPendingJob(brandID string) (bool, error) {
	query := `SELECT COUNT(*) FROM jobs WHERE brand_id = ? AND status IN (?, ?)`
	var count int
	err := q.db.QueryRow(query, brandID, string(StatusPending), string(StatusRunning)).Scan(&count)
	return count > 0, err
}

func (q *SQLiteQueue) Close() error {
	return q.db.Close()
}
