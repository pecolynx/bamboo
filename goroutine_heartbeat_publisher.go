package bamboo

import (
	"context"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/pecolynx/bamboo/internal"
	pb "github.com/pecolynx/bamboo/proto"
)

type goroutineBambooHeartbeatPublisher struct {
	pubsubMap          GoroutineBambooPubSubMap
	heartbeatRespBytes []byte
}

func NewGoroutineBambooHeartbeatPublisher(pubsubMap GoroutineBambooPubSubMap) BambooHeartbeatPublisher {
	heartbeatResp := pb.WorkerResponse{Type: pb.ResponseType_HEARTBEAT}
	heartbeatRespBytes, err := proto.Marshal(&heartbeatResp)
	if err != nil {
		panic(err)
	}

	return &goroutineBambooHeartbeatPublisher{
		pubsubMap:          pubsubMap,
		heartbeatRespBytes: heartbeatRespBytes,
	}
}

func (h *goroutineBambooHeartbeatPublisher) Ping(ctx context.Context) error {
	return nil
}

func (h *goroutineBambooHeartbeatPublisher) Run(ctx context.Context, resultChannel string, heartbeatIntervalMSec int, done <-chan interface{}, aborted <-chan interface{}) error {
	logger := GetLoggerFromContext(ctx, BambooHeartbeatPublisherLoggerContextKey)
	ctx = WithLoggerName(ctx, BambooHeartbeatPublisherLoggerContextKey)

	pubsub, err := h.pubsubMap.GetChannel(resultChannel)
	if err != nil {
		return internal.Errorf("pubsubMap.GetChannel. resultChannel: %s, err: %w", resultChannel, err)
	}

	if heartbeatIntervalMSec == 0 {
		logger.DebugContext(ctx, "heartbeat is disabled because heartbeatIntervalMSec is zero.")
		return nil
	}

	heartbeatInterval := time.Duration(heartbeatIntervalMSec) * time.Millisecond
	pulse := time.NewTicker(heartbeatInterval)

	go func() {
		defer func() {
			logger.DebugContext(ctx, "stop heartbeat loop")
			pulse.Stop()
		}()

		for {
			select {
			case <-ctx.Done():
				logger.DebugContext(ctx, "ctx.Done(). stop heartbeat loop")
				return
			case <-done:
				logger.DebugContext(ctx, "done. stop heartbeat loop")
				return
			case <-aborted:
				logger.DebugContext(ctx, "aborted. stop heartbeat loop")
				return
			case <-pulse.C:
				logger.DebugContext(ctx, "pulse")
				pubsub <- h.heartbeatRespBytes
			}
		}
	}()

	return nil
}
