package botengine

// PredefinedConfig 预定义回复配置
type PredefinedConfig struct {
	Mode     string   `json:"mode"`     // "fixed" | "random" | "template"
	Replies  []string `json:"replies"`  // 回复列表
	Template string   `json:"template"` // 模板字符串（mode=template 时）
}

// LLMConfig LLM 回复配置（用于 mechanism_config 反序列化）
type LLMConfig struct {
	APIURL        string  `json:"api_url"`
	APIKey        string  `json:"api_key"`
	Model         string  `json:"model"`
	SystemPrompt  string  `json:"system_prompt"`
	Temperature   float64 `json:"temperature"`
	MaxTokens     int     `json:"max_tokens"`
	ContextWindow int     `json:"context_window"`
}

// ContextMessage 上下文消息（Go→TS 请求体中使用）
type ContextMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	MsgType string `json:"msgType"`
}
