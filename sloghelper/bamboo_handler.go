package sloghelper

import (
	"context"
	"log/slog"
)

type BambooHandler struct {
	slog.Handler
}

var (
	RequestIDKey  = "bamboo_request_id"
	LoggerNameKey = "bamboo_logger_name"
)

type ContextKey string

const (
	RequestIDContextKey  ContextKey = "RequestIDContextKey"
	LoggerNameContextKey ContextKey = "LoggerNameContextKey"
)

func (h *BambooHandler) Handle(ctx context.Context, record slog.Record) error {
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

func (h *BambooHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &BambooHandler{
		Handler: h.Handler.WithAttrs(attrs),
	}
}

func (h *BambooHandler) WithGroup(name string) slog.Handler {
	return &BambooHandler{
		Handler: h.Handler.WithGroup(name),
	}
}
