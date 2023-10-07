package bamboo

import (
	"context"
	"log/slog"

	"google.golang.org/protobuf/proto"

	"github.com/pecolynx/bamboo/internal"
	pb "github.com/pecolynx/bamboo/proto"
	"github.com/pecolynx/bamboo/sloghelper"
)

type goroutineBambooRequestConsumer struct {
	queue chan []byte
}

func NewGoroutineBambooRequestConsumer(queue chan []byte) BambooRequestConsumer {
	return &goroutineBambooRequestConsumer{
		queue: queue,
	}
}

func (c *goroutineBambooRequestConsumer) Consume(ctx context.Context) (*pb.WorkerParameter, error) {
	logger := sloghelper.FromContext(ctx, sloghelper.BambooRequestConsumerLoggerKey)
	ctx = context.WithValue(ctx, sloghelper.LoggerNameKey, sloghelper.BambooRequestConsumerLoggerKey)
	logger.DebugContext(ctx, "start consuming loop")

	for {
		select {
		case <-ctx.Done():
			return nil, internal.Errorf("ctx.Done(). stop consuming loop. err: %w", ErrContextCanceled)
		case reqBytes := <-c.queue:
			req := pb.WorkerParameter{}
			if err := proto.Unmarshal(reqBytes, &req); err != nil {
				logger.WarnContext(ctx, "invalid parameter. failed to proto.Unmarshal.", slog.Any("err", err))
				continue
			}

			return &req, nil
		}
	}

}

func (c *goroutineBambooRequestConsumer) Ping(ctx context.Context) error {
	return nil
}

func (c *goroutineBambooRequestConsumer) Close(ctx context.Context) error {
	return nil
}
