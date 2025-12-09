package log

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *CustomLogger

type CustomLogger struct {
	*zap.Logger
	*zap.SugaredLogger
	Pid int
}

func (l *CustomLogger) Printf(format string, v ...interface{}) {
	l.SugaredLogger.Infof(format, v...)
}

// LoggerOptions 配置日志选项
type LoggerOptions struct {
	ModuleName string
	LogLevel   zapcore.Level
	MaxSize    int    // MB
	MaxBackups int    // 备份数量
	MaxAge     int    // 天数
	LogDir     string // 日志目录
}

// DefaultLoggerOptions 返回默认的日志配置选项
func DefaultLoggerOptions() LoggerOptions {
	return LoggerOptions{
		ModuleName: "app",
		LogLevel:   zapcore.InfoLevel,
		MaxSize:    500,
		MaxBackups: 200,
		MaxAge:     3,
		LogDir:     "logs",
	}
}

func InitLogger(options LoggerOptions) {
	Logger = NewLogger(options)
}

// NewLogger 使用给定的选项创建一个新的日志记录器
func NewLogger(options LoggerOptions) *CustomLogger {
	// 创建日志文件路径
	if options.LogDir != "" {
		if err := os.MkdirAll(options.LogDir, 0755); err != nil {
			panic(fmt.Sprintf("创建日志目录失败: %v", err))
		}
	}

	// 编码器配置
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:       "time",
		LevelKey:      "level",
		NameKey:       "logger",
		CallerKey:     "caller",
		MessageKey:    "msg",
		StacktraceKey: "stacktrace",
		LineEnding:    zapcore.DefaultLineEnding,
		EncodeLevel:   zapcore.CapitalColorLevelEncoder,
		EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
		},
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 核心配置
	var cores []zapcore.Core

	// 控制台输出
	if options.ModuleName == "" {
		consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)
		consoleCore := zapcore.NewCore(
			consoleEncoder,
			zapcore.AddSync(os.Stdout),
			options.LogLevel,
		)
		cores = append(cores, consoleCore)
	}

	// 文件输出 (如果有指定目录)
	if options.LogDir != "" && options.ModuleName != "" {
		filePrefix := options.ModuleName
		if filePrefix != "" {
			filePrefix += "."
		}

		// 创建自定义的日志轮转器
		rotator := &lumberjack.Logger{
			Filename:   filepath.Join(options.LogDir, fmt.Sprintf("%slog", filePrefix)),
			MaxSize:    options.MaxSize, // MB
			MaxBackups: options.MaxBackups,
			MaxAge:     options.MaxAge, // 天数
			Compress:   true,
			LocalTime:  true,
		}

		fileEncoder := zapcore.NewConsoleEncoder(encoderConfig)
		fileCore := zapcore.NewCore(
			fileEncoder,
			zapcore.AddSync(rotator),
			options.LogLevel,
		)
		cores = append(cores, fileCore)
	}

	// 合并核心
	core := zapcore.NewTee(cores...)

	// 创建日志记录器
	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	sugarLogger := logger.Sugar()

	return &CustomLogger{
		Logger:        logger,
		SugaredLogger: sugarLogger,
		Pid:           os.Getpid(),
	}
}

// 兼容旧接口
func NewPrivateLog(moduleName string, maxAge, compressSize int, logLevel uint32) {
	options := LoggerOptions{
		ModuleName: moduleName,
		LogLevel:   zapcore.InfoLevel,
		MaxSize:    compressSize,
		MaxBackups: maxAge * 24 * 5,
		MaxAge:     maxAge,
		LogDir:     "logs",
	}

	// 根据传入的数字设置日志级别
	switch logLevel {
	case 0:
		options.LogLevel = zap.DebugLevel
	case 1:
		options.LogLevel = zap.InfoLevel
	case 2:
		options.LogLevel = zap.WarnLevel
	case 3:
		options.LogLevel = zap.ErrorLevel
	default:
		options.LogLevel = zap.InfoLevel
	}

	Logger = NewLogger(options)
}

// 将args转换为带空格的字符串
func formatArgs(args ...interface{}) string {
	if len(args) == 0 {
		return ""
	}

	strArgs := make([]string, len(args))
	for i, arg := range args {
		strArgs[i] = fmt.Sprint(arg)
	}
	return strings.Join(strArgs, " ")
}

// Info 记录信息级别日志
func Info(trace string, args ...interface{}) {
	message := fmt.Sprintf("[opId:%s] %s", trace, formatArgs(args...))
	Logger.Logger.Info(message)
}

// Error 记录错误级别日志
func Error(trace string, args ...interface{}) {
	message := fmt.Sprintf("[opId:%s] %s", trace, formatArgs(args...))
	Logger.Logger.Error(message)
}

// Errorf 记录格式化的错误级别日志
func Errorf(trace string, format string, args ...interface{}) {
	message := fmt.Sprintf("[opId:%s] %s", trace, fmt.Sprintf(format, args...))
	Logger.Logger.Error(message)
}

// Debug 记录调试级别日志
func Debug(trace string, args ...interface{}) {
	message := fmt.Sprintf("[opId:%s] %s", trace, formatArgs(args...))
	Logger.Logger.Debug(message)
}

// Warn 记录警告级别日志
func Warn(trace string, args ...interface{}) {
	message := fmt.Sprintf("[opId:%s] %s", trace, formatArgs(args...))
	Logger.Logger.Warn(message)
}
