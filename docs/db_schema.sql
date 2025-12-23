-- 统一爬取目标与结果汇总表：合并站点统计与页面结果（通过字段是否为空区分用途）
CREATE TABLE IF NOT EXISTS crawl_targets (
  id BIGINT AUTO_INCREMENT COMMENT '主键，自增',

  site_name VARCHAR(100) NULL COMMENT '网站名称/站点别名（便于展示与统计）',
  url VARCHAR(2048) NOT NULL COMMENT '目标 URL（站点或页面）',

  -- 爬取/处理时间
  crawled_at DATETIME NULL COMMENT '实际爬取完成时间（可为空，未爬取/失败时为空）',
  llm_processed_at DATETIME NULL COMMENT 'LLM 处理完成时间（可为空，未处理/失败时为空）',

  -- 站点/任务维度的状态与统计
  is_crawled TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否已完成爬取：1=是，0=否',
  crawl_count INT NOT NULL DEFAULT 0 COMMENT '累计爬取次数（该 URL 被爬取的总次数）',
  page_count INT NOT NULL DEFAULT 0 COMMENT '页面数量统计（站点：站点下页面数；页面：相关分页/渲染页数，按业务约定）',

  -- 页面结果维度的数据（站点记录通常为空）
  chunk_count INT NOT NULL DEFAULT 0 COMMENT '内容分块数量（用于分段/向量化等）',
  result_md LONGTEXT NULL COMMENT '抓取到的 Markdown 结果内容（页面为主，可为空）',
  processed_md LONGTEXT NULL COMMENT 'LLM 预处理后的 Markdown（合并文件的 URL）',
  graph_json LONGTEXT NULL COMMENT '从内容抽取的图谱 JSON（可为空）',

  -- 处理耗时（毫秒）
  crawl_duration_ms BIGINT NOT NULL DEFAULT 0 COMMENT '爬取时长（毫秒）',
  llm_duration_ms BIGINT NOT NULL DEFAULT 0 COMMENT 'LLM 处理时长（毫秒）',

  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',

  PRIMARY KEY (id),
  UNIQUE KEY uq_url (url),
  KEY idx_site_name (site_name),
  KEY idx_url_prefix (url(255)),
  KEY idx_is_crawled (is_crawled),
  KEY idx_crawled_at (crawled_at),
  KEY idx_llm_processed_at (llm_processed_at),
  KEY idx_updated_at (updated_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
  COMMENT='爬取目标与结果汇总表：合并站点统计与页面结果（通过字段是否为空区分用途）';

