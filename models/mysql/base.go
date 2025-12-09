package mysql

import (
	"time"

	"gorm.io/gorm"
)

// BaseModel 基础模型
type BaseModel struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// LLMRequest LLM请求记录
type LLMRequest struct {
	BaseModel
	UserID       string  `gorm:"type:varchar(100);index" json:"user_id"`
	Model        string  `gorm:"type:varchar(100)" json:"model"`
	Messages     string  `gorm:"type:text" json:"messages"`
	Temperature  float64 `json:"temperature"`
	MaxTokens    int     `json:"max_tokens"`
	Stream       bool    `json:"stream"`
	RequestID    string  `gorm:"type:varchar(100);index" json:"request_id"`
	Status       int     `gorm:"default:0" json:"status"` // 0:处理中 1:成功 2:失败
	ErrorMessage string  `gorm:"type:text" json:"error_message,omitempty"`
	ResponseData string  `gorm:"type:text" json:"response_data,omitempty"`
	CostTime     int64   `json:"cost_time"` // 耗时(毫秒)
}

// TableName 指定表名
func (LLMRequest) TableName() string {
	return "llm_requests"
}
