package bamboo

import (
	"context"
	"encoding/base64"
	"errors"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/proto"

	"github.com/pecolynx/bamboo/internal"
	pb "github.com/pecolynx/bamboo/proto"
	"github.com/pecolynx/bamboo/sloghelper"
)

type redisBambooRequestConsumer struct {
	consumer           redis.UniversalClient
	consumerChannel    string
	requestWaitTimeout time.Duration
}

func NewRedisBambooRequestConsumer(consumerOptions *redis.UniversalOptions, consumerChannel string, requestWaitTimeout time.Duration) BambooRequestConsumer {
	return &redisBambooRequestConsumer{
		consumer:           redis.NewUniversalClient(consumerOptions),
		consumerChannel:    consumerChannel,
		requestWaitTimeout: requestWaitTimeout,
	}
}

func (c *redisBambooRequestConsumer) Consume(ctx context.Context) (*pb.WorkerParameter, error) {
	logger := sloghelper.FromContext(ctx, sloghelper.BambooRequestConsumerLoggerContextKey)

	for {
		select {
		case <-ctx.Done():
			return nil, ErrContextCanceled
		default:
			m, err := c.consumer.BRPop(ctx, c.requestWaitTimeout, c.consumerChannel).Result()
			if errors.Is(err, redis.Nil) {
				continue
			} else if err != nil {
				return nil, internal.Errorf("consumer.BRPop. err: %w", err)
			}

			if len(m) == 1 {
				return nil, internal.Errorf("received invalid data. m[0]: %s, err: %w", m[0], err)
			} else if len(m) != 2 {
				return nil, internal.Errorf("received invalid data. err: %w", err)
			}

			reqStr := m[1]
			reqBytes, err := base64.StdEncoding.DecodeString(reqStr)
			if err != nil {
				logger.WarnContext(ctx, "invalid parameter. failed to base64.StdEncoding.DecodeString.", slog.Any("err", err))
				continue
			}

			req := pb.WorkerParameter{}
			if err := proto.Unmarshal(reqBytes, &req); err != nil {
				logger.WarnContext(ctx, "invalid parameter. failed to proto.Unmarshal.", slog.Any("err", err))
				continue
			}

			return &req, nil
		}
	}

}

func (c *redisBambooRequestConsumer) Ping(ctx context.Context) error {
	if _, err := c.consumer.Ping(ctx).Result(); err != nil {
		return internal.Errorf("producer.Ping. err: %w", err)
	}

	return nil
}

func (c *redisBambooRequestConsumer) Close(ctx context.Context) error {
	return c.consumer.Close()
}
