package applog

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"wjfcm-go/internal/config"

	"github.com/gin-gonic/gin"
)

func Configure(cfg config.Config) (func() error, error) {
	writer, closeLog, err := logWriter(cfg.Log.Channel)
	if err != nil {
		return func() error { return nil }, err
	}

	log.SetOutput(writer)
	log.SetFlags(log.LstdFlags)
	gin.DefaultWriter = writer
	gin.DefaultErrorWriter = writer

	return closeLog, nil
}

func logWriter(channel string) (io.Writer, func() error, error) {
	switch strings.ToLower(strings.TrimSpace(channel)) {
	case "", "stack", "stdout", "console":
		return os.Stdout, func() error { return nil }, nil
	case "stderr":
		return os.Stderr, func() error { return nil }, nil
	case "single", "file":
		return openLogFile(filepath.Join("storage", "logs", "wjfcm-go.log"))
	case "daily":
		name := "wjfcm-go-" + time.Now().Format("2006-01-02") + ".log"
		return openLogFile(filepath.Join("storage", "logs", name))
	case "null", "discard", "none":
		return io.Discard, func() error { return nil }, nil
	default:
		return os.Stdout, func() error { return nil }, nil
	}
}

func openLogFile(path string) (io.Writer, func() error, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return os.Stdout, func() error { return nil }, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return os.Stdout, func() error { return nil }, err
	}
	return file, file.Close, nil
}
