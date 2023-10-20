package bamboo

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"

	pb "github.com/pecolynx/bamboo/proto"
	"go.opentelemetry.io/otel"
)

type ContextKey string

const (
	BambooWorkerLoggerContextKey             ContextKey = "BambooWorker"
	BambooWorkerClientLoggerContextKey       ContextKey = "BambooWorkerClient"
	BambooWorkerJobLoggerContextKey          ContextKey = "BambooWorkerJob"
	BambooHeartbeatPublisherLoggerContextKey ContextKey = "BambooHeartbeatPublisher"
	BambooRequestConsumerLoggerContextKey    ContextKey = "BambooRequestConsumer"
	BambooRequestProducerLoggerContextKey    ContextKey = "BambooRequestProducer"
	BambooResultPublisherLoggerContextKey    ContextKey = "BambooResultPublisher"
	BambooResultSubscriberLoggerContextKey   ContextKey = "BambooResultSubscriber"
	BambooMetricsServerContextKey            ContextKey = "BambooMetricsServer"
)

var (
	tracer = otel.Tracer("github.com/pecolynx/bamboo")
	keys   = []ContextKey{
		BambooWorkerLoggerContextKey,
		BambooWorkerClientLoggerContextKey,
		BambooWorkerJobLoggerContextKey,
		BambooHeartbeatPublisherLoggerContextKey,
		BambooRequestConsumerLoggerContextKey,
		BambooRequestProducerLoggerContextKey,
		BambooResultPublisherLoggerContextKey,
		BambooResultSubscriberLoggerContextKey,
		BambooMetricsServerContextKey,
	}
	BambooLoggers map[ContextKey]*slog.Logger = make(map[ContextKey]*slog.Logger)
	LogHandlers   map[slog.Level]slog.Handler = make(map[slog.Level]slog.Handler)
	lock          sync.Mutex
)

func init() {
	for _, key := range keys {
		BambooLoggers[key] = slog.New(&BambooLogHandler{Handler: slog.NewJSONHandler(os.Stdout, nil)})
	}

	for _, level := range []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError} {
		LogHandlers[level] = &BambooLogHandler{Handler: slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
		})}
	}
}

type BambooRequestProducer interface {
	Produce(ctx context.Context, resultChannel string, heartbeatIntervalMSec int, jobTimeoutMSec int, headers map[string]string, data []byte) error
	Close(ctx context.Context) error
}

type BambooRequestConsumer interface {
	Ping(ctx context.Context) error
	Consume(ctx context.Context) (*pb.WorkerParameter, error)
	Close(ctx context.Context) error
}

type SubscribeFunc func(ctx context.Context) (*pb.WorkerResponse, error)
type CloseSubscribeConnectionFunc func(ctx context.Context) error

type BambooResultSubscriber interface {
	Ping(ctx context.Context) error

	OpenSubscribeConnection(ctx context.Context, resultChannel string) (SubscribeFunc, CloseSubscribeConnectionFunc, error)
}

type BambooResultPublisher interface {
	Ping(ctx context.Context) error
	Publish(ctx context.Context, resultChannel string, responseType pb.ResponseType, data []byte) error
}

type BambooHeartbeatPublisher interface {
	Ping(ctx context.Context) error
	Run(ctx context.Context, resultChannel string, heartbeatIntervalMSec int, done <-chan interface{}, aborted <-chan interface{}) error
}

type BambooWorker interface {
	Run(ctx context.Context) error
}

type CreateBambooRequestConsumerFunc func(ctx context.Context) BambooRequestConsumer

type WorkerFunc func(ctx context.Context, headers map[string]string, data []byte, aborted <-chan interface{}) ([]byte, error)

type LogConfigFunc func(ctx context.Context, headers map[string]string) context.Context

var ErrConnectTimedout = errors.New("ConnectTimedout")
var ErrJobTimedout = errors.New("JobTimedout")
var ErrInternalError = errors.New("InternalError")
var ErrInvalidArgument = errors.New("InvalidArgument")
var ErrAborted = errors.New("Aborted")
var ErrContextCanceled = errors.New("ContextCanceled")

type ByteArreayResult struct {
	Value []byte
	Error error
}

func InitLogger(ctx context.Context) context.Context {
	for _, key := range keys {
		if _, ok := BambooLoggers[key]; ok {
			ctx = context.WithValue(ctx, key, BambooLoggers[key])
		}
	}
	return ctx
}

func SetLogLevel(logLevel slog.Level) {
	for _, key := range keys {
		if _, ok := BambooLoggers[key]; ok {
			setLogLevel(key, logLevel)
		}
	}
}

func setLogLevel(contextKey ContextKey, logLevel slog.Level) {
	BambooLoggers[contextKey] = slog.New(LogHandlers[logLevel])
}

func WithValue(ctx context.Context, key ContextKey, val any) context.Context {
	return context.WithValue(ctx, key, val)
}

func WithLoggerName(ctx context.Context, val ContextKey) context.Context {
	return context.WithValue(ctx, LoggerNameContextKey, string(val))
}

// GetLoggerFromContext Gets the logger from context
func GetLoggerFromContext(ctx context.Context, key ContextKey) *slog.Logger {
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
		BambooLoggers[key] = slog.New(&BambooLogHandler{Handler: slog.NewJSONHandler(os.Stdout, nil)})
		BambooLoggers[key].WarnContext(ctx, fmt.Sprintf("logger not found. logger: %s", key))
	}

	return BambooLoggers[key]
}
