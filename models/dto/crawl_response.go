package dto

import "GP_back_end_go/models/mysql"

// CrawlResultDetailResponse 单条结果详情响应
type CrawlResultDetailResponse struct {
	Item *mysql.CrawlTarget `json:"item"`
}
