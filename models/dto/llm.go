package dto

import "encoding/json"

// Message 消息结构
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionRequest Chat Completions请求 (已更新为与V2一致)
type ChatCompletionRequest struct {
	GptType           int                    `json:"gpt_type"`
	Messages          []Message              `json:"messages"`
	MaxTokens         int                    `json:"max_tokens,omitempty"`
	Temperature       float64                `json:"temperature,omitempty"`
	TopP              float64                `json:"top_p,omitempty"`
	TopK              int                    `json:"top_k,omitempty"`
	MinP              float64                `json:"min_p,omitempty"`
	RepetitionPenalty float64                `json:"repetition_penalty,omitempty"`
	Stop              []string               `json:"stop,omitempty"`
	Stream            bool                   `json:"stream,omitempty"`
	N                 int                    `json:"n,omitempty"`
	Seed              int                    `json:"seed,omitempty"`
	StreamOptions     map[string]interface{} `json:"stream_options,omitempty"`
}

// ChatCompletionResponse Chat Completions响应 (已更新为与V2一致)
type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	GptType int      `json:"gpt_type"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice 选择项
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage 使用情况 (已更新为与V2一致)
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// TokenDetails 提示词token详情
type TokenDetails struct {
	Tokens         int `json:"tokens"`
	AcceptedTokens int `json:"accepted_tokens"`
	RejectedTokens int `json:"rejected_tokens"`
}

// CompletionTokenDetails 生成内容token详情
type CompletionTokenDetails struct {
	Tokens                   int `json:"tokens"`
	AcceptedTokens           int `json:"accepted_tokens"`
	RejectedTokens           int `json:"rejected_tokens"`
	AcceptedPredictionTokens int `json:"accepted_prediction_tokens"`
	RejectedPredictionTokens int `json:"rejected_prediction_tokens"`
}

// InferenceInvokeRequest 模型调用请求
type InferenceInvokeRequest struct {
	OperationID     string          `json:"operation_id"`
	Source          string          `json:"source"`
	GptType         int             `json:"gpt_type"`
	Prompt          string          `json:"prompt"`
	Platform        string          `json:"platform"`
	AppId           int8            `json:"app_id"`
	InferenceParams InferenceParams `json:"inference_params"`
}

func (iir InferenceInvokeRequest) MarshalJSON() ([]byte, error) {
	// 创建一个可以容纳基础字段以及参数的map
	type Alias InferenceInvokeRequest // 避免无限递归调用MarshalJSON
	temp := map[string]any{
		"operation_id": iir.OperationID,
		"source":       iir.Source,
		"gpt_type":     iir.GptType,
		"prompt":       iir.Prompt,
		"platform":     iir.Platform,
		"app_id":       iir.AppId,
	}
	// 将InferenceParams中的键值添加到map中
	for k, v := range iir.InferenceParams {
		temp[k] = v
	}
	return json.Marshal(temp)
}

// InferenceInvokeResp 模型调用响应
type InferenceInvokeResp struct {
	ErrCode  int            `json:"err_code"`
	ErrMsg   string         `json:"err_msg"`
	GptType  int            `json:"gpt_type"`
	RespList []GenerateText `json:"resp_list"`
	Usage    CustomUsage    `json:"usage"`
}

// GenerateText 生成文本
type GenerateText struct {
	Text            string  `json:"text"`
	RemoveInputText string  `json:"remove_input_text"`
	Score           float64 `json:"score"`
	FinishReason    string  `json:"finish_reason"`
	IsFilter        bool    `json:"is_filter"`
}

// InferenceParams 推理参数
type InferenceParams map[string]any

// CustomUsage 使用统计
type CustomUsage struct {
	PromptTokens        int     `json:"prompt_tokens"`
	CompletionTokens    int     `json:"completion_tokens"`
	TotalTokens         int     `json:"total_tokens"`
	CostTime            int64   `json:"cost_time"`
	SpentCredits        float64 `json:"spent_credits"`
	InputTokenList      []int   `json:"input_token_list"`
	CompletionTokenList []int   `json:"completion_token_list"`
}

// Dialog 对话结构
type Dialog struct {
	Content string `json:"content"`
	Role    bool   `json:"role"` // user-true bot-false
}

// PromptMetaData Prompt元数据
type PromptMetaData struct {
	BotPersona     string    `json:"bot_persona"`
	ExampleDialogs *[]Dialog `json:"example_dialogs"`
	RealDialogs    *[]Dialog `json:"real_dialogs"`
	Username       string    `json:"username"`
	Description    string    `json:"description"`
	BotName        string    `json:"bot_name"`
	UserGender     string    `json:"user_gender"`
	BotGender      string    `json:"bot_gender"`
	BotAge         int64     `json:"bot_age"`
	UserAge        int64     `json:"user_age"`
	BotCareer      string    `json:"bot_career"`
	AppId          int8      `json:"app_id"`
	Memory         string    `json:"memory"`
}

// PersonaData 角色数据
type PersonaData struct {
	Ge           int    `json:"ge"`
	Le           int    `json:"le"`
	Head         string `json:"head"`
	Brief        string `json:"brief"`
	Fact         string `json:"fact"`
	HiddenSecret string `json:"hidden_secret"`
}

// ChatCompletionMessage Chat Completion消息
type ChatCompletionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// InternalFilterReq 内部过滤检查请求
type InternalFilterReq struct {
	Content     string `json:"content"`
	Source      string `json:"source"`
	OperationID string `json:"operation_id"`
}

// InternalFilterResp 内部过滤检查响应
type InternalFilterResp struct {
	Res bool `json:"res"`
}
