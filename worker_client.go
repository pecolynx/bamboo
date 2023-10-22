package bamboo

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/google/uuid"

	"github.com/pecolynx/bamboo/internal"
	pb "github.com/pecolynx/bamboo/proto"
)

type BambooWorkerClient interface {
	Ping(ctx context.Context) error
	Close(ctx context.Context)
	Call(ctx context.Context, heartbeatIntervalMSec, connectTimeoutMSec, jobTimeoutMSec int, headers map[string]string, param []byte) ([]byte, error)
	CallWithRetry(ctx context.Context, heartbeatIntervalMSec, connectTimeoutMSec, jobTimeoutMSec int, headers map[string]string, backOff backoff.BackOff, param []byte) ([]byte, error)
}

type bambooWorkerClient struct {
	requestProducer  BambooRequestProducer
	resultSubscriber BambooResultSubscriber
	logConfigFunc    LogConfigFunc
}

func NewBambooWorkerClient(requestProducer BambooRequestProducer, resultSubscriber BambooResultSubscriber, logConfigFunc LogConfigFunc) BambooWorkerClient {
	return &bambooWorkerClient{
		requestProducer:  requestProducer,
		resultSubscriber: resultSubscriber,
		logConfigFunc:    logConfigFunc,
	}
}

func (c *bambooWorkerClient) Ping(ctx context.Context) error {
	return c.resultSubscriber.Ping(ctx)
}

func (c *bambooWorkerClient) Close(ctx context.Context) {
	defer c.requestProducer.Close(ctx)
}

func (c *bambooWorkerClient) Call(ctx context.Context, heartbeatIntervalMSec, connectTimeoutMSec, jobTimeoutMSec int, headers map[string]string, param []byte) ([]byte, error) {
	logger := GetLoggerFromContext(ctx, BambooWorkerClientLoggerContextKey)
	ctx = WithLoggerName(ctx, BambooWorkerClientLoggerContextKey)
	ctx = c.logConfigFunc(ctx, headers)

	logger.DebugContext(ctx, "Call")

	ctx, cancel := context.WithTimeout(ctx, time.Duration(jobTimeoutMSec+1000)*time.Millisecond)
	defer cancel()

	resultChannel, err := c.newResultChannelString()
	if err != nil {
		return nil, err
	}

	timedout := c.startTimer(ctx, time.Duration(jobTimeoutMSec)*time.Millisecond)

	subscribeFunc, closeSubscribeConnection, err := c.resultSubscriber.OpenSubscribeConnection(ctx, resultChannel)
	if err != nil {
		return nil, internal.Errorf("resultSubscriber.OpenSubscribeConnection. err: %w", err)
	}

	ch := make(chan *ByteArreayResult)

	go func(ctx context.Context) {
		defer close(ch)
		sendResult := func(result *ByteArreayResult) {
			select {
			case <-ctx.Done():
				return
			case <-timedout:
				return
			default:
				ch <- result
			}
		}

		resultBytes, err := c.subscribe(ctx, heartbeatIntervalMSec, connectTimeoutMSec, jobTimeoutMSec, subscribeFunc, closeSubscribeConnection)
		if err != nil {
			sendResult(&ByteArreayResult{Value: nil, Error: err})
			return
		}

		logger.DebugContext(ctx, fmt.Sprintf("result is received. resultChannel: %s", resultChannel))
		sendResult(&ByteArreayResult{Value: resultBytes, Error: nil})
	}(ctx)

	logger.DebugContext(ctx, fmt.Sprintf("produce request. resultChannel: %s, heartbeatIntervalMSec: %d, jobTimeoutMSec: %d", resultChannel, heartbeatIntervalMSec, jobTimeoutMSec))

	if err := c.requestProducer.Produce(ctx, resultChannel, heartbeatIntervalMSec, jobTimeoutMSec, headers, param); err != nil {
		return nil, internal.Errorf("requestProducer.Produce. err: %w", err)
	}

	select {
	case <-ctx.Done():
		return nil, internal.Errorf("context canceled. err: %w", ErrContextCanceled)
	case <-timedout:
		return nil, internal.Errorf("job timedout. err: %w", ErrJobTimedout)
	case result := <-ch:
		if result.Error != nil {
			return nil, result.Error
		}
		return result.Value, nil
	}
}

func (c *bambooWorkerClient) CallWithRetry(ctx context.Context, heartbeatIntervalMSec, connectTimeoutMSec, jobTimeoutMSec int, headers map[string]string, backOff backoff.BackOff, param []byte) ([]byte, error) {
	logger := GetLoggerFromContext(ctx, BambooWorkerClientLoggerContextKey)
	ctx = WithLoggerName(ctx, BambooWorkerClientLoggerContextKey)
	ctx = c.logConfigFunc(ctx, headers)

	numAttempt := 0

	var respBytes []byte
	operation := func() error {
		numAttempt += 1
		ctxCopy := context.WithoutCancel(ctx)
		respBytesTmp, err := c.Call(ctxCopy, heartbeatIntervalMSec, connectTimeoutMSec, jobTimeoutMSec, headers, param)
		if err != nil {
			return err
		}
		respBytes = respBytesTmp
		return nil
	}

	notify := func(err error, d time.Duration) {
		logger.WarnContext(ctx, "failed to Call", slog.Any("err", err), slog.Int("num_attempt", numAttempt))
	}

	if err := backoff.RetryNotify(operation, backOff, notify); err != nil {
		return nil, err
	}

	return respBytes, nil
}

func (c *bambooWorkerClient) subscribe(ctx context.Context, heartbeatIntervalMSec, connectTimeoutMSec, jobTimeoutMSec int, subscribeFunc SubscribeFunc, closeSubscribeConnection CloseSubscribeConnectionFunc) ([]byte, error) {
	logger := GetLoggerFromContext(ctx, BambooWorkerClientLoggerContextKey)
	logger.DebugContext(ctx, "subscribe")

	heartbeat := make(chan int64)
	defer close(heartbeat)

	defer func() {
		if err := closeSubscribeConnection(ctx); err != nil {
			logger.ErrorContext(ctx, "closeSubscribeConnection", slog.Any("err", err))
		}
	}()

	c1 := make(chan ByteArreayResult, 1)
	done := make(chan struct{})

	connectTimedout := make(chan struct{})
	defer close(connectTimedout)

	aborted := make(chan struct{})
	defer close(aborted)

	go func() {
		defer func() {
			logger.DebugContext(ctx, "stop receiving loop")
		}()

		defer close(c1)
		defer close(done)

		c.subscribeLoop(ctx, subscribeFunc, heartbeat, connectTimeoutMSec, connectTimedout, c1)
	}()

	if heartbeatIntervalMSec != 0 {
		c.startHeartbeatCheck(ctx, heartbeatIntervalMSec, done, heartbeat, aborted)
	}

	select {
	case <-ctx.Done():
		return nil, internal.Errorf("context canceled. err: %w", ErrContextCanceled)
	case <-connectTimedout:
		return nil, internal.Errorf("connect timedout. err: %w", ErrConnectTimedout)
	case <-aborted:
		return nil, internal.Errorf("aborted. err: %w", ErrAborted)
	case resp := <-c1:
		if resp.Error != nil {
			if resp.Value != nil {
				return nil, fmt.Errorf("%s", string(resp.Value))
			}
			return nil, resp.Error
		}
		return resp.Value, nil
	}
}

func (c *bambooWorkerClient) subscribeLoop(ctx context.Context, subscribeFunc SubscribeFunc, heartbeat chan<- int64, connectTimeoutMSec int, connectTimedout chan<- struct{}, result chan<- ByteArreayResult) {
	logger := GetLoggerFromContext(ctx, BambooWorkerClientLoggerContextKey)
	connectTimedoutTimer := time.NewTimer(time.Duration(connectTimeoutMSec) * time.Millisecond)

	type RespOrError struct {
		Resp  *pb.WorkerResponse
		Error error
	}

	respOrErrorCh := make(chan RespOrError)

	go func() {
		defer close(respOrErrorCh)
		for {
			resp, err := subscribeFunc(ctx)
			respOrErrorCh <- RespOrError{Resp: resp, Error: err}
			if err != nil {
				return
			}
		}
	}()

	for {
		select {
		case <-connectTimedoutTimer.C:
			if connectTimeoutMSec != 0 {
				connectTimedout <- struct{}{}
				return
			}
		case respOrError := <-respOrErrorCh:
			resp, err := respOrError.Resp, respOrError.Error
			if err != nil {
				if errors.Is(err, ErrContextCanceled) {
					logger.DebugContext(ctx, "context canceled")
					return
				} else {
					result <- ByteArreayResult{Value: nil, Error: err}
					return
				}
			}

			switch resp.Type {
			case pb.ResponseType_HEARTBEAT:
				heartbeat <- time.Now().UnixMilli()
			case pb.ResponseType_DATA:
				result <- ByteArreayResult{Value: resp.Data, Error: nil}
				return
			case pb.ResponseType_ACCEPTED:
				logger.DebugContext(ctx, "job is accepted")
				connectTimedoutTimer.Stop()
			case pb.ResponseType_ABORTED:
			case pb.ResponseType_INTERNAL_ERROR:
				result <- ByteArreayResult{Value: resp.Data, Error: ErrInternalError}
				return
			case pb.ResponseType_INVALID_ARGUMENT:
				result <- ByteArreayResult{Value: resp.Data, Error: ErrInvalidArgument}
				return
			}
		}
	}
}

func (c *bambooWorkerClient) startHeartbeatCheck(ctx context.Context, heartbeatIntervalMSec int, done <-chan struct{}, heartbeat <-chan int64, aborted chan<- struct{}) {
	logger := GetLoggerFromContext(ctx, BambooWorkerClientLoggerContextKey)

	go func() {
		ticker := time.NewTicker(time.Duration(heartbeatIntervalMSec) * time.Millisecond)
		defer func() {
			logger.DebugContext(ctx, "stop heartbeat loop")
			ticker.Stop()
		}()

		last := time.Now().UnixMilli()
		logger.DebugContext(ctx, "start heartbeat loop", slog.Int64("time", last))

		for {
			select {
			case <-ctx.Done():
				logger.DebugContext(ctx, "context canceled")
				return
			case <-done:
				logger.DebugContext(ctx, "done")
				return
			case h := <-heartbeat:
				if h != 0 {
					last = h
					logger.DebugContext(ctx, "heartbeat", slog.Int64("time", h))
				}
			case <-ticker.C:
				if time.Now().UnixMilli()-last > int64(heartbeatIntervalMSec)*2 {
					logger.DebugContext(ctx, "heartbeat couldn't be received")
					aborted <- struct{}{}
				}
			}
		}
	}()
}

func (c *bambooWorkerClient) startTimer(ctx context.Context, timeoutTime time.Duration) <-chan interface{} {
	logger := GetLoggerFromContext(ctx, BambooWorkerClientLoggerContextKey)

	if timeoutTime != 0 {
		timedout := make(chan interface{})
		time.AfterFunc(timeoutTime, func() {
			close(timedout)
		})
		return timedout
	}

	logger.DebugContext(ctx, "timeout time is infinite")
	return nil
}

func (c *bambooWorkerClient) newResultChannelString() (string, error) {
	resultChannel, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}

	return resultChannel.String(), nil
}
