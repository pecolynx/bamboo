package bamboo

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/protobuf/proto"

	"github.com/pecolynx/bamboo/internal"
	pb "github.com/pecolynx/bamboo/proto"
	"github.com/pecolynx/bamboo/sloghelper"
)

type goroutineBambooRequestProducer struct {
	workerName string
	propagator propagation.TextMapPropagator
	queue      chan []byte
}

func NewGoroutineBambooRequestProducer(ctx context.Context, workerName string, queue chan []byte) BambooRequestProducer {
	return &goroutineBambooRequestProducer{
		workerName: workerName,
		queue:      queue,
		propagator: otel.GetTextMapPropagator(),
	}
}

func (p *goroutineBambooRequestProducer) Produce(ctx context.Context, resultChannel string, heartbeatIntervalSec int, jobTimeoutSec int, headers map[string]string, data []byte) error {
	ctx = context.WithValue(ctx, sloghelper.LoggerNameKey, sloghelper.BambooWorkerClientLoggerKey)
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

	p.queue <- reqBytes

	return nil
}

func (p *goroutineBambooRequestProducer) Ping(ctx context.Context) error {
	return nil
}

func (p *goroutineBambooRequestProducer) Close(ctx context.Context) error {
	return nil
}
