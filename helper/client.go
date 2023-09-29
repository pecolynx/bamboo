package helper

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/pecolynx/bamboo"
	"github.com/pecolynx/bamboo/internal"
)

type BambooPrameter interface {
	ToBytes() ([]byte, error)
}

type WorkerClient interface {
	Produce(ctx context.Context, resultChannel string, heartbeatIntervalSec int, jobTimeoutSec int, headers map[string]string, data []byte) error
	Subscribe(ctx context.Context, resultChannel string, heartbeatIntervalSec int, jobTimeoutSec int) ([]byte, error)
	Ping(ctx context.Context) error
	Close(ctx context.Context)
}

type workerClient struct {
	rp bamboo.BambooRequestProducer
	rs bamboo.BambooResultSubscriber
}

func NewWorkerClient(rp bamboo.BambooRequestProducer, rs bamboo.BambooResultSubscriber) WorkerClient {
	return &workerClient{
		rp: rp,
		rs: rs,
	}
}

func (c *workerClient) Produce(ctx context.Context, resultChannel string, heartbeatIntervalSec int, jobTimeoutSec int, headers map[string]string, data []byte) error {
	return c.rp.Produce(ctx, resultChannel, heartbeatIntervalSec, jobTimeoutSec, headers, data)
}

func (c *workerClient) Subscribe(ctx context.Context, redisChannel string, heartbeatIntervalSec int, jobTimeoutSec int) ([]byte, error) {
	return c.rs.Subscribe(ctx, redisChannel, heartbeatIntervalSec, jobTimeoutSec)
}

func (c *workerClient) Ping(ctx context.Context) error {
	return c.rs.Ping(ctx)
}

func (c *workerClient) Close(ctx context.Context) {
	defer c.rp.Close(ctx)
}

type StandardClient struct {
	Clients map[string]WorkerClient
}

func (c *StandardClient) newRedisChannelString() (string, error) {
	redisChannel, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}

	return redisChannel.String(), nil
}

func (c *StandardClient) Call(ctx context.Context, clientName string, heartbeatIntervalSec int, jobTimeoutSec int, headers map[string]string, param []byte) ([]byte, error) {
	logger := internal.FromContext(ctx)

	redisChannel, err := c.newRedisChannelString()
	if err != nil {
		return nil, err
	}

	client, ok := c.Clients[clientName]
	if !ok {
		return nil, errors.New("NotFound" + clientName)
	}

	ch := make(chan bamboo.ByteArreayResult)
	go func() {
		defer func() {
			logger.Debug("END")
		}()

		resultBytes, err := client.Subscribe(ctx, redisChannel, heartbeatIntervalSec, jobTimeoutSec)
		if err != nil {
			ch <- bamboo.ByteArreayResult{Value: nil, Error: err}
			return
		}

		ch <- bamboo.ByteArreayResult{Value: resultBytes, Error: nil}
	}()

	if err := client.Produce(ctx, redisChannel, heartbeatIntervalSec, jobTimeoutSec, headers, param); err != nil {
		return nil, err
	}

	result := <-ch
	if result.Error != nil {
		return nil, result.Error
	}

	return result.Value, nil
}
