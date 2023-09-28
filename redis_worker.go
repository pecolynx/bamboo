package bamboo

import (
	"context"
	"encoding/base64"
	"errors"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/protobuf/proto"

	"github.com/pecolynx/bamboo/internal"
)

type redisBambooWorker struct {
	consumerOptions    *redis.UniversalOptions
	consumerChannel    string
	requestWaitTimeout time.Duration
	resultPublisher    BambooResultPublisher
	heartbeatPublisher BambooHeartbeatPublisher
	workerFunc         WorkerFunc
	numWorkers         int
	logConfigFunc      LogConfigFunc
	workerPool         chan chan internal.Job
}

func NewRedisBambooWorker(consumerOptions *redis.UniversalOptions, consumerChannel string, requestWaitTimeout time.Duration, resultPublisher BambooResultPublisher, heartbeatPublisher BambooHeartbeatPublisher, workerFunc WorkerFunc, numWorkers int, logConfigFunc LogConfigFunc) BambooWorker {
	return &redisBambooWorker{
		consumerOptions:    consumerOptions,
		consumerChannel:    consumerChannel,
		requestWaitTimeout: requestWaitTimeout,
		resultPublisher:    resultPublisher,
		heartbeatPublisher: heartbeatPublisher,
		workerFunc:         workerFunc,
		numWorkers:         numWorkers,
		logConfigFunc:      logConfigFunc,
		workerPool:         make(chan chan internal.Job),
	}
}

func (w *redisBambooWorker) ping(ctx context.Context) error {
	consumer := redis.NewUniversalClient(w.consumerOptions)
	defer consumer.Close()
	if _, err := consumer.Ping(ctx).Result(); err != nil {
		return internal.Errorf("consumer.Ping. err: %w", err)
	}

	if err := w.resultPublisher.Ping(ctx); err != nil {
		return internal.Errorf("publisher.Ping. err: %w", err)
	}

	return nil
}

func (w *redisBambooWorker) Run(ctx context.Context) error {
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

		consumer := redis.NewUniversalClient(w.consumerOptions)
		defer consumer.Close()

		for {
			select {
			case <-ctx.Done():
				return nil
			case worker := <-w.workerPool: // wait for available worker
				logger.Debug("worker is ready")

				job, err := w.waitRequest(ctx, consumer)
				if errors.Is(err, ErrContextCanceled) {
					return nil
				} else if err != nil {
					worker <- internal.NewEmptyJob()
					return err
				}

				logger.Debug("dispatch job to worker")
				worker <- job
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

func (w *redisBambooWorker) waitRequest(ctx context.Context, consumer redis.UniversalClient) (internal.Job, error) {
	logger := internal.FromContext(ctx)

	for {
		select {
		case <-ctx.Done():
			return nil, ErrContextCanceled
		default:
			m, err := consumer.BRPop(ctx, w.requestWaitTimeout, w.consumerChannel).Result()
			if errors.Is(err, redis.Nil) {
				continue
			} else if err != nil {
				return nil, internal.Errorf("consumer.BRPop. err: %w", err)
			}

			if len(m) == 1 {
				return nil, internal.Errorf("received invalid data. m[0]: %s, err: %w", m[0], err)
			} else if len(m) != 2 {
				return nil, internal.Errorf("received invalid data. err: %w", err)
			}

			reqStr := m[1]
			reqBytes, err := base64.StdEncoding.DecodeString(reqStr)
			if err != nil {
				logger.Warnf("invalid parameter. failed to base64.StdEncoding.DecodeString. err: %w", err)
				continue
			}

			req := WorkerParameter{}
			if err := proto.Unmarshal(reqBytes, &req); err != nil {
				logger.Warnf("invalid parameter. failed to proto.Unmarshal. err: %w", err)
				continue
			}

			done := make(chan interface{})
			aborted := make(chan interface{})

			reqCtx := w.logConfigFunc(ctx, req.Headers)

			if req.JobTimeoutSec != 0 {
				time.AfterFunc(time.Duration(req.JobTimeoutSec)*time.Second, func() {
					close(aborted)
				})
			}

			if req.HeartbeatIntervalSec != 0 {
				w.heartbeatPublisher.Run(reqCtx, req.ResultChannel, int(req.HeartbeatIntervalSec), done, aborted)
			}

			var carrier propagation.MapCarrier = req.Carrier
			job := NewWorkerJob(reqCtx, carrier, w.workerFunc, req.Headers, req.Data, w.resultPublisher, req.ResultChannel, done, aborted, w.logConfigFunc)

			return job, nil
		}
	}
}

func (w *redisBambooWorker) newBackOff() backoff.BackOff {
	backOff := backoff.NewExponentialBackOff()
	backOff.MaxElapsedTime = 0
	return backOff
}
