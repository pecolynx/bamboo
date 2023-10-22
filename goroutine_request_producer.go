package bamboo

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

type goroutineBambooRequestProducer struct {
	workerName string
	propagator propagation.TextMapPropagator
	queue      chan<- []byte
}

func NewGoroutineBambooRequestProducer(ctx context.Context, workerName string, queue chan<- []byte) BambooRequestProducer {
	return &goroutineBambooRequestProducer{
		workerName: workerName,
		queue:      queue,
		propagator: otel.GetTextMapPropagator(),
	}
}

func (p *goroutineBambooRequestProducer) Produce(ctx context.Context, resultChannel string, heartbeatIntervalMSec int, jobTimeoutMSec int, headers map[string]string, data []byte) error {
	ctx = WithLoggerName(ctx, BambooWorkerClientLoggerContextKey)
	// carrier := propagation.MapCarrier{}

	// spanCtx, span := tracer.Start(ctx, p.workerName)
	// defer span.End()

	// p.propagator.Inject(spanCtx, carrier)

	// req := pb.WorkerParameter{
	// 	Carrier:               carrier,
	// 	Headers:               headers,
	// 	ResultChannel:         resultChannel,
	// 	HeartbeatIntervalMSec: int32(heartbeatIntervalMSec),
	// 	JobTimeoutMSec:        int32(jobTimeoutMSec),
	// 	Data:                  data,
	// }

	// reqBytes, err := proto.Marshal(&req)
	// if err != nil {
	// 	return internal.Errorf("proto.Marshal. err: %w", err)
	// }
	baseBambooRequestProducer := baseBambooRequestProducer{}
	return baseBambooRequestProducer.Produce(ctx, resultChannel, heartbeatIntervalMSec, jobTimeoutMSec, headers, data, p.workerName, p.propagator, func(ctx context.Context, reqBytes []byte) error {
		p.queue <- reqBytes
		return nil
	})
	// p.queue <- reqBytes

	// return nil
}

func (p *goroutineBambooRequestProducer) Ping(ctx context.Context) error {
	return nil
}

func (p *goroutineBambooRequestProducer) Close(ctx context.Context) error {
	return nil
}
