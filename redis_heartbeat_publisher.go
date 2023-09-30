package bamboo

import (
	"context"
	"encoding/base64"
	"time"

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
	publisher := redis.NewUniversalClient(p.publisherOptions)
	defer publisher.Close()
	if _, err := publisher.Ping(ctx).Result(); err != nil {
		return internal.Errorf("publisher.Ping. err: %w", err)
	}

	return nil
}

func (h *redisBambooHeartbeatPublisher) Run(ctx context.Context, resultChannel string, heartbeatIntervalSec int, done <-chan interface{}, aborted <-chan interface{}) error {
	logger := internal.FromContext(ctx)

	if heartbeatIntervalSec == 0 {
		logger.Debug("heartbeat is disabled because heartbeatIntervalSec is zero.")
		return nil
	}

	heartbeatInterval := time.Duration(heartbeatIntervalSec) * time.Second
	pulse := time.NewTicker(heartbeatInterval)

	go func() {
		defer func() {
			logger.Debug("stop heartbeat loop")
			pulse.Stop()
		}()

		for {
			select {
			case <-ctx.Done():
				logger.Debug("ctx.Done(). stop SendingPulse...")
				return
			case <-done:
				logger.Debug("done. stop SendingPulse...")
				return
			case <-aborted:
				logger.Debug("aborted. stop SendingPulse...")
				return
			case <-pulse.C:
				logger.Debug("pulse")
				publisher := redis.NewUniversalClient(h.publisherOptions)
				defer publisher.Close()

				if _, err := publisher.Publish(ctx, resultChannel, h.heartbeatRespStr).Result(); err != nil {
					logger.Errorf("err: %w", err)
				}
			}
		}
	}()

	return nil
}
