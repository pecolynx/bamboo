package bamboo

import (
	"context"
	"errors"
	"time"

	"github.com/cenkalti/backoff/v4"
	"go.opentelemetry.io/otel/propagation"

	"github.com/pecolynx/bamboo/internal"
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
		return internal.Errorf("publisher.Ping. err: %w", err)
	}

	if err := w.heartbeatPublisher.Ping(ctx); err != nil {
		return internal.Errorf("publisher.Ping. err: %w", err)
	}

	return nil
}

func (w *bambooWorker) Run(ctx context.Context) error {
	logger := internal.FromContext(ctx)

	workers := make([]internal.Worker, w.numWorkers)
	for i := 0; i < w.numWorkers; i++ {
		workers[i] = internal.NewWorker(i, w.workerPool)
		workers[i].Start(ctx)
	}

	operation := func() error {
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
				logger.Debug("worker is ready")

				if err := w.consumeRequestAndDispatchJob(ctx, consumer, worker); err != nil {
					if errors.Is(err, ErrContextCanceled) {
						return nil
					}
					return err
				}
			}
		}
	}

	backOff := backoff.WithContext(w.newBackOff(), ctx)

	notify := func(err error, d time.Duration) {
		logger.Errorf("redis reading error. err: %v", err)
	}

	err := backoff.RetryNotify(operation, backOff, notify)
	if err != nil {
		return err
	}

	return nil
}

func (w *bambooWorker) consumeRequestAndDispatchJob(ctx context.Context, consumer BambooRequestConsumer, worker chan<- internal.Job) error {
	logger := internal.FromContext(ctx)
	logger.Debug("worker is ready")

	req, err := consumer.Consume(ctx)
	if errors.Is(err, ErrContextCanceled) {
		return err
	} else if err != nil {
		worker <- internal.NewEmptyJob()
		return err
	}

	done := make(chan interface{})
	aborted := make(chan interface{})

	reqCtx := w.logConfigFunc(ctx, req.Headers)

	if req.HeartbeatIntervalSec != 0 {
		w.heartbeatPublisher.Run(reqCtx, req.ResultChannel, int(req.HeartbeatIntervalSec), done, aborted)
	}

	var carrier propagation.MapCarrier = req.Carrier
	job := NewWorkerJob(reqCtx, carrier, w.workerFunc, req.Headers, req.Data, w.resultPublisher, req.ResultChannel, done, aborted, w.logConfigFunc)

	logger.Debug("dispatch job to worker")
	worker <- job

	return nil
}

func (w *bambooWorker) newBackOff() backoff.BackOff {
	backOff := backoff.NewExponentialBackOff()
	backOff.MaxElapsedTime = 0
	return backOff
}
