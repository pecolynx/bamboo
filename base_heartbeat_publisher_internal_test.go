package bamboo

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testStruct struct {
	numCallCount bool
}

func (s *testStruct) fn() {
	s.numCallCount = true
}

func Test_baseHeartbeatPublisher_Run(t *testing.T) {
	type inputs struct {
		heartbeatIntervalMSec int
		cancelMSec            int
		doneMSec              int
		abortedMSec           int
		waitMSec              int
	}
	type outputs struct {
		funcCalled bool
		logs       []string
	}
	tests := []struct {
		name    string
		inputs  inputs
		outputs outputs
	}{
		{
			name:    "Heartbeat is disabled",
			inputs:  inputs{},
			outputs: outputs{},
		},
		{
			name: "context is canceled",
			inputs: inputs{
				heartbeatIntervalMSec: 1,
				cancelMSec:            20,
				doneMSec:              30,
				abortedMSec:           30,
				waitMSec:              50,
			},
			outputs: outputs{
				funcCalled: true,
				logs:       []string{"ctx.Done(). stop heartbeat loop"},
			},
		},
		{
			name: "done",
			inputs: inputs{
				heartbeatIntervalMSec: 1,
				cancelMSec:            30,
				doneMSec:              20,
				abortedMSec:           30,
				waitMSec:              50,
			},
			outputs: outputs{
				funcCalled: true,
				logs:       []string{"done. stop heartbeat loop"},
			},
		},
		{
			name: "aborted",
			inputs: inputs{
				heartbeatIntervalMSec: 1,
				cancelMSec:            30,
				doneMSec:              30,
				abortedMSec:           20,
				waitMSec:              50,
			},
			outputs: outputs{
				funcCalled: true,
				logs:       []string{"aborted. stop heartbeat loop"},
			},
		},
	}
	for _, tt := range tests {
		testStruct := testStruct{}
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctx, cancel := context.WithTimeout(ctx, time.Duration(tt.inputs.cancelMSec)*time.Millisecond)
			defer cancel()

			done := make(chan interface{})
			aborted := make(chan interface{})
			time.AfterFunc(time.Duration(tt.inputs.doneMSec)*time.Millisecond, func() {
				done <- struct{}{}
			})
			time.AfterFunc(time.Duration(tt.inputs.abortedMSec)*time.Millisecond, func() {
				aborted <- struct{}{}
			})
			stringList := stringList{list: make([]string, 0)}
			logger := slog.New(&BambooLogHandler{Handler: slog.NewJSONHandler(&stringList, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			})})
			BambooLoggers[BambooHeartbeatPublisherLoggerContextKey] = logger

			h := &baseHeartbeatPublisher{}
			h.Run(ctx, tt.inputs.heartbeatIntervalMSec, done, aborted, testStruct.fn)
			time.Sleep(time.Duration(tt.inputs.waitMSec) * time.Millisecond)
			assert.Equal(t, tt.outputs.funcCalled, testStruct.numCallCount)

			for _, log := range tt.outputs.logs {
				found := false
				for _, s := range stringList.list {
					logStruct := logStruct{}
					err := json.Unmarshal([]byte(s), &logStruct)
					require.NoError(t, err)
					if log == logStruct.Message {
						found = true
					}
					// t.Logf("err %s", s)
				}
				require.True(t, found, fmt.Sprintf("log not found: %s", log))
			}
		})
	}
}
