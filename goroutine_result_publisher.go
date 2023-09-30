package bamboo

import (
	"context"

	"google.golang.org/protobuf/proto"

	"github.com/pecolynx/bamboo/internal"
	pb "github.com/pecolynx/bamboo/proto"
)

type goroutineRedisBambooResultPublisher struct {
	pubsubMap  GoroutineBambooPubSubMap
	workerPool chan chan internal.Job
}

func NewGoroutineBambooResultPublisher(pubsubMap GoroutineBambooPubSubMap) BambooResultPublisher {
	return &goroutineRedisBambooResultPublisher{
		pubsubMap:  pubsubMap,
		workerPool: make(chan chan internal.Job),
	}
}

func (p *goroutineRedisBambooResultPublisher) Ping(ctx context.Context) error {
	return nil
}

func (p *goroutineRedisBambooResultPublisher) Publish(ctx context.Context, resultChannel string, result []byte) error {
	resp := pb.WorkerResponse{Type: pb.ResponseType_DATA, Data: result}
	respBytes, err := proto.Marshal(&resp)
	if err != nil {
		return internal.Errorf("proto.Marshal. err: %w", err)
	}

	pubsub, err := p.pubsubMap.GetChannel(resultChannel)
	if err != nil {
		return err
	}
	pubsub <- respBytes

	return nil
}
