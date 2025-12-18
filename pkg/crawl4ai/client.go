package crawl4ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultBaseURL = "http://43.139.166.203:11235"

// Client 封装 crawl4ai 调用
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// DeepCrawlOptions 深度爬取配置
type DeepCrawlOptions struct {
	MaxDepth   int  `json:"max_depth,omitempty"`
	MaxPages   int  `json:"max_pages,omitempty"`
	SameDomain bool `json:"same_domain,omitempty"`
}

// DeepCrawlRequest 深度爬取请求
type DeepCrawlRequest struct {
	URLs    []string          `json:"urls"`
	Options *DeepCrawlOptions `json:"options,omitempty"`
}

// NewClient 初始化客户端
func NewClient(baseURL string, timeoutSeconds int) *Client {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")
	if timeoutSeconds <= 0 {
		timeoutSeconds = 30
	}
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: time.Duration(timeoutSeconds) * time.Second,
		},
	}
}

// DeepCrawl 使用 crawl4ai 的 /crawl 深度爬取接口
func (c *Client) DeepCrawl(ctx context.Context, req DeepCrawlRequest) ([]byte, error) {
	if len(req.URLs) == 0 {
		return nil, fmt.Errorf("urls empty")
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/crawl", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("crawl4ai status %d: %s", resp.StatusCode, string(respBody))
	}
	return respBody, nil
}
