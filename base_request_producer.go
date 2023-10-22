package bamboo

import (
	"context"

	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/protobuf/proto"

	"github.com/pecolynx/bamboo/internal"
	pb "github.com/pecolynx/bamboo/proto"
)

type baseBambooRequestProducer struct {
}

func (p *baseBambooRequestProducer) Produce(ctx context.Context, resultChannel string, heartbeatIntervalMSec int, jobTimeoutMSec int, headers map[string]string, data []byte, workerName string, propagator propagation.TextMapPropagator, fn func(ctx context.Context, reqBytes []byte) error) error {
	ctx = WithLoggerName(ctx, BambooWorkerClientLoggerContextKey)
	carrier := propagation.MapCarrier{}

	spanCtx, span := tracer.Start(ctx, workerName)
	defer span.End()

	propagator.Inject(spanCtx, carrier)

	req := pb.WorkerParameter{
		Carrier:               carrier,
		Headers:               headers,
		ResultChannel:         resultChannel,
		HeartbeatIntervalMSec: int32(heartbeatIntervalMSec),
		JobTimeoutMSec:        int32(jobTimeoutMSec),
		Data:                  data,
	}

	reqBytes, err := proto.Marshal(&req)
	if err != nil {
		return internal.Errorf("proto.Marshal. err: %w", err)
	}

	return fn(spanCtx, reqBytes)
}
