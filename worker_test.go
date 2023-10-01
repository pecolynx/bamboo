package bamboo_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/pecolynx/bamboo"
	"github.com/pecolynx/bamboo/internal"
	pb_test "github.com/pecolynx/bamboo/proto_test"
	"github.com/pecolynx/bamboo/sloghelper"
)

func TestA(t *testing.T) {

	pubsubMap := bamboo.NewGoroutineBambooPubSubMap()

	queue := make(chan []byte)
	resultPublisher := bamboo.NewGoroutineBambooResultPublisher(pubsubMap)
	heartbeatPublisher := bamboo.NewGoroutineBambooHeartbeatPublisher(pubsubMap)

	createBambooRequestConsumerFunc := func(ctx context.Context) bamboo.BambooRequestConsumer {
		return bamboo.NewGoroutineBambooRequestConsumer(queue)
	}
	workerFunc := func(ctx context.Context, headers map[string]string, reqBytes []byte, aborted <-chan interface{}) ([]byte, error) {
		logger := sloghelper.FromContext(ctx, "bamboo_test")

		req := pb_test.WorkerTestParameter{}
		if err := proto.Unmarshal(reqBytes, &req); err != nil {
			return nil, internal.Errorf("proto.Unmarshal. err: %w", err)
		}

		time.Sleep(time.Second * 1)

		answer := req.X * req.Y
		logger.InfoContext(ctx, fmt.Sprintf("answer: %d", answer))

		resp := pb_test.WorkerTestResponse{Value: answer}
		respBytes, err := proto.Marshal(&resp)
		if err != nil {
			return nil, internal.Errorf("proto.Marshal. err: %w", err)
		}

		return respBytes, nil
	}

	logConfigFunc := func(ctx context.Context, headers map[string]string) context.Context {
		for k, v := range headers {
			if k == sloghelper.RequestIDKey {
				ctx = context.WithValue(ctx, sloghelper.RequestIDKey, v)
			}
		}
		return ctx
	}

	worker, err := bamboo.NewBambooWorker(createBambooRequestConsumerFunc, resultPublisher, heartbeatPublisher, workerFunc, 1, logConfigFunc)
	if err != nil {
		panic(err)
	}

	fmt.Println(worker)
}
