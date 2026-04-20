package botengine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// PredefinedConfig 预定义回复配置
type PredefinedConfig struct {
	Mode     string   `json:"mode"`     // "fixed" | "random" | "template"
	Replies  []string `json:"replies"`  // 回复列表
	Template string   `json:"template"` // 模板字符串（mode=template 时）
}

// LLMConfig LLM 回复配置
type LLMConfig struct {
	APIURL        string  `json:"api_url"`
	APIKey        string  `json:"api_key"`
	Model         string  `json:"model"`
	SystemPrompt  string  `json:"system_prompt"`
	Temperature   float64 `json:"temperature"`
	MaxTokens     int     `json:"max_tokens"`
	ContextWindow int     `json:"context_window"` // 上下文消息窗口大小
}

// ContextMessage 上下文消息
type ContextMessage struct {
	Role    string `json:"role"` // "user" | "assistant" | "system"
	Content string `json:"content"`
}

// CallLLM 调用 LLM 生成回复（独立函数，供机制和事件链复用）
func CallLLM(ctx context.Context, config *LLMConfig, input string, messages []ContextMessage) (string, error) {
	if config == nil || config.APIURL == "" {
		return "...", nil
	}

	// 构建消息列表
	apiMessages := []map[string]string{}

	// 添加 system prompt
	if config.SystemPrompt != "" {
		apiMessages = append(apiMessages, map[string]string{
			"role":    "system",
			"content": config.SystemPrompt,
		})
	}

	// 截取上下文窗口
	contextWindow := config.ContextWindow
	if contextWindow <= 0 {
		contextWindow = 20
	}
	if contextWindow > 0 && len(messages) > contextWindow {
		messages = messages[len(messages)-contextWindow:]
	}

	// 添加上下文消息
	for _, msg := range messages {
		apiMessages = append(apiMessages, map[string]string{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}

	// 添加当前用户消息
	apiMessages = append(apiMessages, map[string]string{
		"role":    "user",
		"content": input,
	})

	// 构建请求体
	reqBody := map[string]any{
		"model":      config.Model,
		"messages":   apiMessages,
		"max_tokens": config.MaxTokens,
	}
	if config.Temperature > 0 {
		reqBody["temperature"] = config.Temperature
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal LLM request: %w", err)
	}

	// 创建 HTTP 请求
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "POST", config.APIURL, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create LLM request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+config.APIKey)
	}

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("LLM request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("LLM returned status %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var llmResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&llmResp); err != nil {
		return "", fmt.Errorf("failed to decode LLM response: %w", err)
	}

	if len(llmResp.Choices) > 0 && llmResp.Choices[0].Message.Content != "" {
		return llmResp.Choices[0].Message.Content, nil
	}

	return "...", nil
}
