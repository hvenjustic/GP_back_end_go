package mysql

import (
	"GP_back_end_go/models/mysql"
	"GP_back_end_go/pkg/db"
	"time"
)

type UserApiKeyDAO struct{}

func NewUserApiKeyDAO() *UserApiKeyDAO {
	return &UserApiKeyDAO{}
}

// GetByApiKey 根据API Key获取记录
func (dao *UserApiKeyDAO) GetByApiKey(apiKey string) (*mysql.UserApiKey, error) {
	var userApiKey mysql.UserApiKey
	err := db.DB.MysqlDB.DB().Where("api_key = ? AND deleted_at IS NULL", apiKey).First(&userApiKey).Error
	if err != nil {
		return nil, err
	}
	return &userApiKey, nil
}

// IsApiKeyValid 检查API Key是否有效（存在且未过期）
func (dao *UserApiKeyDAO) IsApiKeyValid(apiKey string) (bool, error) {
	userApiKey, err := dao.GetByApiKey(apiKey)
	if err != nil {
		return false, err
	}

	// 检查是否过期
	currentTime := uint64(time.Now().UnixMilli())
	if userApiKey.ExpireTime > 0 && currentTime > userApiKey.ExpireTime {
		return false, nil
	}

	return true, nil
}
