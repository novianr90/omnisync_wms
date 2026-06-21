package logger

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type GormLogger struct {
	SlowThreshold time.Duration
}

func NewGormLogger() *GormLogger {
	return &GormLogger{
		SlowThreshold: 200 * time.Millisecond,
	}
}

func (l *GormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	return l
}

func (l *GormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	slog.InfoContext(ctx, msg, data...)
}

func (l *GormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	slog.WarnContext(ctx, msg, data...)
}

func (l *GormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	slog.ErrorContext(ctx, msg, data...)
}

func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		slog.ErrorContext(ctx, "database error",
			slog.String("error", err.Error()),
			slog.Int64("latency_ms", elapsed.Milliseconds()),
			slog.Int64("rows", rows),
			slog.String("sql", sql),
		)
		return
	}

	if l.SlowThreshold != 0 && elapsed > l.SlowThreshold {
		slog.WarnContext(ctx, "slow database query",
			slog.Int64("latency_ms", elapsed.Milliseconds()),
			slog.Int64("rows", rows),
			slog.String("sql", sql),
		)
		return
	}

	// Always log normal queries at debug level
	slog.DebugContext(ctx, "database query",
		slog.Int64("latency_ms", elapsed.Milliseconds()),
		slog.Int64("rows", rows),
		slog.String("sql", sql),
	)
}
