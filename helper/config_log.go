package helper

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/pecolynx/bamboo"
)

type LogConfig struct {
	Level string `yaml:"level"`
}

func InitLog(appName bamboo.ContextKey, cfg *LogConfig) error {
	var logLevel slog.Level

	switch strings.ToLower(cfg.Level) {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		slog.Info(fmt.Sprintf("Unsupported log level: %s", cfg.Level))
		logLevel = slog.LevelWarn
	}

	bamboo.BambooLoggers[appName] = slog.New(bamboo.LogHandlers[logLevel])

	bamboo.SetLogLevel(logLevel)

	return nil
}
