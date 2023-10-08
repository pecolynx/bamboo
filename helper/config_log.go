package helper

import (
	"fmt"
	"log/slog"
	"os"
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

	handler := &bamboo.BambooLogHandler{Handler: slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})}

	bamboo.BambooLoggers[appName] = slog.New(handler)
	bamboo.BambooLoggers[bamboo.BambooWorkerLoggerContextKey] = slog.New(handler)
	bamboo.BambooLoggers[bamboo.BambooHeartbeatPublisherLoggerContextKey] = slog.New(handler)
	bamboo.BambooLoggers[bamboo.BambooRequestProducerLoggerContextKey] = slog.New(handler)
	bamboo.BambooLoggers[bamboo.BambooRequestConsumerLoggerContextKey] = slog.New(handler)
	bamboo.BambooLoggers[bamboo.BambooResultPublisherLoggerContextKey] = slog.New(handler)
	bamboo.BambooLoggers[bamboo.BambooResultSubscriberLoggerContextKey] = slog.New(handler)

	return nil
}
