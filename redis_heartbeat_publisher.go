package bamboo

import (
	"context"
	"encoding/base64"
	"time"

	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/proto"

	"github.com/pecolynx/bamboo/internal"
)

type RedisBambooHeartbeatPublisher interface {
	Run(ctx context.Context)
}

type redisBambooHeartbeatPublisher struct {
	publisherOptions     *redis.UniversalOptions
	resultChannel        string
	heartbeatIntervalSec int
	done                 <-chan interface{}
	aborted              <-chan interface{}
	heartbeatRespStr     string
}

func NewRedisBambooHeartbeatPublisher(publisherOptions *redis.UniversalOptions, resultChannel string, heartbeatIntervalSec int, done <-chan interface{}, aborted <-chan interface{}) RedisBambooHeartbeatPublisher {
	heartbeatResp := WorkerResponse{Type: ResponseType_HEARTBEAT}
	heartbeatRespBytes, err := proto.Marshal(&heartbeatResp)
	if err != nil {
		panic(err)
	}

	heartbeatRespStr := base64.StdEncoding.EncodeToString(heartbeatRespBytes)

	return &redisBambooHeartbeatPublisher{
		publisherOptions:     publisherOptions,
		resultChannel:        resultChannel,
		heartbeatIntervalSec: heartbeatIntervalSec,
		done:                 done,
		aborted:              aborted,
		heartbeatRespStr:     heartbeatRespStr,
	}
}

func (h *redisBambooHeartbeatPublisher) Run(ctx context.Context) {
	logger := internal.FromContext(ctx)

	if h.heartbeatIntervalSec == 0 {
		logger.Debug("heartbeat is disabled because heartbeatIntervalSec is zero.")
		return
	}

	heartbeatInterval := time.Duration(h.heartbeatIntervalSec) * time.Second
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
			case <-h.done:
				logger.Debug("done. stop SendingPulse...")
				return
			case <-h.aborted:
				logger.Debug("aborted. stop SendingPulse...")
				return
			case <-pulse.C:
				logger.Debug("pulse")
				publisher := redis.NewUniversalClient(h.publisherOptions)
				defer publisher.Close()

				if _, err := publisher.Publish(ctx, h.resultChannel, h.heartbeatRespStr).Result(); err != nil {
					logger.Errorf("err: %w", err)
				}
			}
		}
	}()
}
