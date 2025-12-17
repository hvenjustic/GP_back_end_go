package crawl

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"image-api/internal/mysql"
	"image-api/pkg/db"
	"image-api/pkg/log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const defaultQueueKey = "crawl_tasks"

type TaskItem struct {
	URL      string `json:"url"`
	SiteName string `json:"site_name,omitempty"`
	MaxDepth *int   `json:"max_depth,omitempty"`
	MaxPages *int   `json:"max_pages,omitempty"`
}

type SubmitTasksRequest struct {
	URLs     []TaskItem `json:"urls"`
	MaxDepth *int       `json:"max_depth,omitempty"`
	MaxPages *int       `json:"max_pages,omitempty"`
}

type SubmitTasksResponse struct {
	Queued   int    `json:"queued"`
	QueueKey string `json:"queue_key"`
	Pending  int64  `json:"pending"`
}

type StatusResponse struct {
	Pending  int64  `json:"pending"`
	QueueKey string `json:"queue_key"`
}

type TaskResultCallbackRequest struct {
	ID uint64 `json:"id,omitempty"`

	SiteName string `json:"site_name,omitempty"`
	URL      string `json:"url"`

	CrawledAt      *string `json:"crawled_at,omitempty"`       // RFC3339 或 "2006-01-02 15:04:05"
	LLMProcessedAt *string `json:"llm_processed_at,omitempty"` // RFC3339 或 "2006-01-02 15:04:05"

	IsCrawled  *bool `json:"is_crawled,omitempty"`
	PageCount  *int  `json:"page_count,omitempty"`
	ChunkCount *int  `json:"chunk_count,omitempty"`

	ResultMD  *string `json:"result_md,omitempty"`
	GraphJSON *string `json:"graph_json,omitempty"`

	CrawlDurationMs *int64 `json:"crawl_duration_ms,omitempty"`
	LLMDurationMs   *int64 `json:"llm_duration_ms,omitempty"`
}

func queueKey() string {
	return defaultQueueKey
}

func parseTimeFlexible(input *string) (*time.Time, error) {
	if input == nil {
		return nil, nil
	}
	text := strings.TrimSpace(*input)
	if text == "" {
		return nil, nil
	}
	if t, err := time.Parse(time.RFC3339, text); err == nil {
		return &t, nil
	}
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", text, time.Local); err == nil {
		return &t, nil
	}
	return nil, errors.New("invalid time format")
}

func deriveSiteName(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	if u.Host != "" {
		return u.Host
	}
	return ""
}

func SubmitTasks(c *gin.Context) {
	var req SubmitTasksRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.URLs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "urls empty"})
		return
	}

	ctx := context.Background()
	dao := mysql.NewCrawlTargetDAO()

	queued := 0
	for _, item := range req.URLs {
		rawURL := strings.TrimSpace(item.URL)
		if rawURL == "" {
			continue
		}
		siteName := strings.TrimSpace(item.SiteName)
		if siteName == "" {
			siteName = deriveSiteName(rawURL)
		}

		target, err := dao.UpsertForSubmission(rawURL, siteName)
		if err != nil {
			log.Error("SubmitTasks", "upsert failed", err.Error(), "url", rawURL)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		maxDepth := req.MaxDepth
		if item.MaxDepth != nil {
			maxDepth = item.MaxDepth
		}
		maxPages := req.MaxPages
		if item.MaxPages != nil {
			maxPages = item.MaxPages
		}

		payload := map[string]any{
			"id":        target.ID,
			"url":       target.URL,
			"site_name": target.SiteName,
		}
		if maxDepth != nil {
			payload["max_depth"] = *maxDepth
		}
		if maxPages != nil {
			payload["max_pages"] = *maxPages
		}
		b, _ := json.Marshal(payload)
		if err := db.DB.RDB.RPush(ctx, queueKey(), string(b)).Err(); err != nil {
			log.Error("SubmitTasks", "rpush failed", err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		queued++
	}

	pending, err := db.DB.RDB.LLen(ctx, queueKey()).Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, SubmitTasksResponse{
		Queued:   queued,
		QueueKey: queueKey(),
		Pending:  pending,
	})
}

func GetStatus(c *gin.Context) {
	ctx := context.Background()
	pending, err := db.DB.RDB.LLen(ctx, queueKey()).Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, StatusResponse{Pending: pending, QueueKey: queueKey()})
}

func PostTaskResult(c *gin.Context) {
	var req TaskResultCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.URL = strings.TrimSpace(req.URL)
	if req.ID == 0 && req.URL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id/url empty"})
		return
	}

	crawledAt, err := parseTimeFlexible(req.CrawledAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "crawled_at invalid"})
		return
	}
	llmProcessedAt, err := parseTimeFlexible(req.LLMProcessedAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "llm_processed_at invalid"})
		return
	}

	now := time.Now()
	isCrawled := true
	if req.IsCrawled != nil {
		isCrawled = *req.IsCrawled
	}
	if isCrawled && crawledAt == nil {
		crawledAt = &now
	}

	patch := map[string]any{
		"updated_at": now,
	}
	if req.SiteName != "" {
		patch["site_name"] = req.SiteName
	}
	if req.URL != "" {
		patch["url"] = req.URL
	}
	patch["is_crawled"] = isCrawled
	if isCrawled && crawledAt != nil {
		patch["crawled_at"] = *crawledAt
	}
	if llmProcessedAt != nil {
		patch["llm_processed_at"] = *llmProcessedAt
	}
	if req.PageCount != nil {
		patch["page_count"] = *req.PageCount
	}
	if req.ChunkCount != nil {
		patch["chunk_count"] = *req.ChunkCount
	}
	if req.ResultMD != nil {
		patch["result_md"] = *req.ResultMD
	}
	if req.GraphJSON != nil {
		patch["graph_json"] = *req.GraphJSON
	}
	if req.CrawlDurationMs != nil {
		patch["crawl_duration_ms"] = *req.CrawlDurationMs
	}
	if req.LLMDurationMs != nil {
		patch["llm_duration_ms"] = *req.LLMDurationMs
	}

	// 回传即视为一次尝试，累加 crawl_count
	patch["crawl_count"] = gorm.Expr("crawl_count + 1")

	dao := mysql.NewCrawlTargetDAO()
	var idPtr *uint64
	if req.ID > 0 {
		idPtr = &req.ID
	}
	out, err := dao.ApplyResultByIDOrURL(idPtr, req.URL, patch)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "data": out})
}

func ListResults(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	dao := mysql.NewCrawlTargetDAO()
	items, total, err := dao.List(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	c.JSON(http.StatusOK, gin.H{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func GetResultDetail(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalid"})
		return
	}
	dao := mysql.NewCrawlTargetDAO()
	out, err := dao.GetDetailByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}
