package botengine

// TriggerRule 触发规则
type TriggerRule struct {
	Type         string `json:"type"`          // "keyword" | "regex" | "command" | "equals"
	Pattern      string `json:"pattern"`       // 匹配模式
	CaseSensitive bool   `json:"case_sensitive"` // 是否区分大小写
}
