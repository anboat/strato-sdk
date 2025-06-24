package logging

import (
	"github.com/anboat/strato-sdk/config/types"
	"os"
	"path/filepath"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	logger *zap.Logger
	once   sync.Once
)

// InitLoggerFromConfig initializes the logger from a configuration object.
// It sets up the logger with specified settings for level, file path, rotation, and console output.
func InitLoggerFromConfig(cfg *types.LogConfig) *zap.Logger {
	// Initialize logger
	Setup(
		cfg.Level,
		cfg.FilePath,
		cfg.MaxSize,
		cfg.MaxBackups,
		cfg.MaxAge,
		cfg.Compress,
		cfg.Env,
		cfg.EnableConsole,
	)
	return GetLogger()
}

// Setup initializes the logging system.
// It configures the logger's level, output path, rotation policies, and console logging based on the provided parameters.
// This function is designed to be called only once.
func Setup(level string, filePath string, maxSize, maxBackups, maxAge int, compress bool, env string, enableConsole bool) {
	once.Do(func() {
		// Ensure the directory exists
		if filePath != "" {
			dir := filepath.Dir(filePath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				panic(err)
			}
		}

		// Parse log level
		var zapLevel zapcore.Level
		switch level {
		case "debug":
			zapLevel = zapcore.DebugLevel
		case "info":
			zapLevel = zapcore.InfoLevel
		case "warn":
			zapLevel = zapcore.WarnLevel
		case "error":
			zapLevel = zapcore.ErrorLevel
		default:
			zapLevel = zapcore.InfoLevel
		}

		// Create Encoder configuration
		encoderConfig := zapcore.EncoderConfig{
			TimeKey:        "time",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}

		// Create core configurations
		var cores []zapcore.Core

		// Console output (enabled based on configuration)
		if enableConsole {
			consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)
			consoleWriter := zapcore.Lock(os.Stdout)
			cores = append(cores, zapcore.NewCore(consoleEncoder, consoleWriter, zapLevel))
		}

		// File output (if a file path is configured)
		if filePath != "" {
			// Configure log rotation
			fileWriter := zapcore.AddSync(&lumberjack.Logger{
				Filename:   filePath,
				MaxSize:    maxSize,    // Max size of a single file in MB
				MaxBackups: maxBackups, // Number of old log files to keep
				MaxAge:     maxAge,     // Days to retain old log files
				Compress:   compress,   // Whether to compress old logs
			})

			// Use a more friendly console encoder in development, and JSON in production
			var fileEncoder zapcore.Encoder
			if env == "development" || env == "dev" {
				fileEncoder = zapcore.NewConsoleEncoder(encoderConfig)
			} else {
				fileEncoder = zapcore.NewJSONEncoder(encoderConfig)
			}

			cores = append(cores, zapcore.NewCore(fileEncoder, fileWriter, zapLevel))
		}

		// Create the core
		core := zapcore.NewTee(cores...)

		// Create the logger
		logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	})
}

// GetLogger returns the logger instance.
// If the logger has not been initialized, it sets it up with a default configuration.
func GetLogger() *zap.Logger {
	if logger == nil {
		// If not yet initialized, use default configuration
		Setup("info", "", 0, 0, 0, false, "development", true)
	}
	return logger
}

// Debug logs a message at the debug level.
func Debug(msg string, fields ...zap.Field) {
	GetLogger().Debug(msg, fields...)
}

// Info logs a message at the info level.
func Info(msg string, fields ...zap.Field) {
	GetLogger().Info(msg, fields...)
}

// Warn logs a message at the warn level.
func Warn(msg string, fields ...zap.Field) {
	GetLogger().Warn(msg, fields...)
}

// Error logs a message at the error level.
func Error(msg string, fields ...zap.Field) {
	GetLogger().Error(msg, fields...)
}

// Fatal logs a message at the fatal level, which causes the program to exit.
func Fatal(msg string, fields ...zap.Field) {
	GetLogger().Fatal(msg, fields...)
}

// Sync flushes any buffered log entries.
func Sync() {
	if logger != nil {
		logger.Sync()
	}
}

// Sugar returns a `SugaredLogger` for more convenient logging methods.
func Sugar() *zap.SugaredLogger {
	return GetLogger().Sugar()
}

// Infof logs a formatted message at the info level.
func Infof(format string, args ...interface{}) {
	Sugar().Infof(format, args...)
}

// Warnf logs a formatted message at the warn level.
func Warnf(format string, args ...interface{}) {
	Sugar().Warnf(format, args...)
}

// Errorf logs a formatted message at the error level.
func Errorf(format string, args ...interface{}) {
	Sugar().Errorf(format, args...)
}

// Debugf logs a formatted message at the debug level.
func Debugf(format string, args ...interface{}) {
	Sugar().Debugf(format, args...)
}
