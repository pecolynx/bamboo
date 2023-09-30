package bamboo

import (
	"context"
	"time"

	"google.golang.org/protobuf/proto"

	pb "github.com/pecolynx/bamboo/proto"
	"github.com/pecolynx/bamboo/sloghelper"
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

func (p *goroutineBambooHeartbeatPublisher) Ping(ctx context.Context) error {
	return nil
}

func (h *goroutineBambooHeartbeatPublisher) Run(ctx context.Context, resultChannel string, heartbeatIntervalSec int, done <-chan interface{}, aborted <-chan interface{}) error {
	logger := sloghelper.FromContext(ctx, sloghelper.BambooHeartbeatPublisherLoggerKey)
	ctx = context.WithValue(ctx, "logger_name", sloghelper.BambooHeartbeatPublisherLoggerKey)

	pubsub, err := h.pubsubMap.GetChannel(resultChannel)
	if err != nil {
		return err
	}

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
