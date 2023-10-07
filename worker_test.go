package bamboo_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/pecolynx/bamboo"
	"github.com/pecolynx/bamboo/internal"
	pb_test "github.com/pecolynx/bamboo/proto_test"
	"github.com/pecolynx/bamboo/sloghelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

type logStruct struct {
	Level      string `json:"level"`
	ClientName string `json:"client_name"`
	LoggerName string `json:"bamboo_logger_name"`
	Message    string `json:"msg"`
}

type stringList struct {
	list   []string
	writer io.Writer
}

func (s *stringList) Write(p []byte) (n int, err error) {
	s.writer.Write(p)
	s.list = append(s.list, string(p))
	return len(p), nil
}

func Test_WorkerClient_Call(t *testing.T) {
	ctx := context.Background()
	stringList := stringList{list: make([]string, 0), writer: os.Stdout}

	logger := slog.New(&sloghelper.BambooHandler{Handler: slog.NewJSONHandler(&stringList, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})})

	sloghelper.BambooLoggers["bamboo_test"] = logger
	sloghelper.BambooLoggers[sloghelper.BambooWorkerLoggerKey] = logger
	sloghelper.BambooLoggers[sloghelper.BambooWorkerClientLoggerKey] = logger
	sloghelper.BambooLoggers[sloghelper.BambooWorkerJobLoggerKey] = logger
	sloghelper.BambooLoggers[sloghelper.BambooRequestProducerLoggerKey] = logger
	sloghelper.BambooLoggers[sloghelper.BambooRequestConsumerLoggerKey] = logger

	pubsubMap := bamboo.NewGoroutineBambooPubSubMap()

	queue := make(chan []byte, 1)
	resultPublisher := bamboo.NewGoroutineBambooResultPublisher(pubsubMap)
	heartbeatPublisher := bamboo.NewGoroutineBambooHeartbeatPublisher(pubsubMap)

	createBambooRequestConsumerFunc := func(ctx context.Context) bamboo.BambooRequestConsumer {
		return bamboo.NewGoroutineBambooRequestConsumer(queue)
	}
	workerFunc := func(ctx context.Context, headers map[string]string, reqBytes []byte, aborted <-chan interface{}) ([]byte, error) {
		logger := sloghelper.FromContext(ctx, "bamboo_test")
		ctx = context.WithValue(ctx, sloghelper.LoggerNameKey, "bamboo_test")

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
	rp := bamboo.NewGoroutineBambooRequestProducer(ctx, "WORKER-NAME", queue)
	rs := bamboo.NewGoroutineBambooResultSubscriber(ctx, "WORKER-NAME", pubsubMap)
	workerClient := bamboo.NewBambooWorkerClient(rp, rs)

	ctxCanel, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		time.AfterFunc(5000*time.Millisecond, cancel)
		worker.Run(ctxCanel)
	}()

	req := pb_test.WorkerTestParameter{X: 3, Y: 5}
	reqBytes, err := proto.Marshal(&req)
	require.Nil(t, err)

	respBytes, err := workerClient.Call(ctx, 0, 3, map[string]string{}, reqBytes)
	require.Nil(t, err)

	resp := pb_test.WorkerTestResponse{}
	err = proto.Unmarshal(respBytes, &resp)
	require.Nil(t, err)

	assert.Equal(t, int(resp.Value), 15)
}
