package tests

import (
	"os"
	"testing"

	"purr-chat-server/pkg/logger"
)

// TestLoggerBasic 测试基本的日志功能
func TestLoggerBasic(t *testing.T) {
	// 初始化日志
	logger.Init()

	// 测试基本日志功能
	logger.Info("This is a test info message")
	logger.Error("This is a test error message")
	logger.Infof("This is a formatted info message: %s", "test")
	logger.Errorf("This is a formatted error message: %d", 404)
}

// TestLoggerWithCaller 测试带调用者信息的日志功能
func TestLoggerWithCaller(t *testing.T) {
	// 初始化日志
	logConfig := &logger.LogConfig{
		Directory: "logs",
		MaxFiles:  5,
		MaxLines:  1000,
	}

	if err := logger.InitWithConfig(logConfig); err != nil {
		logger.Init()
		t.Fatalf("Failed to initialize file logger: %v", err)
	}

	// 测试带调用者信息的日志
	logger.Info("This is a test info message")
	logger.Error("This is a test error message")
	logger.Infof("This is a formatted info message: %s", "test")
	logger.Errorf("This is a formatted error message: %d", 404)

	logger.InfoWithCaller("This is a test info message with caller")
	logger.ErrorWithCaller("This is a test error message with caller")
	logger.InfofWithCaller("This is a formatted info message with caller: %s", "test")
	logger.ErrorfWithCaller("This is a formatted error message with caller: %d", 404)
}

// TestLoggerConsoleOutput 测试终端输出控制功能
func TestLoggerConsoleOutput(t *testing.T) {
	// 初始化日志（不使用文件日志）
	logger.Init()

	// 测试正常输出
	logger.Info("This should be visible in console")
	logger.Error("This error should be visible in console")

	// 禁用终端输出
	logger.DisableConsoleOutput()
	logger.Info("This should NOT be visible in console (only in file)")
	logger.Error("This error should NOT be visible in console (only in file)")

	// 重新启用终端输出
	logger.EnableConsoleOutput()
	logger.Info("This should be visible in console again")
	logger.Error("This error should be visible in console again")
}

// TestLoggerFileOutput 测试文件输出功能
func TestLoggerFileOutput(t *testing.T) {
	// 创建临时日志目录
	tempDir := os.TempDir()
	logDir := tempDir + "/purr-chat-test-logs"
	defer os.RemoveAll(logDir)

	logConfig := &logger.LogConfig{
		Directory: logDir,
		MaxFiles:  5,
		MaxLines:  1000,
	}

	if err := logger.InitWithConfig(logConfig); err != nil {
		t.Fatalf("Failed to initialize file logger: %v", err)
	}

	// 测试日志输出
	testMessage := "Test message for file output"
	logger.Info(testMessage)
	logger.InfoWithCaller("Test message with caller")

	// 注意：由于日志是异步写入文件的，我们无法立即验证文件内容
	// 这里只是测试日志函数不会崩溃
	t.Log("Logger file output test completed")
}
