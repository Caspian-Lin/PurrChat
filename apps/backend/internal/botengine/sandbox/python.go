package sandbox

import (
    "context"
    "encoding/json"
    "fmt"
    "os/exec"
    "time"
)

// PythonSandbox Python 沙箱执行器
type PythonSandbox struct {
    maxTimeout time.Duration
}

// NewPythonSandbox 创建 Python 沙箱
func NewPythonSandbox() *PythonSandbox {
    return &PythonSandbox{
        maxTimeout: 5 * time.Second,
    }
}

// Execute 在 Python 沙箱中执行代码
func (s *PythonSandbox) Execute(ctx context.Context, code string, input map[string]any) (map[string]any, error) {
    // 1. AST 安全检查
    if err := CheckPythonSafety(code); err != nil {
        return nil, fmt.Errorf("safety check failed: %w", err)
    }

    // 2. 构造 Python wrapper 脚本
    script := buildPythonScript(code)

    // 3. 带超时执行
    execCtx, cancel := context.WithTimeout(ctx, s.maxTimeout)
    defer cancel()

    cmd := exec.CommandContext(execCtx, "python3", "-c", script)

    // 4. 通过 stdin 传递 JSON 输入
    inputJSON, err := json.Marshal(input)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal input: %w", err)
    }

    cmd.Stdin = stringsReader(string(inputJSON))

    // 5. 获取 stdout 输出
    output, err := cmd.Output()
    if err != nil {
        if execCtx.Err() == context.DeadlineExceeded {
            return nil, fmt.Errorf("python execution timed out after %v", s.maxTimeout)
        }
        return nil, fmt.Errorf("python execution failed: %w", err)
    }

    // 6. 解析 JSON 输出
    var result map[string]any
    if err := json.Unmarshal(output, &result); err != nil {
        return nil, fmt.Errorf("failed to parse python output: %w (output: %s)", err, string(output))
    }

    return result, nil
}

// buildPythonScript 构造 Python wrapper 脚本
func buildPythonScript(code string) string {
    // 注入安全环境和输入变量
    return fmt.Sprintf(`import json, sys

__input__ = json.loads(sys.stdin.read())
__output__ = {}

def run(context, input_data):
    %s

try:
    result = run(__input__, __input__)
    if isinstance(result, dict):
        __output__ = result
    else:
        __output__["result"] = str(result)
except Exception as e:
    __output__["error"] = str(e)

print(json.dumps(__output__))
`, code)
}

// stringsReader 简单的字符串 Reader
type stringsReader string

func (s stringsReader) Read(b []byte) (int, error) {
    if len(s) == 0 {
        return 0, fmt.Errorf("EOF")
    }
    n := copy(b, string(s))
    _ = s[n:] // consume bytes
    return n, nil
}

// ExecutePythonEvent 执行 Python 事件（供 specialmode.go 调用的便捷函数）
func ExecutePythonEvent(ctx context.Context, config map[string]any, input string) (string, error) {
    code, ok := config["code"].(string)
    if !ok || code == "" {
        return "", fmt.Errorf("python code is empty")
    }

    timeoutMs := 5000
    if v, ok := config["timeout_ms"].(float64); ok && v > 0 {
        timeoutMs = int(v)
    }

    sandbox := &PythonSandbox{
        maxTimeout: time.Duration(timeoutMs) * time.Millisecond,
    }

    inputMap := map[string]any{
        "input": input,
    }

    // 如果有 input_schema 中的额外字段，从 variables 中获取（由调用方传入）
    result, err := sandbox.Execute(ctx, code, inputMap)
    if err != nil {
        return "", err
    }

    // 提取输出
    if resultStr, ok := result["result"].(string); ok {
        return resultStr, nil
    }
    if errStr, ok := result["error"].(string); ok {
        return "", fmt.Errorf("python error: %s", errStr)
    }

    outputJSON, _ := json.Marshal(result)
    return string(outputJSON), nil
}
