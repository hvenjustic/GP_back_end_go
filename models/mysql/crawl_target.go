package mysql

import "time"

// CrawlTarget 爬取目标与结果汇总表：合并站点统计与页面结果（通过字段是否为空区分用途）
type CrawlTarget struct {
	ID uint64 `gorm:"column:id;primaryKey;autoIncrement;type:bigint" json:"id"`

	SiteName *string `gorm:"column:site_name;type:varchar(100);index:idx_site_name" json:"site_name"`
	URL      string  `gorm:"column:url;type:varchar(255);not null;uniqueIndex:uq_url" json:"url"`

	CrawledAt      *time.Time `gorm:"column:crawled_at;index:idx_crawled_at" json:"crawled_at"`
	LLMProcessedAt *time.Time `gorm:"column:llm_processed_at;index:idx_llm_processed_at" json:"llm_processed_at"`

	IsCrawled  bool `gorm:"column:is_crawled;not null;default:0;index:idx_is_crawled" json:"is_crawled"`
	CrawlCount int  `gorm:"column:crawl_count;not null;default:0" json:"crawl_count"`
	PageCount  int  `gorm:"column:page_count;not null;default:0" json:"page_count"`

	ChunkCount  int     `gorm:"column:chunk_count;not null;default:0" json:"chunk_count"`
	ResultMD    *string `gorm:"column:result_md;type:longtext" json:"result_md"`
	ProcessedMD *string `gorm:"column:processed_md;type:longtext" json:"processed_md"`
	GraphJSON   *string `gorm:"column:graph_json;type:longtext" json:"graph_json"`

	// 额外：按需求保留耗时（毫秒），前端列表暂不展示
	CrawlDurationMs int64 `gorm:"column:crawl_duration_ms;not null;default:0" json:"crawl_duration_ms"`
	LLMDurationMs   int64 `gorm:"column:llm_duration_ms;not null;default:0" json:"llm_duration_ms"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime;index:idx_updated_at" json:"updated_at"`
}

func (CrawlTarget) TableName() string {
	return "crawl_targets"
}
