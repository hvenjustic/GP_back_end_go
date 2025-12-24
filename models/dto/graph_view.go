package dto

// GraphVisualNode 面向前端渲染的节点
type GraphVisualNode struct {
	ID          string            `json:"id"`
	Name        string            `json:"name,omitempty"`
	Type        string            `json:"type,omitempty"`
	Label       string            `json:"label,omitempty"`
	Description string            `json:"description,omitempty"`
	Extra       map[string]any    `json:"extra,omitempty"`
	Raw         map[string]any    `json:"raw,omitempty"`
	Meta        map[string]string `json:"meta,omitempty"`
}

// GraphVisualEdge 面向前端渲染的边
type GraphVisualEdge struct {
	ID     string            `json:"id"`
	Source string            `json:"source"`
	Target string            `json:"target"`
	Type   string            `json:"type,omitempty"`
	Label  string            `json:"label,omitempty"`
	Raw    map[string]any    `json:"raw,omitempty"`
	Meta   map[string]string `json:"meta,omitempty"`
}

// GraphVisualResponse 前端可直接使用的节点/边集合
type GraphVisualResponse struct {
	Nodes []GraphVisualNode `json:"nodes"`
	Edges []GraphVisualEdge `json:"edges"`
}
