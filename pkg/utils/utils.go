package utils

import (
	"encoding/json"
)

// StructToJsonString 将结构体转换为JSON字符串
func StructToJsonString(obj interface{}) string {
	bytes, err := json.Marshal(obj)
	if err != nil {
		return ""
	}
	return string(bytes)
}
