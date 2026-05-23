package database

import (
	"context"
	"errors"
	"fmt"
	"log"
	"runtime"
	"strings"
	"time"

	"wjfcm-go/internal/config"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type SQLLogger struct {
	level         logger.LogLevel
	logSQL        bool
	logSlowSQL    bool
	logErrorSQL   bool
	slowThreshold time.Duration
}

func NewSQLLogger(cfg config.DBConfig) logger.Interface {
	if !cfg.LogSQL && !cfg.LogSlowSQL && !cfg.LogErrorSQL {
		return logger.Default.LogMode(logger.Silent)
	}

	return SQLLogger{
		level:         parseLogLevel(cfg.LogLevel),
		logSQL:        cfg.LogSQL,
		logSlowSQL:    cfg.LogSlowSQL,
		logErrorSQL:   cfg.LogErrorSQL,
		slowThreshold: time.Duration(cfg.SlowThresholdMS) * time.Millisecond,
	}
}

func (l SQLLogger) LogMode(level logger.LogLevel) logger.Interface {
	l.level = level
	return l
}

func (l SQLLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.level >= logger.Info {
		log.Printf("[gorm] [info] "+msg, data...)
	}
}

func (l SQLLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.level >= logger.Warn {
		log.Printf("[gorm] [warn] "+msg, data...)
	}
}

func (l SQLLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.level >= logger.Error {
		log.Printf("[gorm] [error] "+msg, data...)
	}
}

func (l SQLLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.level <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()
	source := callerSource()
	rowsText := "-"
	if rows >= 0 {
		rowsText = fmt.Sprintf("%d", rows)
	}

	switch {
	case err != nil && l.logErrorSQL && !errors.Is(err, gorm.ErrRecordNotFound):
		log.Printf("[gorm] [error] [%s] [%.2fms] [rows:%s] %s | %v", source, float64(elapsed.Nanoseconds())/1e6, rowsText, sql, err)
	case l.logSlowSQL && l.slowThreshold > 0 && elapsed > l.slowThreshold:
		log.Printf("[gorm] [slow] [%s] [%.2fms] [rows:%s] %s", source, float64(elapsed.Nanoseconds())/1e6, rowsText, sql)
	case l.logSQL && l.level >= logger.Info:
		log.Printf("[gorm] [sql] [%s] [%.2fms] [rows:%s] %s", source, float64(elapsed.Nanoseconds())/1e6, rowsText, sql)
	}
}

func parseLogLevel(value string) logger.LogLevel {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "silent":
		return logger.Silent
	case "error":
		return logger.Error
	case "warn", "warning":
		return logger.Warn
	default:
		return logger.Info
	}
}

func callerSource() string {
	pcs := make([]uintptr, 32)
	n := runtime.Callers(4, pcs)
	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		if isApplicationFrame(frame.File) {
			return fmt.Sprintf("%s:%d", normalizePath(frame.File), frame.Line)
		}
		if !more {
			break
		}
	}
	return "unknown:0"
}

func isApplicationFrame(file string) bool {
	if file == "" {
		return false
	}
	file = strings.ReplaceAll(file, "\\", "/")
	if strings.Contains(file, "gorm.io/") ||
		strings.Contains(file, "database/sql") ||
		strings.Contains(file, "internal/database/sql_logger.go") ||
		strings.Contains(file, "runtime/") {
		return false
	}
	return strings.Contains(file, "/wjfcm-go/server/") ||
		strings.Contains(file, "/internal/handler/") ||
		strings.Contains(file, "/internal/service/") ||
		strings.Contains(file, "/internal/router/") ||
		strings.Contains(file, "/cmd/api/")
}

func normalizePath(file string) string {
	file = strings.ReplaceAll(file, "\\", "/")
	if index := strings.Index(file, "/wjfcm-go/server/"); index >= 0 {
		return file[index+len("/wjfcm-go/server/"):]
	}
	return file
}
