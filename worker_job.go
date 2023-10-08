package bamboo

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/pecolynx/bamboo/internal"
	"github.com/pecolynx/bamboo/sloghelper"
)

type WorkerJob interface {
	Run(ctx context.Context) error
}

type workerJob struct {
	carrier         propagation.MapCarrier
	workerFunc      WorkerFunc
	headers         map[string]string
	parameter       []byte
	resultPublisher BambooResultPublisher
	resultChannel   string
	done            chan<- interface{}
	aborted         <-chan interface{}
	logConfigFunc   LogConfigFunc
}

func NewWorkerJob(ctx context.Context, carrier propagation.MapCarrier, workerFunc WorkerFunc, headers map[string]string, parameter []byte, resultPublisher BambooResultPublisher, resultChannel string, done chan<- interface{}, aborted <-chan interface{}, logConfigFunc LogConfigFunc) WorkerJob {
	return &workerJob{
		carrier:         carrier,
		workerFunc:      workerFunc,
		headers:         headers,
		parameter:       parameter,
		resultPublisher: resultPublisher,
		resultChannel:   resultChannel,
		done:            done,
		aborted:         aborted,
		logConfigFunc:   logConfigFunc,
	}
}

func (j *workerJob) Run(ctx context.Context) error {
	logger := sloghelper.FromContext(ctx, sloghelper.BambooWorkerJobLoggerContextKey)
	ctx = sloghelper.WithLoggerName(ctx, sloghelper.BambooWorkerLoggerContextKey)
	defer close(j.done)

	logger.DebugContext(ctx, fmt.Sprintf("start job. resultChannel: %s", j.resultChannel))

	propagator := otel.GetTextMapPropagator()
	ctx = propagator.Extract(ctx, j.carrier)

	attrs := make([]attribute.KeyValue, 0)
	for k, v := range j.headers {
		attrs = append(attrs, attribute.KeyValue{Key: attribute.Key(k), Value: attribute.StringValue(v)})
	}
	ctx = j.logConfigFunc(ctx, j.headers)

	opts := []trace.SpanStartOption{
		trace.WithAttributes(attrs...),
		trace.WithSpanKind(trace.SpanKindConsumer),
	}
	ctx, span := tracer.Start(ctx, "Run", opts...)
	defer span.End()

	result, err := j.workerFunc(ctx, j.headers, j.parameter, j.aborted)
	if err != nil {
		return internal.Errorf("workerFunc. err: %w", err)
	}

	logger.DebugContext(ctx, fmt.Sprintf("publish result. resultChannel: %s", j.resultChannel))
	if err := j.resultPublisher.Publish(ctx, j.resultChannel, result); err != nil {
		return internal.Errorf("publisher.Publish. err: %w", err)
	}

	return nil
}
