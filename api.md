# Crawl4AI REST API（以运行中服务 OpenAPI 为准）

本文档以当前线上 Crawl4AI 服务实际暴露的 OpenAPI 为准：

- OpenAPI：`http://43.139.166.203:11235/openapi.json`

> 说明：
> - **端点路径**与**请求体字段**：以 OpenAPI 为准。
> - 对于请求体里的 `object` 类型（如 `browser_config` / `crawler_config`）：OpenAPI 只描述为“任意 object”，而 Crawl4AI 的 Python 配置对象在 REST 场景有更严格的序列化约定；这部分以本文档的“配置对象 JSON 序列化规则”说明为准。

## Base URL

* `http://<host>:11235`

## Content-Type

* `Content-Type: application/json`

## 配置对象的 JSON 序列化规则（很重要）

在 REST API 里，`BrowserConfig` / `CrawlerRunConfig` 这类“Python 配置对象”要写成：

* `{"type": "<类名>", "params": { ... }}`
* 如果 `params` 内部包含 **dict / list** 等复杂类型，通常需要包一层：`{"type": "dict", "value": {...}}` / `{"type":"list","value":[...]}`

---

## 1) 同步爬取：POST `/crawl`

### 请求

**Path**

* `POST /crawl`

**Body（JSON）**

| 字段               |                               类型 |  必填 | 说明                                                                |
| ---------------- | -------------------------------: | :-: | ----------------------------------------------------------------- |
| `urls`           |                       `string[]` |  是  | 要爬取的 URL 列表（单个也用数组）；OpenAPI 限制：`minItems=1`，`maxItems=100` |
| `browser_config` |                         `object \| null` |  否  | 浏览器层配置（UA、代理、headers、viewport、是否启用 JS 等）；对象序列化规则见上文 |
| `crawler_config` |                         `object \| null` |  否  | 单次爬取运行配置（缓存、JS 执行、等待策略、抽取、链接过滤、表格抽取等）；对象序列化规则见上文 |
| `hooks`          |                  `HookConfig \| null` |  否  | 可选 Hook（OpenAPI：`code`/`timeout`），一般用不到可不传 |

> 注意：如果你**只要“非流式”**，确保 `crawler_config.params.stream = false`（默认就是 false；Self-Hosting Guide 的同步示例也明确写了 `stream: False`）。 

### 你给的配置（整理 + 字段解释）

你给的是（我保持你的写法），含义如下：

```json
{
  "browser": {
    "type": "BrowserConfig",
    "params": {
      "headers": {
        "type": "dict",
        "value": {
          "sec-ch-ua": "\"Chromium\";v=\"116\", \"Not_A Brand\";v=\"8\", \"Google Chrome\";v=\"116\""
        }
      }
    }
  },
  "crawler": {
    "type": "CrawlerRunConfig",
    "params": {
      "scraping_strategy": { "type": "LXMLWebScrapingStrategy", "params": {} },
      "table_extraction": { "type": "DefaultTableExtraction", "params": {} },
      "exclude_social_media_domains": [
        "facebook.com","twitter.com","x.com","linkedin.com","instagram.com",
        "pinterest.com","tiktok.com","snapchat.com","reddit.com"
      ]
    }
  }
}
```

把它映射到 REST API 的标准字段名，应该是：

* `browser` → `browser_config`
* `crawler` → `crawler_config`

各字段含义：

* `browser_config.type = "BrowserConfig"`：表示这是浏览器配置对象。
* `browser_config.params.headers`：为**每个请求**附加额外 HTTP Header（文档里就叫 `headers: dict`）。你用 `{"type":"dict","value":{...}}` 是符合 REST 序列化习惯的。
* `crawler_config.params.scraping_strategy = LXMLWebScrapingStrategy`：使用 LXML 做内容抓取/解析（文档里说明 `scraping_strategy` 默认就是 `LXMLWebScrapingStrategy()`，也支持自定义）。
* `crawler_config.params.table_extraction = DefaultTableExtraction`：启用默认表格抽取策略；一般优先用它（LLM 表格抽取更贵更慢）。
* `crawler_config.params.exclude_social_media_domains`：扩展/覆盖要剔除的社媒域名列表；文档也给了默认社媒列表（基本与你这份一致）。

### 强 JS（动态站）怎么配（同步也适用）

要“强 JS”，核心是两块：

* 浏览器层：`BrowserConfig.java_script_enabled`（默认 true）
* 爬取层：`CrawlerRunConfig.js_code` / `wait_for` / `wait_until` / `page_timeout` 等

常用字段（放在 `crawler_config.params`）：

* `wait_until`: `"domcontentloaded"` / `"networkidle"`（控制导航完成条件）
* `wait_for`: `"css:selector"` 或 `"js:() => boolean"`（等待某个元素/条件出现再抽取）
* `js_code`: `string | string[]`（页面加载后执行 JS：点击“加载更多”、滚动等）
* `page_timeout`: 导航/脚本超时（毫秒）

---

## 2) 同步响应（Response）

### 2.1 `/crawl` 顶层响应形态

OpenAPI 对响应 schema 标注较少（常见为 `{}`），但实际服务通常会返回以下两种形态（以 README / Self-Hosting Guide 的行为为参考）：

* **若直接完成**：响应 JSON 里会包含 `results`
* **若未立即完成**：会返回 `task_id`，随后可用 `GET /task/{task_id}` 拉取结果

因此建议你按这两种形态兼容：

**A. 已完成**

```json
{
  "results": [ /* CrawlResult[] */ ]
}
```

**B. 返回任务 ID**

```json
{
  "task_id": "string"
}
```

### 2.2 `CrawlResult` 结构体（单页结果）

`results[]` 的元素是 `CrawlResult`。官方文档给了模型字段（核心字段如下）。

| 字段                  | 类型        | 说明                                                        |                                                                                                 |
| ------------------- | --------- | --------------------------------------------------------- | ----------------------------------------------------------------------------------------------- |
| `url`               | `string`  | 最终 URL（含重定向后的最终地址）               |                                                                                                 |
| `success`           | `boolean` | 是否成功；失败时看 `error_message`        |                                                                                                 |
| `status_code`       | `number?` | HTTP 状态码（可能为空）                   |                                                                                                 |
| `error_message`     | `string?` | 失败原因                             |                                                                                                 |
| `html`              | `string`  | 原始 HTML                          |                                                                                                 |
| `cleaned_html`      | `string?` | 清洗后的 HTML（去脚本/样式、按配置剔除标签等）       |                                                                                                 |
| `fit_html`          | `string?` | “适配抽取”的预处理 HTML                  |                                                                                                 |
| `markdown`          | `string   | object?`                                                  | Markdown 或 `MarkdownGenerationResult`（可能包含 citations / fit_markdown 等） |
| `extracted_content` | `string?` | 结构化抽取结果（通常是 JSON 字符串）            |                                                                                                 |
| `links`             | `object`  | 链接信息（常见分 internal/external）      |                                                                                                 |
| `media`             | `object`  | 图片/音视频等媒体信息                      |                                                                                                 |
| `tables`            | `array`   | 表格抽取结果列表（开启 table_extraction 后）  |                                                                                                 |
| `screenshot`        | `string?` | base64 截图（需配置 `screenshot=true`） |                                                                                                 |
| `pdf`               | `bytes?`  | PDF（需 `pdf=true`）                |                                                                                                 |
| `mhtml`             | `string?` | MHTML 快照（需 `capture_mhtml=true`） |                                                                                                 |
| `response_headers`  | `object?` | 响应头                              |                                                                                                 |
| `session_id`        | `string?` | 会话 ID（复用浏览器上下文）                  |                                                                                                 |

---

## 3) 站点内爬取（只在站内走，不出域）

你要的是两种“站内”概念，分别配置：

### A) **深度爬取（会跟随链接）**：不出域

在深度爬取策略里设置：

* `include_external = false`（不跟随外部域名链接）
  并可叠加：
* `DomainFilter.allowed_domains=[...]` 限定允许域名

### B) **单页爬取但想清理外链输出**

在 `CrawlerRunConfig` 里设置：

* `exclude_external_links = true`（把站外链接从结果里移除/剔除）

---

---

# Crawl4AI REST API（异步 Job Queue）

> 适用：任务队列模式（提交任务 → 轮询状态/取结果），非流式。

## 1) 提交异步任务：POST `/crawl/job`

**Path**

* `POST /crawl/job`

**Body（JSON）**
字段与同步 `/crawl` 基本一致，并额外支持 webhook（如果你启用）。

| 字段               | 类型 | 必填 | 说明 |
| ---------------- | ---: | :-: | --- |
| `urls`           | `string[]` | 是 | 要爬取的 URL 列表（单个也用数组） |
| `browser_config` | `object` | 否 | 浏览器层配置；对象序列化规则见上文（OpenAPI 默认 `{}`） |
| `crawler_config` | `object` | 否 | 运行配置；对象序列化规则见上文（OpenAPI 默认 `{}`） |
| `webhook_config` | `WebhookConfig \| null` | 否 | webhook 通知配置（见下） |

### `WebhookConfig`（OpenAPI）

| 字段 | 类型 | 必填 | 说明 |
| --- | ---: | :-: | --- |
| `webhook_url` | `string` | 是 | webhook 回调地址（uri） |
| `webhook_data_in_payload` | `boolean` | 否 | 是否把结果数据放进 payload（默认 `false`） |
| `webhook_headers` | `object \| null` | 否 | 额外 header（string→string） |

**Response（提交成功）**

```json
{
  "task_id": "string",
  "message": "Task queued successfully."
}
```

 

---

## 2) 查询任务状态 / 获取结果：GET `/crawl/job/{task_id}`

**Path**

* `GET /crawl/job/{task_id}`

**Response（示例：进行中）**

```json
{
  "task_id": "string",
  "status": "queued | running",
  "message": "Task is in progress."
}
```

**Response（示例：已完成）**

```json
{
  "task_id": "string",
  "status": "completed",
  "result": { /* CrawlResult 或 CrawlResult[]（取决于你提交的 urls 数量） */ }
}
```

**Response（示例：失败）**

```json
{
  "task_id": "string",
  "status": "failed",
  "message": "error message"
}
```

> `status` 与 “完成后返回 result” 的模式来自 Self-Hosting Guide 的 Job Queue 说明。 

---

## 3) 兼容端点：GET `/task/{task_id}`

有些版本/镜像可能提供不同的兼容端点（README 的 Docker 快速测试示例里出现过）：

* `GET /task/{task_id}` 

如果你在某些版本/镜像里发现 `/crawl/job/{task_id}` 不通、但 `/task/{task_id}` 可用（或相反），建议做兼容分支（客户端侧依次尝试）。

---

## 4) 异步结果结构：`result` 是什么？

* 若你提交的是多 URL（`urls: [...]`），最稳妥的处理方式是把 `result` 当成“可能是单个 CrawlResult，也可能是 CrawlResult 列表 / 包装对象”，然后做结构探测。
* 真正的“单页结果字段”，以官方 `CrawlResult` 为准（见上一个文档的 `CrawlResult` 表）。 

---

## 5) 常用配置片段（异步也完全一样）

### 强 JS（动态内容）

* `crawler_config.params.wait_for`
* `crawler_config.params.js_code`
* `crawler_config.params.page_timeout`


### 站内（不出域/不输出外链）

* 深度爬取：`include_external=false`（在 deep crawl strategy 上） 
* 清理外链：`exclude_external_links=true`（在 CrawlerRunConfig 上） 
