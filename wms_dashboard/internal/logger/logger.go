package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"
)

func InitLogger() {
	logLevelEnv := strings.ToUpper(os.Getenv("LOG_LEVEL"))
	var level slog.Level
	switch logLevelEnv {
	case "DEBUG":
		level = slog.LevelDebug
	case "INFO":
		level = slog.LevelInfo
	case "WARN":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	logFile := os.Getenv("LOG_FILE")
	var writers []io.Writer
	writers = append(writers, os.Stdout)

	if logFile != "" {
		fileWriter := &lumberjack.Logger{
			Filename:   logFile,
			MaxSize:    50, // megabytes
			MaxBackups: 0,
			MaxAge:     7,  // days
			Compress:   true,
		}
		writers = append(writers, fileWriter)
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	handler := slog.NewJSONHandler(io.MultiWriter(writers...), opts)

	logger := slog.New(handler)
	slog.SetDefault(logger)
}
