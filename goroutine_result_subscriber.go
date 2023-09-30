package bamboo

import (
	"context"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/pecolynx/bamboo/internal"
	pb "github.com/pecolynx/bamboo/proto"
)

type goroutineBambooResultSubscriber struct {
	workerName string
	pubsubMap  GoroutineBambooPubSubMap
}

func NewGoroutineBambooResultSubscriber(ctx context.Context, workerName string, pubsubMap GoroutineBambooPubSubMap) BambooResultSubscriber {
	return &goroutineBambooResultSubscriber{
		workerName: workerName,
		pubsubMap:  pubsubMap,
	}
}

func (s *goroutineBambooResultSubscriber) Ping(ctx context.Context) error {
	return nil
}

func (s *goroutineBambooResultSubscriber) Subscribe(ctx context.Context, resultChannel string, heartbeatIntervalSec int, jobTimeoutSec int) ([]byte, error) {
	logger := internal.FromContext(ctx)

	pubsub := s.pubsubMap.CreateChannel(resultChannel)
	defer func() {
		if err := s.pubsubMap.CloseSubscribeChannel(resultChannel); err != nil {
			logger.Errorf(".pubsubMap.CloseSubscribeChannel. err: %v", err)
		}
	}()

	c1 := make(chan ByteArreayResult, 1)
	done := make(chan interface{})
	heartbeat := make(chan int64)
	defer close(heartbeat)

	aborted := make(chan interface{})
	defer close(aborted)

	timedout := s.startTimer(ctx, time.Duration(jobTimeoutSec)*time.Second)

	go func() {
		defer func() {
			logger.Debug("stop receiving loop")
		}()

		defer close(c1)
		defer close(done)
		for {
			select {
			case <-ctx.Done():
				logger.Debug("ctx.Done() stop receiving loop")
				return
			case respBytes := <-pubsub:
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

	if heartbeatIntervalSec != 0 {
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
	}

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

func (s *goroutineBambooResultSubscriber) startTimer(ctx context.Context, timeoutTime time.Duration) <-chan interface{} {
	logger := internal.FromContext(ctx)
	if timeoutTime != 0 {
		timedout := make(chan interface{})
		time.AfterFunc(timeoutTime, func() {
			close(timedout)
		})
		return timedout
	}

	logger.Debug("timeout time is infinite")
	return nil

}
