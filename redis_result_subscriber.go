package bamboo

import (
	"context"
	"encoding/base64"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/proto"

	pb "github.com/pecolynx/bamboo/proto"
	"github.com/pecolynx/bamboo/sloghelper"
)

type ByteArreayResult struct {
	Value []byte
	Error error
}

type redisBambooResultSubscriber struct {
	workerName string
	subscriber redis.UniversalClient
}

func NewRedisBambooResultSubscriber(ctx context.Context, workerName string, subscriberConfig *redis.UniversalOptions) BambooResultSubscriber {
	subscriber := redis.NewUniversalClient(subscriberConfig)

	return &redisBambooResultSubscriber{
		workerName: workerName,
		subscriber: subscriber,
	}
}

func (s *redisBambooResultSubscriber) Ping(ctx context.Context) error {
	if _, err := s.subscriber.Ping(ctx).Result(); err != nil {
		return err
	}

	return nil
}

func (s *redisBambooResultSubscriber) Subscribe(ctx context.Context, resultChannel string, heartbeatIntervalSec int, jobTimeoutSec int) ([]byte, error) {
	logger := sloghelper.FromContext(ctx, sloghelper.BambooResultSubscriberLoggerKey)

	pubsub := s.subscriber.Subscribe(ctx, resultChannel)
	defer pubsub.Close()

	c1 := make(chan ByteArreayResult, 1)
	done := make(chan interface{})
	heartbeat := make(chan int64)
	defer close(heartbeat)

	aborted := make(chan interface{})
	defer close(aborted)

	timedout := s.startTimer(ctx, time.Duration(jobTimeoutSec)*time.Second)

	go func() {
		defer func() {
			logger.DebugContext(ctx, "stop receiving loop")
		}()

		defer close(c1)
		defer close(done)
		for {
			msg, err := pubsub.ReceiveMessage(ctx)
			if err != nil {
				c1 <- ByteArreayResult{Value: nil, Error: err}
				return
			}

			respBytes, err := base64.StdEncoding.DecodeString(msg.Payload)
			if err != nil {
				c1 <- ByteArreayResult{Value: nil, Error: err}
				return
			}

			resp := pb.WorkerResponse{}
			if err := proto.Unmarshal(respBytes, &resp); err != nil {
				c1 <- ByteArreayResult{Value: nil, Error: err}
				return
			}

			switch resp.Type {
			case pb.ResponseType_HEARTBEAT:
				heartbeat <- time.Now().Unix()
			// case pb.ResponseType_ACCEPTED:
			// 	heartbeat <- time.Now().Unix()
			// case pb.ResponseType_ABORTED:
			// 	heartbeat <- time.Now().Unix()
			// case pb.ResponseType_INVALID_ARGUMENT:
			// 	heartbeat <- time.Now().Unix()
			// case pb.ResponseType_INTERNAL_ERROR:
			// 	heartbeat <- time.Now().Unix()
			case pb.ResponseType_DATA:
				c1 <- ByteArreayResult{Value: resp.Data, Error: nil}
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(time.Duration(heartbeatIntervalSec) * time.Second)
		defer func() {
			logger.DebugContext(ctx, "stop heartbeat loop")
			ticker.Stop()
		}()

		last := time.Now().Unix()

		for {
			select {
			case <-done:
				logger.DebugContext(ctx, "done")
				return
			case h := <-heartbeat:
				if h != 0 {
					last = h
					logger.DebugContext(ctx, "heartbeat", slog.Int64("time", h))
				}
			case <-timedout:
				logger.DebugContext(ctx, "timedout")
				return
			case <-ticker.C:
				if time.Now().Unix()-last > int64(heartbeatIntervalSec)*2 {
					logger.DebugContext(ctx, "heartbeat couldn't be received")
					aborted <- struct{}{}
				}
			}
		}
	}()

	select {
	case res := <-c1:
		if res.Error != nil {
			return nil, res.Error
		}
		return res.Value, nil
	case <-aborted:
		return nil, ErrAborted
	case <-timedout:
		return nil, ErrTimedout
	}
}

func (s *redisBambooResultSubscriber) startTimer(ctx context.Context, timeoutTime time.Duration) <-chan interface{} {
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
