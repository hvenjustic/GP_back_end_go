package dto

// IDRequest 用于简单的按 ID 请求体
type IDRequest struct {
	ID uint64 `json:"id" binding:"required"`
}

// PreprocessResponse 返回预处理结果
type PreprocessResponse struct {
	Status        string `json:"status"`
	ProcessedMD   string `json:"processed_md,omitempty"`
	LLMDurationMs int64  `json:"llm_duration_ms,omitempty"`
}

// GraphBuildResponse 返回图谱抽取结果
type GraphBuildResponse struct {
	Status        string `json:"status"`
	GraphJSON     string `json:"graph_json,omitempty"`
	LLMDurationMs int64  `json:"llm_duration_ms,omitempty"`
}

// QueueAckResponse 统一的异步入队响应
type QueueAckResponse struct {
	Queued   int    `json:"queued"`
	QueueKey string `json:"queue_key"`
	Pending  int64  `json:"pending"`
}
