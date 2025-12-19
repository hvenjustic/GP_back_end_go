package crawl4ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"resty.dev/v3"

	"image-api/pkg/log"
)

const defaultBaseURL = "http://43.139.166.203:11235"

type Client struct {
	baseURL string
	resty   *resty.Client
}

type DeepCrawlOptions struct {
	MaxDepth   int  `json:"max_depth,omitempty"`
	MaxPages   int  `json:"max_pages,omitempty"`
	SameDomain bool `json:"same_domain,omitempty"`
}

type DeepCrawlRequest struct {
	URLs    []string          `json:"urls"`
	Options *DeepCrawlOptions `json:"options,omitempty"`
}

func NewClient(baseURL string, timeoutSeconds int) *Client {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")
	if timeoutSeconds <= 0 {
		timeoutSeconds = 30
	}
	r := resty.New().
		SetBaseURL(baseURL).
		SetTimeout(time.Duration(timeoutSeconds) * time.Second)
	return &Client{baseURL: baseURL, resty: r}
}

// DeepCrawl 使用 crawl4ai 的 /crawl 同步接口（历史逻辑保留，但项目新流程应使用异步 Job Queue）。
func (c *Client) DeepCrawl(ctx context.Context, req DeepCrawlRequest) ([]byte, error) {
	if len(req.URLs) == 0 {
		return nil, fmt.Errorf("urls empty")
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	log.Info("Crawl4AI", "deepcrawl_request", string(body))
	resp, err := c.resty.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetBody(req).
		Post("/crawl")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() >= 300 {
		return nil, fmt.Errorf("crawl4ai status %d: %s", resp.StatusCode(), resp.String())
	}
	return resp.Bytes(), nil
}

type CrawlJobPayload struct {
	URLs          []string `json:"urls"`
	Priority      *int     `json:"priority,omitempty"`
	BrowserConfig any      `json:"browser_config,omitempty"`
	CrawlerConfig any      `json:"crawler_config,omitempty"`
	WebhookConfig any      `json:"webhook_config,omitempty"`
}

type CrawlJobEnqueueResponse struct {
	TaskID  string `json:"task_id"`
	Message string `json:"message,omitempty"`
}

type CrawlJobStatusResponse struct {
	TaskID  string     `json:"task_id"`
	Status  string     `json:"status"`
	Message string     `json:"message,omitempty"`
	Result  *JobResult `json:"result,omitempty"`
}

type JobResult struct {
	Success bool            `json:"success"`
	Results []JobPageResult `json:"results,omitempty"`
}

type JobPageResult struct {
	URL          string `json:"url,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`

	// markdown 可能是 string 或对象（MarkdownGenerationResult）
	Markdown any `json:"markdown,omitempty"`
}

type MarkdownGenerationResult struct {
	RawMarkdown        string `json:"raw_markdown,omitempty"`
	ReferencesMarkdown string `json:"references_markdown,omitempty"`
}

// EnqueueCrawlJob 提交异步任务：POST /crawl/job
func (c *Client) EnqueueCrawlJob(ctx context.Context, payload CrawlJobPayload) (string, error) {
	if len(payload.URLs) == 0 {
		return "", fmt.Errorf("urls empty")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	log.Info("Crawl4AI", "enqueue_job_request", string(body))
	resp, err := c.resty.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetBody(payload).
		Post("/crawl/job")
	if err != nil {
		return "", err
	}
	if resp.StatusCode() >= 300 {
		return "", fmt.Errorf("crawl4ai status %d: %s", resp.StatusCode(), resp.String())
	}

	respBody := resp.Bytes()
	var out CrawlJobEnqueueResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return "", err
	}
	out.TaskID = strings.TrimSpace(out.TaskID)
	if out.TaskID == "" {
		return "", fmt.Errorf("crawl4ai missing task_id: %s", string(respBody))
	}
	return out.TaskID, nil
}

// GetCrawlJob 查询任务状态/获取结果。
// 兼容不同镜像版本：优先 /crawl/job/{task_id}，再尝试 /job/{task_id} 与 /task/{task_id}。
func (c *Client) GetCrawlJob(ctx context.Context, taskID string) (*CrawlJobStatusResponse, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, fmt.Errorf("task_id empty")
	}

	candidates := []string{
		c.baseURL + "/crawl/job/" + taskID,
		c.baseURL + "/job/" + taskID,
		c.baseURL + "/task/" + taskID,
	}
	var lastErr error
	for _, u := range candidates {
		out, err := c.getJobOnce(ctx, u)
		if err == nil {
			return out, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func (c *Client) getJobOnce(ctx context.Context, url string) (*CrawlJobStatusResponse, error) {
	resp, err := c.resty.R().
		SetContext(ctx).
		Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() >= 300 {
		return nil, fmt.Errorf("crawl4ai status %d: %s", resp.StatusCode(), resp.String())
	}
	var out CrawlJobStatusResponse
	if err := json.Unmarshal(resp.Bytes(), &out); err != nil {
		return nil, err
	}
	return &out, nil
}
