package bamboo

import (
	"context"

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
	ctx = WithLoggerName(ctx, BambooHeartbeatPublisherLoggerContextKey)

	pubsub, err := h.pubsubMap.GetChannel(resultChannel)
	if err != nil {
		return internal.Errorf("pubsubMap.GetChannel. resultChannel: %s, err: %w", resultChannel, err)
	}

	baseHeartbeatPublisher := baseHeartbeatPublisher{}
	baseHeartbeatPublisher.Run(ctx, heartbeatIntervalMSec, done, aborted, func() {
		pubsub <- h.heartbeatRespBytes
	})

	return nil
}
