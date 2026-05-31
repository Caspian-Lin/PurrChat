// Deprecated: Python 沙箱安全检查器。随 python.go 一起废弃。
package sandbox

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// CheckPythonSafety 检查 Python 代码安全性
// 使用 Python AST 模块进行静态分析
func CheckPythonSafety(code string) error {
	// 构建 AST 检查脚本
	script := fmt.Sprintf(`
import ast, json, sys

code = %q

forbidden_modules = %s
forbidden_builtins = %s

try:
    tree = ast.parse(code)
    issues = []

    for node in ast.walk(tree):
        if isinstance(node, ast.Import):
            for alias in node.names:
                if alias.name in forbidden_modules:
                    issues.append("forbidden_import:" + alias.name)
        elif isinstance(node, ast.ImportFrom):
            if node.module in forbidden_modules:
                issues.append("forbidden_import:" + node.module)
        elif isinstance(node, ast.Call):
            if isinstance(node.func, ast.Name) and node.func.id in forbidden_builtins:
                issues.append("forbidden_call:" + node.func.id)
            elif isinstance(node.func, ast.Attribute):
                if node.func.attr in ("exec", "eval", "__import__"):
                    issues.append("forbidden_call:" + node.func.attr)
        elif isinstance(node, ast.Attribute):
            if node.attr.startswith("__") and node.attr.endswith("__"):
                issues.append("forbidden_access:" + node.attr)

    if issues:
        print(json.dumps(issues))
        sys.exit(1)
    else:
        print(json.dumps([]))
except SyntaxError as e:
    print(json.dumps(["syntax_error:" + str(e)]))
    sys.exit(1)
except Exception as e:
    print(json.dumps(["check_error:" + str(e)]))
    sys.exit(1)
`, code, formatModuleList(forbiddenModules), formatModuleList(forbiddenBuiltins))

	cmd := exec.Command("python3", "-c", script)
	output, err := cmd.CombinedOutput()

	if err != nil {
		var issues []string
		if json.Unmarshal(output, &issues) == nil && len(issues) > 0 {
			for _, issue := range issues {
				if strings.HasPrefix(issue, "forbidden_import:") {
					return fmt.Errorf("forbidden import: %s", strings.TrimPrefix(issue, "forbidden_import:"))
				}
				if strings.HasPrefix(issue, "forbidden_call:") {
					return fmt.Errorf("forbidden call: %s", strings.TrimPrefix(issue, "forbidden_call:"))
				}
				if strings.HasPrefix(issue, "forbidden_access:") {
					return fmt.Errorf("forbidden access: %s", strings.TrimPrefix(issue, "forbidden_access:"))
				}
				if strings.HasPrefix(issue, "syntax_error:") {
					return fmt.Errorf("python syntax error: %s", strings.TrimPrefix(issue, "syntax_error:"))
				}
			}
		}
		return fmt.Errorf("python safety check failed: %s", string(output))
	}

	return nil
}

// forbiddenModules 禁止导入的模块
var forbiddenModules = map[string]bool{
	"os":              true,
	"subprocess":      true,
	"shutil":          true,
	"pathlib":         true,
	"socket":          true,
	"http":            true,
	"urllib":          true,
	"ctypes":          true,
	"importlib":       true,
	"signal":          true,
	"multiprocessing": true,
	"threading":       true,
	"webbrowser":      true,
}

// forbiddenBuiltins 禁止的内置函数调用
var forbiddenBuiltins = map[string]bool{
	"exec":       true,
	"eval":       true,
	"compile":    true,
	"__import__": true,
	"open":       true,
	"input":      true,
	"globals":    true,
	"locals":     true,
	"getattr":    true,
	"setattr":    true,
	"delattr":    true,
	"breakpoint": true,
}

// formatModuleList 将 map 格式化为 Python set 字面量字符串
func formatModuleList(m map[string]bool) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, fmt.Sprintf("%q", k))
	}
	return "{" + strings.Join(keys, ", ") + "}"
}
