package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"image-api/internal/mysql"
	"image-api/models/dto"
	"image-api/pkg/config"
	"image-api/pkg/constants"
	"image-api/pkg/db"
	"image-api/pkg/log"

	go_redis "github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

type QueueItem struct {
	ID       uint64 `json:"id"`
	URL      string `json:"url"`
	SiteName string `json:"site_name,omitempty"`
	MaxDepth *int   `json:"max_depth,omitempty"`
	MaxPages *int   `json:"max_pages,omitempty"`
}

type CrawlScheduler struct {
	queueKey     string
	activeSetKey string
	taskKeyPref  string

	maxConcurrent int64
	pollInterval  time.Duration

	client *Crawl4AIClient
	dao    *mysql.CrawlTargetDAO
}

func NewCrawlScheduler() *CrawlScheduler {
	maxConcurrent := int64(3)
	pollInterval := 10 * time.Second
	if config.Config.Crawl4AI.DefaultMaxDepth <= 0 {
		config.Config.Crawl4AI.DefaultMaxDepth = 2
	}
	if config.Config.Crawl4AI.DefaultMaxPages <= 0 {
		config.Config.Crawl4AI.DefaultMaxPages = 10
	}
	if config.Config.Crawl4AI.TimeoutSeconds <= 0 {
		config.Config.Crawl4AI.TimeoutSeconds = 30
	}
	return &CrawlScheduler{
		queueKey:      constants.CrawlQueueKey,
		activeSetKey:  constants.CrawlActiveSetKey,
		taskKeyPref:   constants.CrawlTaskKeyPref,
		maxConcurrent: maxConcurrent,
		pollInterval:  pollInterval,
		client:        NewCrawl4AIClient(config.Config.Crawl4AI.BaseURL, config.Config.Crawl4AI.TimeoutSeconds),
		dao:           mysql.NewCrawlTargetDAO(),
	}
}

func (s *CrawlScheduler) Start(ctx context.Context) {
	t := time.NewTicker(s.pollInterval)
	defer t.Stop()

	_ = s.Tick(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			_ = s.Tick(ctx)
		}
	}
}

func (s *CrawlScheduler) Tick(ctx context.Context) error {
	if err := s.pollActive(ctx); err != nil {
		log.Error("CrawlScheduler", "pollActive failed", err.Error())
	}
	if err := s.startNew(ctx); err != nil {
		log.Error("CrawlScheduler", "startNew failed", err.Error())
	}
	return nil
}

func (s *CrawlScheduler) startNew(ctx context.Context) error {
	if db.DB.RDB == nil {
		return fmt.Errorf("redis not initialized")
	}
	defaultMaxPages := config.Config.Crawl4AI.DefaultMaxPages
	if defaultMaxPages <= 0 {
		defaultMaxPages = 10
	}
	for {
		activeCount, err := db.DB.RDB.SCard(ctx, s.activeSetKey).Result()
		if err != nil {
			return err
		}
		if activeCount >= s.maxConcurrent {
			return nil
		}

		raw, err := db.DB.RDB.LPop(ctx, s.queueKey).Result()
		if err == go_redis.Nil {
			return nil
		}
		if err != nil {
			return err
		}

		item, err := parseQueueItem(raw)
		if err != nil {
			log.Error("CrawlScheduler", "drop invalid queue item", err.Error(), "raw", raw)
			continue
		}

		maxDepth := config.Config.Crawl4AI.DefaultMaxDepth
		if item.MaxDepth != nil && *item.MaxDepth > 0 {
			maxDepth = *item.MaxDepth
		}
		maxPages := defaultMaxPages
		if item.MaxPages != nil && *item.MaxPages > 0 {
			maxPages = *item.MaxPages
		}

		crawlerParams := map[string]any{
			"include_external":       false,
			"exclude_external_links": true,
		}
		if maxDepth > 0 {
			crawlerParams["max_depth"] = maxDepth
		}
		if maxPages > 0 {
			crawlerParams["max_pages"] = maxPages
		}
		crawlerCfg := map[string]any{
			"type":   "CrawlerRunConfig",
			"params": crawlerParams,
		}

		taskID, err := s.client.EnqueueCrawlJob(ctx, dto.CrawlJobPayload{
			URLs:          []string{item.URL},
			CrawlerConfig: crawlerCfg,
		})
		if err != nil {
			_ = db.DB.RDB.RPush(ctx, s.queueKey, raw).Err()
			return err
		}

		taskKey := s.taskKey(taskID)
		now := time.Now().UTC().Format(time.RFC3339Nano)
		if _, err := db.DB.RDB.Pipelined(ctx, func(p go_redis.Pipeliner) error {
			p.SAdd(ctx, s.activeSetKey, taskID)
			p.HSet(ctx, taskKey, map[string]any{
				"task_id":     taskID,
				"target_id":   item.ID,
				"url":         item.URL,
				"site_name":   item.SiteName,
				"status":      "enqueued",
				"enqueued_at": now,
			})
			return nil
		}); err != nil {
			return err
		}

		log.Info("CrawlScheduler", "enqueued", "task_id", taskID, "url", item.URL)
	}
}

func (s *CrawlScheduler) pollActive(ctx context.Context) error {
	if db.DB.RDB == nil {
		return fmt.Errorf("redis not initialized")
	}

	taskIDs, err := db.DB.RDB.SMembers(ctx, s.activeSetKey).Result()
	if err != nil {
		return err
	}
	for _, taskID := range taskIDs {
		taskID = strings.TrimSpace(taskID)
		if taskID == "" {
			continue
		}

		job, err := s.client.GetCrawlJob(ctx, taskID)
		if err != nil {
			log.Error("CrawlScheduler", "get job failed", err.Error(), "task_id", taskID)
			continue
		}
		if raw, err := json.Marshal(job); err != nil {
			log.Error("CrawlScheduler", "marshal job resp failed", err.Error(), "task_id", taskID)
		} else {
			log.Info("CrawlScheduler", "poll progress", "task_id", taskID, "resp", string(raw))
		}

		status := strings.ToLower(strings.TrimSpace(job.Status))
		if status == "" {
			status = "unknown"
		}

		taskKey := s.taskKey(taskID)
		_ = db.DB.RDB.HSet(ctx, taskKey, map[string]any{
			"status":         status,
			"last_polled_at": time.Now().UTC().Format(time.RFC3339Nano),
		}).Err()

		switch status {
		case "completed":
			targetID, urlStr, err := s.loadTaskMeta(ctx, taskKey)
			if err != nil {
				log.Error("CrawlScheduler", "loadTaskMeta failed", err.Error(), "task_id", taskID)
			}
			md, pageCount := extractMarkdown(job)
			now := time.Now()

			if targetID > 0 {
				patch := map[string]any{
					"updated_at":  now,
					"is_crawled":  true,
					"crawled_at":  now,
					"page_count":  pageCount,
					"result_md":   md,
					"crawl_count": gorm.Expr("crawl_count + 1"),
				}
				if _, err := s.dao.ApplyResultByIDOrURL(&targetID, "", patch); err != nil {
					log.Error("CrawlScheduler", "apply result failed", err.Error(), "task_id", taskID, "target_id", targetID)
				}
			} else if urlStr != "" {
				patch := map[string]any{
					"updated_at":  now,
					"is_crawled":  true,
					"crawled_at":  now,
					"page_count":  pageCount,
					"result_md":   md,
					"crawl_count": gorm.Expr("crawl_count + 1"),
				}
				if _, err := s.dao.ApplyResultByIDOrURL(nil, urlStr, patch); err != nil {
					log.Error("CrawlScheduler", "apply result failed", err.Error(), "task_id", taskID, "url", urlStr)
				}
			}

			_, _ = db.DB.RDB.Pipelined(ctx, func(p go_redis.Pipeliner) error {
				p.SRem(ctx, s.activeSetKey, taskID)
				p.Del(ctx, taskKey)
				return nil
			})
			log.Info("CrawlScheduler", "completed", "task_id", taskID, "page_count", pageCount, "md_len", len(md))

		case "failed", "error":
			targetID, urlStr, _ := s.loadTaskMeta(ctx, taskKey)
			now := time.Now()
			patch := map[string]any{
				"updated_at":  now,
				"crawl_count": gorm.Expr("crawl_count + 1"),
			}
			if targetID > 0 {
				_, _ = s.dao.ApplyResultByIDOrURL(&targetID, "", patch)
			} else if urlStr != "" {
				_, _ = s.dao.ApplyResultByIDOrURL(nil, urlStr, patch)
			}

			_, _ = db.DB.RDB.Pipelined(ctx, func(p go_redis.Pipeliner) error {
				p.SRem(ctx, s.activeSetKey, taskID)
				p.Del(ctx, taskKey)
				return nil
			})
			log.Info("CrawlScheduler", "failed", "task_id", taskID)
		default:
		}
	}
	return nil
}

func (s *CrawlScheduler) taskKey(taskID string) string {
	return s.taskKeyPref + taskID
}

func parseQueueItem(raw string) (QueueItem, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return QueueItem{}, fmt.Errorf("empty")
	}
	var item QueueItem
	if strings.HasPrefix(raw, "{") {
		if err := json.Unmarshal([]byte(raw), &item); err != nil {
			return QueueItem{}, err
		}
	} else {
		item.URL = raw
	}
	item.URL = strings.TrimSpace(item.URL)
	if item.URL == "" {
		return QueueItem{}, fmt.Errorf("missing url")
	}
	return item, nil
}

func extractMarkdown(job *dto.CrawlJobStatusResponse) (md string, pageCount int) {
	if job == nil || job.Result == nil || len(job.Result.Results) == 0 {
		return "", 0
	}
	pageCount = len(job.Result.Results)

	var b strings.Builder
	for _, r := range job.Result.Results {
		u := strings.TrimSpace(r.URL)
		if u != "" {
			b.WriteString("## ")
			b.WriteString(u)
			b.WriteString("\n\n")
		}

		switch v := r.Markdown.(type) {
		case string:
			v = strings.TrimSpace(v)
			if v != "" {
				b.WriteString(v)
				if !strings.HasSuffix(v, "\n") {
					b.WriteString("\n")
				}
				b.WriteString("\n")
			}
		case map[string]any:
			raw, _ := v["raw_markdown"].(string)
			raw = strings.TrimSpace(raw)
			if raw != "" {
				b.WriteString(raw)
				if !strings.HasSuffix(raw, "\n") {
					b.WriteString("\n")
				}
				b.WriteString("\n")
			}
			refs, _ := v["references_markdown"].(string)
			refs = strings.TrimSpace(refs)
			if refs != "" {
				b.WriteString(refs)
				if !strings.HasSuffix(refs, "\n") {
					b.WriteString("\n")
				}
				b.WriteString("\n")
			}
		default:
		}
	}
	return strings.TrimSpace(b.String()), pageCount
}

func (s *CrawlScheduler) loadTaskMeta(ctx context.Context, taskKey string) (targetID uint64, url string, err error) {
	m, err := db.DB.RDB.HGetAll(ctx, taskKey).Result()
	if err != nil {
		return 0, "", err
	}
	url = strings.TrimSpace(m["url"])
	if v := strings.TrimSpace(m["target_id"]); v != "" {
		var n uint64
		_, scanErr := fmt.Sscanf(v, "%d", &n)
		if scanErr == nil {
			targetID = n
		}
	}
	return targetID, url, nil
}
