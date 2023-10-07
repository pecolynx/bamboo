package bamboo

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/pecolynx/bamboo/internal"
	pb "github.com/pecolynx/bamboo/proto"
	"github.com/pecolynx/bamboo/sloghelper"
)

type BambooWorkerClient interface {
	Ping(ctx context.Context) error
	Close(ctx context.Context)
	Call(ctx context.Context, heartbeatIntervalMSec int, jobTimeoutMSec int, headers map[string]string, param []byte) ([]byte, error)
}

type bambooWorkerClient struct {
	requestProducer  BambooRequestProducer
	resultSubscriber BambooResultSubscriber
}

func NewBambooWorkerClient(requestProducer BambooRequestProducer, resultSubscriber BambooResultSubscriber) BambooWorkerClient {
	return &bambooWorkerClient{
		requestProducer:  requestProducer,
		resultSubscriber: resultSubscriber,
	}
}

func (c *bambooWorkerClient) Ping(ctx context.Context) error {
	return c.resultSubscriber.Ping(ctx)
}

func (c *bambooWorkerClient) Close(ctx context.Context) {
	defer c.requestProducer.Close(ctx)
}

func (c *bambooWorkerClient) Call(ctx context.Context, heartbeatIntervalMSec int, jobTimeoutMSec int, headers map[string]string, param []byte) ([]byte, error) {
	logger := sloghelper.FromContext(ctx, sloghelper.BambooWorkerClientLoggerKey)
	ctx = context.WithValue(ctx, sloghelper.LoggerNameKey, sloghelper.BambooWorkerClientLoggerKey)
	logger.DebugContext(ctx, "Call")

	resultChannel, err := c.newResultChannelString()
	if err != nil {
		return nil, err
	}

	timedout := c.startTimer(ctx, time.Duration(jobTimeoutMSec)*time.Millisecond)

	ch := make(chan *ByteArreayResult)
	defer close(ch)
	go func() {
		sendResult := func(result *ByteArreayResult) {
			select {
			case <-ctx.Done():
			case <-timedout:
			default:
				ch <- result
			}
		}

		resultBytes, err := c.subscribe(ctx, resultChannel, heartbeatIntervalMSec, jobTimeoutMSec)
		if err != nil {
			sendResult(&ByteArreayResult{Value: nil, Error: err})
			return
		}

		logger.DebugContext(ctx, fmt.Sprintf("result is received. resultChannel: %s", resultChannel))
		sendResult(&ByteArreayResult{Value: resultBytes, Error: nil})
	}()

	logger.DebugContext(ctx, fmt.Sprintf("produce request. resultChannel: %s, heartbeatIntervalMSec: %d, jobTimeoutMSec: %d", resultChannel, heartbeatIntervalMSec, jobTimeoutMSec))
	if err := c.requestProducer.Produce(ctx, resultChannel, heartbeatIntervalMSec, jobTimeoutMSec, headers, param); err != nil {
		return nil, internal.Errorf("requestProducer.Produce. err: %w", err)
	}

	select {
	case <-ctx.Done():
		return nil, internal.Errorf("context canceled. err: %w", ErrContextCanceled)
	case <-timedout:
		return nil, internal.Errorf("timedout. err: %w", ErrTimedout)
	case result := <-ch:
		if result.Error != nil {
			return nil, result.Error
		}
		return result.Value, nil
	}
}

func (c *bambooWorkerClient) subscribe(ctx context.Context, resultChannel string, heartbeatIntervalMSec int, jobTimeoutMSec int) ([]byte, error) {
	logger := sloghelper.FromContext(ctx, sloghelper.BambooWorkerClientLoggerKey)
	logger.DebugContext(ctx, "subscribe")

	heartbeat := make(chan int64)
	defer close(heartbeat)

	subscribeFunc, closeSubscribeConnection, err := c.resultSubscriber.OpenSubscribeConnection(ctx, resultChannel)
	if err != nil {
		return nil, internal.Errorf("resultSubscriber.OpenSubscribeConnection. err: %w", err)
	}

	defer func() {
		if err := closeSubscribeConnection(ctx); err != nil {
			logger.ErrorContext(ctx, "closeSubscribeConnection", slog.Any("err", err))
		}
	}()

	c1 := make(chan ByteArreayResult, 1)
	done := make(chan interface{})

	aborted := make(chan interface{})
	defer close(aborted)

	go func() {
		defer func() {
			logger.DebugContext(ctx, "stop receiving loop")
		}()

		defer close(c1)
		defer close(done)

		for {
			resp, err := subscribeFunc(ctx)
			if err != nil {
				if errors.Is(err, ErrContextCanceled) {
					logger.DebugContext(ctx, "context canceled")
					return
				} else {
					c1 <- ByteArreayResult{Value: nil, Error: err}
					return
				}
			}

			switch resp.Type {
			case pb.ResponseType_HEARTBEAT:
				heartbeat <- time.Now().UnixMilli()
			case pb.ResponseType_DATA:
				c1 <- ByteArreayResult{Value: resp.Data, Error: nil}
				return
			}
		}
	}()

	if heartbeatIntervalMSec != 0 {
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

	select {
	case <-ctx.Done():
		return nil, internal.Errorf("context canceled. err: %w", ErrContextCanceled)
	case <-aborted:
		return nil, internal.Errorf("aborted. err: %w", ErrAborted)
	case resp := <-c1:
		if resp.Error != nil {
			return nil, resp.Error
		}
		return resp.Value, nil
	}

}

func (c *bambooWorkerClient) startTimer(ctx context.Context, timeoutTime time.Duration) <-chan interface{} {
	logger := sloghelper.FromContext(ctx, sloghelper.BambooResultSubscriberLoggerKey)
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
