package bamboo

import (
	"context"
	"encoding/base64"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/protobuf/proto"

	"github.com/pecolynx/bamboo/internal"
	pb "github.com/pecolynx/bamboo/proto"
)

type redisBambooRequestProducer struct {
	workerName      string
	producerOptions redis.UniversalOptions
	producerChannel string
	propagator      propagation.TextMapPropagator
}

func NewRedisBambooRequestProducer(ctx context.Context, workerName string, producerOptions redis.UniversalOptions, producerChannel string, propagator propagation.TextMapPropagator) BambooRequestProducer {
	return &redisBambooRequestProducer{
		workerName:      workerName,
		producerOptions: producerOptions,
		producerChannel: producerChannel,
		propagator:      propagator,
	}
}

func (p *redisBambooRequestProducer) Produce(ctx context.Context, resultChannel string, heartbeatIntervalSec int, jobTimeoutSec int, headers map[string]string, data []byte) error {
	// logger := sloghelper.FromContext(ctx)

	carrier := propagation.MapCarrier{}

	spanCtx, span := tracer.Start(ctx, p.workerName)
	defer span.End()

	p.propagator.Inject(spanCtx, carrier)

	req := pb.WorkerParameter{
		Carrier:              carrier,
		Headers:              headers,
		ResultChannel:        resultChannel,
		HeartbeatIntervalSec: int32(heartbeatIntervalSec),
		JobTimeoutSec:        int32(jobTimeoutSec),
		Data:                 data,
	}

	reqBytes, err := proto.Marshal(&req)
	if err != nil {
		return internal.Errorf("proto.Marshal. err: %w", err)
	}
	reqStr := base64.StdEncoding.EncodeToString(reqBytes)

	producer := redis.NewUniversalClient(&p.producerOptions)
	defer producer.Close()

	if _, err := producer.LPush(ctx, p.producerChannel, reqStr).Result(); err != nil {
		return internal.Errorf("producer.LPush. err: %w", err)
	}

	return nil
}

func (p *redisBambooRequestProducer) Ping(ctx context.Context) error {
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
