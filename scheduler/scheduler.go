package scheduler

import (
	"content-creator-agent/memory"
	"content-creator-agent/models"
	"fmt"
	"time"
)

// Scheduler manages the per-brand recurring job cycles.
type Scheduler struct {
	Store memory.Store
	Queue Queue
}

func NewScheduler(s memory.Store, q Queue) *Scheduler {
	return &Scheduler{
		Store: s,
		Queue: q,
	}
}

// Start initiates the scheduling loop that ensures all brands have active jobs.
func (s *Scheduler) Start() {
	fmt.Println("⏰ Scheduler started. Managing recurring brand cycles...")
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	// Initial sync
	s.SyncAllBrands()
	s.CheckScheduledPosts()

	for range ticker.C {
		s.SyncAllBrands()
		s.CheckScheduledPosts()
	}
}

// SyncAllBrands ensures every brand in the store has a scheduled job in the queue.
func (s *Scheduler) SyncAllBrands() {
	brands, err := s.Store.ListAllBrands()
	if err != nil {
		fmt.Printf("Scheduler failed to list brands: %v\n", err)
		return
	}

	for _, b := range brands {
		s.EnsureScheduled(b.ID, b.ScheduleIntervalHours)
	}
}

// CheckScheduledPosts looks for posts that are due to be published.
func (s *Scheduler) CheckScheduledPosts() {
	posts, err := s.Store.GetPendingScheduledPosts()
	if err != nil {
		fmt.Printf("Scheduler failed to get pending posts: %v\n", err)
		return
	}

	for _, p := range posts {
		fmt.Printf("⏰ Enqueuing publish job for post %s (Brand: %s)\n", p.ID, p.BrandID)
		// We use 0 delay because it's already due or past due
		s.Queue.Enqueue(p.BrandID, JobTypePublish, 0, p.ID)
		// Mark as scheduled in the calendar table so we don't enqueue it again next time
		s.Store.UpdateScheduledPostStatus(p.ID, models.StatusScheduled)
	}
}

// EnsureScheduled checks if a brand needs a new job and enqueues it.
func (s *Scheduler) EnsureScheduled(brandID string, intervalHours int) {
	if intervalHours <= 0 {
		intervalHours = 4 // Default
	}

	exists, err := s.Queue.HasPendingJob(brandID)
	if err != nil {
		fmt.Printf("Scheduler error checking job existence for %s: %v\n", brandID, err)
		return
	}

	if exists {
		return
	}

	// Find when the next run should be.
	// We'll check the latest post time.
	history, err := s.Store.GetHistory(brandID)
	var nextRunDelay time.Duration

	if err == nil && len(history) > 0 {
		lastPost := history[0].CreatedAt
		nextRunAt := lastPost.Add(time.Duration(intervalHours) * time.Hour)
		nextRunDelay = time.Until(nextRunAt)
		if nextRunDelay < 0 {
			nextRunDelay = 10 * time.Second // Run soon if overdue
		}
	} else {
		// First time run
		nextRunDelay = 1 * time.Minute
	}

	fmt.Printf("⏰ Scheduling next run for brand %s in %v\n", brandID, nextRunDelay)
	s.Queue.Enqueue(brandID, JobTypeRun, nextRunDelay, "")
}
