package bamboo

import (
	"context"
	"errors"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/protobuf/proto"

	"github.com/pecolynx/bamboo/internal"
)

type kafkaBambooWorker struct {
	consumerOptions    kafka.ReaderConfig
	requestWaitTimeout time.Duration
	resultPublisher    BambooResultPublisher
	heartbeatPublisher BambooHeartbeatPublisher
	workerFunc         WorkerFunc
	numWorkers         int
	logConfigFunc      LogConfigFunc
	workerPool         chan chan internal.Job
}

func NewKafkaBambooWorker(consumerOptions kafka.ReaderConfig, requestWaitTimeout time.Duration, resultPublisher BambooResultPublisher, heartbeatPublisher BambooHeartbeatPublisher, workerFunc WorkerFunc, numWorkers int, logConfigFunc LogConfigFunc) BambooWorker {
	return &kafkaBambooWorker{
		consumerOptions:    consumerOptions,
		requestWaitTimeout: requestWaitTimeout,
		resultPublisher:    resultPublisher,
		heartbeatPublisher: heartbeatPublisher,
		workerFunc:         workerFunc,
		numWorkers:         numWorkers,
		logConfigFunc:      logConfigFunc,
		workerPool:         make(chan chan internal.Job),
	}
}

func (w *kafkaBambooWorker) ping(ctx context.Context) error {
	if len(w.consumerOptions.Brokers) == 0 {
		return errors.New("broker size is 0")
	}

	conn, err := kafka.Dial("tcp", w.consumerOptions.Brokers[0])
	if err != nil {
		return internal.Errorf("kafka.Dial. err: %w", err)
	}
	defer conn.Close()

	if _, err := conn.ReadPartitions(); err != nil {
		return internal.Errorf("conn.ReadPartitions. err: %w", err)
	}

	if err := w.resultPublisher.Ping(ctx); err != nil {
		return internal.Errorf("publisher.Ping. err: %w", err)
	}

	return nil
}

func (w *kafkaBambooWorker) Run(ctx context.Context) error {
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

		consumer := kafka.NewReader(w.consumerOptions)
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

func (w *kafkaBambooWorker) waitRequest(ctx context.Context, consumer *kafka.Reader) (internal.Job, error) {
	logger := internal.FromContext(ctx)

	for {
		select {
		case <-ctx.Done():
			return nil, ErrContextCanceled
		default:
			m, err := consumer.ReadMessage(ctx)
			if err != nil {
				return nil, internal.Errorf("kafka.ReadMessage. err: %w", err)
			}

			if len(m.Key) == 0 && len(m.Value) == 0 {
				return nil, errors.New("size of kafkaMessage received is invalid")
			}

			// fmt.Printf("message at offset %d: %s = %s\n", m.Offset, string(m.Key), string(m.Value))

			req := WorkerParameter{}
			if err := proto.Unmarshal(m.Value, &req); err != nil {
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

func (w *kafkaBambooWorker) newBackOff() backoff.BackOff {
	backOff := backoff.NewExponentialBackOff()
	backOff.MaxElapsedTime = 0
	return backOff
}
