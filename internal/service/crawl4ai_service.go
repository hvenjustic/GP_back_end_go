package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"resty.dev/v3"

	"image-api/models/dto"
	"image-api/pkg/constants"
	"image-api/pkg/log"
)

type Crawl4AIClient struct {
	baseURL string
	resty   *resty.Client
}

func NewCrawl4AIClient(baseURL string, timeoutSeconds int) *Crawl4AIClient {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = constants.DefaultCrawl4AIBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")
	if timeoutSeconds <= 0 {
		timeoutSeconds = 30
	}
	r := resty.New().
		SetBaseURL(baseURL).
		SetTimeout(time.Duration(timeoutSeconds) * time.Second)
	return &Crawl4AIClient{baseURL: baseURL, resty: r}
}

// DeepCrawl 使用 crawl4ai 的 /crawl 同步接口（历史逻辑保留，但项目新流程应使用异步 Job Queue）。
func (c *Crawl4AIClient) DeepCrawl(ctx context.Context, req dto.DeepCrawlRequest) ([]byte, error) {
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

// EnqueueCrawlJob 提交异步任务：POST /crawl/job
func (c *Crawl4AIClient) EnqueueCrawlJob(ctx context.Context, payload dto.CrawlJobPayload) (string, error) {
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
	var out dto.CrawlJobEnqueueResponse
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
func (c *Crawl4AIClient) GetCrawlJob(ctx context.Context, taskID string) (*dto.CrawlJobStatusResponse, error) {
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

func (c *Crawl4AIClient) getJobOnce(ctx context.Context, url string) (*dto.CrawlJobStatusResponse, error) {
	resp, err := c.resty.R().
		SetContext(ctx).
		Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() >= 300 {
		return nil, fmt.Errorf("crawl4ai status %d: %s", resp.StatusCode(), resp.String())
	}
	var out dto.CrawlJobStatusResponse
	if err := json.Unmarshal(resp.Bytes(), &out); err != nil {
		return nil, err
	}
	return &out, nil
}
