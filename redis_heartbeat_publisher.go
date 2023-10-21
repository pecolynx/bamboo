package bamboo

import (
	"context"
	"encoding/base64"
	"log/slog"

	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/proto"

	"github.com/pecolynx/bamboo/internal"
	pb "github.com/pecolynx/bamboo/proto"
)

type redisBambooHeartbeatPublisher struct {
	publisherOptions *redis.UniversalOptions
	heartbeatRespStr string
}

func NewRedisBambooHeartbeatPublisher(publisherOptions *redis.UniversalOptions) BambooHeartbeatPublisher {
	heartbeatResp := pb.WorkerResponse{Type: pb.ResponseType_HEARTBEAT}
	heartbeatRespBytes, err := proto.Marshal(&heartbeatResp)
	if err != nil {
		panic(err)
	}

	heartbeatRespStr := base64.StdEncoding.EncodeToString(heartbeatRespBytes)

	return &redisBambooHeartbeatPublisher{
		publisherOptions: publisherOptions,
		heartbeatRespStr: heartbeatRespStr,
	}
}

func (p *redisBambooHeartbeatPublisher) Ping(ctx context.Context) error {
	ctx = WithLoggerName(ctx, BambooHeartbeatPublisherLoggerContextKey)

	publisher := redis.NewUniversalClient(p.publisherOptions)
	defer publisher.Close()
	if _, err := publisher.Ping(ctx).Result(); err != nil {
		return internal.Errorf("publisher.Ping. err: %w", err)
	}

	return nil
}

func (h *redisBambooHeartbeatPublisher) Run(ctx context.Context, resultChannel string, heartbeatIntervalMSec int, done <-chan interface{}, aborted <-chan interface{}) error {
	logger := GetLoggerFromContext(ctx, BambooHeartbeatPublisherLoggerContextKey)
	ctx = WithLoggerName(ctx, BambooHeartbeatPublisherLoggerContextKey)

	baseHeartbeatPublisher := baseHeartbeatPublisher{}
	baseHeartbeatPublisher.Run(ctx, heartbeatIntervalMSec, done, aborted, func() {
		publisher := redis.NewUniversalClient(h.publisherOptions)
		defer publisher.Close()

		if _, err := publisher.Publish(ctx, resultChannel, h.heartbeatRespStr).Result(); err != nil {
			logger.ErrorContext(ctx, "publisher.Publish", slog.Any("err", err))
		}
	})

	return nil
}
