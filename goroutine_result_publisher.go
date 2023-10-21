package bamboo

import (
	"context"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/pecolynx/bamboo/internal"
	pb "github.com/pecolynx/bamboo/proto"
)

type goroutineRedisBambooResultPublisher struct {
	pubsubMap         GoroutineBambooPubSubMap
	debugPublishDelay time.Duration
}

func NewGoroutineBambooResultPublisher(pubsubMap GoroutineBambooPubSubMap, debugPublishDelay time.Duration) BambooResultPublisher {
	return &goroutineRedisBambooResultPublisher{
		pubsubMap:         pubsubMap,
		debugPublishDelay: debugPublishDelay,
	}
}

func (p *goroutineRedisBambooResultPublisher) Ping(ctx context.Context) error {
	return nil
}

func (p *goroutineRedisBambooResultPublisher) Publish(ctx context.Context, resultChannel string, responseType pb.ResponseType, data []byte) error {
	ctx = WithLoggerName(ctx, BambooResultPublisherLoggerContextKey)

	// debug delay
	time.Sleep(p.debugPublishDelay)

	resp := pb.WorkerResponse{Type: responseType, Data: data}
	respBytes, err := proto.Marshal(&resp)
	if err != nil {
		return internal.Errorf("proto.Marshal. err: %w", err)
	}

	pubsub, err := p.pubsubMap.GetChannel(resultChannel)
	if err != nil {
		return internal.Errorf("pubsubMap.GetChannel. err: %w", err)
	}

	pubsub <- respBytes

	return nil
}
