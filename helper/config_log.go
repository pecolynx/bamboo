package helper

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/pecolynx/bamboo/sloghelper"
)

type LogConfig struct {
	Level string `yaml:"level"`
}

func InitLog(appName string, cfg *LogConfig) error {
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

	handler := &sloghelper.BambooHandler{Handler: slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})}

	sloghelper.BambooLoggers[appName] = slog.New(handler)
	sloghelper.BambooLoggers[sloghelper.BambooWorkerLoggerKey] = slog.New(handler)
	sloghelper.BambooLoggers[sloghelper.BambooHeartbeatPublisherLoggerKey] = slog.New(handler)
	sloghelper.BambooLoggers[sloghelper.BambooRequestProducerLoggerKey] = slog.New(handler)
	sloghelper.BambooLoggers[sloghelper.BambooRequestConsumerLoggerKey] = slog.New(handler)
	sloghelper.BambooLoggers[sloghelper.BambooResultPublisherLoggerKey] = slog.New(handler)
	sloghelper.BambooLoggers[sloghelper.BambooResultSubscriberLoggerKey] = slog.New(handler)

	return nil
}
