package sloghelper

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
)

var (
	BambooLoggers                     map[string]*slog.Logger
	BambooWorkerLoggerKey             = "BambooWorker"
	BambooHeartbeatPublisherLoggerKey = "BambooHeartbeatPublisher"
	BambooRequestConsumerLoggerKey    = "BambooRequestConsumer"
	BambooRequestProducerLoggerKey    = "BambooRequestProducer"
	BambooResultPublisherLoggerKey    = "BambooResultPublisher"
	BambooResultSubscriberLoggerKey   = "BambooResultSubscriber"
	keys                              = []string{
		BambooWorkerLoggerKey,
		BambooHeartbeatPublisherLoggerKey,
		BambooRequestConsumerLoggerKey,
		BambooRequestProducerLoggerKey,
		BambooResultPublisherLoggerKey,
		BambooResultSubscriberLoggerKey,
	}
	lock sync.Mutex
)

func init() {
	BambooLoggers = make(map[string]*slog.Logger)

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

// FromContext Gets the logger from context
func FromContext(ctx context.Context, key string) *slog.Logger {
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
