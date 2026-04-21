package botengine

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
)

// argsIndexRe 匹配 {args:N} 模式，N 为非负整数
var argsIndexRe = regexp.MustCompile(`\{args:(\d+)\}`)

// ReplaceArgsVars 在模板字符串中替换 {args} 和 {args:N} 变量。
// {args} 返回 input 按空格分隔后跳过索引 0 的所有部分（空格连接）。
// {args:N} 返回第 N 个部分，索引越界返回空字符串。
func ReplaceArgsVars(result string, input string) string {
	parts := strings.Fields(input)

	// 先替换 {args:N}（带索引），再替换 {args}（无索引）
	result = argsIndexRe.ReplaceAllStringFunc(result, func(match string) string {
		indexStr := match[len("{args:") : len(match)-1]
		index, err := strconv.Atoi(indexStr)
		if err != nil || index < 0 || index >= len(parts) {
			return ""
		}
		return parts[index]
	})

	if len(parts) > 1 {
		result = strings.ReplaceAll(result, "{args}", strings.Join(parts[1:], " "))
	} else {
		result = strings.ReplaceAll(result, "{args}", "")
	}
	return result
}

// ===== 机制配置模型 =====

// MechanismConfig Bot 的完整机制配置
type MechanismConfig struct {
	Mechanisms []Mechanism `json:"mechanisms"`
}

// Mechanism 单个机制 = 触发规则 + 回复设置
type Mechanism struct {
	ID      string      `json:"id"`
	Name    string      `json:"name"`
	Enabled bool        `json:"enabled"`
	Trigger TriggerSpec `json:"trigger"`
	Reply   ReplySpec   `json:"reply"`
}

// TriggerSpec 触发规格
type TriggerSpec struct {
	Type        string        `json:"type"`                  // "rule" | "probability"
	Rules       []TriggerRule `json:"rules,omitempty"`       // 仅 rule 类型
	Probability float64       `json:"probability,omitempty"` // 仅 probability 类型 (0.0-1.0)
}

// ReplySpec 回复规格
type ReplySpec struct {
	Type        string            `json:"type"` // "predefined" | "llm" | "special_mode"
	Predefined  *PredefinedConfig `json:"predefined,omitempty"`
	LLM         *LLMConfig        `json:"llm,omitempty"`
	SpecialMode *SpecialModeSpec  `json:"special_mode,omitempty"`
}

// SpecialModeSpec 特殊模式规格（嵌套在机制中）
type SpecialModeSpec struct {
	Events        []SpecialModeEvent `json:"events"`
	Connections   []FlowConnection   `json:"connections,omitempty"` // 端口化连线（新流程引擎）
	EndConditions []EndCondition     `json:"end_conditions"`
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

// DefaultMechanismConfig 创建默认的机制配置（一个空规则 + 随机回复机制）
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
				Reply: ReplySpec{
					Type: "predefined",
					Predefined: &PredefinedConfig{
						Mode:    "random",
						Replies: []string{"..."},
					},
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

		// 验证回复
		if err := validateReplySpec(&mech.Reply); err != nil {
			return fmt.Errorf("mechanism[%d] (%s): reply invalid: %w", i, mech.ID, err)
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

func validateReplySpec(rs *ReplySpec) error {
	switch rs.Type {
	case "predefined":
		if rs.Predefined == nil {
			return fmt.Errorf("predefined config is required when type is 'predefined'")
		}
	case "llm":
		if rs.LLM == nil {
			return fmt.Errorf("llm config is required when type is 'llm'")
		}
	case "special_mode":
		if rs.SpecialMode == nil {
			return fmt.Errorf("special_mode config is required when type is 'special_mode'")
		}
		if len(rs.SpecialMode.EndConditions) == 0 {
			return fmt.Errorf("special_mode must have at least one end condition")
		}
	default:
		return fmt.Errorf("invalid reply type: %q", rs.Type)
	}
	return nil
}

// ===== 触发评估 =====

// Evaluate 评估消息是否触发该机制
func (ts *TriggerSpec) Evaluate(content string) bool {
	switch ts.Type {
	case "rule":
		return ts.evaluateRules(content)
	case "probability":
		if ts.Probability <= 0 {
			return false
		}
		return rand.Float64() < ts.Probability
	default:
		return false
	}
}

// evaluateRules 规则触发评估（任一规则匹配即触发）
func (ts *TriggerSpec) evaluateRules(content string) bool {
	if len(ts.Rules) == 0 {
		return true // 无规则时默认触发
	}

	for _, rule := range ts.Rules {
		switch rule.Type {
		case "keyword":
			checkContent := content
			checkPattern := rule.Pattern
			if !rule.CaseSensitive {
				checkContent = strings.ToLower(content)
				checkPattern = strings.ToLower(rule.Pattern)
			}
			if strings.Contains(checkContent, checkPattern) {
				return true
			}

		case "regex":
			flags := ""
			if !rule.CaseSensitive {
				flags = "(?i)"
			}
			re, err := regexp.Compile(flags + rule.Pattern)
			if err != nil {
				continue // 无效正则跳过
			}
			if re.MatchString(content) {
				return true
			}

		case "command":
			// 命令前缀匹配
			checkContent := content
			checkPattern := rule.Pattern
			if !rule.CaseSensitive {
				checkContent = strings.ToLower(content)
				checkPattern = strings.ToLower(rule.Pattern)
			}
			if strings.HasPrefix(checkContent, checkPattern) {
				return true
			}

		case "equals":
			// 精确匹配
			checkContent := strings.TrimSpace(content)
			checkPattern := rule.Pattern
			if !rule.CaseSensitive {
				checkContent = strings.ToLower(checkContent)
				checkPattern = strings.ToLower(rule.Pattern)
			}
			if checkContent == checkPattern {
				return true
			}
		}
	}

	return false
}

// ===== 回复生成 =====

// GenerateReply 生成回复
func (rs *ReplySpec) GenerateReply(input string, contextVars map[string]string, messages []ContextMessage, senderName string) (string, error) {
	switch rs.Type {
	case "predefined":
		return generatePredefinedReply(rs.Predefined, input, contextVars, senderName)
	case "llm":
		return generateLLMReply(rs.LLM, input, messages)
	default:
		return "", fmt.Errorf("unsupported reply type: %q", rs.Type)
	}
}

// generatePredefinedReply 生成预定义回复（复用 reply.go 中的逻辑）
func generatePredefinedReply(config *PredefinedConfig, input string, vars map[string]string, senderName string) (string, error) {
	if config == nil {
		return "...", nil
	}

	switch config.Mode {
	case "fixed":
		if len(config.Replies) > 0 {
			return config.Replies[0], nil
		}
		return "...", nil

	case "random":
		if len(config.Replies) > 0 {
			return config.Replies[rand.Intn(len(config.Replies))], nil
		}
		return "...", nil

	case "template":
		result := config.Template
		if result == "" {
			result = "{input}"
		}

		// 替换变量
		result = strings.ReplaceAll(result, "{input}", input)
		result = strings.ReplaceAll(result, "{username}", senderName)

		// 替换 args 变量（{args}, {args:N}）
		result = ReplaceArgsVars(result, input)

		// 替换自定义变量
		for k, v := range vars {
			result = strings.ReplaceAll(result, "{"+k+"}", v)
		}

		return result, nil

	default:
		return "...", nil
	}
}

// generateLLMReply 调用 LLM 生成回复（复用 reply.go 中的 CallLLM）
func generateLLMReply(config *LLMConfig, input string, messages []ContextMessage) (string, error) {
	if config == nil || config.APIURL == "" {
		return "...", nil
	}
	return CallLLM(context.TODO(), config, input, messages)
}

// ===== 辅助函数 =====

// FindSpecialModeMechanism 在机制列表中查找特殊模式类型的机制
func FindSpecialModeMechanism(mechanisms []Mechanism) *Mechanism {
	for i := range mechanisms {
		if mechanisms[i].Reply.Type == "special_mode" && mechanisms[i].Enabled {
			return &mechanisms[i]
		}
	}
	return nil
}

// GetMechanismSummary 获取机制的简要描述
func GetMechanismSummary(mechanism Mechanism) (triggerSummary string, replySummary string) {
	// 触发摘要
	switch mechanism.Trigger.Type {
	case "rule":
		ruleCount := len(mechanism.Trigger.Rules)
		if ruleCount == 0 {
			triggerSummary = "始终触发"
		} else {
			triggerSummary = fmt.Sprintf("%d 条规则", ruleCount)
		}
	case "probability":
		triggerSummary = fmt.Sprintf("概率 %.0f%%", mechanism.Trigger.Probability*100)
	}

	// 回复摘要
	switch mechanism.Reply.Type {
	case "predefined":
		if mechanism.Reply.Predefined != nil {
			switch mechanism.Reply.Predefined.Mode {
			case "fixed":
				replySummary = "固定回复"
			case "random":
				replySummary = fmt.Sprintf("随机回复 (%d条)", len(mechanism.Reply.Predefined.Replies))
			case "template":
				replySummary = "模板回复"
			}
		}
	case "llm":
		replySummary = "LLM 回复"
	case "special_mode":
		if mechanism.Reply.SpecialMode != nil {
			replySummary = fmt.Sprintf("特殊模式 (%d事件)", len(mechanism.Reply.SpecialMode.Events))
		}
	}

	return triggerSummary, replySummary
}
