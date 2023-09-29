package bamboo

import (
	"context"
	"encoding/base64"
	"time"

	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/proto"

	"github.com/pecolynx/bamboo/internal"
	pb "github.com/pecolynx/bamboo/proto"
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
	logger := internal.FromContext(ctx)

	pubsub := s.subscriber.Subscribe(ctx, resultChannel)
	defer pubsub.Close()

	c1 := make(chan ByteArreayResult, 1)
	done := make(chan interface{})
	heartbeat := make(chan int64)
	defer close(heartbeat)

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
				logger.Debug("done. stop receiving message...")
				return
			// case <-timedout:
			// 	logger.Debug("timedout. stop receiving message...")
			// 	return
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
			case h := <-heartbeat:
				if h != 0 {
					last = h
					logger.Debugf("heartbeat, %d", h)
				}
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
