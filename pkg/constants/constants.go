package constants

// LLM请求状态
const (
	LLMRequestStatusProcessing = 0 // 处理中
	LLMRequestStatusSuccess    = 1 // 成功
	LLMRequestStatusFailed     = 2 // 失败
)

// 响应码
const (
	CodeSuccess      = 200
	CodeBadRequest   = 400
	CodeUnauthorized = 401
	CodeForbidden    = 403
	CodeNotFound     = 404
	CodeServerError  = 500
)

// 缓存键前缀
const (
	CachePrefixLLMResponse = "llm:response:"
	CachePrefixUserCount   = "llm:user:count:"
	CachePrefixModelConfig = "llm:model:config:"
)

// 默认配置
const (
	DefaultMaxTokens   = 2048
	DefaultTemperature = 0.7
	DefaultCacheExpire = 3600 // 1小时
	DefaultRateLimit   = 100  // 每小时100次请求
)

// Story模型类型
const (
	Model133GptType = 133 // Gemini-2.5-pro
	Model137GptType = 137 // Grok-3
	Model163GptType = 163 // grok-4-fast-reasoning
	Model164GptType = 164 // grok-4-fast-None-reasoning
)

// Story模型类型列表
var StoryGptTypes = []int{
	Model133GptType, // Gemini-2.5-pro
	Model137GptType, // Grok-3
	Model163GptType, // grok-4-fast-reasoning
	Model164GptType, // grok-4-fast-None-reasoning
}

// 消息模式
type MessageMode int

const (
	MMStandard      MessageMode = iota // 标准模式
	MMStory                            // Story模式
	MMStoryEconomy                     // Story经济模式
	MMStoryLoveMode                    // Story爱情模式
)

// 新的GPT类型（对应mode）
const (
	Model1GptType = 1 // 对应MMStory
	Model2GptType = 2 // 对应MMStoryEconomy
	Model3GptType = 3 // 对应MMStoryLoveMode
)

// 新接入的GPT类型映射
const (
	Model8001GptType = 8001 // 映射到1
	Model8101GptType = 8101 // 映射到78
	Model8102GptType = 8102 // 映射到133
	Model8201GptType = 8201 // 映射到137
	Model8202GptType = 8202 // 映射到163
	Model8203GptType = 8203 // 映射到164
)

// GPT类型映射表
var GptTypeMapping = map[int]int{
	Model8001GptType: 1,               // 8001 -> 1
	Model8101GptType: 78,              // 8101 -> 78
	Model8102GptType: Model133GptType, // 8102 -> 133
	Model8201GptType: Model137GptType, // 8201 -> 137
	Model8202GptType: Model163GptType, // 8202 -> 163
	Model8203GptType: Model164GptType, // 8203 -> 164
}

// 新接入的GPT类型列表
var NewGptTypes = []int{
	Model8001GptType,
	Model8101GptType,
	Model8102GptType,
	Model8201GptType,
	Model8202GptType,
	Model8203GptType,
}
