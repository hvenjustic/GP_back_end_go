下面是一份**基于你刚才实际跑出来的服务行为**整理的「异步爬取（Job）请求 / 进度查询 / 响应结构体使用说明」Markdown 文档，你可以直接保存成 `crawl4ai_async_job_api.md` 用作在线文档。

---

````md
# Crawl4AI Docker 异步 Job API 使用文档（请求 / 进度查询 / 响应结构体）

> 本文档基于你当前服务（`http://43.139.166.203:11235`）的实际调用结果整理：
> - 提交异步任务：`POST /crawl/job`
> - 查询任务进度与结果：`GET /crawl/job/{task_id}`
>
> 重点：你的服务返回的 Markdown **在** `result.results[i].markdown`（对象），而不是 `extracted_content`。

---

## 1. 基本信息

### Base URL
- `http://43.139.166.203:11235`

### Content-Type
- `Content-Type: application/json`

---

## 2. API 列表

| 场景 | 方法 | 路径 |
|---|---|---|
| 提交异步爬取任务（Job） | POST | `/crawl/job` |
| 查询任务进度/结果 | GET | `/crawl/job/{task_id}` |
| 查看服务端 schema（你当前返回的是一个“示例模板”） | GET | `/schema` |

---

## 3. 提交任务：POST `/crawl/job`

### 3.1 请求体（Request Body）结构

你的服务对顶层字段（`urls / browser_config / crawler_config / extractor_config`）支持“直接 JSON 对象”写法；  
但像 `deep_crawl_strategy / filter_chain` 这种 **策略类字段**必须使用 `type + params` 的结构。

#### 顶层字段

| 字段 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| `urls` | string[] | ✅ | 起始 URL 列表（深度爬取会从这些起点扩展） |
| `browser_config` | object | ❌ | 浏览器配置（例如 `headless`） |
| `crawler_config` | object | ❌ | 爬虫运行配置（深度策略、是否剔除外链、JS 等） |
| `extractor_config` | object | ❌ | 抽取器配置（你这次用 `MarkdownExtractor`；但 Markdown 实际落在 `markdown` 字段，而非 `extracted_content`） |

---

### 3.2 站点内深度爬取（你现在跑通的标准写法）

下面这个就是你当前“可用且生效”的请求结构（**非流式**、**站点内**、**深度爬取**）：

```bash
curl -X POST http://43.139.166.203:11235/crawl/job \
  -H "Content-Type: application/json" \
  -d '{
    "urls": ["https://bio.whu.edu.cn/"],
    "browser_config": { "headless": true },
    "crawler_config": {
      "exclude_external_links": true,
      "deep_crawl_strategy": {
        "type": "BFSDeepCrawlStrategy",
        "params": {
          "max_depth": 3,
          "max_pages": 1000,
          "include_external": false,
          "filter_chain": {
            "type": "FilterChain",
            "params": {
              "filters": [
                { "type": "DomainFilter", "params": { "allowed_domains": ["bio.whu.edu.cn"] } },
                { "type": "ContentTypeFilter", "params": { "allowed_types": ["text/html"] } }
              ]
            }
          }
        }
      }
    },
    "extractor_config": { "type": "MarkdownExtractor" }
  }'
````

#### crawler_config 关键字段说明

| 字段                       | 类型                | 说明                            |
| ------------------------ | ----------------- | ----------------------------- |
| `exclude_external_links` | boolean           | 输出时剔除站外链接（减少噪音；不等于“是否跟随外链去爬”） |
| `deep_crawl_strategy`    | object (Strategy) | 启用深度爬取策略（不配置就只抓 `urls` 里的起始页） |

#### deep_crawl_strategy（BFSDeepCrawlStrategy）参数说明

| 参数                 | 类型                   | 说明                          |
| ------------------ | -------------------- | --------------------------- |
| `max_depth`        | number               | 最大爬取深度（从起始 URL 为 0 开始）      |
| `max_pages`        | number               | 最大页面数上限（防止无限扩展）             |
| `include_external` | boolean              | 是否允许跟随站外链接（站内爬取应设为 `false`） |
| `filter_chain`     | object (FilterChain) | 过滤器链，用于限制域名、路径、内容类型等        |

#### filter_chain（FilterChain）说明

FilterChain 的 `filters` 是一个数组，按顺序对候选 URL/资源进行过滤。

你当前用到的两种 Filter：

* `DomainFilter.allowed_domains`

  * 只允许列出的域名
  * 建议站内爬取一定配，作为“双保险”

* `ContentTypeFilter.allowed_types`

  * 只允许指定内容类型，例如 `text/html`（避免误抓 pdf、图片等二进制资源）

---

### 3.3 提交任务响应（Response）

#### 响应示例

```json
{ "task_id": "crawl_3959d6f8" }
```

字段说明：

| 字段        | 类型     | 说明             |
| --------- | ------ | -------------- |
| `task_id` | string | 任务 ID，用于后续进度查询 |

---

## 4. 进度查询：GET `/crawl/job/{task_id}`

### 4.1 请求

```bash
curl -X GET "http://43.139.166.203:11235/crawl/job/<task_id>"
```

---

### 4.2 响应结构（顶层 Envelope）

你实际返回的顶层结构（已脱敏/简化）类似：

```json
{
  "task_id": "crawl_5e3d3a58",
  "status": "completed",
  "created_at": "2025-12-19T18:26:13.515650",
  "url": "[\"https://bio.whu.edu.cn/\"]",
  "_links": {
    "self":    { "href": "http://43.139.166.203:11235//llm/crawl_5e3d3a58" },
    "refresh": { "href": "http://43.139.166.203:11235//llm/crawl_5e3d3a58" }
  },
  "result": {
    "success": true,
    "results": [ /* CrawlResult[] */ ]
  }
}
```

字段说明：

| 字段           | 类型     | 说明                                                           |
| ------------ | ------ | ------------------------------------------------------------ |
| `task_id`    | string | 任务 ID                                                        |
| `status`     | string | 任务状态（你已见到 `completed`；其它可能是 `queued/processing/failed` 等）    |
| `created_at` | string | 任务创建时间（ISO 时间戳）                                              |
| `url`        | string | **注意：你这个服务返回的是“字符串形式的 JSON 数组”**（例如 `"[\\"https://...\\"]"`） |
| `_links`     | object | 自描述链接（你当前给的是 `//llm/...`，可能用于服务内部或管理界面）                      |
| `result`     | object | 当 `status=completed` 时提供结果；失败时可能提供错误信息                       |

---

## 5. result 结构体（完成时）

```json
"result": {
  "success": true,
  "results": [ /* CrawlResult[] */ ]
}
```

| 字段        | 类型      | 说明                   |
| --------- | ------- | -------------------- |
| `success` | boolean | 任务是否整体成功             |
| `results` | array   | 每个页面一个 `CrawlResult` |

---

## 6. CrawlResult 结构体（你这台服务实际返回的字段集合）

你通过 `jq '.result.results[0] | keys'` 得到的字段如下（每个页面都可能包含这些字段；有些会是 null）：

* `cleaned_html`
* `console_messages`
* `dispatch_result`
* `downloaded_files`
* `error_message`
* `extracted_content`
* `fit_html`
* `html`
* `js_execution_result`
* `links`
* `markdown`
* `media`
* `metadata`
* `mhtml`
* `network_requests`
* `pdf`
* `redirected_url`
* `response_headers`
* `screenshot`
* `session_id`
* `ssl_certificate`
* `status_code`
* `success`
* `tables`
* `url`

### 6.1 最常用字段说明（建议优先看这几个）

| 字段                          | 类型          | 说明                                    |
| --------------------------- | ----------- | ------------------------------------- |
| `url`                       | string      | 当前页面 URL                              |
| `success`                   | boolean     | 当前页抓取是否成功                             |
| `status_code`               | number/null | HTTP 状态码                              |
| `error_message`             | string/null | 失败原因                                  |
| `metadata`                  | object      | 元信息（深度爬取时通常会含 depth/parent 等，具体以返回为准） |
| `links`                     | object      | 链接集合（internal/external 等）             |
| `markdown`                  | object      | **MarkdownGenerationResult（重点）**      |
| `html`                      | string      | 原始 HTML（很大）                           |
| `cleaned_html` / `fit_html` | string/null | 清洗/适配抽取的 HTML（也很大）                    |
| `extracted_content`         | any         | 你这次为 `null`（说明没走结构化抽取输出）              |

---

## 7. markdown 字段：MarkdownGenerationResult（你已验证是 object）

你已验证：

* `.markdown | type` 为 `"object"`
* `.markdown | keys` 为：

```json
[
  "fit_html",
  "fit_markdown",
  "markdown_with_citations",
  "raw_markdown",
  "references_markdown"
]
```

字段说明：

| 字段                        | 类型     | 推荐用途                             |
| ------------------------- | ------ | -------------------------------- |
| `raw_markdown`            | string | 最“完整”的 Markdown（噪音略多）            |
| `fit_markdown`            | string | 更偏“正文提取/去噪”的 Markdown（通常更适合做知识库） |
| `markdown_with_citations` | string | Markdown + 引用编号                  |
| `references_markdown`     | string | 引用列表（和 citations 对应）             |
| `fit_html`                | string | 生成 `fit_markdown` 的对应 HTML 片段    |

---

## 8. 常用 jq 用法（强烈建议）

### 8.1 看爬了多少页

```bash
curl -s "http://43.139.166.203:11235/crawl/job/<task_id>" | jq '.result.results | length'
```

### 8.2 列出所有 URL（前 50 条）

```bash
curl -s "http://43.139.166.203:11235/crawl/job/<task_id>" \
| jq -r '.result.results[].url' | head -n 50
```

### 8.3 导出所有页面 fit_markdown 成一个 Markdown 文件

```bash
curl -s "http://43.139.166.203:11235/crawl/job/<task_id>" \
| jq -r '
  .result.results[]
  | "## " + .url + "\n\n"
    + (.markdown.fit_markdown // .markdown.raw_markdown // "")
    + "\n\n---\n"
' > site-fit.md
```

### 8.4 只看“瘦身后的 JSON”（避免被 html 淹没）

```bash
curl -s "http://43.139.166.203:11235/crawl/job/<task_id>" \
| jq '
  {
    task_id, status, created_at,
    count: (.result.results | length),
    results: (.result.results | map({
      url, success, status_code, error_message,
      markdown_preview: ((.markdown.fit_markdown // .markdown.raw_markdown // "")[0:200])
    }))
  }
'
```

---

