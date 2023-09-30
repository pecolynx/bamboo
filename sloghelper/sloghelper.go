package sloghelper

import (
	"context"
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
	lock                              sync.Mutex
)

func init() {
	BambooLoggers = make(map[string]*slog.Logger)
	// BambooLoggers[BambooWorkerLoggerKey] = slog.New(&RequestIDHandler{Handler: slog.NewJSONHandler(os.Stdout, nil)})
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
	}

	return BambooLoggers[key]
}
