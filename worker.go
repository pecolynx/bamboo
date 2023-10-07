package bamboo

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/cenkalti/backoff/v4"
	"go.opentelemetry.io/otel/propagation"

	"github.com/pecolynx/bamboo/internal"
	"github.com/pecolynx/bamboo/sloghelper"
)

type bambooWorker struct {
	createRequestConsumerFunc CreateBambooRequestConsumerFunc
	resultPublisher           BambooResultPublisher
	heartbeatPublisher        BambooHeartbeatPublisher
	workerFunc                WorkerFunc
	numWorkers                int
	logConfigFunc             LogConfigFunc
	workerPool                chan chan internal.Job
}

func NewBambooWorker(createRequestConsumerFunc CreateBambooRequestConsumerFunc, resultPublisher BambooResultPublisher, heartbeatPublisher BambooHeartbeatPublisher, workerFunc WorkerFunc, numWorkers int, logConfigFunc LogConfigFunc) (BambooWorker, error) {

	if resultPublisher == nil {
		return nil, errors.New("nil")
	}

	return &bambooWorker{
		createRequestConsumerFunc: createRequestConsumerFunc,
		resultPublisher:           resultPublisher,
		heartbeatPublisher:        heartbeatPublisher,
		workerFunc:                workerFunc,
		numWorkers:                numWorkers,
		logConfigFunc:             logConfigFunc,
		workerPool:                make(chan chan internal.Job),
	}, nil
}

func (w *bambooWorker) ping(ctx context.Context) error {
	consumer := w.createRequestConsumerFunc(ctx)
	defer consumer.Close(ctx)
	if err := consumer.Ping(ctx); err != nil {
		return internal.Errorf("consumer.Ping. err: %w", err)
	}

	if err := w.resultPublisher.Ping(ctx); err != nil {
		return internal.Errorf("resultPublisher.Ping. err: %w", err)
	}

	if err := w.heartbeatPublisher.Ping(ctx); err != nil {
		return internal.Errorf("heartbeatPublisher.Ping. err: %w", err)
	}

	return nil
}

func (w *bambooWorker) Run(ctx context.Context) error {
	logger := sloghelper.FromContext(ctx, sloghelper.BambooWorkerLoggerKey)
	ctx = context.WithValue(ctx, sloghelper.LoggerNameKey, sloghelper.BambooWorkerLoggerKey)

	workers := make([]internal.Worker, w.numWorkers)
	for i := 0; i < w.numWorkers; i++ {
		workers[i] = internal.NewWorker(i, w.workerPool)
		workers[i].Start(ctx)
	}

	operation := func() error { return w.run(ctx) }

	backOff := backoff.WithContext(w.newBackOff(), ctx)

	notify := func(err error, d time.Duration) {
		logger.ErrorContext(ctx, "redis reading error", slog.Any("err", err))
	}

	err := backoff.RetryNotify(operation, backOff, notify)
	if err != nil {
		return err
	}

	return nil
}

func (w *bambooWorker) run(ctx context.Context) error {
	logger := sloghelper.FromContext(ctx, sloghelper.BambooWorkerLoggerKey)
	logger.DebugContext(ctx, "run")
	if err := w.ping(ctx); err != nil {
		return internal.Errorf("ping. err: %w", err)
	}

	consumer := w.createRequestConsumerFunc(ctx)
	defer consumer.Close(ctx)

	for {
		select {
		case <-ctx.Done():
			return nil
		case worker := <-w.workerPool: // wait for available worker
			if err := w.consumeRequestAndDispatchJob(ctx, consumer, worker); err != nil {
				if errors.Is(err, ErrContextCanceled) {
					return nil
				}
				return internal.Errorf("consumeRequestAndDispatchJob. err: %w", err)
			}
		}
	}
}

func (w *bambooWorker) consumeRequestAndDispatchJob(ctx context.Context, consumer BambooRequestConsumer, worker chan<- internal.Job) error {
	logger := sloghelper.FromContext(ctx, sloghelper.BambooWorkerLoggerKey)
	logger.DebugContext(ctx, "worker is ready")

	req, err := consumer.Consume(ctx)
	if errors.Is(err, ErrContextCanceled) {
		return internal.Errorf("consumer.Consume. err: %w", err)
	} else if err != nil {
		worker <- internal.NewEmptyJob()
		return internal.Errorf("consumer.Consume. err: %w", err)
	}

	done := make(chan interface{})
	aborted := make(chan interface{})

	reqCtx := w.logConfigFunc(ctx, req.Headers)
	logger.DebugContext(reqCtx, "request is received")

	if req.HeartbeatIntervalMSec != 0 {
		if err := w.heartbeatPublisher.Run(reqCtx, req.ResultChannel, int(req.HeartbeatIntervalMSec), done, aborted); err != nil {
			worker <- internal.NewEmptyJob()
			return internal.Errorf("heartbeatPublisher.Run. err: %w", err)
		}
	}

	var carrier propagation.MapCarrier = req.Carrier
	job := NewWorkerJob(reqCtx, carrier, w.workerFunc, req.Headers, req.Data, w.resultPublisher, req.ResultChannel, done, aborted, w.logConfigFunc)

	logger.DebugContext(ctx, "dispatch job to worker")
	worker <- job

	return nil
}

func (w *bambooWorker) newBackOff() backoff.BackOff {
	backOff := backoff.NewExponentialBackOff()
	backOff.MaxElapsedTime = 0
	return backOff
}
