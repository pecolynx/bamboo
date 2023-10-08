package bamboo

import (
	"context"
	"log/slog"
)

type BambooLogHandler struct {
	slog.Handler
}

var (
	RequestIDKey  = "bamboo_request_id"
	LoggerNameKey = "bamboo_logger_name"
)

const (
	RequestIDContextKey  ContextKey = "RequestIDContextKey"
	LoggerNameContextKey ContextKey = "LoggerNameContextKey"
)

func (h *BambooLogHandler) Handle(ctx context.Context, record slog.Record) error {
	requestID, ok := ctx.Value(RequestIDContextKey).(string)
	if ok {
		record.AddAttrs(slog.String(RequestIDKey, requestID))
	}

	loggerName, ok := ctx.Value(LoggerNameContextKey).(string)
	if ok {
		record.AddAttrs(slog.String(LoggerNameKey, loggerName))
	}

	return h.Handler.Handle(ctx, record)
}

func (h *BambooLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &BambooLogHandler{
		Handler: h.Handler.WithAttrs(attrs),
	}
}

func (h *BambooLogHandler) WithGroup(name string) slog.Handler {
	return &BambooLogHandler{
		Handler: h.Handler.WithGroup(name),
	}
}
