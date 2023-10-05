package bamboo

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/pecolynx/bamboo/internal"
	mocks "github.com/pecolynx/bamboo/mocks"
	pb "github.com/pecolynx/bamboo/proto"
	"github.com/pecolynx/bamboo/sloghelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var anythingOfContext = mock.MatchedBy(func(_ context.Context) bool { return true })

type TestBambooHandler struct {
	slog.Handler
}

var (
	ClientNameKey = "client_name"
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

func Test_consumeRequestAndDispatchJob(t *testing.T) {
	stringList := stringList{list: make([]string, 0)}
	logger := slog.New(&sloghelper.BambooHandler{Handler: slog.NewJSONHandler(&stringList, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})})
	sloghelper.BambooLoggers[sloghelper.BambooWorkerLoggerKey] = logger
	ctx := context.Background()
	ctx = context.WithValue(ctx, sloghelper.LoggerNameKey, sloghelper.BambooWorkerLoggerKey)

	req := pb.WorkerParameter{
		Headers: map[string]string{
			ClientNameKey: "CLIENT-NAME",
		},
		HeartbeatIntervalSec: 0,
		ResultChannel:        "RESULT-CHANNEL",
	}
	consumer := mocks.BambooRequestConsumer{}
	consumer.On("Consume", ctx).Return(&req, nil)
	// heartbeatPublisher := mocks.BambooHeartbeatPublisher{}
	// heartbeatPublisher.On("Run", anythingOfContext, "RESULT-CHANNEL", 0)
	var reqCtx context.Context
	logConfigFunc := func(ctx context.Context, headers map[string]string) context.Context {
		for k, v := range headers {
			if k == sloghelper.RequestIDKey {
				ctx = context.WithValue(ctx, sloghelper.RequestIDKey, v)
			}
		}
		reqCtx = ctx
		return reqCtx
	}
	worker := bambooWorker{
		logConfigFunc: logConfigFunc,
	}

	// worker, err := bamboo.NewBambooWorker(
	// 	createBambooRequestConsumerFunc,
	// 	nil,
	// 	nil,
	// 	nil,
	// 	0,
	// 	logConfigFunc,
	// )
	// assert.Nil(t, err)

	jobQueue := make(chan internal.Job, 1)

	worker.consumeRequestAndDispatchJob(ctx, &consumer, jobQueue)

	assert.Len(t, stringList.list, 3)
	for i, s := range stringList.list {
		logStruct := logStruct{}
		if err := json.Unmarshal([]byte(s), &logStruct); err != nil {
			t.Fatal(err)
		}
		switch i {
		case 0:
			assert.Equal(t, logStruct.Level, "DEBUG")
			assert.Equal(t, logStruct.LoggerName, "BambooWorker")
			assert.Equal(t, logStruct.Message, "worker is ready")
		case 1:
			assert.Equal(t, logStruct.Level, "DEBUG")
		}
		t.Logf("err %s", s)
	}
	job := <-jobQueue
	t.Logf("%s", job)
}
