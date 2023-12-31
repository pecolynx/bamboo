package bamboo

import (
	"context"
	"encoding/base64"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"github.com/pecolynx/bamboo/internal"
)

type redisBambooRequestProducer struct {
	workerName      string
	producerOptions redis.UniversalOptions
	producerChannel string
	propagator      propagation.TextMapPropagator
}

func NewRedisBambooRequestProducer(ctx context.Context, workerName string, producerOptions redis.UniversalOptions, producerChannel string) BambooRequestProducer {
	return &redisBambooRequestProducer{
		workerName:      workerName,
		producerOptions: producerOptions,
		producerChannel: producerChannel,
		propagator:      otel.GetTextMapPropagator(),
	}
}

func (p *redisBambooRequestProducer) Produce(ctx context.Context, resultChannel string, heartbeatIntervalMSec int, jobTimeoutMSec int, headers map[string]string, data []byte) error {
	ctx = WithLoggerName(ctx, BambooRequestProducerLoggerContextKey)

	baseBambooRequestProducer := baseBambooRequestProducer{}
	return baseBambooRequestProducer.Produce(ctx, resultChannel, heartbeatIntervalMSec, jobTimeoutMSec, headers, data, p.workerName, p.propagator, func(ctx context.Context, reqBytes []byte) error {
		reqStr := base64.StdEncoding.EncodeToString(reqBytes)

		producer := redis.NewUniversalClient(&p.producerOptions)
		defer producer.Close()

		if _, err := producer.LPush(ctx, p.producerChannel, reqStr).Result(); err != nil {
			return internal.Errorf("producer.LPush. err: %w", err)
		}
		return nil
	})
}

func (p *redisBambooRequestProducer) Ping(ctx context.Context) error {
	ctx = WithLoggerName(ctx, BambooRequestProducerLoggerContextKey)

	producer := redis.NewUniversalClient(&p.producerOptions)
	defer producer.Close()

	if _, err := producer.Ping(ctx).Result(); err != nil {
		return internal.Errorf("producer.Ping. err: %w", err)
	}

	return nil
}

func (p *redisBambooRequestProducer) Close(ctx context.Context) error {
	return nil
}
