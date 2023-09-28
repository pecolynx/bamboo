package bamboo

import (
	"context"
	"encoding/base64"

	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/proto"

	"github.com/pecolynx/bamboo/internal"
)

type redisRedisBambooResultPublisher struct {
	publisherOptions *redis.UniversalOptions
	workerFunc       WorkerFunc
	numWorkers       int
	logConfigFunc    LogConfigFunc
	workerPool       chan chan internal.Job
}

func NewRedisBambooResultPublisher() BambooResultPublisher {
	return &redisRedisBambooResultPublisher{}
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
	resp := WorkerResponse{Type: ResponseType_DATA, Data: result}
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
