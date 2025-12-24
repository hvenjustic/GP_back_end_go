package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"resty.dev/v3"

	"GP_back_end_go/internal/mysql"
	"GP_back_end_go/models/dto"
	"GP_back_end_go/pkg/config"
	"GP_back_end_go/pkg/log"
	"GP_back_end_go/pkg/utils"
)

type markdownStore struct {
	Nums                 int      `json:"nums"`
	RawMarkdown          []string `json:"raw_markdown"`
	FitMarkdown          []string `json:"fit_markdown"`
	MarkdownWithCitation []string `json:"markdown_with_citations"`
}

type preprocessLLMResp struct {
	HasRelevantInfo bool            `json:"has_relevant_info"`
	Body            json.RawMessage `json:"body"`
	Reason          string          `json:"reason"`
}

func (r preprocessLLMResp) bodyText() string {
	raw := strings.TrimSpace(string(r.Body))
	if raw == "" || raw == "{}" {
		return ""
	}
	var asStr string
	if err := json.Unmarshal(r.Body, &asStr); err == nil {
		return strings.TrimSpace(asStr)
	}
	return raw
}

// RunPreprocessLLM 下载 fit markdown，调用 LLM 过滤并上传合并文件。
func RunPreprocessLLM(ctx context.Context, id uint64) (dto.PreprocessResponse, error) {
	if id == 0 {
		return dto.PreprocessResponse{}, fmt.Errorf("%w: id invalid", ErrBadRequest)
	}

	dao := mysql.NewCrawlTargetDAO()
	record, err := dao.GetDetailByID(id)
	if err != nil {
		return dto.PreprocessResponse{}, err
	}
	store, err := parseMarkdownStore(record.ResultMD)
	if err != nil {
		return dto.PreprocessResponse{}, err
	}
	if len(store.FitMarkdown) == 0 {
		return dto.PreprocessResponse{}, fmt.Errorf("%w: fit_markdown 为空，无法预处理", ErrBadRequest)
	}

	prompt, err := loadPrompt(config.Config.Prompts.PreprocessInline, config.Config.Prompts.PreprocessPath, "## 网页预处理提示词")
	if err != nil {
		return dto.PreprocessResponse{}, err
	}

	llm := NewChatLLMClient(config.Config.PagePreprocessLLM)
	if llm == nil {
		return dto.PreprocessResponse{}, fmt.Errorf("llm client not initialized")
	}

	start := time.Now()
	var sections []string
	failures := 0
	totalChunks := len(store.FitMarkdown)

	for idx, link := range store.FitMarkdown {
		text, err := downloadText(ctx, link)
		if err != nil {
			log.Error("PreprocessLLM", "download failed", err.Error(), "url", link)
			failures++
			continue
		}
		if strings.TrimSpace(text) == "" {
			failures++
			continue
		}

		promptText := strings.ReplaceAll(prompt, "{{text_block}}", text)
		log.Warn("PreprocessLLM", "llm_call_progress", fmt.Sprintf("%d/%d", idx+1, totalChunks))
		respText, err := llm.Chat(ctx, promptText)
		if err != nil {
			log.Error("PreprocessLLM", "llm call failed", err.Error(), "url", link)
			failures++
			continue
		}

		respText = stripCodeFence(respText)
		var parsed preprocessLLMResp
		if err := json.Unmarshal([]byte(respText), &parsed); err != nil {
			log.Error("PreprocessLLM", "parse llm resp failed", err.Error(), "raw", respText)
			failures++
			continue
		}
		body := strings.TrimSpace(parsed.bodyText())
		if body == "" {
			if !parsed.HasRelevantInfo {
				failures++
				continue
			}
			body = respText
		}
		section := fmt.Sprintf("# chunk%d\n%s", idx, body)
		sections = append(sections, section)
		log.Info("PreprocessLLM", "chunk_ok", "chunk_idx", idx, "url", link, "body_len", len(body))
	}

	if len(sections) == 0 {
		return dto.PreprocessResponse{}, fmt.Errorf("无可用预处理内容，处理失败（失败片段 %d 个）", failures)
	}

	merged := strings.Join(sections, "\n\n")
	uploader := InitOSSUploader()
	if uploader == nil {
		return dto.PreprocessResponse{}, fmt.Errorf("oss uploader init failed")
	}
	objectKey := fmt.Sprintf("processed/%s-%d-%d.md", SafeObjectKeyFromURL(record.URL, int(record.ID)), record.ID, time.Now().Unix())
	url, err := uploader.UploadString(ctx, objectKey, merged)
	if err != nil {
		return dto.PreprocessResponse{}, err
	}

	duration := time.Since(start).Milliseconds()
	patch := map[string]any{
		"processed_md":     url,
		"llm_processed_at": time.Now(),
		"llm_duration_ms":  duration,
	}
	if _, err := dao.ApplyResultByIDOrURL(&id, "", patch); err != nil {
		return dto.PreprocessResponse{}, err
	}

	return dto.PreprocessResponse{
		Status:        "ok",
		ProcessedMD:   url,
		LLMDurationMs: duration,
	}, nil
}

// BuildGraphFromProcessed 读取 processed_md，按 3 个 chunk 一轮调用 LLM，最终写入 graph_json。
func BuildGraphFromProcessed(ctx context.Context, id uint64) (dto.GraphBuildResponse, error) {
	if id == 0 {
		return dto.GraphBuildResponse{}, fmt.Errorf("%w: id invalid", ErrBadRequest)
	}
	dao := mysql.NewCrawlTargetDAO()
	record, err := dao.GetDetailByID(id)
	if err != nil {
		return dto.GraphBuildResponse{}, err
	}
	if record.ProcessedMD == nil || strings.TrimSpace(*record.ProcessedMD) == "" {
		return dto.GraphBuildResponse{}, fmt.Errorf("%w: processed_md 为空，请先完成预处理", ErrBadRequest)
	}

	content, err := downloadText(ctx, *record.ProcessedMD)
	if err != nil {
		return dto.GraphBuildResponse{}, fmt.Errorf("下载 processed_md 失败: %w", err)
	}
	chunks := splitChunks(content)
	if len(chunks) == 0 {
		return dto.GraphBuildResponse{}, fmt.Errorf("%w: processed_md 未包含 chunk 内容", ErrBadRequest)
	}

	prompt, err := loadPrompt(config.Config.Prompts.EntityInline, config.Config.Prompts.EntityPath, "## 实体识别提取提示词")
	if err != nil {
		return dto.GraphBuildResponse{}, err
	}

	llm := NewChatLLMClient(config.Config.EntityExtractionLLM)
	if llm == nil {
		return dto.GraphBuildResponse{}, fmt.Errorf("llm client not initialized")
	}

	currentJSON := map[string]any{
		"entities":  []any{},
		"relations": []any{},
	}
	if record.GraphJSON != nil && strings.TrimSpace(*record.GraphJSON) != "" {
		var tmp map[string]any
		if err := json.Unmarshal([]byte(*record.GraphJSON), &tmp); err == nil {
			currentJSON = tmp
		}
	}
	if _, ok := currentJSON["entities"]; !ok {
		currentJSON["entities"] = []any{}
	}
	if _, ok := currentJSON["relations"]; !ok {
		currentJSON["relations"] = []any{}
	}

	start := time.Now()
	successBatches := 0
	failures := 0
	totalChunks := len(chunks)
	for i := 0; i < len(chunks); i += 3 {
		end := i + 3
		if end > len(chunks) {
			end = len(chunks)
		}
		batch := strings.Join(chunks[i:end], "\n\n")
		promptText := strings.ReplaceAll(prompt, "{{markdown}}", batch)
		promptText = strings.ReplaceAll(promptText, "{{oldjson}}", utils.StructToJsonString(currentJSON))

		attempts := 0
		for {
			log.Warn("GraphLLM", "llm_call_progress", fmt.Sprintf("%d-%d/%d", i+1, end, totalChunks))
			respText, err := llm.Chat(ctx, promptText)
			if err != nil {
				attempts++
				log.Error("GraphLLM", "llm chat failed", err.Error(), "batch_start", i, "batch_end", end, "attempt", attempts)
				if attempts >= 2 {
					failures++
					break
				}
				time.Sleep(10 * time.Second)
				continue
			}
			respText = stripCodeFence(respText)

			var next map[string]any
			if err := json.Unmarshal([]byte(respText), &next); err != nil {
				attempts++
				snippet := truncateResp(respText, 200)
				log.Error("GraphLLM", "unmarshal_failed", err.Error(), "resp_snippet", snippet, "attempt", attempts)
				if attempts >= 2 {
					failures++
					break
				}
				time.Sleep(10 * time.Second)
				continue
			}

			currentJSON = next
			successBatches++
			log.Info("GraphLLM", "batch_ok", "batch_start", i, "batch_end", end, "resp_len", len(respText))
			break
		}
	}

	if successBatches == 0 {
		return dto.GraphBuildResponse{}, fmt.Errorf("图谱生成失败，所有批次均出错（失败批次 %d 个）", failures)
	}

	graph := utils.StructToJsonString(currentJSON)
	duration := time.Since(start).Milliseconds()
	patch := map[string]any{
		"graph_json":       graph,
		"llm_processed_at": time.Now(),
		"llm_duration_ms":  duration,
	}
	if _, err := dao.ApplyResultByIDOrURL(&id, "", patch); err != nil {
		return dto.GraphBuildResponse{}, err
	}

	return dto.GraphBuildResponse{
		Status:        "ok",
		GraphJSON:     graph,
		LLMDurationMs: duration,
	}, nil
}

type graphEntity struct {
	Name        string         `json:"name"`
	Type        string         `json:"type"`
	Aliases     []string       `json:"aliases"`
	Description string         `json:"description"`
	Extra       map[string]any `json:"extra"`
}

type graphRelation struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"`
}

type graphJSON struct {
	Entities  []graphEntity   `json:"entities"`
	Relations []graphRelation `json:"relations"`
}

var invalidIDChars = regexp.MustCompile(`[^a-zA-Z0-9:_-]+`)

// GetGraphVisual 将 graph_json 转换为前端可直接渲染的节点/边结构
func GetGraphVisual(ctx context.Context, id uint64) (dto.GraphVisualResponse, error) {
	if id == 0 {
		return dto.GraphVisualResponse{}, fmt.Errorf("%w: id invalid", ErrBadRequest)
	}

	dao := mysql.NewCrawlTargetDAO()
	record, err := dao.GetDetailByID(id)
	if err != nil {
		return dto.GraphVisualResponse{}, err
	}
	if record.GraphJSON == nil || strings.TrimSpace(*record.GraphJSON) == "" {
		return dto.GraphVisualResponse{}, fmt.Errorf("%w: graph_json 为空", ErrBadRequest)
	}

	var parsed graphJSON
	if err := json.Unmarshal([]byte(*record.GraphJSON), &parsed); err != nil {
		return dto.GraphVisualResponse{}, fmt.Errorf("%w: 解析 graph_json 失败: %v", ErrBadRequest, err)
	}

	nodes, edges := buildVisualElements(parsed)
	return dto.GraphVisualResponse{
		Nodes: nodes,
		Edges: edges,
	}, nil
}

func buildVisualElements(g graphJSON) ([]dto.GraphVisualNode, []dto.GraphVisualEdge) {
	usedNodeIDs := make(map[string]bool)
	nameIndex := make(map[string]string)
	var nodes []dto.GraphVisualNode

	genID := func(base string) string {
		clean := strings.TrimSpace(base)
		clean = invalidIDChars.ReplaceAllString(clean, "_")
		if clean == "" {
			clean = "node"
		}
		original := clean
		suffix := 1
		for usedNodeIDs[clean] {
			clean = fmt.Sprintf("%s_%d", original, suffix)
			suffix++
		}
		usedNodeIDs[clean] = true
		return clean
	}

	// 先遍历实体，生成 type:name 形式的 id，并记录 name -> id 的映射
	for _, ent := range g.Entities {
		name := strings.TrimSpace(ent.Name)
		typ := strings.TrimSpace(ent.Type)
		label := name
		if label == "" {
			label = "未知实体"
		}
		nodeID := genID(fmt.Sprintf("%s:%s", typ, name))
		if name != "" {
			nameIndex[strings.ToLower(name)] = nodeID
		}
		nodes = append(nodes, dto.GraphVisualNode{
			ID:          nodeID,
			Name:        name,
			Type:        typ,
			Label:       label,
			Description: ent.Description,
			Extra:       ent.Extra,
			Raw: map[string]any{
				"aliases": ent.Aliases,
				"extra":   ent.Extra,
			},
		})
	}

	usedEdgeIDs := make(map[string]int)
	var edges []dto.GraphVisualEdge
	for idx, rel := range g.Relations {
		srcName := strings.TrimSpace(rel.Source)
		dstName := strings.TrimSpace(rel.Target)
		if srcName == "" || dstName == "" {
			continue
		}
		srcID, okSrc := nameIndex[strings.ToLower(srcName)]
		dstID, okDst := nameIndex[strings.ToLower(dstName)]
		if !okSrc || !okDst {
			continue
		}

		edgeType := strings.TrimSpace(rel.Type)
		if edgeType == "" {
			edgeType = "RELATED_TO"
		}

		baseID := fmt.Sprintf("%s:%s:%s", srcID, edgeType, dstID)
		edgeID := baseID
		if count := usedEdgeIDs[baseID]; count > 0 {
			edgeID = fmt.Sprintf("%s_%d", baseID, count)
		}
		usedEdgeIDs[baseID]++

		edges = append(edges, dto.GraphVisualEdge{
			ID:     edgeID,
			Source: srcID,
			Target: dstID,
			Type:   edgeType,
			Label:  edgeType,
			Raw: map[string]any{
				"index":       idx,
				"source_name": srcName,
				"target_name": dstName,
			},
		})
	}

	return nodes, edges
}

func parseMarkdownStore(raw *string) (*markdownStore, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil, fmt.Errorf("%w: result_md 为空", ErrBadRequest)
	}
	var store markdownStore
	if err := json.Unmarshal([]byte(*raw), &store); err != nil {
		return nil, fmt.Errorf("%w: 解析 result_md 失败: %v", ErrBadRequest, err)
	}
	return &store, nil
}

func downloadText(ctx context.Context, url string) (string, error) {
	client := resty.New().SetTimeout(60 * time.Second)
	resp, err := client.R().SetContext(ctx).Get(url)
	if err != nil {
		return "", err
	}
	if resp.StatusCode() >= 300 {
		return "", fmt.Errorf("download status %d: %s", resp.StatusCode(), resp.String())
	}
	if text := resp.String(); text != "" {
		return text, nil
	}
	if resp.Body == nil {
		return "", fmt.Errorf("download body empty")
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func loadPrompt(inline string, path string, marker string) (string, error) {
	if text := strings.TrimSpace(inline); text != "" {
		return text, nil
	}
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("prompt path empty")
	}
	abs, _ := filepath.Abs(path)
	b, err := os.ReadFile(abs)
	if err != nil {
		return "", err
	}
	content := string(b)
	if marker == "" {
		return strings.TrimSpace(content), nil
	}
	idx := strings.Index(content, marker)
	if idx < 0 {
		log.Error("Prompt", "marker not found", marker, "path", abs)
		return strings.TrimSpace(content), nil
	}
	section := content[idx+len(marker):]
	nextIdx := strings.Index(section, "## ")
	if nextIdx >= 0 {
		section = section[:nextIdx]
	}
	return strings.TrimSpace(section), nil
}

func splitChunks(content string) []string {
	lines := strings.Split(content, "\n")
	var chunks []string
	var current []string
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "# chunk") && len(current) > 0 {
			chunks = appendChunk(chunks, current)
			current = []string{line}
			continue
		}
		current = append(current, line)
	}
	if len(current) > 0 {
		chunks = appendChunk(chunks, current)
	}
	return chunks
}

func appendChunk(list []string, lines []string) []string {
	text := strings.TrimSpace(strings.Join(lines, "\n"))
	if text != "" {
		list = append(list, text)
	}
	return list
}

func truncateResp(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
