package bamboo

import (
	"context"
	"encoding/base64"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/proto"

	"github.com/pecolynx/bamboo/internal"
	pb "github.com/pecolynx/bamboo/proto"
	"github.com/pecolynx/bamboo/sloghelper"
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
	publisher := redis.NewUniversalClient(p.publisherOptions)
	defer publisher.Close()
	if _, err := publisher.Ping(ctx).Result(); err != nil {
		return internal.Errorf("publisher.Ping. err: %w", err)
	}

	return nil
}

func (h *redisBambooHeartbeatPublisher) Run(ctx context.Context, resultChannel string, heartbeatIntervalSec int, done <-chan interface{}, aborted <-chan interface{}) error {
	logger := sloghelper.FromContext(ctx, sloghelper.BambooHeartbeatPublisherLoggerKey)
	ctx = context.WithValue(ctx, sloghelper.LoggerNameKey, sloghelper.BambooHeartbeatPublisherLoggerKey)

	if heartbeatIntervalSec == 0 {
		logger.DebugContext(ctx, "heartbeat is disabled because heartbeatIntervalSec is zero.")
		return nil
	}

	heartbeatInterval := time.Duration(heartbeatIntervalSec) * time.Second
	pulse := time.NewTicker(heartbeatInterval)

	go func() {
		defer func() {
			logger.DebugContext(ctx, "stop heartbeat loop")
			pulse.Stop()
		}()

		for {
			select {
			case <-ctx.Done():
				logger.DebugContext(ctx, "ctx.Done(). stop SendingPulse...")
				return
			case <-done:
				logger.DebugContext(ctx, "done. stop SendingPulse...")
				return
			case <-aborted:
				logger.DebugContext(ctx, "aborted. stop SendingPulse...")
				return
			case <-pulse.C:
				logger.DebugContext(ctx, "pulse")
				publisher := redis.NewUniversalClient(h.publisherOptions)
				defer publisher.Close()

				if _, err := publisher.Publish(ctx, resultChannel, h.heartbeatRespStr).Result(); err != nil {
					logger.ErrorContext(ctx, "publisher.Publish", slog.Any("err", err))
				}
			}
		}
	}()

	return nil
}
