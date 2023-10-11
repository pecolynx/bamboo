package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/protobuf/proto"

	"github.com/pecolynx/bamboo"
	"github.com/pecolynx/bamboo/helper"
	"github.com/pecolynx/bamboo/internal"
)

var tracer = otel.Tracer("github.com/pecolynx/bamboo/example/calc-app")
var appNameContextKey bamboo.ContextKey

type expr struct {
	workerClients map[string]bamboo.BambooWorkerClient
	err           error
	mu            sync.Mutex
}

func getValue(values ...string) string {
	for _, v := range values {
		if len(v) != 0 {
			return v
		}
	}
	return ""
}

func (e *expr) getError() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.err != nil {
		return e.err
	}
	return nil
}

func (e *expr) workerRedisRedis(ctx context.Context, x, y int, jobTimeoutSec int, jobSec int) int {
	logger := bamboo.GetLoggerFromContext(ctx, appNameContextKey)

	request_id, _ := ctx.Value(bamboo.RequestIDContextKey).(string)
	headers := map[string]string{
		bamboo.RequestIDKey: request_id,
	}

	if err := e.getError(); err != nil {
		logger.InfoContext(ctx, "", slog.Any("err", err))
		return 0
	}

	p1 := RedisRedisParameter{X: int32(x), Y: int32(y), JobSec: int32(jobSec)}
	paramBytes, err := proto.Marshal(&p1)
	if err != nil {
		e.setError(internal.Errorf("proto.Marshal. err: %w", err))
		return 0
	}

	workerClient, ok := e.workerClients["worker-redis-redis"]
	if !ok {
		e.setError(fmt.Errorf("worker client not found. name: %s", "worker-redis-redis"))
		return 0
	}

	respBytes, err := workerClient.Call(ctx, 2000, jobTimeoutSec*1000, headers, paramBytes)
	if err != nil {
		e.setError(internal.Errorf("app.Call(worker-redis-redis). err: %w", err))
		return 0
	}

	resp := RedisRedisResponse{}
	if err := proto.Unmarshal(respBytes, &resp); err != nil {
		e.setError(internal.Errorf("proto.Unmarshal. worker-redis-redis response is invalid. err: %w", err))
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
	fmt.Println(os.Getenv("APP_MODE"))
	fmt.Println(os.Getenv("NUM_REQUESTS"))
	ctx := context.Background()
	appModeParam := flag.String("app_mode", "", "")
	flag.Parse()
	appMode := getValue(*appModeParam, os.Getenv("APP_MODE"), "debug")

	cfg, tp := initialize(ctx, appMode)
	defer tp.ForceFlush(ctx) // flushes any pending spans

	bamboo.InitLogger(ctx)

	appNameContextKey = bamboo.ContextKey(cfg.App.Name)
	logger := bamboo.GetLoggerFromContext(ctx, appNameContextKey)
	ctx = bamboo.WithLoggerName(ctx, appNameContextKey)

	logger.DebugContext(ctx, fmt.Sprintf("cfg: %+v", cfg))

	factory := helper.NewBambooFactory()
	workerClients := map[string]bamboo.BambooWorkerClient{}
	for k, v := range cfg.Workers {
		workerClient, err := factory.CreateBambooWorkerClient(ctx, k, v)
		if err != nil {
			panic(err)
		}
		defer workerClient.Close(ctx)
		workerClients[k] = workerClient
	}

	logger.InfoContext(ctx, fmt.Sprintf("Started %s", cfg.App.Name))

	wg := sync.WaitGroup{}
	for i := 0; i < cfg.App.NumRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			spanCtx, span := tracer.Start(ctx, cfg.App.Name)
			defer span.End()

			requestID, err := uuid.NewRandom()
			if err != nil {
				panic(err)
			}

			logCtx := bamboo.WithValue(spanCtx, bamboo.RequestIDContextKey, requestID.String())

			expr := expr{workerClients: workerClients}

			a := expr.workerRedisRedis(logCtx, 3, 5, cfg.App.JobTimeoutSec, cfg.App.JobSec)
			// b := expr.workerRedisRedis(logCtx, a, 7)

			if expr.getError() != nil {
				logger.ErrorContext(logCtx, "failed to run (3 * 5 * 7)", slog.Any("err", expr.getError()))
			} else {
				logger.InfoContext(logCtx, fmt.Sprintf("3 * 5 = %d", a))
			}

			// if expr.getError() != nil {
			// 	logger.Errorf("failed to run (3 * 5 * 7). err: %v", expr.getError())
			// } else {
			// 	logger.Infof("3 * 5 * 7= %d", b)
			// }
		}()
	}
	wg.Wait()
}

func initialize(ctx context.Context, appMode string) (*Config, *sdktrace.TracerProvider) {
	cfg, err := LoadConfig(appMode)
	if err != nil {
		panic(err)
	}

	// init log
	if err := helper.InitLog(bamboo.ContextKey(cfg.App.Name), cfg.Log); err != nil {
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
