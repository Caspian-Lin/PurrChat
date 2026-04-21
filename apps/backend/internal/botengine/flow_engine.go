package botengine

import (
	"context"
	"fmt"
	"strings"

	"purr-chat-server/pkg/logger"
)

// ExecutionContext 端口化流程引擎的执行上下文
type ExecutionContext struct {
	Events      []SpecialModeEvent
	Connections []FlowConnection
	PortValues  map[string]any // "nodeID:portID" -> value
	Variables   map[string]string
	Session     *SpecialModeSession
}

// NewExecutionContext 创建执行上下文
func NewExecutionContext(events []SpecialModeEvent, connections []FlowConnection, session *SpecialModeSession) *ExecutionContext {
	events = ensureEventPorts(events)
	return &ExecutionContext{
		Events:      events,
		Connections: connections,
		PortValues:  make(map[string]any),
		Variables:   session.Variables,
		Session:     session,
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
		// 没有 trigger 节点，回退到旧 BFS 模式
		return engine.executeEventChain(engineCtx, ctx.Session, input)
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
		condition := ctx.getPortValue(event.ID, "in_condition")
		condBool, ok := condition.(bool)
		if !ok {
			condStr, _ := condition.(string)
			condBool = ctx.evaluateCondition(condStr)
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

	case "wait":
		// 等待节点：将最近的用户输入写入 out_user_input
		if len(ctx.Session.ContextBuffer) > 0 {
			userMsg := ctx.Session.ContextBuffer[len(ctx.Session.ContextBuffer)-1].Content
			ctx.PortValues[event.ID+":out_user_input"] = userMsg
		}

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

// findOutputConnection 找到指定节点指定类型输出端口对应的连接
func (ctx *ExecutionContext) findOutputConnection(nodeID string, dataType string) *FlowConnection {
	// 找到源节点的对应输出端口
	var outPortID string
	for i := range ctx.Events {
		if ctx.Events[i].ID == nodeID {
			for _, port := range ctx.Events[i].Ports {
				if port.Direction == "output" && port.DataType == dataType {
					outPortID = port.ID
					break
				}
			}
			break
		}
	}
	if outPortID == "" {
		return nil
	}

	for i := range ctx.Connections {
		if ctx.Connections[i].SourceNodeID == nodeID && ctx.Connections[i].SourcePortID == outPortID {
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
func (ctx *ExecutionContext) replaceVariables(s string) string {
	// 替换端口值引用 $nodeID:portID
	for key, val := range ctx.PortValues {
		strVal, ok := val.(string)
		if ok {
			s = strings.ReplaceAll(s, "$"+key, strVal)
		}
	}

	// 替换事件输出引用 $evtID.output
	for evtID, output := range ctx.Session.EventOutputs {
		s = strings.ReplaceAll(s, "$"+evtID+".output", output)
	}

	// 替换会话变量
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

// ===== 端口兼容性 =====

// ensureEventPorts 确保事件有端口定义（向后兼容旧数据）
func ensureEventPorts(events []SpecialModeEvent) []SpecialModeEvent {
	for i := range events {
		if len(events[i].Ports) == 0 {
			events[i].Ports = getDefaultPortsForType(events[i].Type)
		}
	}
	return events
}

// getDefaultPortsForType 返回事件类型的默认端口
func getDefaultPortsForType(eventType string) []EventPort {
	switch eventType {
	case "trigger":
		return []EventPort{
			{ID: "out_exec", Name: "执行", DataType: "trigger", Direction: "output"},
		}
	case "end":
		return []EventPort{
			{ID: "in_exec", Name: "执行", DataType: "trigger", Direction: "input"},
		}
	case "reply":
		return []EventPort{
			{ID: "in_exec", Name: "执行", DataType: "trigger", Direction: "input"},
			{ID: "in_content", Name: "内容", DataType: "string", Direction: "input"},
		}
	case "if":
		return []EventPort{
			{ID: "in_exec", Name: "执行", DataType: "trigger", Direction: "input"},
			{ID: "in_condition", Name: "条件", DataType: "boolean", Direction: "input"},
			{ID: "out_true", Name: "真", DataType: "trigger", Direction: "output"},
			{ID: "out_false", Name: "假", DataType: "trigger", Direction: "output"},
		}
	case "loop":
		return []EventPort{
			{ID: "in_exec", Name: "执行", DataType: "trigger", Direction: "input"},
			{ID: "in_condition", Name: "条件", DataType: "boolean", Direction: "input"},
			{ID: "out_body", Name: "循环体", DataType: "trigger", Direction: "output"},
			{ID: "out_done", Name: "完成", DataType: "trigger", Direction: "output"},
		}
	case "wait":
		return []EventPort{
			{ID: "in_exec", Name: "执行", DataType: "trigger", Direction: "input"},
			{ID: "out_user_input", Name: "用户输入", DataType: "string", Direction: "output"},
			{ID: "out_exec", Name: "执行", DataType: "trigger", Direction: "output"},
		}
	default:
		// 处理节点（llm, builtin, python, template）默认端口
		return []EventPort{
			{ID: "in_exec", Name: "执行", DataType: "trigger", Direction: "input"},
			{ID: "in_prompt", Name: "输入", DataType: "string", Direction: "input"},
			{ID: "out_exec", Name: "执行", DataType: "trigger", Direction: "output"},
			{ID: "out_output", Name: "输出", DataType: "string", Direction: "output"},
		}
	}
}

// HasPortedFlow 检查 SpecialModeSpec 是否使用端口化连线
func HasPortedFlow(spec *SpecialModeSpec) bool {
	return spec != nil && len(spec.Connections) > 0
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
