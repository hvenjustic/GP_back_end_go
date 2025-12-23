package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"resty.dev/v3"

	"GP_back_end_go/pkg/config"
	"GP_back_end_go/pkg/log"
)

// ChatLLMClient 统一的 Chat Completions 客户端
type ChatLLMClient struct {
	endpoint string
	model    string
	apiKey   string
	resty    *resty.Client
}

// NewChatLLMClient 根据配置构造客户端
func NewChatLLMClient(cfg config.LLMConfig) *ChatLLMClient {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 60
	}
	client := resty.New().SetTimeout(time.Duration(timeout) * time.Second)
	return &ChatLLMClient{
		endpoint: strings.TrimSpace(cfg.BaseURL),
		model:    strings.TrimSpace(cfg.Model),
		apiKey:   strings.TrimSpace(cfg.APIKey),
		resty:    client,
	}
}

// Chat 发送单轮对话并返回模型内容
func (c *ChatLLMClient) Chat(ctx context.Context, content string) (string, error) {
	if c == nil {
		return "", fmt.Errorf("llm client is nil")
	}
	endpoint := strings.TrimSpace(c.endpoint)
	if endpoint == "" {
		return "", fmt.Errorf("llm endpoint empty")
	}
	if strings.TrimSpace(content) == "" {
		return "", fmt.Errorf("prompt is empty")
	}

	body := map[string]any{
		"model": c.model,
		"messages": []map[string]string{
			{"role": "user", "content": content},
		},
	}
	req := c.resty.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetBody(body)
	if c.apiKey != "" {
		req.SetHeader("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := req.Post(endpoint)
	if err != nil {
		log.Error("LLM", "request_failed", err.Error(), "endpoint", endpoint, "model", c.model)
		return "", err
	}
	if resp.StatusCode() >= 300 {
		log.Error("LLM", "status_not_ok", fmt.Sprintf("%d", resp.StatusCode()), "endpoint", endpoint, "model", c.model, "resp", resp.String())
		return "", fmt.Errorf("llm status %d: %s", resp.StatusCode(), resp.String())
	}

	bodyText := resp.String()
	if bodyText == "" && resp.Body != nil {
		if b, err := io.ReadAll(resp.Body); err == nil {
			bodyText = string(b)
		}
	}

	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal([]byte(bodyText), &out); err != nil {
		log.Error("LLM", "unmarshal_failed", err.Error(), "endpoint", endpoint, "model", c.model, "body", truncate(bodyText, 200))
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("llm response empty")
	}

	text := strings.TrimSpace(out.Choices[0].Message.Content)
	text = stripCodeFence(text)
	if text == "" {
		return "", fmt.Errorf("llm response content empty")
	}
	log.Info("LLM", "chat_success", "endpoint", endpoint, "model", c.model, "resp_len", len(text))
	return text, nil
}

// stripCodeFence 去除 ```json 包裹的内容
func stripCodeFence(text string) string {
	t := strings.TrimSpace(text)
	if strings.HasPrefix(t, "```") {
		t = strings.TrimPrefix(t, "```json")
		t = strings.TrimPrefix(t, "```")
		t = strings.TrimSpace(t)
		if idx := strings.LastIndex(t, "```"); idx >= 0 {
			t = t[:idx]
		}
	}
	return strings.TrimSpace(t)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
