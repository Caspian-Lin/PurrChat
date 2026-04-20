package botengine

import (
	"fmt"
	"math/rand"
	"strings"
)

// executeBuiltinHandler 执行内置事件
// Step 2 中会扩展完善，当前实现基础功能
func executeBuiltinHandler(config map[string]any, input string, variables map[string]string) (string, error) {
	builtinType := getStringField(config, "builtin_type")

	switch builtinType {
	case "random_number":
		return builtinRandomNumber(config)
	case "haiku":
		return builtinHaiku(config, input)
	case "echo":
		return builtinEcho(config, input)
	case "count":
		return builtinCount(config, variables)
	case "template":
		return builtinTemplate(config, input, variables)
	default:
		return "", fmt.Errorf("unknown builtin type: %s", builtinType)
	}
}

// builtinRandomNumber 生成随机数
func builtinRandomNumber(config map[string]any) (string, error) {
	minVal := 0
	maxVal := 100
	isInteger := true

	if v, ok := config["min"].(float64); ok {
		minVal = int(v)
	}
	if v, ok := config["max"].(float64); ok {
		maxVal = int(v)
	}
	if v, ok := config["integer"].(bool); ok {
		isInteger = v
	}

	if minVal > maxVal {
		minVal, maxVal = maxVal, minVal
	}

	if isInteger {
		return fmt.Sprintf("%d", rand.Intn(maxVal-minVal+1)+minVal), nil
	}
	return fmt.Sprintf("%f", rand.Float64()*float64(maxVal-minVal)+float64(minVal)), nil
}

// builtinHaiku 生成俳句（模板化）
func builtinHaiku(config map[string]any, input string) (string, error) {
	topic := getStringField(config, "topic")
	if topic == "" {
		topic = input
	}

	// 简单的俳句模板（5-7-5 结构）
	haikus := []string{
		fmt.Sprintf("%s的光芒\n照亮了前行的路\n脚步不停歇", topic),
		fmt.Sprintf("%s轻声吟唱\n风中传来回响\n万物皆有灵", topic),
		fmt.Sprintf("静观%s变\n一叶落而知秋至\n心如止水", topic),
		fmt.Sprintf("%s如流水\n昼夜不息向前\n奔向大海", topic),
	}

	return haikus[rand.Intn(len(haikus))], nil
}

// builtinEcho 回显输入
func builtinEcho(config map[string]any, input string) (string, error) {
	prefix := getStringField(config, "prefix")
	suffix := getStringField(config, "suffix")

	result := input
	if prefix != "" {
		result = prefix + result
	}
	if suffix != "" {
		result = result + suffix
	}

	return result, nil
}

// builtinCount 统计消息计数
func builtinCount(config map[string]any, variables map[string]string) (string, error) {
	counterKey := getStringField(config, "counter_key")
	if counterKey == "" {
		counterKey = "message_count"
	}

	count := 0
	if val, ok := variables[counterKey]; ok {
		_, _ = fmt.Sscanf(val, "%d", &count)
	}
	count++

	variables[counterKey] = fmt.Sprintf("%d", count)
	return fmt.Sprintf("%d", count), nil
}

// builtinTemplate 模板渲染
func builtinTemplate(config map[string]any, input string, variables map[string]string) (string, error) {
	template := getStringField(config, "template")
	if template == "" {
		return "", fmt.Errorf("template is empty")
	}

	result := template
	for key, value := range variables {
		result = strings.ReplaceAll(result, "{"+key+"}", value)
	}

	// 替换 args 变量（{args}, {args:N}）
	result = ReplaceArgsVars(result, input)

	return result, nil
}
