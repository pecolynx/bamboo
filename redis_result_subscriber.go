package bamboo

import (
	"context"
	"encoding/base64"

	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/proto"

	pb "github.com/pecolynx/bamboo/proto"
)

type redisBambooResultSubscriber struct {
	workerName string
	subscriber redis.UniversalClient
}

func NewRedisBambooResultSubscriber(ctx context.Context, workerName string, subscriberConfig *redis.UniversalOptions) BambooResultSubscriber {
	subscriber := redis.NewUniversalClient(subscriberConfig)

	return &redisBambooResultSubscriber{
		workerName: workerName,
		subscriber: subscriber,
	}
}

func (s *redisBambooResultSubscriber) Ping(ctx context.Context) error {
	if _, err := s.subscriber.Ping(ctx).Result(); err != nil {
		return err
	}

	return nil
}

func (s *redisBambooResultSubscriber) OpenSubscribeConnection(ctx context.Context, resultChannel string) (SubscribeFunc, CloseSubscribeConnectionFunc, error) {
	pubsub := s.subscriber.Subscribe(ctx, resultChannel)

	subscribeFunc := func(ctx context.Context) (*pb.WorkerResponse, error) {
		msg, err := pubsub.ReceiveMessage(ctx)
		if err != nil {
			return nil, err
		}

		respBytes, err := base64.StdEncoding.DecodeString(msg.Payload)
		if err != nil {
			return nil, err
		}

		resp := pb.WorkerResponse{}
		if err := proto.Unmarshal(respBytes, &resp); err != nil {
			return nil, err
		}

		return &resp, nil
	}

	closeSubscribeConnectionFunc := func(ctx context.Context) error {
		return pubsub.Close()
	}

	return subscribeFunc, closeSubscribeConnectionFunc, nil
}
