package agent

import (
	"content-creator-agent/memory"
	"content-creator-agent/models"
	"content-creator-agent/tools"
	"fmt"
	"strings"
	"time"
)

// Agent represents the autonomous content creator brain.
type Agent struct {
	Brand     models.BrandProfile
	Search    tools.SearchTool
	LLM       tools.LLMTool
	Social    tools.SocialClient
	Store     memory.Store
	Vector    memory.VectorStore
	Embedding tools.EmbeddingTool
	Analytics tools.AnalyticsFetcher
}

func NewAgent(brand models.BrandProfile, search tools.SearchTool, llm tools.LLMTool, social tools.SocialClient, store memory.Store, vector memory.VectorStore, embedding tools.EmbeddingTool, analytics tools.AnalyticsFetcher) *Agent {
	return &Agent{
		Brand:     brand,
		Search:    search,
		LLM:       llm,
		Social:    social,
		Store:     store,
		Vector:    vector,
		Embedding: embedding,
		Analytics: analytics,
	}
}

// Run executes one full cycle of the agent loop.
func (a *Agent) Run() error {
	fmt.Printf("Starting autonomous loop for brand: %s\n", a.Brand.Name)

	// 1. Research
	fmt.Println("Step 1: Researching latest trends...")
	query := fmt.Sprintf("latest trends in %s", a.Brand.Industry)
	trends, err := a.Search.Search(query)
	if err != nil {
		return fmt.Errorf("research failed: %w", err)
	}

	// 2. Planning
	fmt.Println("Step 2: Planning content strategy...")
	plan, err := a.Plan(trends)
	if err != nil {
		return fmt.Errorf("planning failed: %w", err)
	}
	fmt.Printf("Selected Topic: %s\n", plan)

	// 3. Generation & Evaluation Loop
	fmt.Println("Step 3: Generating and refining content...")
	var finalPost *models.Post
	for i := 0; i < 3; i++ { // Allow up to 3 iterations
		draft, err := a.Generate(plan)
		if err != nil {
			return err
		}

		critique, score, err := a.Evaluate(draft)
		if err != nil {
			return err
		}

		fmt.Printf("Draft Iteration %d (Score: %d/10)\n", i+1, score)
		if score >= 8 {
			finalPost = &models.Post{
				ID:        fmt.Sprintf("post-%d", time.Now().Unix()),
				BrandID:   a.Brand.ID,
				Topic:     plan,
				Content:   draft,
				Platform:  "LinkedIn/X",
				Status:    models.StatusApproved,
				CreatedAt: time.Now(),
			}
			break
		}
		fmt.Printf("Feedback: %s\n", critique)
	}

	if finalPost == nil {
		return fmt.Errorf("failed to generate satisfactory content after 3 attempts")
	}

	// 4. Posting
	fmt.Println("Step 4: Publishing...")
	if err := a.Social.Post(finalPost); err != nil {
		return fmt.Errorf("posting failed: %w", err)
	}

	// 5. Memory
	fmt.Println("Step 5: Saving to long-term memory...")
	if err := a.Store.SavePost(*finalPost); err != nil {
		return fmt.Errorf("memory storage failed: %w", err)
	}

	// 5b. Vector Memory
	if a.Embedding != nil && a.Vector != nil {
		fmt.Println("Step 5b: Generating embeddings and indexing post...")
		embedding, err := a.Embedding.Embed(finalPost.Content)
		if err == nil {
			a.Vector.Add(memory.VectorRecord{
				ID:     finalPost.ID,
				Vector: embedding,
				Metadata: map[string]interface{}{
					"topic":   finalPost.Topic,
					"content": finalPost.Content,
					"brand":   a.Brand.ID,
				},
			})
		} else {
			fmt.Printf("Warning: Failed to create embedding: %v\n", err)
		}
	}

	fmt.Println("Autonomous cycle completed successfully!")
	return nil
}

// PlanBatch researches and generates a series of posts to be scheduled for the future.
func (a *Agent) PlanBatch(postCount int) error {
	fmt.Printf("🎯 Planning batch of %d posts for brand: %s\n", postCount, a.Brand.Name)

	// 1. Research
	fmt.Println("Step 1: Researching latest trends for batch...")
	query := fmt.Sprintf("latest trends in %s", a.Brand.Industry)
	trends, err := a.Search.Search(query)
	if err != nil {
		return fmt.Errorf("research failed: %w", err)
	}

	// 2. Generate multiple plans
	var topics []string
	for i := 0; i < postCount; i++ {
		topic, err := a.Plan(trends)
		if err != nil {
			return err
		}
		topics = append(topics, topic)
		fmt.Printf("Planned topic %d: %s\n", i+1, topic)
	}

	// 3. For each topic, generate and schedule
	for i, topic := range topics {
		fmt.Printf("Step 3.%d: Generating content for: %s\n", i+1, topic)

		var draft string
		var score int
		for retry := 0; retry < 3; retry++ {
			draft, err = a.Generate(topic)
			if err != nil {
				return err
			}
			_, score, err = a.Evaluate(draft)
			if err != nil {
				return err
			}
			if score >= 7 {
				break
			}
		}

		// Schedule them evenly over the next week (simplified logic)
		scheduleTime := time.Now().Add(time.Duration((i+1)*24) * time.Hour)

		sp := models.ScheduledPost{
			ID:          fmt.Sprintf("sp-%d-%d", time.Now().Unix(), i),
			BrandID:     a.Brand.ID,
			Topic:       topic,
			Content:     draft,
			Platform:    "LinkedIn/X",
			Status:      models.StatusPending,
			ScheduledAt: scheduleTime,
			CreatedAt:   time.Now(),
		}

		if err := a.Store.SaveScheduledPost(sp); err != nil {
			fmt.Printf("Warning: Failed to save scheduled post: %v\n", err)
		} else {
			fmt.Printf("✅ Scheduled post %d for %v\n", i+1, scheduleTime.Format(time.RFC822))
		}
	}

	return nil
}

// PublishScheduledPost takes a previously planned post and pushes it to social media.
func (a *Agent) PublishScheduledPost(sp models.ScheduledPost) error {
	fmt.Printf("🚀 Publishing scheduled post: %s\n", sp.ID)

	post := models.Post{
		ID:        fmt.Sprintf("p-%d", time.Now().Unix()),
		BrandID:   sp.BrandID,
		Topic:     sp.Topic,
		Content:   sp.Content,
		Platform:  sp.Platform,
		Status:    models.StatusPublished,
		CreatedAt: time.Now(),
	}

	if err := a.Social.Post(&post); err != nil {
		return err
	}

	// Update status and save to history
	if err := a.Store.SavePost(post); err != nil {
		return err
	}

	return a.Store.UpdateScheduledPostStatus(sp.ID, models.StatusPublished)
}

// Plan uses the LLM to select the best trend.
func (a *Agent) Plan(trends []models.Trend) (string, error) {
	var trendList []string
	for _, t := range trends {
		trendList = append(trendList, fmt.Sprintf("- %s: %s", t.Title, t.Snippet))
	}

	history, _ := a.Store.GetHistory(a.Brand.ID)
	var pastTopics []string
	for _, p := range history {
		pastTopics = append(pastTopics, p.Topic)
	}

	// 2b. Semantic context
	var semanticContext string
	if a.Embedding != nil && a.Vector != nil {
		queryEmbed, err := a.Embedding.Embed(fmt.Sprintf("content about %s in %s industry", a.Brand.Name, a.Brand.Industry))
		if err == nil {
			matches, _ := a.Vector.Query(queryEmbed, 3)
			if len(matches) > 0 {
				var contexts []string
				for _, m := range matches {
					contexts = append(contexts, fmt.Sprintf("- Past Topic: %s", m.Metadata["topic"]))
				}
				semanticContext = "\nRelevant semantic memories from past successes:\n" + strings.Join(contexts, "\n")
			}
		}
	}

	systemPrompt := "You are a content strategist."
	userPrompt := fmt.Sprintf(`Based on the following trends in %s, select ONE topic to write a high-relevance post about. 
Trends:
%s

Past topics we covered: %s
%s
Avoid duplicating recent topics. Highlight why this topic is trending. Output ONLY the topic title.`,
		a.Brand.Industry, strings.Join(trendList, "\n"), strings.Join(pastTopics, ", "), semanticContext)

	return a.LLM.Generate(systemPrompt, userPrompt)
}

// Generate creates the content draft.
func (a *Agent) Generate(topic string) (string, error) {
	systemPrompt := fmt.Sprintf("You are the Content Creator for %s. Your brand voice is: %s. Your audience is %s.",
		a.Brand.Name, a.Brand.Voice, a.Brand.TargetAudience)

	userPrompt := fmt.Sprintf("Write a professional and engaging social media post (approx 150 words) about: %s. Include relevant hashtags.", topic)

	return a.LLM.Generate(systemPrompt, userPrompt)
}

// Evaluate provides a critique and score.
func (a *Agent) Evaluate(content string) (string, int, error) {
	systemPrompt := "You are a Brand Quality Critic. Your job is to ensure content matches brand voice and quality."
	userPrompt := fmt.Sprintf(`Evaluate the following post for brand: %s. 
Voice requirement: %s
Target Audience: %s

Post Content:
"%s"

Provide a critique and a score from 1 to 10. Format: "Critique: [text] Score: [number]"`,
		a.Brand.Name, a.Brand.Voice, a.Brand.TargetAudience, content)

	response, err := a.LLM.Generate(systemPrompt, userPrompt)
	if err != nil {
		return "", 0, err
	}

	// Simple heuristic to extract score
	score := 7 // Default if parsing fails
	if strings.Contains(response, "Score:") {
		fmt.Sscanf(strings.Split(response, "Score:")[1], "%d", &score)
	}

	return response, score, nil
}

// SyncAnalytics fetches latest performance data for past posts and updates memory.
func (a *Agent) SyncAnalytics() error {
	if a.Analytics == nil {
		return fmt.Errorf("analytics fetcher not configured")
	}

	history, err := a.Store.GetHistory(a.Brand.ID)
	if err != nil {
		return err
	}

	fmt.Printf("Syncing analytics for %d posts...\n", len(history))
	for _, p := range history {
		if p.SocialID == "" {
			continue
		}

		fmt.Printf("Fetching metrics for post %s (%s)...\n", p.ID, p.Platform)
		metrics, err := a.Analytics.Fetch(&p)
		if err != nil {
			fmt.Printf("Warning: Failed to fetch metrics for %s: %v\n", p.ID, err)
			continue
		}

		// Update both history and vector metadata
		a.Store.UpdateAnalytics(a.Brand.ID, p.ID, metrics)
		if a.Vector != nil {
			a.Vector.UpdateMetadata(p.ID, map[string]interface{}{
				"likes":    metrics.Likes,
				"shares":   metrics.Shares,
				"comments": metrics.Comments,
				"score":    metrics.Likes + (metrics.Shares * 2), // Simple performance score
			})
		}
	}

	return nil
}

// Start runs the agent loop autonomously at the specified interval.
func (a *Agent) Start(interval time.Duration) {
	fmt.Printf("Agent started in daemon mode. Cycle interval: %v\n", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run once immediately
	a.runAndSync()

	for range ticker.C {
		a.runAndSync()
	}
}

func (a *Agent) runAndSync() {
	fmt.Printf("\n--- [%s] Starting Autonomous Cycle ---\n", time.Now().Format(time.RFC822))
	if err := a.Run(); err != nil {
		fmt.Printf("Cycle error: %v\n", err)
	}

	fmt.Printf("[%s] Syncing analytics...\n", time.Now().Format(time.RFC822))
	if err := a.SyncAnalytics(); err != nil {
		fmt.Printf("Sync error: %v\n", err)
	}
	fmt.Printf("--- [%s] Cycle Finished ---\n", time.Now().Format(time.RFC822))
}
