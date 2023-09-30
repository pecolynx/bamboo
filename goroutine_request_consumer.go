package bamboo

import (
	"context"

	"google.golang.org/protobuf/proto"

	"github.com/pecolynx/bamboo/internal"
	pb "github.com/pecolynx/bamboo/proto"
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
	logger := internal.FromContext(ctx)

	for {
		select {
		case <-ctx.Done():
			return nil, ErrContextCanceled
		case reqBytes := <-c.queue:
			req := pb.WorkerParameter{}
			if err := proto.Unmarshal(reqBytes, &req); err != nil {
				logger.Warnf("invalid parameter. failed to proto.Unmarshal. err: %w", err)
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
