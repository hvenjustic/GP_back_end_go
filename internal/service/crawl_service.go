package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"GP_back_end_go/internal/mysql"
	"GP_back_end_go/models/dto"
	"GP_back_end_go/pkg/config"
	"GP_back_end_go/pkg/constants"
	"GP_back_end_go/pkg/db"
	"GP_back_end_go/pkg/log"
	"GP_back_end_go/pkg/utils"

	"gorm.io/gorm"
)

var (
	// ErrBadRequest 用于标记请求参数非法
	ErrBadRequest = errors.New("bad request")
	// ErrRedisUnavailable 表示 Redis 未初始化
	ErrRedisUnavailable = errors.New("redis not initialized")
)

// SubmitCrawlTasks 提交爬虫任务入队
func SubmitCrawlTasks(ctx context.Context, req dto.SubmitTasksRequest) (dto.SubmitTasksResponse, error) {
	if len(req.URLs) == 0 {
		return dto.SubmitTasksResponse{}, fmt.Errorf("%w: urls empty", ErrBadRequest)
	}
	if db.DB.RDB == nil {
		return dto.SubmitTasksResponse{}, fmt.Errorf("%w", ErrRedisUnavailable)
	}

	dao := mysql.NewCrawlTargetDAO()
	defaultDepth := config.Config.Crawl4AI.DefaultMaxDepth
	defaultPages := config.Config.Crawl4AI.DefaultMaxPages
	if defaultPages <= 0 {
		defaultPages = 10
	}

	success := 0
	for _, item := range req.URLs {
		rawURL := strings.TrimSpace(item.URL)
		if rawURL == "" {
			continue
		}
		parsed, err := url.Parse(rawURL)
		if err != nil || parsed.Host == "" {
			continue
		}

		siteName := strings.TrimSpace(item.SiteName)
		if siteName == "" {
			siteName = utils.DeriveSiteName(rawURL)
		}

		target, err := dao.UpsertForSubmission(rawURL, siteName)
		if err != nil {
			log.Error("SubmitCrawlTasks", "upsert failed", err.Error(), "url", rawURL)
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

		maxPages := defaultPages
		if req.MaxPages != nil {
			maxPages = *req.MaxPages
		}
		if item.MaxPages != nil {
			maxPages = *item.MaxPages
		}
		if maxPages <= 0 {
			maxPages = defaultPages
		}

		queueItem := map[string]any{
			"id":        target.ID,
			"url":       target.URL,
			"site_name": siteName,
			"max_depth": depthVal,
			"max_pages": maxPages,
		}
		b, _ := json.Marshal(queueItem)
		if err := db.DB.RDB.RPush(ctx, constants.CrawlQueueKey, string(b)).Err(); err != nil {
			log.Error("SubmitCrawlTasks", "enqueue failed", err.Error(), "url", target.URL)
			continue
		}
		success++
	}

	pending, _ := db.DB.RDB.LLen(ctx, constants.CrawlQueueKey).Result()
	return dto.SubmitTasksResponse{
		Queued:   success,
		QueueKey: constants.CrawlQueueKey,
		Pending:  pending,
	}, nil
}

// GetCrawlStatus 获取任务队列状态
func GetCrawlStatus(ctx context.Context) (dto.StatusResponse, error) {
	if db.DB.RDB == nil {
		return dto.StatusResponse{}, ErrRedisUnavailable
	}
	pendingQueue, err := db.DB.RDB.LLen(ctx, constants.CrawlQueueKey).Result()
	if err != nil {
		return dto.StatusResponse{}, err
	}
	activeCount, err := db.DB.RDB.SCard(ctx, constants.CrawlActiveSetKey).Result()
	if err != nil {
		return dto.StatusResponse{}, err
	}
	pending := pendingQueue + activeCount
	return dto.StatusResponse{
		Pending:  pending,
		QueueKey: constants.CrawlQueueKey,
	}, nil
}

// ClearQueue 清空指定的 Redis 队列/集合
func ClearQueue(ctx context.Context, queueName string) (dto.ClearQueueResponse, error) {
	if queueName == "" {
		return dto.ClearQueueResponse{}, fmt.Errorf("%w: queue_name empty", ErrBadRequest)
	}
	if db.DB.RDB == nil {
		return dto.ClearQueueResponse{}, ErrRedisUnavailable
	}
	removed, err := db.DB.RDB.Del(ctx, queueName).Result()
	if err != nil {
		return dto.ClearQueueResponse{}, err
	}
	return dto.ClearQueueResponse{
		QueueName:   queueName,
		RemovedKeys: removed,
	}, nil
}

// ApplyTaskResult 处理爬虫结果回传
func ApplyTaskResult(ctx context.Context, req dto.TaskResultCallbackRequest) (dto.TaskResultResponse, error) {
	req.URL = strings.TrimSpace(req.URL)
	if req.ID == 0 && req.URL == "" {
		return dto.TaskResultResponse{}, fmt.Errorf("%w: id/url empty", ErrBadRequest)
	}

	crawledAt, err := utils.ParseTimeFlexible(req.CrawledAt)
	if err != nil {
		return dto.TaskResultResponse{}, fmt.Errorf("%w: crawled_at invalid", ErrBadRequest)
	}
	llmProcessedAt, err := utils.ParseTimeFlexible(req.LLMProcessedAt)
	if err != nil {
		return dto.TaskResultResponse{}, fmt.Errorf("%w: llm_processed_at invalid", ErrBadRequest)
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
		"is_crawled": isCrawled,
	}
	if req.SiteName != "" {
		patch["site_name"] = req.SiteName
	}
	if req.URL != "" {
		patch["url"] = req.URL
	}
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
	if req.ProcessedMD != nil {
		patch["processed_md"] = *req.ProcessedMD
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
		return dto.TaskResultResponse{}, err
	}
	return dto.TaskResultResponse{
		Status: "ok",
		Data:   out,
	}, nil
}

// ListCrawlResults 分页查询爬取结果
func ListCrawlResults(ctx context.Context, page, pageSize int) (dto.ListResultsResponse, error) {
	dao := mysql.NewCrawlTargetDAO()
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	items, total, err := dao.List(page, pageSize)
	if err != nil {
		return dto.ListResultsResponse{}, err
	}
	return dto.ListResultsResponse{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// GetResultDetail 查询单条爬取结果详情
func GetResultDetail(ctx context.Context, id uint64) (dto.CrawlResultDetailResponse, error) {
	if id == 0 {
		return dto.CrawlResultDetailResponse{}, fmt.Errorf("%w: id invalid", ErrBadRequest)
	}
	dao := mysql.NewCrawlTargetDAO()
	out, err := dao.GetDetailByID(id)
	if err != nil {
		return dto.CrawlResultDetailResponse{}, err
	}
	return dto.CrawlResultDetailResponse{
		Item: out,
	}, nil
}
