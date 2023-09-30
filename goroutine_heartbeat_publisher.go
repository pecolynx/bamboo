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

func (p *goroutineBambooHeartbeatPublisher) Ping(ctx context.Context) error {
	return nil
}

func (h *goroutineBambooHeartbeatPublisher) Run(ctx context.Context, resultChannel string, heartbeatIntervalSec int, done <-chan interface{}, aborted <-chan interface{}) error {
	logger := internal.FromContext(ctx)

	pubsub, err := h.pubsubMap.GetChannel(resultChannel)
	if err != nil {
		return err
	}

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
				logger.Debug("ctx.Done(). stop heartbeat loop")
				return
			case <-done:
				logger.Debug("done. stop heartbeat loop")
				return
			case <-aborted:
				logger.Debug("aborted. stop heartbeat loop")
				return
			case <-pulse.C:
				logger.Debug("pulse")
				pubsub <- h.heartbeatRespBytes
			}
		}
	}()

	return nil
}
