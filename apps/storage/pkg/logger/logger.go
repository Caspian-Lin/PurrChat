package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

var (
	InfoLogger      *log.Logger
	ErrorLogger     *log.Logger
	fileLogger      *fileLoggerImpl
	logConfig       *LogConfig
	lineCount       int
	fileMutex       sync.Mutex
	consoleDisabled bool
)

// LogConfig 日志配置
type LogConfig struct {
	Directory string
	MaxFiles  int
	MaxLines  int
}

type fileLoggerImpl struct {
	infoFile  *os.File
	errorFile *os.File
}

// getCallerInfo 获取调用者信息（文件名和行号）
func getCallerInfo(skip int) string {
	_, file, line, ok := runtime.Caller(skip + 1)
	if !ok {
		return "unknown:0"
	}
	// 获取文件名的最后一部分（不含路径）
	parts := strings.Split(file, "/")
	filename := parts[len(parts)-1]
	return fmt.Sprintf("%s:%d", filename, line)
}

// formatLog 格式化日志输出
func formatLog(level string, caller string, message string) string {
	timestamp := time.Now().Format("2006/01/02 15:04:05")
	if caller != "" {
		return fmt.Sprintf("[%s]%s [%s] %s", level, timestamp, caller, message)
	}
	return fmt.Sprintf("[%s]%s %s", level, timestamp, message)
}

// formatLogWithColor 格式化日志输出（带颜色，仅用于终端）
func formatLogWithColor(level string, caller string, message string) string {
	timestamp := time.Now().Format("2006/01/02 15:04:05")
	colorReset := "\033[0m"
	levelColor := "\033[32m" // 绿色
	if level == "ERROR" {
		levelColor = "\033[31m" // 红色
	}

	if caller != "" {
		return fmt.Sprintf("%s[%s]%s%s %s %s", levelColor, level, colorReset, timestamp, caller, message)
	}
	return fmt.Sprintf("%s[%s]%s%s %s", levelColor, level, colorReset, timestamp, message)
}

// Init 初始化日志
func Init() {
	consoleDisabled = false
	initLoggers()
}

// InitWithConfig 使用配置初始化日志
func InitWithConfig(config *LogConfig) error {
	logConfig = config
	lineCount = 0
	consoleDisabled = false

	// 确保日志目录存在
	if err := os.MkdirAll(config.Directory, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	if err := initFileLogger(); err != nil {
		return err
	}

	initLoggers()
	return nil
}

// DisableConsoleOutput 禁用终端输出（用于Gin框架日志）
func DisableConsoleOutput() {
	consoleDisabled = true
	initLoggers()
}

// EnableConsoleOutput 启用终端输出
func EnableConsoleOutput() {
	consoleDisabled = false
	initLoggers()
}

// initLoggers 初始化日志记录器
func initLoggers() {
	var infoWriter io.Writer
	var errorWriter io.Writer

	if consoleDisabled {
		// 禁用终端输出
		if fileLogger != nil && fileLogger.infoFile != nil {
			infoWriter = fileLogger.infoFile
		} else {
			infoWriter = io.Discard
		}
		if fileLogger != nil && fileLogger.errorFile != nil {
			errorWriter = fileLogger.errorFile
		} else {
			errorWriter = io.Discard
		}
	} else {
		// 启用终端输出，使用自定义writer来处理颜色
		if fileLogger != nil {
			infoWriter = &logWriter{
				console: os.Stdout,
				file:    fileLogger.infoFile,
				isError: false,
			}
			errorWriter = &logWriter{
				console: os.Stderr,
				file:    fileLogger.errorFile,
				isError: true,
			}
		} else {
			infoWriter = &logWriter{
				console: os.Stdout,
				file:    nil,
				isError: false,
			}
			errorWriter = &logWriter{
				console: os.Stderr,
				file:    nil,
				isError: true,
			}
		}
	}

	// 创建日志记录器，不使用log.Lshortfile，因为我们在格式化时自己添加
	InfoLogger = log.New(infoWriter, "", 0)
	ErrorLogger = log.New(errorWriter, "", 0)
}

// logWriter 自定义writer，处理终端和文件的输出
type logWriter struct {
	console *os.File
	file    *os.File
	isError bool
}

func (w *logWriter) Write(p []byte) (n int, err error) {
	// 解析日志内容
	content := strings.TrimSpace(string(p))

	// 写入终端（带颜色）
	if w.console != nil {
		var level string
		if w.isError {
			level = "ERROR"
		} else {
			level = "INFO"
		}

		// 解析日志格式: [INFO]timestamp [caller] message 或 [INFO]timestamp message
		var caller string
		var message string

		// 提取日志级别
		if strings.HasPrefix(content, "[INFO]") {
			content = content[6:] // 移除 [INFO]
		} else if strings.HasPrefix(content, "[ERROR]") {
			content = content[7:] // 移除 [ERROR]
		}

		// 提取时间戳
		if len(content) >= 19 { // 格式: 2006/01/02 15:04:05
			content = strings.TrimSpace(content[19:])
		}

		// 检查是否有调用者信息
		if strings.HasPrefix(content, "[") {
			// 找到闭合的 ]
			idx := strings.Index(content, "]")
			if idx != -1 {
				caller = content[:idx+1]
				message = strings.TrimSpace(content[idx+1:])
			}
		} else {
			message = content
		}

		// 生成带颜色的日志
		colored := formatLogWithColor(level, caller, message)
		_, _ = w.console.WriteString(colored + "\n")
	}

	// 写入文件（不带颜色）
	if w.file != nil {
		_, _ = w.file.Write(p)
	}

	return len(p), nil
}

// initFileLogger 初始化文件日志记录器
func initFileLogger() error {
	if logConfig == nil {
		return nil
	}

	// 创建新的日志文件
	infoFile, err := createLogFile("info")
	if err != nil {
		return err
	}

	errorFile, err := createLogFile("error")
	if err != nil {
		infoFile.Close()
		return err
	}

	// 关闭旧的文件（如果存在）
	if fileLogger != nil {
		if fileLogger.infoFile != nil {
			fileLogger.infoFile.Close()
		}
		if fileLogger.errorFile != nil {
			fileLogger.errorFile.Close()
		}
	}

	fileLogger = &fileLoggerImpl{
		infoFile:  infoFile,
		errorFile: errorFile,
	}

	return nil
}

// createLogFile 创建日志文件
func createLogFile(logType string) (*os.File, error) {
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("%s/%s-%s.log", logConfig.Directory, timestamp, logType)

	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	return file, nil
}

// rotateLogFiles 轮转日志文件
func rotateLogFiles() error {
	if logConfig == nil || logConfig.MaxFiles <= 0 {
		return nil
	}

	// 获取日志目录中的所有日志文件
	files, err := os.ReadDir(logConfig.Directory)
	if err != nil {
		return err
	}

	// 收集所有日志文件并提取时间戳
	type fileInfo struct {
		name      string
		timestamp time.Time
	}

	var logFiles []fileInfo
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".log") {
			// 从文件名中提取时间戳，格式: 20060102-150405-info.log
			name := file.Name()
			parts := strings.Split(name, "-")
			if len(parts) >= 3 {
				dateStr := parts[0] + parts[1] // 20060102-150405
				timestamp, err := time.Parse("20060102150405", dateStr)
				if err == nil {
					logFiles = append(logFiles, fileInfo{
						name:      name,
						timestamp: timestamp,
					})
				}
			}
		}
	}

	// 如果文件数量超过限制，删除最旧的文件
	if len(logFiles) > logConfig.MaxFiles {
		// 按时间戳排序，最早的在前
		for i := 0; i < len(logFiles)-1; i++ {
			for j := i + 1; j < len(logFiles); j++ {
				if logFiles[i].timestamp.After(logFiles[j].timestamp) {
					logFiles[i], logFiles[j] = logFiles[j], logFiles[i]
				}
			}
		}

		// 删除最旧的文件
		for i := 0; i < len(logFiles)-logConfig.MaxFiles; i++ {
			filename := filepath.Join(logConfig.Directory, logFiles[i].name)
			os.Remove(filename)
		}
	}

	return nil
}

// checkAndRotateFile 检查并轮转当前日志文件
func checkAndRotateFile() {
	fileMutex.Lock()
	defer fileMutex.Unlock()

	if logConfig == nil || logConfig.MaxLines <= 0 {
		return
	}

	lineCount++
	if lineCount >= logConfig.MaxLines {
		// 重置行计数器
		lineCount = 0

		// 轮转日志文件
		if err := initFileLogger(); err != nil {
			ErrorLogger.Printf("Failed to rotate log file: %v", err)
		}

		// 清理旧文件
		if err := rotateLogFiles(); err != nil {
			ErrorLogger.Printf("Failed to rotate old log files: %v", err)
		}
	}
}

// Info 记录信息日志
func Info(v ...interface{}) {
	checkAndRotateFile()
	message := fmt.Sprint(v...)
	InfoLogger.Println(formatLog("INFO", "", message))
}

// Error 记录错误日志
func Error(v ...interface{}) {
	checkAndRotateFile()
	message := fmt.Sprint(v...)
	ErrorLogger.Println(formatLog("ERROR", "", message))
}

// Infof 记录格式化信息日志
func Infof(format string, v ...interface{}) {
	checkAndRotateFile()
	message := fmt.Sprintf(format, v...)
	InfoLogger.Println(formatLog("INFO", "", message))
}

// Errorf 记录格式化错误日志
func Errorf(format string, v ...interface{}) {
	checkAndRotateFile()
	message := fmt.Sprintf(format, v...)
	ErrorLogger.Println(formatLog("ERROR", "", message))
}

// InfoWithCaller 记录带调用者信息的信息日志
func InfoWithCaller(v ...interface{}) {
	checkAndRotateFile()
	caller := getCallerInfo(1)
	message := fmt.Sprint(v...)
	InfoLogger.Println(formatLog("INFO", caller, message))
}

// ErrorWithCaller 记录带调用者信息的错误日志
func ErrorWithCaller(v ...interface{}) {
	checkAndRotateFile()
	caller := getCallerInfo(1)
	message := fmt.Sprint(v...)
	ErrorLogger.Println(formatLog("ERROR", caller, message))
}

// InfofWithCaller 记录带调用者信息的格式化信息日志
func InfofWithCaller(format string, v ...interface{}) {
	checkAndRotateFile()
	caller := getCallerInfo(1)
	message := fmt.Sprintf(format, v...)
	InfoLogger.Println(formatLog("INFO", caller, message))
}

// ErrorfWithCaller 记录带调用者信息的格式化错误日志
func ErrorfWithCaller(format string, v ...interface{}) {
	checkAndRotateFile()
	caller := getCallerInfo(1)
	message := fmt.Sprintf(format, v...)
	ErrorLogger.Println(formatLog("ERROR", caller, message))
}
