package bamboo

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/pecolynx/bamboo/internal"
	mocks "github.com/pecolynx/bamboo/mocks"
	pb "github.com/pecolynx/bamboo/proto"
	"github.com/pecolynx/bamboo/sloghelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var anythingOfContext = mock.MatchedBy(func(_ context.Context) bool { return true })
var anythingOfChanIn = mock.MatchedBy(func(_ <-chan interface{}) bool { return true })

type TestBambooHandler struct {
	slog.Handler
}

var (
	ClientNameKey = "client_name"

	logConfigFunc = func(ctx context.Context, headers map[string]string) context.Context {
		for k, v := range headers {
			if k == sloghelper.RequestIDKey {
				ctx = sloghelper.WithValue(ctx, sloghelper.RequestIDContextKey, v)
			}
		}
		return ctx
	}
)

func (h *TestBambooHandler) Handle(ctx context.Context, record slog.Record) error {
	requestID, ok := ctx.Value(ClientNameKey).(string)
	if ok {
		record.AddAttrs(slog.String(ClientNameKey, requestID))
	}
	return h.Handler.Handle(ctx, record)
}

func (h *TestBambooHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &TestBambooHandler{
		Handler: h.Handler.WithAttrs(attrs),
	}
}

func (h *TestBambooHandler) WithGroup(name string) slog.Handler {
	return &TestBambooHandler{
		Handler: h.Handler.WithGroup(name),
	}
}

type logStruct struct {
	Level      string `json:"level"`
	ClientName string `json:"client_name"`
	LoggerName string `json:"bamboo_logger_name"`
	Message    string `json:"msg"`
}

type stringList struct {
	list []string
}

func (s *stringList) Write(p []byte) (n int, err error) {
	s.list = append(s.list, string(p))
	return len(p), nil
}

func Test_bambooWorker_run(t *testing.T) {
	ctx := context.Background()
	ctx = sloghelper.WithLoggerName(ctx, sloghelper.BambooWorkerLoggerContextKey)

	req := pb.WorkerParameter{
		Headers: map[string]string{
			ClientNameKey: "CLIENT-NAME",
		},
		HeartbeatIntervalMSec: 0,
		ResultChannel:         "RESULT-CHANNEL",
	}

	type inputs struct {
		consumeError error
	}
	type outputs struct {
		wantError bool
	}
	tests := []struct {
		name    string
		inputs  inputs
		outputs outputs
	}{
		{
			name:   "Run_HeartBeatPublisher",
			inputs: inputs{},
			outputs: outputs{
				wantError: false,
			},
		},
		{
			name: "Run_HeartBeatPublisher",
			inputs: inputs{
				consumeError: errors.New("CONSUME_ERROR"),
			},
			outputs: outputs{
				wantError: true,
			},
		},
		{
			name: "Run_HeartBeatPublisher",
			inputs: inputs{
				consumeError: ErrContextCanceled,
			},
			outputs: outputs{
				wantError: false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctxCancel, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
			defer cancel()

			consumer := mocks.BambooRequestConsumer{}
			consumer.On("Consume", ctxCancel).Return(&req, tt.inputs.consumeError)
			consumer.On("Close", ctxCancel).Return(nil)
			consumer.On("Ping", ctxCancel).Return(nil)
			resultPublisher := mocks.BambooResultPublisher{}
			resultPublisher.On("Ping", ctxCancel).Return(nil)
			heartbeatPublisher := mocks.BambooHeartbeatPublisher{}
			heartbeatPublisher.On("Ping", ctxCancel).Return(nil)

			createRequestConsumerFunc := func(ctx context.Context) BambooRequestConsumer {
				return &consumer
			}
			workerPool := make(chan chan internal.Job, 1)
			worker := bambooWorker{
				createRequestConsumerFunc: createRequestConsumerFunc,
				resultPublisher:           &resultPublisher,
				heartbeatPublisher:        &heartbeatPublisher,
				logConfigFunc:             logConfigFunc,
				workerPool:                workerPool,
			}

			// time.AfterFunc(100*time.Millisecond, cancel)
			jobQueue := make(chan internal.Job, 1)
			workerPool <- jobQueue
			err := worker.run(ctxCancel)
			if tt.outputs.wantError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func Test_bambooWorker_consumeRequestAndDispatchJob(t *testing.T) {
	ctx := context.Background()
	ctx = sloghelper.WithLoggerName(ctx, sloghelper.BambooWorkerLoggerContextKey)

	type inputs struct {
		heartbeatIntervalMSec       int
		consumeError                error
		heartbeatPublishderRunError error
	}
	type outputs struct {
		hasJob                             bool
		numberOfCallsHeartbeatPublisherRun int
		numberOfLogs                       int
		wantError                          bool
	}
	tests := []struct {
		name    string
		inputs  inputs
		outputs outputs
	}{
		{
			name: "Run_HeartBeatPublisher",
			inputs: inputs{
				heartbeatIntervalMSec: 1000,
			},
			outputs: outputs{
				hasJob:                             true,
				numberOfCallsHeartbeatPublisherRun: 1,
				numberOfLogs:                       3,
				wantError:                          false,
			},
		},
		{
			name: "DoNotRun_HeartBeatPublisher",
			inputs: inputs{
				heartbeatIntervalMSec: 0,
			},
			outputs: outputs{
				hasJob:                             true,
				numberOfCallsHeartbeatPublisherRun: 0,
				numberOfLogs:                       3,
				wantError:                          false,
			},
		},
		{
			name: "Raise_Error",
			inputs: inputs{
				heartbeatIntervalMSec: 0,
				consumeError:          errors.New("CONSUME_ERROR"),
			},
			outputs: outputs{
				hasJob:                             true,
				numberOfCallsHeartbeatPublisherRun: 0,
				numberOfLogs:                       1,
				wantError:                          true,
			},
		},
		{
			name: "Raise_ErrContextCanceled",
			inputs: inputs{
				heartbeatIntervalMSec: 0,
				consumeError:          ErrContextCanceled,
			},
			outputs: outputs{
				hasJob:                             false,
				numberOfCallsHeartbeatPublisherRun: 0,
				numberOfLogs:                       1,
				wantError:                          true,
			},
		},
		{
			name: "HeartBeatPublishder_Raise_Error",
			inputs: inputs{
				heartbeatIntervalMSec:       1000,
				heartbeatPublishderRunError: errors.New("Run"),
			},
			outputs: outputs{
				hasJob:                             true,
				numberOfCallsHeartbeatPublisherRun: 1,
				numberOfLogs:                       2,
				wantError:                          true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jobQueue := make(chan internal.Job, 1)
			stringList := stringList{list: make([]string, 0)}
			logger := slog.New(&sloghelper.BambooHandler{Handler: slog.NewJSONHandler(&stringList, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			})})
			sloghelper.BambooLoggers[sloghelper.BambooWorkerLoggerContextKey] = logger

			// given
			req := pb.WorkerParameter{
				Headers: map[string]string{
					ClientNameKey: "CLIENT-NAME",
				},
				HeartbeatIntervalMSec: int32(tt.inputs.heartbeatIntervalMSec),
				ResultChannel:         "RESULT-CHANNEL",
			}

			consumer := mocks.BambooRequestConsumer{}
			consumer.On("Consume", ctx).Return(&req, tt.inputs.consumeError)
			heartbeatPublisher := mocks.BambooHeartbeatPublisher{}
			heartbeatPublisher.On("Run", anythingOfContext, "RESULT-CHANNEL", tt.inputs.heartbeatIntervalMSec, anythingOfChanIn, anythingOfChanIn).Return(tt.inputs.heartbeatPublishderRunError)

			worker := bambooWorker{
				heartbeatPublisher: &heartbeatPublisher,
				logConfigFunc:      logConfigFunc,
			}

			// when
			err := worker.consumeRequestAndDispatchJob(ctx, &consumer, jobQueue)

			var hasJob bool
			select {
			case <-jobQueue:
				hasJob = true
			case <-time.After(100 * time.Millisecond):
			}

			// then
			if tt.outputs.wantError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, hasJob, tt.outputs.hasJob)
			assert.Len(t, stringList.list, tt.outputs.numberOfLogs)

			for i, s := range stringList.list {
				logStruct := logStruct{}
				err := json.Unmarshal([]byte(s), &logStruct)
				require.NoError(t, err)

				switch i {
				case 0:
					assert.Equal(t, logStruct.Level, "DEBUG")
					assert.Equal(t, logStruct.LoggerName, "BambooWorker")
					assert.Equal(t, logStruct.Message, "worker is ready")
				case 1:
					assert.Equal(t, logStruct.Level, "DEBUG")
					assert.Equal(t, logStruct.LoggerName, "BambooWorker")
					assert.Equal(t, logStruct.Message, "request is received")
				case 2:
					assert.Equal(t, logStruct.Level, "DEBUG")
					assert.Equal(t, logStruct.LoggerName, "BambooWorker")
					assert.Equal(t, logStruct.Message, "dispatch job to worker")
				}
				t.Logf("err %s", s)
			}
			heartbeatPublisher.AssertNumberOfCalls(t, "Run", tt.outputs.numberOfCallsHeartbeatPublisherRun)
		})
	}
}
