package bamboo

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/pecolynx/bamboo/internal"
	pb "github.com/pecolynx/bamboo/proto"
)

type WorkerJob interface {
	Run(ctx context.Context) error
}

type workerJob struct {
	carrier             propagation.MapCarrier
	workerFunc          WorkerFunc
	headers             map[string]string
	parameter           []byte
	resultPublisher     BambooResultPublisher
	resultChannel       string
	done                chan<- interface{}
	aborted             <-chan interface{}
	logConfigFunc       LogConfigFunc
	metricsEventHandler MetricsEventHandler
}

func NewWorkerJob(ctx context.Context, carrier propagation.MapCarrier, workerFunc WorkerFunc, headers map[string]string, parameter []byte, resultPublisher BambooResultPublisher, resultChannel string, done chan<- interface{}, aborted <-chan interface{}, logConfigFunc LogConfigFunc, metricsEventHandler MetricsEventHandler) WorkerJob {
	return &workerJob{
		carrier:             carrier,
		workerFunc:          workerFunc,
		headers:             headers,
		parameter:           parameter,
		resultPublisher:     resultPublisher,
		resultChannel:       resultChannel,
		done:                done,
		aborted:             aborted,
		logConfigFunc:       logConfigFunc,
		metricsEventHandler: metricsEventHandler,
	}
}

func (j *workerJob) Run(ctx context.Context) error {
	logger := GetLoggerFromContext(ctx, BambooWorkerJobLoggerContextKey)
	ctx = WithLoggerName(ctx, BambooWorkerJobLoggerContextKey)
	defer close(j.done)
	start := time.Now()

	j.metricsEventHandler.OnIncrNumRunningWorkers()
	defer j.metricsEventHandler.OnDecrNumRunningWorkers()

	propagator := otel.GetTextMapPropagator()
	ctx = propagator.Extract(ctx, j.carrier)

	attrs := make([]attribute.KeyValue, 0)
	for k, v := range j.headers {
		attrs = append(attrs, attribute.KeyValue{Key: attribute.Key(k), Value: attribute.StringValue(v)})
	}
	ctx = j.logConfigFunc(ctx, j.headers)

	logger.DebugContext(ctx, fmt.Sprintf("start job. resultChannel: %s", j.resultChannel))

	opts := []trace.SpanStartOption{
		trace.WithAttributes(attrs...),
		trace.WithSpanKind(trace.SpanKindConsumer),
	}
	ctx, span := tracer.Start(ctx, "Run", opts...)
	defer span.End()

	result, err := j.workerFunc(ctx, j.headers, j.parameter, j.aborted)
	if err != nil {
		j.metricsEventHandler.OnInternalErrorJob()
		if err := j.resultPublisher.Publish(ctx, j.resultChannel, pb.ResponseType_INTERNAL_ERROR, []byte(err.Error())); err != nil {
			j.metricsEventHandler.OnInternalErrorJob()
			return internal.Errorf("resultPublisher.Publish. err: %w", err)
		}

		return internal.Errorf("workerFunc. err: %w", err)
	}

	logger.DebugContext(ctx, fmt.Sprintf("publish result. resultChannel: %s", j.resultChannel))
	if err := j.resultPublisher.Publish(ctx, j.resultChannel, pb.ResponseType_DATA, result); err != nil {
		j.metricsEventHandler.OnInternalErrorJob()
		return internal.Errorf("resultPublisher.Publish. err: %w", err)
	}

	end := time.Now()
	logger.DebugContext(ctx, "done", slog.Float64("time", end.Sub(start).Seconds()))
	j.metricsEventHandler.OnSuccessJob()
	return nil
}
