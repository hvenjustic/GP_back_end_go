package crawl

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"image-api/internal/mysql"
	"image-api/models/dto"
	model "image-api/models/mysql"
	"image-api/pkg/config"
	"image-api/pkg/crawl4ai"
	"image-api/pkg/db"
	"image-api/pkg/log"
	"image-api/pkg/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const defaultQueueKey = "crawl4ai"

func queueKey() string {
	return defaultQueueKey
}

func SubmitTasks(c *gin.Context) {
	var req dto.SubmitTasksRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.URLs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "urls empty"})
		return
	}

	dao := mysql.NewCrawlTargetDAO()
	client := crawl4ai.NewClient(config.Config.Crawl4AI.BaseURL, config.Config.Crawl4AI.TimeoutSeconds)
	defaultDepth := config.Config.Crawl4AI.DefaultMaxDepth
	success := 0
	failures := 0

	for _, item := range req.URLs {
		rawURL := strings.TrimSpace(item.URL)
		if rawURL == "" {
			failures++
			continue
		}
		parsed, err := url.Parse(rawURL)
		if err != nil || parsed.Host == "" {
			failures++
			continue
		}

		siteName := strings.TrimSpace(item.SiteName)
		if siteName == "" {
			siteName = utils.DeriveSiteName(rawURL)
		}

		target, err := dao.UpsertForSubmission(rawURL, siteName)
		if err != nil {
			log.Error("SubmitTasks", "upsert failed", err.Error(), "url", rawURL)
			failures++
			continue
		}

		maxDepth := req.MaxDepth
		if item.MaxDepth != nil {
			maxDepth = item.MaxDepth
		}
		depthVal := defaultDepth
		if maxDepth != nil {
			depthVal = *maxDepth
		}
		if depthVal <= 0 {
			depthVal = 2
		}
		maxPages := 0
		if req.MaxPages != nil {
			maxPages = *req.MaxPages
		}
		if item.MaxPages != nil {
			maxPages = *item.MaxPages
		}

		respBody, err := client.DeepCrawl(c.Request.Context(), crawl4ai.DeepCrawlRequest{
			URLs: []string{target.URL},
			Options: &crawl4ai.DeepCrawlOptions{
				MaxDepth:   depthVal,
				MaxPages:   maxPages,
				SameDomain: true,
			},
		})
		if err != nil {
			log.Error("SubmitTasks", "crawl4ai failed", err.Error(), "url", target.URL)
			failures++
			continue
		}

		now := time.Now()
		patch := map[string]any{
			"updated_at": now,
			"is_crawled": true,
			"crawled_at": now,
			"result_md":  string(respBody),
		}
		if _, err := dao.ApplyResultByIDOrURL(&target.ID, target.URL, patch); err != nil {
			log.Error("SubmitTasks", "apply result failed", err.Error(), "url", target.URL)
			failures++
			continue
		}
		success++
	}

	c.JSON(http.StatusOK, dto.SubmitTasksResponse{
		Queued:   success,
		QueueKey: queueKey(),
		Pending:  0,
	})
}

func GetStatus(c *gin.Context) {
	var pending int64
	if err := db.DB.MysqlDB.ReadDB().Model(&model.CrawlTarget{}).Where("is_crawled = ?", false).Count(&pending).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, dto.StatusResponse{Pending: pending, QueueKey: queueKey()})
}

func PostTaskResult(c *gin.Context) {
	var req dto.TaskResultCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.URL = strings.TrimSpace(req.URL)
	if req.ID == 0 && req.URL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id/url empty"})
		return
	}

	crawledAt, err := utils.ParseTimeFlexible(req.CrawledAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "crawled_at invalid"})
		return
	}
	llmProcessedAt, err := utils.ParseTimeFlexible(req.LLMProcessedAt)
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
	c.JSON(http.StatusOK, dto.TaskResultResponse{
		Status: "ok",
		Data:   out,
	})
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
	c.JSON(http.StatusOK, dto.ListResultsResponse{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
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
