package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"

	"github.com/pecolynx/bamboo"
	"github.com/pecolynx/bamboo/helper"
	"github.com/pecolynx/bamboo/internal"
	"github.com/pecolynx/bamboo/sloghelper"
)

var tracer = otel.Tracer("github.com/pecolynx/bamboo/example/goroutine-app")
var appName string

type expr struct {
	app *helper.StandardClient
	err error
	mu  sync.Mutex
}

func (e *expr) getError() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.err != nil {
		return e.err
	}
	return nil
}

func (e *expr) workerGoroutine(ctx context.Context, x, y int) int {
	logger := sloghelper.FromContext(ctx, appName)

	request_id, _ := ctx.Value(sloghelper.RequestIDKey).(string)
	headers := map[string]string{
		sloghelper.RequestIDKey: request_id,
	}

	if err := e.getError(); err != nil {
		logger.InfoContext(ctx, "", slog.Any("err", err))
		return 0
	}

	p1 := GoroutineAppParameter{X: int32(x), Y: int32(y)}
	paramBytes, err := proto.Marshal(&p1)
	if err != nil {
		e.setError(internal.Errorf("proto.Marshal. err: %w", err))
		return 0
	}

	respBytes, err := e.app.Call(ctx, "worker-goroutine", 0, 0, headers, paramBytes)
	if err != nil {
		e.setError(internal.Errorf("app.Call(worker-goroutine). err: %w", err))
		return 0
	}

	resp := GoroutineAppResponse{}
	if err := proto.Unmarshal(respBytes, &resp); err != nil {
		e.setError(internal.Errorf("proto.Unmarshal. worker-goroutine response is invalid. err: %w", err))
		return 0
	}

	return int(resp.Value)
}

func (e *expr) setError(err error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.err = err
}

func main() {
	ctx := context.Background()
	appMode := "debug"

	cfg, tp := initialize(ctx, appMode)
	defer tp.ForceFlush(ctx) // flushes any pending spans

	appName = cfg.App.Name

	logger := sloghelper.FromContext(ctx, appName)
	ctx = context.WithValue(ctx, sloghelper.LoggerNameKey, cfg.App.Name)

	factory := helper.NewBambooFactory()
	worker, err := factory.CreateBambooWorker(cfg.Worker, workerFunc)
	if err != nil {
		panic(err)
	}

	clients := map[string]helper.WorkerClient{}
	for k, v := range cfg.Workers {
		var err error
		clients[k], err = factory.CreateBambooWorkerClient(ctx, k, v, otel.GetTextMapPropagator())
		if err != nil {
			panic(err)
		}
		defer clients[k].Close(ctx)
	}

	app := helper.StandardClient{Clients: clients}

	logger.InfoContext(ctx, fmt.Sprintf("Started %s", appName))

	result := run(ctx, worker, app)
	time.Sleep(time.Second)

	logger.InfoContext(ctx, "exited")
	os.Exit(result)
}

func run(ctx context.Context, worker bamboo.BambooWorker, app helper.StandardClient) int {
	ctx, cancel := context.WithCancel(ctx)
	eg, ctx := errgroup.WithContext(ctx)
	logger := sloghelper.FromContext(ctx, appName)

	eg.Go(func() error {
		done := make(chan interface{})

		go func() {
			spanCtx, span := tracer.Start(ctx, appName)
			defer span.End()

			requestID, err := uuid.NewRandom()
			if err != nil {
				panic(err)
			}

			logCtx := context.WithValue(spanCtx, sloghelper.RequestIDKey, requestID.String())

			expr := expr{app: &app}

			a := expr.workerGoroutine(logCtx, 3, 5)
			// b := expr.workerGoroutine(logCtx, a, 7)

			if expr.getError() != nil {
				logger.ErrorContext(logCtx, "failed to run (3 * 5 * 7)", expr.getError())
			} else {
				logger.InfoContext(logCtx, fmt.Sprintf("3 * 5 * 7 = %d", a))
			}

			// if expr.getError() != nil {
			// 	logger.Errorf("failed to run (3 * 5 * 7). err: %v", expr.getError())
			// } else {
			// 	logger.Infof("3 * 5 * 7= %d", b)
			// }

			done <- struct{}{}
			cancel()
		}()

		select {
		case <-ctx.Done():
			break
		case <-done:
			break
		}

		return nil
	})
	eg.Go(func() error {
		return worker.Run(ctx)
	})
	eg.Go(func() error {
		return helper.SignalWatchProcess(ctx)
	})
	eg.Go(func() error {
		<-ctx.Done()
		return ctx.Err() // nolint:wrapcheck
	})

	if err := eg.Wait(); err != nil {
		if errors.Is(err, context.Canceled) {
			logger.InfoContext(ctx, "", slog.Any("err", err))
			return 0
		} else {
			logger.ErrorContext(ctx, "", slog.Any("err", err))
			return 1
		}
	}
	return 0
}

func initialize(ctx context.Context, appMode string) (*Config, *sdktrace.TracerProvider) {
	cfg, err := LoadConfig(appMode)
	if err != nil {
		panic(err)
	}

	// init log
	if err := helper.InitLog(cfg.App.Name, cfg.Log); err != nil {
		panic(err)
	}

	// init tracer
	tp, err := helper.InitTracerProvider(cfg.App.Name, cfg.Trace)
	if err != nil {
		panic(err)
	}
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return cfg, tp
}

func workerFunc(ctx context.Context, headers map[string]string, reqBytes []byte, aborted <-chan interface{}) ([]byte, error) {
	logger := sloghelper.FromContext(ctx, appName)

	req := GoroutineAppParameter{}
	if err := proto.Unmarshal(reqBytes, &req); err != nil {
		return nil, internal.Errorf("proto.Unmarshal. err: %w", err)
	}

	time.Sleep(time.Second * 1)

	answer := req.X * req.Y
	logger.InfoContext(ctx, fmt.Sprintf("answer: %d", answer))

	resp := GoroutineAppResponse{Value: answer}
	respBytes, err := proto.Marshal(&resp)
	if err != nil {
		return nil, internal.Errorf("proto.Marshal. err: %w", err)
	}

	return respBytes, nil
}
