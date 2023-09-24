package bamboo

import (
	"context"
	"encoding/base64"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/protobuf/proto"

	"github.com/pecolynx/bamboo/internal"
)

type redisRedisBambooWorker struct {
	consumerOptions  *redis.UniversalOptions
	consumerChannel  string
	publisherOptions *redis.UniversalOptions
	workerFunc       WorkerFunc
	numWorkers       int
	logConfigFunc    LogConfigFunc
}

func NewRedisRedisBambooWorker(consumerOptions *redis.UniversalOptions, consumerChannel string, publisherOptions *redis.UniversalOptions, workerFunc WorkerFunc, numWorkers int, logConfigFunc LogConfigFunc) BambooWorker {
	return &redisRedisBambooWorker{
		consumerOptions:  consumerOptions,
		consumerChannel:  consumerChannel,
		publisherOptions: publisherOptions,
		workerFunc:       workerFunc,
		numWorkers:       numWorkers,
		logConfigFunc:    logConfigFunc,
	}
}

func (w *redisRedisBambooWorker) ping(ctx context.Context) error {
	consumer := redis.NewUniversalClient(w.consumerOptions)
	defer consumer.Close()
	if _, err := consumer.Ping(ctx).Result(); err != nil {
		return internal.Errorf("consumer.Ping. err: %w", err)
	}

	publisher := redis.NewUniversalClient(w.publisherOptions)
	defer publisher.Close()
	if _, err := publisher.Ping(ctx).Result(); err != nil {
		return internal.Errorf("publisher.Ping. err: %w", err)
	}

	return nil
}

func (w *redisRedisBambooWorker) Run(ctx context.Context) error {
	logger := internal.FromContext(ctx)

	operation := func() error {
		if err := w.ping(ctx); err != nil {
			return internal.Errorf("ping. err: %w", err)
		}

		dispatcher := internal.NewDispatcher()
		defer dispatcher.Stop(ctx)
		dispatcher.Start(ctx, w.numWorkers)

		consumer := redis.NewUniversalClient(w.publisherOptions)
		defer consumer.Close()

		for {
			m, err := consumer.BRPop(ctx, 0, w.consumerChannel).Result()
			if err != nil {
				return internal.Errorf("consumer.BRPop. err: %w", err)
			}

			if len(m) == 1 {
				return internal.Errorf("received invalid data. m[0]: %s, err: %w", m[0], err)
			} else if len(m) != 2 {
				return internal.Errorf("received invalid data. err: %w", err)
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

			if req.JobTimeoutSec != 0 {
				time.AfterFunc(time.Duration(req.JobTimeoutSec)*time.Second, func() {
					close(aborted)
				})
			}

			reqCtx := w.logConfigFunc(ctx, req.Headers)
			heartbeatPublisher := NewRedisBambooHeartbeatPublisher(w.publisherOptions, req.ResultChannel, int(req.HeartbeatIntervalSec), done, aborted)
			heartbeatPublisher.Run(reqCtx)

			var carrier propagation.MapCarrier = req.Carrier
			dispatcher.AddJob(NewRedisJob(reqCtx, carrier, w.workerFunc, req.Headers, req.Data, w.publisherOptions, req.ResultChannel, done, aborted, w.logConfigFunc))

		}
	}

	backOff := backoff.NewExponentialBackOff()
	backOff.MaxElapsedTime = 0

	notify := func(err error, d time.Duration) {
		logger.Errorf("redis reading error. err: %v", err)
	}

	err := backoff.RetryNotify(operation, backOff, notify)
	if err != nil {
		return err
	}

	return nil
}
