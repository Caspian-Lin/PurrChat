package botengine

import (
	"encoding/json"
	"fmt"
)

// ===== 机制配置模型 =====

// ContextMessage 上下文消息（Go→TS 请求体中使用）
type ContextMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	MsgType string `json:"msgType"`
}

// MechanismConfig Bot 的完整机制配置
type MechanismConfig struct {
	Mechanisms []Mechanism `json:"mechanisms"`
}

// Mechanism 单个机制 = 触发规则（回复行为由 mechanism 级工作流文档定义）
type Mechanism struct {
	ID      string      `json:"id"`
	Name    string      `json:"name"`
	Enabled bool        `json:"enabled"`
	Trigger TriggerSpec `json:"trigger"`
}

// TriggerSpec 触发规格
type TriggerSpec struct {
	Type        string        `json:"type"`                  // "rule" | "probability"
	Rules       []TriggerRule `json:"rules,omitempty"`       // 仅 rule 类型
	Probability float64       `json:"probability,omitempty"` // 仅 probability 类型 (0.0-1.0)
}

// ===== 解析函数 =====

// ParseMechanismConfig 从 JSON 解析机制配置
func ParseMechanismConfig(raw json.RawMessage) (*MechanismConfig, error) {
	if len(raw) == 0 || string(raw) == "[]" || string(raw) == "null" {
		return &MechanismConfig{Mechanisms: []Mechanism{}}, nil
	}

	var config MechanismConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		return nil, fmt.Errorf("failed to parse mechanism config: %w", err)
	}

	return &config, nil
}

// DefaultMechanismConfig 创建默认的机制配置（一个空规则触发机制）
func DefaultMechanismConfig() json.RawMessage {
	config := MechanismConfig{
		Mechanisms: []Mechanism{
			{
				ID:      "mech_default",
				Name:    "默认机制",
				Enabled: true,
				Trigger: TriggerSpec{
					Type:  "rule",
					Rules: []TriggerRule{},
				},
			},
		},
	}
	data, _ := json.Marshal(config)
	return json.RawMessage(data)
}

// ===== 验证函数 =====

// ValidateMechanisms 验证机制配置的合法性
func ValidateMechanisms(config *MechanismConfig) error {
	if config == nil {
		return fmt.Errorf("mechanism config is nil")
	}

	probabilityCount := 0
	for i, mech := range config.Mechanisms {
		if mech.ID == "" {
			return fmt.Errorf("mechanism[%d]: id is required", i)
		}
		if mech.Name == "" {
			return fmt.Errorf("mechanism[%d] (%s): name is required", i, mech.ID)
		}

		// 验证触发
		if err := validateTriggerSpec(&mech.Trigger); err != nil {
			return fmt.Errorf("mechanism[%d] (%s): trigger invalid: %w", i, mech.ID, err)
		}

		// 统计概率机制数量
		if mech.Trigger.Type == "probability" {
			probabilityCount++
		}
	}

	if probabilityCount > 1 {
		return fmt.Errorf("only one probability trigger mechanism is allowed, found %d", probabilityCount)
	}

	return nil
}

func validateTriggerSpec(ts *TriggerSpec) error {
	switch ts.Type {
	case "rule":
		for j, rule := range ts.Rules {
			if rule.Type != "keyword" && rule.Type != "regex" && rule.Type != "command" && rule.Type != "equals" {
				return fmt.Errorf("rule[%d]: invalid type %q", j, rule.Type)
			}
		}
	case "probability":
		if ts.Probability <= 0 || ts.Probability > 1 {
			return fmt.Errorf("probability must be between 0 and 1, got %f", ts.Probability)
		}
	default:
		return fmt.Errorf("invalid trigger type: %q", ts.Type)
	}
	return nil
}

// ===== 辅助函数 =====

// GetMechanismTriggerSummary 获取机制的触发摘要
func GetMechanismTriggerSummary(mechanism Mechanism) string {
	switch mechanism.Trigger.Type {
	case "rule":
		ruleCount := len(mechanism.Trigger.Rules)
		if ruleCount == 0 {
			return "始终触发"
		}
		return fmt.Sprintf("%d 条规则", ruleCount)
	case "probability":
		return fmt.Sprintf("概率 %.0f%%", mechanism.Trigger.Probability*100)
	}
	return ""
}
