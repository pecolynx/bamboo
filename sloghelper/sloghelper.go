package sloghelper

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
)

var (
	BambooLoggers                            map[ContextKey]*slog.Logger
	BambooWorkerLoggerContextKey             ContextKey = "BambooWorker"
	BambooWorkerClientLoggerContextKey       ContextKey = "BambooWorkerClient"
	BambooWorkerJobLoggerContextKey          ContextKey = "BambooWorkerJob"
	BambooHeartbeatPublisherLoggerContextKey ContextKey = "BambooHeartbeatPublisher"
	BambooRequestConsumerLoggerContextKey    ContextKey = "BambooRequestConsumer"
	BambooRequestProducerLoggerContextKey    ContextKey = "BambooRequestProducer"
	BambooResultPublisherLoggerContextKey    ContextKey = "BambooResultPublisher"
	BambooResultSubscriberLoggerContextKey   ContextKey = "BambooResultSubscriber"
	keys                                                = []ContextKey{
		BambooWorkerLoggerContextKey,
		BambooWorkerClientLoggerContextKey,
		BambooWorkerJobLoggerContextKey,
		BambooHeartbeatPublisherLoggerContextKey,
		BambooRequestConsumerLoggerContextKey,
		BambooRequestProducerLoggerContextKey,
		BambooResultPublisherLoggerContextKey,
		BambooResultSubscriberLoggerContextKey,
	}
	lock sync.Mutex
)

func init() {
	BambooLoggers = make(map[ContextKey]*slog.Logger)

	for _, key := range keys {
		BambooLoggers[key] = slog.New(&BambooHandler{Handler: slog.NewJSONHandler(os.Stdout, nil)})
	}

}

func Init(ctx context.Context) context.Context {
	for _, key := range keys {
		if _, ok := BambooLoggers[key]; ok {
			ctx = context.WithValue(ctx, key, BambooLoggers[key])
		}
	}
	return ctx
}

func WithValue(ctx context.Context, key ContextKey, val any) context.Context {
	return context.WithValue(ctx, key, val)
}

func WithLoggerName(ctx context.Context, val ContextKey) context.Context {
	return context.WithValue(ctx, LoggerNameContextKey, string(val))
}

// FromContext Gets the logger from context
func FromContext(ctx context.Context, key ContextKey) *slog.Logger {
	if ctx == nil {
		panic("nil context")
	}

	logger, ok := ctx.Value(key).(*slog.Logger)
	if ok {
		return logger
	}

	lock.Lock()
	defer lock.Unlock()

	if _, ok := BambooLoggers[key]; !ok {
		BambooLoggers[key] = slog.New(&BambooHandler{Handler: slog.NewJSONHandler(os.Stdout, nil)})
		BambooLoggers[key].WarnContext(ctx, fmt.Sprintf("logger not found. logger: %s", key))
	}

	return BambooLoggers[key]
}
