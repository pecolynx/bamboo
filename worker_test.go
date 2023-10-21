package bamboo_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/pecolynx/bamboo"
	"github.com/pecolynx/bamboo/internal"
	pb_test "github.com/pecolynx/bamboo/proto_test"
)

var testAppNameContextKey bamboo.ContextKey = bamboo.ContextKey("bamboo_test")

// type logStruct struct {
// 	Level      string `json:"level"`
// 	ClientName string `json:"client_name"`
// 	LoggerName string `json:"bamboo_logger_name"`
// 	Message    string `json:"msg"`
// }

type stringList struct {
	list   []string
	writer io.Writer
}

func (s *stringList) Write(p []byte) (n int, err error) {
	if _, err := s.writer.Write(p); err != nil {
		panic(err)
	}
	s.list = append(s.list, string(p))
	return len(p), nil
}

type emptyBambooHeartbeatPublisher struct {
}

func NewEmptyBambooHeartbeatPublisher() bamboo.BambooHeartbeatPublisher {
	return &emptyBambooHeartbeatPublisher{}
}

func (h *emptyBambooHeartbeatPublisher) Ping(ctx context.Context) error {
	return nil
}

func (h *emptyBambooHeartbeatPublisher) Run(ctx context.Context, resultChannel string, heartbeatIntervalMSec int, done <-chan interface{}, aborted <-chan interface{}) error {
	return nil
}

func initLog() stringList {
	stringList := stringList{list: make([]string, 0), writer: os.Stdout}

	logger := slog.New(&bamboo.BambooLogHandler{Handler: slog.NewJSONHandler(&stringList, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})})

	bamboo.BambooLoggers[testAppNameContextKey] = logger
	bamboo.BambooLoggers[bamboo.BambooWorkerLoggerContextKey] = logger
	bamboo.BambooLoggers[bamboo.BambooWorkerClientLoggerContextKey] = logger
	bamboo.BambooLoggers[bamboo.BambooWorkerJobLoggerContextKey] = logger
	bamboo.BambooLoggers[bamboo.BambooRequestProducerLoggerContextKey] = logger
	bamboo.BambooLoggers[bamboo.BambooRequestConsumerLoggerContextKey] = logger
	bamboo.BambooLoggers[bamboo.BambooHeartbeatPublisherLoggerContextKey] = logger
	bamboo.BambooLoggers[bamboo.BambooResultPublisherLoggerContextKey] = logger
	bamboo.BambooLoggers[bamboo.BambooResultSubscriberLoggerContextKey] = logger

	return stringList
}

var (
	logConfigFunc = func(ctx context.Context, headers map[string]string) context.Context {
		for k, v := range headers {
			if k == bamboo.RequestIDKey {
				ctx = bamboo.WithValue(ctx, bamboo.RequestIDContextKey, v)
			}
		}
		return ctx
	}

	workerFunc = func(ctx context.Context, headers map[string]string, reqBytes []byte, aborted <-chan interface{}) ([]byte, error) {
		logger := bamboo.GetLoggerFromContext(ctx, testAppNameContextKey)
		ctx = bamboo.WithLoggerName(ctx, testAppNameContextKey)

		req := pb_test.WorkerTestParameter{}
		if err := proto.Unmarshal(reqBytes, &req); err != nil {
			return nil, internal.Errorf("proto.Unmarshal. err: %w", err)
		}

		if req.Fail {
			return nil, errors.New("FAIL")
		}

		time.Sleep(time.Duration(req.WaitMSec) * time.Millisecond)

		answer := req.X * req.Y
		logger.InfoContext(ctx, fmt.Sprintf("answer: %d", answer))

		resp := pb_test.WorkerTestResponse{Value: answer}
		respBytes, err := proto.Marshal(&resp)
		if err != nil {
			return nil, internal.Errorf("proto.Marshal. err: %w", err)
		}

		return respBytes, nil
	}
	emptyEventHandler = bamboo.NewEmptyEventHandler()
)

func Test_WorkerClient_Call(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	initLog()
	internal.UseXerrorsErrorf()
	logger := bamboo.GetLoggerFromContext(ctx, "bamboo_test")

	type inputs struct {
		debugPublishDelay       time.Duration
		heartbeatIntervalMSec   int
		connectTimeoutMSec      int
		jobTimeoutMSec          int
		waitMSec                int
		failJob                 bool
		emptyHeartbeatPublisher bool
	}
	type outputs struct {
		callError error
	}
	tests := []struct {
		name    string
		inputs  inputs
		outputs outputs
	}{
		{
			name: "Success_without_Heartbeat",
			inputs: inputs{
				heartbeatIntervalMSec: 0,
				jobTimeoutMSec:        200,
				waitMSec:              100,
			},
			outputs: outputs{
				callError: nil,
			},
		},
		{
			name: "ConnectTimedout",
			inputs: inputs{
				debugPublishDelay:     2000 * time.Millisecond,
				heartbeatIntervalMSec: 0,
				connectTimeoutMSec:    10,
				jobTimeoutMSec:        5000,
				waitMSec:              100,
			},
			outputs: outputs{
				callError: bamboo.ErrConnectTimedout,
			},
		},
		{
			name: "JobTimedout",
			inputs: inputs{
				heartbeatIntervalMSec: 0,
				jobTimeoutMSec:        100,
				waitMSec:              200,
			},
			outputs: outputs{
				callError: bamboo.ErrJobTimedout,
			},
		},
		{
			name: "Success_with_Heartbeat",
			inputs: inputs{
				heartbeatIntervalMSec: 200,
				jobTimeoutMSec:        800,
				waitMSec:              400,
			},
			outputs: outputs{
				callError: nil,
			},
		},
		{
			name: "Aborted",
			inputs: inputs{
				heartbeatIntervalMSec:   100,
				jobTimeoutMSec:          800,
				waitMSec:                400,
				emptyHeartbeatPublisher: true,
			},
			outputs: outputs{
				callError: bamboo.ErrAborted,
			},
		},
		{
			name: "Failure",
			inputs: inputs{
				heartbeatIntervalMSec: 200,
				jobTimeoutMSec:        800,
				waitMSec:              400,
				failJob:               true,
			},
			outputs: outputs{
				callError: bamboo.ErrInternalError,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pubsubMap := bamboo.NewGoroutineBambooPubSubMap()
			queue := make(chan []byte, 1)
			resultPublisher := bamboo.NewGoroutineBambooResultPublisher(pubsubMap, tt.inputs.debugPublishDelay)
			heartbeatPublisher := bamboo.NewGoroutineBambooHeartbeatPublisher(pubsubMap)
			if tt.inputs.emptyHeartbeatPublisher {
				heartbeatPublisher = NewEmptyBambooHeartbeatPublisher()
			}

			createBambooRequestConsumerFunc := func(ctx context.Context) bamboo.BambooRequestConsumer {
				return bamboo.NewGoroutineBambooRequestConsumer(queue)
			}

			requestProducer := bamboo.NewGoroutineBambooRequestProducer(ctx, "WORKER-NAME", queue)
			resultSubscriber := bamboo.NewGoroutineBambooResultSubscriber(ctx, "WORKER-NAME", pubsubMap)
			workerClient := bamboo.NewBambooWorkerClient(requestProducer, resultSubscriber, logConfigFunc)
			worker, err := bamboo.NewBambooWorker(createBambooRequestConsumerFunc, resultPublisher, heartbeatPublisher, workerFunc, 1, logConfigFunc, emptyEventHandler)
			require.Nil(t, err)

			go func() {
				time.Sleep(10 * time.Millisecond)
				time.AfterFunc(5000*time.Millisecond, cancel)
				err := worker.Run(ctx)
				assert.NoError(t, err)
			}()

			req := pb_test.WorkerTestParameter{
				X:        3,
				Y:        5,
				WaitMSec: int32(tt.inputs.waitMSec),
				Fail:     tt.inputs.failJob,
			}
			reqBytes, err := proto.Marshal(&req)
			require.Nil(t, err)

			respBytes, err := workerClient.Call(ctx, tt.inputs.heartbeatIntervalMSec, tt.inputs.connectTimeoutMSec, tt.inputs.jobTimeoutMSec, map[string]string{}, reqBytes)
			if tt.outputs.callError != nil {
				logger.ErrorContext(ctx, fmt.Sprintf("%+v", err))
				assert.ErrorIs(t, err, tt.outputs.callError)
				return
			}

			require.Nil(t, err)

			resp := pb_test.WorkerTestResponse{}
			err = proto.Unmarshal(respBytes, &resp)
			require.Nil(t, err)

			assert.Equal(t, 15, int(resp.Value))
		})
	}
}
