package bamboo

import (
	"context"
	"encoding/base64"
	"time"

	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/proto"

	"github.com/pecolynx/bamboo/internal"
)

type ByteArreayResult struct {
	Value []byte
	Error error
}

type RedisResultSubscriber struct {
	workerName string
	subscriber redis.UniversalClient
}

func NewRedisResultSubscriber(ctx context.Context, workerName string, subscriberConfig *redis.UniversalOptions) BambooResultSubscriber {
	subscriber := redis.NewUniversalClient(subscriberConfig)

	return &RedisResultSubscriber{
		workerName: workerName,
		subscriber: subscriber,
	}
}

func (s *RedisResultSubscriber) Ping(ctx context.Context) error {
	if _, err := s.subscriber.Ping(ctx).Result(); err != nil {
		return err
	}

	return nil
}

func (s *RedisResultSubscriber) Subscribe(ctx context.Context, resultChannel string, heartbeatIntervalSec int, jobTimeoutSec int) ([]byte, error) {
	logger := internal.FromContext(ctx)

	pubsub := s.subscriber.Subscribe(ctx, resultChannel)
	defer pubsub.Close()
	c1 := make(chan ByteArreayResult, 1)
	done := make(chan interface{})
	heartbeat := make(chan int64)

	aborted := make(chan interface{})
	timedout := make(chan interface{})

	if jobTimeoutSec != 0 {
		time.AfterFunc(time.Duration(jobTimeoutSec)*time.Second, func() {
			close(timedout)
		})
	} else {
		logger.Debug("timeout time is infinite")
	}

	go func() {
		defer func() {
			logger.Debug("stop receiving loop")
		}()

		defer close(c1)
		defer close(done)
		for {
			select {
			case <-done:
				logger.Debug("done. stop ReceivingMessage...")
				return
			default:
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

				resp := WorkerResponse{}
				if err := proto.Unmarshal(respBytes, &resp); err != nil {
					c1 <- ByteArreayResult{Value: nil, Error: err}
					return
				}
				if resp.Type == ResponseType_HEARTBEAT {
					heartbeat <- time.Now().Unix()
				} else {
					c1 <- ByteArreayResult{Value: resp.Data, Error: nil}
				}
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(time.Duration(heartbeatIntervalSec) * time.Second)
		defer func() {
			logger.Debug("stop heartbeat loop")
			ticker.Stop()
		}()

		last := time.Now().Unix()

		for {
			select {
			case <-done:
				logger.Debug("done")
				return
			case last = <-heartbeat:
				logger.Debug("heartbeat")
			case <-timedout:
				logger.Debug("timedout")
				return
			case <-ticker.C:
				if time.Now().Unix()-last > int64(heartbeatIntervalSec)*2 {
					logger.Debug("heartbeat couldn't be received")
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
