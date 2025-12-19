package dto

// DeepCrawlOptions 控制深度爬取的选项
type DeepCrawlOptions struct {
	MaxDepth   int  `json:"max_depth,omitempty"`
	MaxPages   int  `json:"max_pages,omitempty"`
	SameDomain bool `json:"same_domain,omitempty"`
}

// DeepCrawlRequest 同步深度爬取请求
type DeepCrawlRequest struct {
	URLs    []string          `json:"urls"`
	Options *DeepCrawlOptions `json:"options,omitempty"`
}

// CrawlJobPayload 异步任务入队请求体
type CrawlJobPayload struct {
	URLs          []string `json:"urls"`
	Priority      *int     `json:"priority,omitempty"`
	BrowserConfig any      `json:"browser_config,omitempty"`
	CrawlerConfig any      `json:"crawler_config,omitempty"`
	WebhookConfig any      `json:"webhook_config,omitempty"`
}

// CrawlJobEnqueueResponse 异步任务入队响应
type CrawlJobEnqueueResponse struct {
	TaskID  string `json:"task_id"`
	Message string `json:"message,omitempty"`
}

// CrawlJobStatusResponse 爬取任务状态/结果响应
type CrawlJobStatusResponse struct {
	TaskID  string     `json:"task_id"`
	Status  string     `json:"status"`
	Message string     `json:"message,omitempty"`
	Result  *JobResult `json:"result,omitempty"`
}

// JobResult 爬取结果
type JobResult struct {
	Success bool            `json:"success"`
	Results []JobPageResult `json:"results,omitempty"`
}

// JobPageResult 单页结果
type JobPageResult struct {
	URL          string `json:"url,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
	Markdown     any    `json:"markdown,omitempty"` // string 或 MarkdownGenerationResult
}

// MarkdownGenerationResult markdown 生成结果
type MarkdownGenerationResult struct {
	RawMarkdown        string `json:"raw_markdown,omitempty"`
	ReferencesMarkdown string `json:"references_markdown,omitempty"`
}
