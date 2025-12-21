package dto

import "GP_back_end_go/models/mysql"

// SubmitTaskItem 单个爬取任务参数
type SubmitTaskItem struct {
	URL      string `json:"url"`
	SiteName string `json:"site_name,omitempty"`
	MaxDepth *int   `json:"max_depth,omitempty"`
	MaxPages *int   `json:"max_pages,omitempty"`
}

// SubmitTasksRequest 任务提交请求
type SubmitTasksRequest struct {
	URLs     []SubmitTaskItem `json:"urls"`
	MaxDepth *int             `json:"max_depth,omitempty"`
	MaxPages *int             `json:"max_pages,omitempty"`
}

// SubmitTasksResponse 任务提交响应
type SubmitTasksResponse struct {
	Queued   int    `json:"queued"`
	QueueKey string `json:"queue_key"`
	Pending  int64  `json:"pending"`
}

// StatusResponse 队列状态响应
type StatusResponse struct {
	Pending  int64  `json:"pending"`
	QueueKey string `json:"queue_key"`
}

// ClearQueueRequest 清空指定队列请求
type ClearQueueRequest struct {
	QueueName string `json:"queue_name" binding:"required"`
}

// ClearQueueResponse 清空指定队列响应
type ClearQueueResponse struct {
	QueueName   string `json:"queue_name"`
	RemovedKeys int64  `json:"removed_keys"`
}

// TaskResultCallbackRequest Python回传爬虫结果
type TaskResultCallbackRequest struct {
	ID  uint64 `json:"id,omitempty"`
	URL string `json:"url"`

	SiteName string `json:"site_name,omitempty"`

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

// TaskResultResponse 回传结果响应
type TaskResultResponse struct {
	Status string             `json:"status"`
	Data   *mysql.CrawlTarget `json:"data,omitempty"`
}

// ListResultsResponse 结果列表响应
type ListResultsResponse struct {
	Items    []mysql.CrawlTarget `json:"items"`
	Total    int64               `json:"total"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"page_size"`
}
