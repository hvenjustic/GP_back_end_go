package dto

// HealthStatusResponse 健康检查响应
type HealthStatusResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}
