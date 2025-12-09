package mysql

// UserApiKey 用户API Key表
type UserApiKey struct {
	BaseModel
	ApiKey     string `gorm:"column:api_key" json:"api_key"`
	ExpireTime uint64 `gorm:"column:expire_time" json:"expire_time"`
}

// TableName 指定表名
func (UserApiKey) TableName() string {
	return "user_api_keys"
}
