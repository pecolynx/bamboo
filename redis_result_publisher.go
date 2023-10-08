package bamboo

import (
	"context"
	"encoding/base64"

	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/proto"

	"github.com/pecolynx/bamboo/internal"
	pb "github.com/pecolynx/bamboo/proto"
)

type redisRedisBambooResultPublisher struct {
	publisherOptions *redis.UniversalOptions
	workerPool       chan chan internal.Job
}

func NewRedisBambooResultPublisher(publisherOptions *redis.UniversalOptions) BambooResultPublisher {
	return &redisRedisBambooResultPublisher{
		publisherOptions: publisherOptions,
		workerPool:       make(chan chan internal.Job),
	}
}

func (p *redisRedisBambooResultPublisher) Ping(ctx context.Context) error {
	publisher := redis.NewUniversalClient(p.publisherOptions)
	defer publisher.Close()
	if _, err := publisher.Ping(ctx).Result(); err != nil {
		return internal.Errorf("publisher.Ping. err: %w", err)
	}

	return nil
}

func (p *redisRedisBambooResultPublisher) Publish(ctx context.Context, resultChannel string, result []byte) error {
	resp := pb.WorkerResponse{Type: pb.ResponseType_DATA, Data: result}
	respBytes, err := proto.Marshal(&resp)
	if err != nil {
		return internal.Errorf("proto.Marshal. err: %w", err)
	}

	respStr := base64.StdEncoding.EncodeToString(respBytes)

	publisher := redis.NewUniversalClient(p.publisherOptions)
	defer publisher.Close()

	if _, err := publisher.Publish(ctx, resultChannel, respStr).Result(); err != nil {
		return internal.Errorf("publisher.Publish. err: %w", err)
	}

	return nil
}
