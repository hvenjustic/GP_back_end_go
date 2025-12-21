package utils

import (
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"time"
)

// StructToJsonString 将结构体转换为JSON字符串
func StructToJsonString(obj interface{}) string {
	bytes, err := json.Marshal(obj)
	if err != nil {
		return ""
	}
	return string(bytes)
}

// ParseTimeFlexible 解析多种时间格式，支持RFC3339与"2006-01-02 15:04:05"
func ParseTimeFlexible(input *string) (*time.Time, error) {
	if input == nil {
		return nil, nil
	}
	text := strings.TrimSpace(*input)
	if text == "" {
		return nil, nil
	}
	if t, err := time.Parse(time.RFC3339, text); err == nil {
		return &t, nil
	}
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", text, time.Local); err == nil {
		return &t, nil
	}
	return nil, errors.New("invalid time format")
}

// DeriveSiteName 根据URL推导站点名（host）
func DeriveSiteName(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	if u.Host != "" {
		return u.Host
	}
	return ""
}
