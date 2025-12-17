# back_end_go API（爬虫任务）

默认端口：`5010`（见 `config/config-prod.yaml`）。

## 任务提交

- `POST /api/tasks`

请求体示例：

```json
{
  "urls": [
    { "url": "https://example.com", "max_depth": 2, "max_pages": 50 }
  ],
  "max_depth": 2,
  "max_pages": 50
}
```

响应示例：

```json
{
  "queued": 1,
  "queue_key": "crawl_tasks",
  "pending": 5
}
```

说明：
- `urls[*].site_name` 可选；未传则从 URL 自动推导（host）。
- 入队 payload 会包含 `id/url/site_name/max_depth/max_pages`，供 Python worker 消费。

## 队列状态

- `GET /api/tasks/status`

返回 Redis 队列长度：

```json
{ "pending": 5, "queue_key": "crawl_tasks" }
```

## Python 回传结果（入库）

- `POST /api/tasks/result`

请求体（字段可按需要补充）：
- `id` 或 `url` 二选一（建议带 `id`）
- `is_crawled`：成功/失败
- `result_md`、`graph_json`、`page_count`、`chunk_count`
- `crawled_at`、`llm_processed_at`（支持 RFC3339 或 `YYYY-MM-DD HH:MM:SS`）
- `crawl_duration_ms`、`llm_duration_ms`

## 结果列表（分页）

- `GET /api/results?page=1&page_size=20`

返回字段为前端列表所需的精简列。

## 结果详情

- `GET /api/results/:id`

直接返回该行的完整 JSON。

