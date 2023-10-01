package bamboo

import (
	"context"
	"errors"

	pb "github.com/pecolynx/bamboo/proto"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("github.com/pecolynx/bamboo")

type BambooRequestProducer interface {
	Produce(ctx context.Context, resultChannel string, heartbeatIntervalSec int, jobTimeoutSec int, headers map[string]string, data []byte) error
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
	Publish(ctx context.Context, resultChannel string, result []byte) error
}

type BambooHeartbeatPublisher interface {
	Ping(ctx context.Context) error
	Run(ctx context.Context, resultChannel string, heartbeatIntervalSec int, done <-chan interface{}, aborted <-chan interface{}) error
}

type BambooWorker interface {
	Run(ctx context.Context) error
}

type CreateBambooRequestConsumerFunc func(ctx context.Context) BambooRequestConsumer

type WorkerFunc func(ctx context.Context, headers map[string]string, data []byte, aborted <-chan interface{}) ([]byte, error)

type LogConfigFunc func(ctx context.Context, headers map[string]string) context.Context

var ErrTimedout = errors.New("Timedout")
var ErrAborted = errors.New("Aborted")
var ErrContextCanceled = errors.New("ContextCanceled")

type ByteArreayResult struct {
	Value []byte
	Error error
}
