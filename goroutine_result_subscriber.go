package bamboo

import (
	"context"

	"google.golang.org/protobuf/proto"

	pb "github.com/pecolynx/bamboo/proto"
)

type goroutineBambooResultSubscriber struct {
	workerName string
	pubsubMap  GoroutineBambooPubSubMap
}

func NewGoroutineBambooResultSubscriber(ctx context.Context, workerName string, pubsubMap GoroutineBambooPubSubMap) BambooResultSubscriber {
	return &goroutineBambooResultSubscriber{
		workerName: workerName,
		pubsubMap:  pubsubMap,
	}
}

func (s *goroutineBambooResultSubscriber) Ping(ctx context.Context) error {
	return nil
}

func (s *goroutineBambooResultSubscriber) OpenSubscribeConnection(ctx context.Context, resultChannel string) (SubscribeFunc, CloseSubscribeConnectionFunc, error) {
	pubsub := s.pubsubMap.CreateChannel(resultChannel)

	subscribeFunc := func(ctx context.Context) (*pb.WorkerResponse, error) {
		// for {
		select {
		case <-ctx.Done():
			return nil, ErrContextCanceled
		case respBytes := <-pubsub:
			resp := pb.WorkerResponse{}
			if err := proto.Unmarshal(respBytes, &resp); err != nil {
				return nil, err
			}

			return &resp, nil
		}
		// }
	}

	closeSubscribeConnectionFunc := func(ctx context.Context) error {
		if err := s.pubsubMap.CloseSubscribeChannel(resultChannel); err != nil {
			return err
		}
		return nil
	}

	return subscribeFunc, closeSubscribeConnectionFunc, nil
}
