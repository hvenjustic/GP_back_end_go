package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
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
	log.Info("CrawlScheduler", "start scheduler loop", "poll_interval", s.pollInterval, "max_concurrent", s.maxConcurrent)
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
	log.Info("CrawlScheduler", "tick begin")
	if err := s.pollActive(ctx); err != nil {
		log.Error("CrawlScheduler", "pollActive failed", err.Error())
	}
	if err := s.startNew(ctx); err != nil {
		log.Error("CrawlScheduler", "startNew failed", err.Error())
	}
	log.Info("CrawlScheduler", "tick end")
	return nil
}

func (s *CrawlScheduler) startNew(ctx context.Context) error {
	if db.DB.RDB == nil {
		return fmt.Errorf("redis not initialized")
	}
	log.Info("CrawlScheduler", "startNew check", "queue_key", s.queueKey, "active_set_key", s.activeSetKey)
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

		payload := s.buildCrawlJobPayload(item, maxDepth, maxPages)
		taskID, err := s.client.EnqueueCrawlJob(ctx, payload)
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
	log.Info("CrawlScheduler", "pollActive check", "active_set_key", s.activeSetKey)

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
			errMsg := strings.TrimSpace(job.Message)
			pageErr := ""
			if job.Result != nil {
				for _, r := range job.Result.Results {
					if e := strings.TrimSpace(r.ErrorMessage); e != "" {
						pageErr = e
						break
					}
				}
			}
			log.Info("CrawlScheduler", "failed detail", "task_id", taskID, "message", errMsg, "page_error", pageErr)

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
	for idx, r := range job.Result.Results {
		u := strings.TrimSpace(r.URL)
		if u != "" {
			b.WriteString("## ")
			b.WriteString(u)
			b.WriteString("\n\n")
		}

		content, refs := pickMarkdownContent(r.Markdown)
		if content != "" {
			b.WriteString(content)
			if !strings.HasSuffix(content, "\n") {
				b.WriteString("\n")
			}
			b.WriteString("\n")
		}
		if refs != "" {
			b.WriteString(refs)
			if !strings.HasSuffix(refs, "\n") {
				b.WriteString("\n")
			}
			b.WriteString("\n")
		}
		b.WriteString("---")
		if idx < pageCount-1 {
			b.WriteString("\n\n")
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

func (s *CrawlScheduler) buildCrawlJobPayload(item QueueItem, maxDepth, maxPages int) dto.CrawlJobPayload {
	domain := deriveAllowedDomain(item.URL)

	filters := []any{
		map[string]any{
			"type": "ContentTypeFilter",
			"params": map[string]any{
				"allowed_types": []string{"text/html"},
			},
		},
	}
	if domain != "" {
		filters = append([]any{
			map[string]any{
				"type": "DomainFilter",
				"params": map[string]any{
					"allowed_domains": []string{domain},
				},
			},
		}, filters...)
	}

	filterChain := map[string]any{
		"type": "FilterChain",
		"params": map[string]any{
			"filters": filters,
		},
	}

	deepCrawlStrategy := map[string]any{
		"type": "BFSDeepCrawlStrategy",
		"params": map[string]any{
			"max_depth":        maxDepth,
			"max_pages":        maxPages,
			"include_external": false,
			"filter_chain":     filterChain,
		},
	}

	return dto.CrawlJobPayload{
		URLs: []string{item.URL},
		BrowserConfig: map[string]any{
			"headless": true,
		},
		CrawlerConfig: map[string]any{
			"exclude_external_links": true,
			"deep_crawl_strategy":    deepCrawlStrategy,
		},
		ExtractorConfig: map[string]any{
			"type": "MarkdownExtractor",
		},
	}
}

func deriveAllowedDomain(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	host := strings.TrimSpace(parsed.Hostname())
	return host
}

func pickMarkdownContent(md any) (content string, refs string) {
	switch v := md.(type) {
	case string:
		content = strings.TrimSpace(v)
	case map[string]any:
		fit := pickString(v, "fit_markdown")
		raw := pickString(v, "raw_markdown")
		withCitations := pickString(v, "markdown_with_citations")
		refs = pickString(v, "references_markdown")
		content = firstNonEmpty(fit, raw, withCitations)
	default:
		// 支持 json.RawMessage 等类型：尝试反序列化为通用 map
		b, err := json.Marshal(v)
		if err != nil {
			return "", ""
		}
		var m map[string]any
		if err := json.Unmarshal(b, &m); err != nil {
			return "", ""
		}
		return pickMarkdownContent(m)
	}
	return content, refs
}

func pickString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func firstNonEmpty(items ...string) string {
	for _, v := range items {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
