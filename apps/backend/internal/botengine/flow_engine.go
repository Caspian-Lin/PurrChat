package botengine

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"regexp"
	"time"

	"purr-chat-server/pkg/logger"
)

// ExecutionContext 端口化流程引擎的执行上下文
type ExecutionContext struct {
	Events       []SpecialModeEvent
	Connections  []FlowConnection
	PortValues   map[string]any    // "nodeID:portID" -> value
	Variables    map[string]string
	Session      *SpecialModeSession
	nameResolver map[string]string // "nodeName.portName" -> "nodeID:portID"
}

// NewExecutionContext 创建执行上下文
func NewExecutionContext(events []SpecialModeEvent, connections []FlowConnection, session *SpecialModeSession) *ExecutionContext {
	ctx := &ExecutionContext{
		Events:      events,
		Connections: connections,
		PortValues:  make(map[string]any),
		Variables:   session.Variables,
		Session:     session,
	}
	ctx.buildNameResolver()
	return ctx
}

// buildNameResolver 构建 nodeName.portName → nodeID:portID 的反向映射
// 用于支持 {name.port} 人类可读变量引用格式
func (ctx *ExecutionContext) buildNameResolver() {
	ctx.nameResolver = make(map[string]string)
	for _, evt := range ctx.Events {
		for _, port := range evt.Ports {
			if port.Direction == "output" {
				ctx.nameResolver[evt.Name+"."+port.Name] = evt.ID + ":" + port.ID
			}
		}
	}
}

// ExecuteFlow 从 trigger 节点开始执行事件链
func (ctx *ExecutionContext) ExecuteFlow(engineCtx context.Context, engine *BotEngine, input string) (string, error) {
	// 1. 找到 trigger 节点
	var triggerEvent *SpecialModeEvent
	for i := range ctx.Events {
		if ctx.Events[i].Type == "trigger" {
			triggerEvent = &ctx.Events[i]
			break
		}
	}
	if triggerEvent == nil {
		return "", fmt.Errorf("no trigger event found in flow")
	}

	// 2. 找到 trigger 的 exec 输出连接
	outConn := ctx.findOutputConnection(triggerEvent.ID, "trigger")
	if outConn == nil {
		return "", nil // trigger 没有连接任何节点
	}

	// 3. 沿控制流执行
	var finalReply string
	err := ctx.followControlFlow(engineCtx, engine, outConn.TargetNodeID, &finalReply)
	return finalReply, err
}

// followControlFlow 递归沿控制流执行
func (ctx *ExecutionContext) followControlFlow(engineCtx context.Context, engine *BotEngine, nodeID string, finalReply *string) error {
	event := ctx.findEvent(nodeID)
	if event == nil {
		return nil
	}

	switch event.Type {
	case "reply":
		// 读取 content 端口的值，替换变量后作为最终回复
		content := ctx.getPortValue(event.ID, "in_content")
		contentStr, _ := content.(string)
		contentStr = ctx.replaceVariables(contentStr)
		*finalReply = contentStr
		return nil

	case "llm", "builtin", "python", "template":
		// 处理节点：读取输入端口，执行，写入输出端口
		prompt := ctx.getPortValue(event.ID, "in_prompt")
		inputStr, _ := prompt.(string)

		output, err := engine.executeEvent(engineCtx, ctx.Session, event, inputStr)
		if err != nil {
			logger.ErrorfWithCaller("[FlowEngine] Event %s failed: %v", event.ID, err)
			output = ""
		}
		ctx.PortValues[event.ID+":out_output"] = output
		ctx.Session.EventOutputs[event.ID] = output

	case "if":
		var condBool bool

		if rawConditions, ok := event.Config["conditions"].([]any); ok && len(rawConditions) > 0 {
			// 新格式：多条件 + AND/OR 逻辑
			logic := "AND"
			if l, ok := event.Config["logic"].(string); ok {
				logic = strings.ToUpper(l)
			}

			if logic == "AND" {
				condBool = true
				for _, rc := range rawConditions {
					condMap, ok := rc.(map[string]any)
					if !ok {
						continue
					}
					leftStr := ctx.replaceVariables(getStringField(condMap, "left"))
					operator := getStringField(condMap, "operator")
					rightStr := ctx.replaceVariables(getStringField(condMap, "right"))
					if operator == "" {
						operator = "=="
					}
					if !ctx.evaluateOperatorCondition(leftStr, rightStr, operator) {
						condBool = false
						break
					}
				}
			} else {
				condBool = false
				for _, rc := range rawConditions {
					condMap, ok := rc.(map[string]any)
					if !ok {
						continue
					}
					leftStr := ctx.replaceVariables(getStringField(condMap, "left"))
					operator := getStringField(condMap, "operator")
					rightStr := ctx.replaceVariables(getStringField(condMap, "right"))
					if operator == "" {
						operator = "=="
					}
					if ctx.evaluateOperatorCondition(leftStr, rightStr, operator) {
						condBool = true
						break
					}
				}
			}
		} else {
			// 向后兼容：旧单条件端口模式
			operator := "=="
			if op, ok := event.Config["operator"].(string); ok {
				operator = op
			}
			leftVal := ctx.getPortValue(event.ID, "in_left")
			rightVal := ctx.getPortValue(event.ID, "in_right")
			leftStr, _ := leftVal.(string)
			if leftStr == "" {
				if def, ok := event.Config["left_default"].(string); ok {
					leftStr = def
				}
			}
			rightStr, _ := rightVal.(string)
			if rightStr == "" {
				if def, ok := event.Config["right_default"].(string); ok {
					rightStr = def
				}
			}
			condBool = ctx.evaluateOperatorCondition(leftStr, rightStr, operator)
		}

		var nextPort string
		if condBool {
			nextPort = "out_true"
		} else {
			nextPort = "out_false"
		}
		conn := ctx.findOutputConnection(event.ID, nextPort)
		if conn != nil {
			return ctx.followControlFlow(engineCtx, engine, conn.TargetNodeID, finalReply)
		}
		return nil

	case "loop":
		maxIterations := 10
		if v, ok := event.Config["max_iterations"].(float64); ok {
			maxIterations = int(v)
		}

		for i := 0; i < maxIterations; i++ {
			// 注入循环迭代变量
			ctx.PortValues[event.ID+":loop_index"] = i
			ctx.PortValues[event.ID+":loop_iterations"] = maxIterations

			condition := ctx.getPortValue(event.ID, "in_condition")
			condBool, ok := condition.(bool)
			if !ok {
				condStr, _ := condition.(string)
				condBool = ctx.evaluateCondition(condStr)
			}
			if !condBool {
				break
			}

			bodyConn := ctx.findOutputConnection(event.ID, "out_body")
			if bodyConn == nil {
				break
			}
			if err := ctx.followControlFlow(engineCtx, engine, bodyConn.TargetNodeID, finalReply); err != nil {
				return err
			}
		}

		doneConn := ctx.findOutputConnection(event.ID, "out_done")
		if doneConn != nil {
			return ctx.followControlFlow(engineCtx, engine, doneConn.TargetNodeID, finalReply)
		}
		return nil

	case "switch":
		// Switch 节点：根据匹配值路由到对应分支
		inputVal := ctx.getPortValue(event.ID, "in_value")
		inputStr, _ := inputVal.(string)
		inputStr = strings.TrimSpace(inputStr)
		inputStr = ctx.replaceVariables(inputStr)

		var nextPort string
		if rawCases, ok := event.Config["cases"].([]interface{}); ok {
			matched := false
			for i, rc := range rawCases {
				caseMap, ok := rc.(map[string]interface{})
				if !ok {
					continue
				}
				caseValue, _ := caseMap["value"].(string)
				if caseValue != "" && inputStr == caseValue {
					nextPort = fmt.Sprintf("out_case_%d", i)
					matched = true
					break
				}
			}
			if !matched {
				nextPort = "out_default"
			}
		} else {
			nextPort = "out_default"
		}

		conn := ctx.findOutputConnection(event.ID, nextPort)
		if conn != nil {
			return ctx.followControlFlow(engineCtx, engine, conn.TargetNodeID, finalReply)
		}
		return nil

	case "merge":
		// Merge 节点：passthrough，直接沿 out_exec 继续执行

	case "tool":
		// Tool 节点：HTTP 请求
		method, _ := event.Config["method"].(string)
		if method == "" {
			method = "GET"
		}
		rawURL, _ := event.Config["url"].(string)
		rawURL = strings.TrimSpace(ctx.replaceVariables(rawURL))

		body := ""
		if bodyVal := ctx.getPortValue(event.ID, "in_body"); bodyVal != "" {
			body, _ = bodyVal.(string)
		}

		if rawURL != "" {
			output, status := ctx.executeHTTPRequest(engineCtx, method, rawURL, body, event.Config)
			ctx.PortValues[event.ID+":out_output"] = output
			ctx.PortValues[event.ID+":out_status"] = status
			ctx.Session.EventOutputs[event.ID] = output
		}

	case "dify":
		// Dify 节点：调用 Dify Workflow / Chatflow API
		apiBase := strings.TrimRight(getStringField(event.Config, "api_base"), "/")
		apiKey := getStringField(event.Config, "api_key")
		if apiBase == "" || apiKey == "" {
			ctx.PortValues[event.ID+":out_error"] = "api_base and api_key are required"
			break
		}

		appType := getStringField(event.Config, "app_type")
		if appType == "" {
			appType = "workflow"
		}

		// 解析输入变量映射
		inputsMapping := getStringField(event.Config, "inputs_mapping")
		inputs := ctx.resolveInputsMapping(inputsMapping)

		// 如果映射为空，尝试从 in_input 端口获取值
		if len(inputs) == 0 {
			if inputVal := ctx.getPortValue(event.ID, "in_input"); inputVal != "" {
				if inputStr, ok := inputVal.(string); ok && inputStr != "" {
					inputs = map[string]any{"query": inputStr}
				}
			}
		}

		// 根据应用类型选择端点并构造请求
		var endpoint string
		var reqBody map[string]any

		switch appType {
		case "chatflow":
			endpoint = apiBase + "/v1/chat-messages"
			query, _ := inputs["query"].(string)
			reqBody = map[string]any{
				"query":         query,
				"inputs":        inputs,
				"response_mode": getStringField(event.Config, "response_mode"),
				"user":          "purrchat",
			}
			// Chatflow 多轮上下文：传递 conversation_id
			convKey := "dify_conversation_" + event.ID
			if convID, ok := ctx.Session.Variables[convKey]; ok && convID != "" {
				reqBody["conversation_id"] = convID
			}
		default: // workflow
			endpoint = apiBase + "/v1/workflows/run"
			reqBody = map[string]any{
				"inputs":        inputs,
				"response_mode": getStringField(event.Config, "response_mode"),
				"user":          "purrchat",
			}
		}

		reqBodyBytes, _ := json.Marshal(reqBody)

		// 设置认证 header
		difyConfig := make(map[string]any)
		for k, v := range event.Config {
			difyConfig[k] = v
		}
		difyConfig["headers"] = fmt.Sprintf(
			`{"Authorization":"Bearer %s","Content-Type":"application/json"}`,
			apiKey,
		)

		respBody, statusCode := ctx.executeHTTPRequest(engineCtx, "POST", endpoint, string(reqBodyBytes), difyConfig)

		// 错误处理
		if statusCode == 0 {
			ctx.PortValues[event.ID+":out_error"] = respBody
			ctx.PortValues[event.ID+":out_output"] = ""
			break
		}
		if statusCode != 200 {
			ctx.PortValues[event.ID+":out_error"] = fmt.Sprintf("HTTP %d: %s", statusCode, respBody)
			ctx.PortValues[event.ID+":out_output"] = ""
			break
		}

		// 解析响应 JSON
		var result map[string]any
		if err := json.Unmarshal([]byte(respBody), &result); err != nil {
			ctx.PortValues[event.ID+":out_error"] = "JSON parse error: " + err.Error()
			ctx.PortValues[event.ID+":out_output"] = ""
			break
		}

		// 检查工作流执行状态
		outputPath := getStringField(event.Config, "output_path")
		output, err := ctx.extractDifyOutput(result, outputPath)
		if err != nil {
			ctx.PortValues[event.ID+":out_error"] = err.Error()
			ctx.PortValues[event.ID+":out_output"] = ""
		} else {
			ctx.PortValues[event.ID+":out_output"] = output
			ctx.Session.EventOutputs[event.ID] = output
		}

		// Chatflow：保存 conversation_id 用于后续多轮
		if appType == "chatflow" {
			convKey := "dify_conversation_" + event.ID
			if convID := extractJSONPath(result, "data.conversation_id"); convID != nil {
				if idStr, ok := convID.(string); ok {
					ctx.Session.Variables[convKey] = idStr
				}
			}
		}

	case "n8n":
		// n8n 节点：调用 n8n Webhook
		webhookURL := getStringField(event.Config, "webhook_url")
		webhookURL = strings.TrimSpace(ctx.replaceVariables(webhookURL))
		if webhookURL == "" {
			ctx.PortValues[event.ID+":out_error"] = "webhook_url is required"
			break
		}

		method := getStringField(event.Config, "method")
		if method == "" {
			method = "POST"
		}

		// 获取请求体：优先 config.body，其次 in_input 端口
		body := getStringField(event.Config, "body")
		if body == "" {
			if inputVal := ctx.getPortValue(event.ID, "in_input"); inputVal != "" {
				body, _ = inputVal.(string)
			}
		}
		body = ctx.replaceVariables(body)

		// 构造认证 header
		authType := getStringField(event.Config, "auth_type")
		var authHeaders string
		switch authType {
		case "header":
			cred := getStringField(event.Config, "auth_credential")
			parts := strings.SplitN(cred, ":", 2)
			if len(parts) == 2 {
				h, _ := json.Marshal(map[string]string{parts[0]: parts[1]})
				authHeaders = string(h)
			}
		case "basic":
			cred := getStringField(event.Config, "auth_credential")
			parts := strings.SplitN(cred, ":", 2)
			if len(parts) == 2 {
				h, _ := json.Marshal(map[string]string{"Authorization": "Basic " + b64encode(parts[0]+":"+parts[1])})
				authHeaders = string(h)
			}
		}

		n8nConfig := make(map[string]any)
		for k, v := range event.Config {
			n8nConfig[k] = v
		}
		if authHeaders != "" {
			n8nConfig["headers"] = authHeaders
		}

		respBody, statusCode := ctx.executeHTTPRequest(engineCtx, method, webhookURL, body, n8nConfig)

		if statusCode == 0 {
			ctx.PortValues[event.ID+":out_error"] = respBody
			ctx.PortValues[event.ID+":out_output"] = ""
		} else if statusCode >= 400 {
			ctx.PortValues[event.ID+":out_error"] = fmt.Sprintf("HTTP %d: %s", statusCode, respBody)
			ctx.PortValues[event.ID+":out_output"] = ""
		} else {
			ctx.PortValues[event.ID+":out_output"] = respBody
			ctx.Session.EventOutputs[event.ID] = respBody
		}

	case "wait":
		// 等待节点：将最近的用户输入写入 out_user_input
		if len(ctx.Session.ContextBuffer) > 0 {
			userMsg := ctx.Session.ContextBuffer[len(ctx.Session.ContextBuffer)-1].Content
			ctx.PortValues[event.ID+":out_user_input"] = userMsg
		}

	case "history":
		// 历史消息节点：获取前 N 条消息并格式化为 prompt 字符串
		n := 20 // 默认 20 条
		if v, ok := event.Config["count"].(float64); ok && v > 0 {
			n = int(v)
		}
		// 从输入端口读取 N（如果有连接）
		if nVal := ctx.getPortValue(event.ID, "in_count"); nVal != "" {
			if nStr, ok := nVal.(string); ok {
				if parsed, err := strconv.Atoi(nStr); err == nil && parsed > 0 {
					n = parsed
				}
			}
		}
		historyPrompt := ctx.buildHistoryPrompt(n)
		ctx.PortValues[event.ID+":out_history"] = historyPrompt

	case "end":
		return nil

	default:
		// 未知类型尝试作为通用处理节点执行
		output, err := engine.executeEvent(engineCtx, ctx.Session, event, "")
		if err != nil {
			logger.ErrorfWithCaller("[FlowEngine] Unknown event type %s failed: %v", event.Type, err)
			return nil
		}
		ctx.PortValues[event.ID+":out_output"] = output
		ctx.Session.EventOutputs[event.ID] = output
	}

	// 对于处理节点，执行后继续沿 trigger 类型输出连接
	outConn := ctx.findOutputConnection(event.ID, "trigger")
	if outConn != nil {
		return ctx.followControlFlow(engineCtx, engine, outConn.TargetNodeID, finalReply)
	}
	return nil
}

// collectNodeOrder 收集 flow 的节点执行顺序（用于调试）
// 不执行事件，仅按控制流遍历收集节点 ID 列表
func (ctx *ExecutionContext) collectNodeOrder() []string {
	// 找到 trigger 节点
	var triggerEvent *SpecialModeEvent
	for i := range ctx.Events {
		if ctx.Events[i].Type == "trigger" {
			triggerEvent = &ctx.Events[i]
			break
		}
	}
	if triggerEvent == nil {
		return nil
	}

	var order []string
	visited := map[string]bool{}
	ctx.collectNodeOrderDFS(triggerEvent.ID, &order, visited)
	return order
}

// collectNodeOrderDFS 递归收集节点顺序
func (ctx *ExecutionContext) collectNodeOrderDFS(nodeID string, order *[]string, visited map[string]bool) {
	if visited[nodeID] {
		return
	}
	visited[nodeID] = true
	*order = append(*order, nodeID)

	event := ctx.findEvent(nodeID)
	if event == nil {
		return
	}

	// 根据节点类型决定遍历哪些输出端口
	var portIDs []string
	switch event.Type {
	case "if":
		portIDs = []string{"out_true", "out_false"}
	case "loop":
		portIDs = []string{"out_body", "out_done"}
	default:
		portIDs = []string{"trigger"}
	}

	for _, portID := range portIDs {
		conn := ctx.findOutputConnection(nodeID, portID)
		if conn != nil {
			ctx.collectNodeOrderDFS(conn.TargetNodeID, order, visited)
		}
	}
}

// findOutputConnection 找到指定节点指定输出端口的连接
func (ctx *ExecutionContext) findOutputConnection(nodeID string, portID string) *FlowConnection {
	for i := range ctx.Connections {
		if ctx.Connections[i].SourceNodeID == nodeID && ctx.Connections[i].SourcePortID == portID {
			return &ctx.Connections[i]
		}
	}
	return nil
}

// findEvent 按 ID 查找事件
func (ctx *ExecutionContext) findEvent(id string) *SpecialModeEvent {
	for i := range ctx.Events {
		if ctx.Events[i].ID == id {
			return &ctx.Events[i]
		}
	}
	return nil
}

// getPortValue 获取端口值
// 规则：1. 如果端口有直接存储的值，直接返回
//
//  2. 否则查找输入连接，从源端口获取值
//  3. 都没有时，对于 trigger/exec 类型端口返回 true，否则返回空字符串
func (ctx *ExecutionContext) getPortValue(nodeID, portID string) any {
	key := nodeID + ":" + portID

	// 如果已有值，直接返回
	if val, ok := ctx.PortValues[key]; ok {
		return val
	}

	// 查找输入连接，获取源端口值
	for _, conn := range ctx.Connections {
		if conn.TargetNodeID == nodeID && conn.TargetPortID == portID {
			srcKey := conn.SourceNodeID + ":" + conn.SourcePortID
			if val, ok := ctx.PortValues[srcKey]; ok {
				return val
			}
		}
	}

	// 对于 exec/trigger 输入端口，返回 true（触发信号）
	if strings.Contains(portID, "exec") || strings.Contains(portID, "trigger") {
		return true
	}

	return ""
}

// replaceVariables 替换模板中的变量引用
// 支持两种格式：
//   - {nodeName.portName} — 人类可读格式（优先解析）
//   - $nodeID:portID / $evtID.output / $variable — 机器格式（向后兼容）
func (ctx *ExecutionContext) replaceVariables(s string) string {
	// 替换 {name.port} 格式（最高优先级）
	if ctx.nameResolver != nil {
		re := regexp.MustCompile(`\{([^}]+)\}`)
		s = re.ReplaceAllStringFunc(s, func(match string) string {
			ref := match[1 : len(match)-1] // 去掉 { }
			if mappedKey, ok := ctx.nameResolver[ref]; ok {
				if val, ok := ctx.PortValues[mappedKey]; ok {
					if strVal, ok := val.(string); ok {
						return strVal
					}
				}
			}
			return match // 未找到映射，原样返回
		})
	}

	// 替换端口值引用 $nodeID:portID（向后兼容）
	for key, val := range ctx.PortValues {
		strVal, ok := val.(string)
		if ok {
			s = strings.ReplaceAll(s, "$"+key, strVal)
		}
	}

	// 替换事件输出引用 $evtID.output（向后兼容）
	for evtID, output := range ctx.Session.EventOutputs {
		s = strings.ReplaceAll(s, "$"+evtID+".output", output)
	}

	// 替换会话变量（向后兼容）
	for key, value := range ctx.Variables {
		s = strings.ReplaceAll(s, "$"+key, value)
	}

	return s
}

// evaluateCondition 求值条件表达式
// 支持格式：
//   - "true" / "false" — 字面值
//   - "$nodeID:portID" — 端口值引用（通过 replaceVariables 解析）
//   - "left == right" / "left != right" — 字符串比较
//   - 非空字符串视为 true
func (ctx *ExecutionContext) evaluateCondition(condition string) bool {
	if condition == "" {
		return false
	}

	// 替换变量引用
	resolved := ctx.replaceVariables(condition)
	resolved = strings.TrimSpace(resolved)

	// 布尔字面值
	if resolved == "true" {
		return true
	}
	if resolved == "false" {
		return false
	}

	// 不等于比较（必须在 == 之前检查，避免 != 被 == 截断）
	if strings.Contains(resolved, "!=") {
		parts := strings.SplitN(resolved, "!=", 2)
		return strings.TrimSpace(parts[0]) != strings.TrimSpace(parts[1])
	}

	// 等于比较
	if strings.Contains(resolved, "==") {
		parts := strings.SplitN(resolved, "==", 2)
		return strings.TrimSpace(parts[0]) == strings.TrimSpace(parts[1])
	}

	// 非空字符串视为 true
	return resolved != ""
}

// evaluateOperatorCondition 使用指定的运算符比较两个值
func (ctx *ExecutionContext) evaluateOperatorCondition(left, right, operator string) bool {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)

	switch operator {
	case "==":
		return left == right
	case "!=":
		return left != right
	case "contains":
		return strings.Contains(left, right)
	case ">":
		return compareNumeric(left, right) > 0
	case "<":
		return compareNumeric(left, right) < 0
	case "startsWith":
		return strings.HasPrefix(left, right)
	case "endsWith":
		return strings.HasSuffix(left, right)
	case "regex":
		matched, err := regexp.MatchString(right, left)
		return err == nil && matched
	default:
		return left == right
	}
}

// compareNumeric 尝试将两个字符串解析为 float64 进行比较
// 无法解析时回退到字符串比较
func compareNumeric(a, b string) int {
	af, aErr := strconv.ParseFloat(a, 64)
	bf, bErr := strconv.ParseFloat(b, 64)
	if aErr != nil || bErr != nil {
		// 无法解析为数字时回退到字符串比较
		if a < b {
			return -1
		} else if a > b {
			return 1
		}
		return 0
	}
	if af < bf {
		return -1
	} else if af > bf {
		return 1
	}
	return 0
}

// buildHistoryPrompt 从 ContextBuffer 中取最近 N 条消息，格式化为 prompt 字符串
// 图片消息用 "[图片]" 占位
func (ctx *ExecutionContext) buildHistoryPrompt(n int) string {
	buf := ctx.Session.ContextBuffer
	if len(buf) == 0 {
		return ""
	}

	start := 0
	if len(buf) > n {
		start = len(buf) - n
	}

	var sb strings.Builder
	for i := start; i < len(buf); i++ {
		msg := buf[i]
		roleLabel := msg.Role
		if roleLabel == "assistant" {
			roleLabel = "助手"
		} else if roleLabel == "system" {
			roleLabel = "系统"
		} else {
			roleLabel = "用户"
		}

		content := msg.Content
		if msg.MsgType == "image" {
			content = "[图片]"
		}

		sb.WriteString(roleLabel)
		sb.WriteString(": ")
		sb.WriteString(content)
		if i < len(buf)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// FindTriggerNode 在事件列表中查找 trigger 类型的事件
func FindTriggerNode(events []SpecialModeEvent) *SpecialModeEvent {
	for i := range events {
		if events[i].Type == "trigger" {
			return &events[i]
		}
	}
	return nil
}

// ValidatePortedFlow 验证端口化流程配置的合法性
func ValidatePortedFlow(events []SpecialModeEvent, connections []FlowConnection) error {
	trigger := FindTriggerNode(events)
	if trigger == nil {
		return fmt.Errorf("ported flow requires a trigger node")
	}

	// 验证所有连接引用的节点和端口是否存在
	eventMap := make(map[string]*SpecialModeEvent, len(events))
	portMap := make(map[string]map[string]EventPort) // nodeID -> portID -> port
	for i := range events {
		eventMap[events[i].ID] = &events[i]
		portMap[events[i].ID] = make(map[string]EventPort)
		for _, p := range events[i].Ports {
			portMap[events[i].ID][p.ID] = p
		}
	}

	for i, conn := range connections {
		if _, ok := eventMap[conn.SourceNodeID]; !ok {
			return fmt.Errorf("connection[%d]: source node %s not found", i, conn.SourceNodeID)
		}
		if _, ok := eventMap[conn.TargetNodeID]; !ok {
			return fmt.Errorf("connection[%d]: target node %s not found", i, conn.TargetNodeID)
		}
		if _, ok := portMap[conn.SourceNodeID][conn.SourcePortID]; !ok {
			return fmt.Errorf("connection[%d]: source port %s not found on node %s", i, conn.SourcePortID, conn.SourceNodeID)
		}
		if _, ok := portMap[conn.TargetNodeID][conn.TargetPortID]; !ok {
			return fmt.Errorf("connection[%d]: target port %s not found on node %s", i, conn.TargetPortID, conn.TargetNodeID)
		}
	}

	return nil
}

// executeHTTPRequest 执行 HTTP 请求并返回响应体和状态码
func (ctx *ExecutionContext) executeHTTPRequest(engineCtx context.Context, method, rawURL, body string, config map[string]interface{}) (string, int) {
	timeout := 30 * time.Second
	if v, ok := config["timeout_ms"].(float64); ok && v > 0 {
		timeout = time.Duration(v) * time.Millisecond
	}

	ctxHTTP, cancel := context.WithTimeout(engineCtx, timeout)
	defer cancel()

	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctxHTTP, method, rawURL, bodyReader)
	if err != nil {
		logger.ErrorfWithCaller("[FlowEngine] Failed to create request: %v", err)
		return fmt.Sprintf("request creation error: %v", err), 0
	}

	// 设置请求头
	if rawHeaders, ok := config["headers"].(string); ok && rawHeaders != "" {
		var headers map[string]string
		if err := json.Unmarshal([]byte(rawHeaders), &headers); err == nil {
			for k, v := range headers {
				req.Header.Set(k, v)
			}
		}
	}
	if req.Header.Get("Content-Type") == "" && body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.ErrorfWithCaller("[FlowEngine] HTTP request failed: %v", err)
		return fmt.Sprintf("request error: %v", err), 0
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.ErrorfWithCaller("[FlowEngine] Failed to read response body: %v", err)
		return fmt.Sprintf("read body error: %v", err), resp.StatusCode
	}

	return string(respBody), resp.StatusCode
}

// ─── Dify / n8n 辅助函数 ─────────────────────────────────────

// extractJSONPath 按 dot 分割的路径从嵌套 map 中取值
// 例如 extractJSONPath(data, "data.outputs.text") → data["data"]["outputs"]["text"]
func extractJSONPath(data map[string]any, path string) any {
	keys := strings.Split(path, ".")
	var current any = data
	for _, key := range keys {
		m, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = m[key]
	}
	return current
}

// resolveInputsMapping 解析 JSON 格式的输入变量映射，替换其中的变量引用
// 输入: {"query": "$triggerID:out_input"} → 输出: {"query": "实际值"}
func (ctx *ExecutionContext) resolveInputsMapping(rawMapping string) map[string]any {
	result := make(map[string]any)
	if rawMapping == "" {
		return result
	}
	var mapping map[string]any
	if err := json.Unmarshal([]byte(rawMapping), &mapping); err != nil {
		logger.InfofWithCaller("[FlowEngine] Failed to parse inputs_mapping JSON: %v", err)
		return result
	}
	for k, v := range mapping {
		if strVal, ok := v.(string); ok {
			result[k] = ctx.replaceVariables(strVal)
		} else {
			result[k] = v
		}
	}
	return result
}

// extractDifyOutput 从 Dify API 响应中提取有意义的输出
// outputPath 为空时，智能提取 data.outputs（单字段直接取值，多字段返回 JSON）
func (ctx *ExecutionContext) extractDifyOutput(data map[string]any, outputPath string) (string, error) {
	if outputPath != "" {
		val := extractJSONPath(data, outputPath)
		if val == nil {
			return "", fmt.Errorf("output_path '%s' returned nil", outputPath)
		}
		return anyToString(val), nil
	}

	// 默认行为：检查工作流状态
	if status, _ := extractJSONPath(data, "data.status").(string); status == "failed" {
		errMsg, _ := extractJSONPath(data, "data.error").(string)
		return "", fmt.Errorf("Dify workflow failed: %s", errMsg)
	}

	// 提取 data.outputs
	outputsRaw := extractJSONPath(data, "data.outputs")
	if outputsRaw == nil {
		return "", fmt.Errorf("data.outputs is nil")
	}

	outputs, ok := outputsRaw.(map[string]any)
	if !ok {
		return anyToString(outputsRaw), nil
	}

	// 单字段：直接返回值
	if len(outputs) == 1 {
		for _, v := range outputs {
			return anyToString(v), nil
		}
	}

	// 多字段：序列化为 JSON
	b, _ := json.Marshal(outputs)
	return string(b), nil
}

// b64encode 返回 base64 编码字符串
func b64encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

// anyToString 将任意类型转为字符串
func anyToString(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	case map[string]any, []any:
		b, _ := json.Marshal(val)
		return string(b)
	default:
		return fmt.Sprintf("%v", val)
	}
}
