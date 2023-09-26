package main

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/protobuf/proto"

	"github.com/pecolynx/bamboo/helper"
	"github.com/pecolynx/bamboo/internal"
)

var tracer = otel.Tracer("github.com/pecolynx/bamboo/example/calc-app")

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

func (e *expr) workerRedisRedis(ctx context.Context, x, y int) int {
	logger := internal.FromContext(ctx)

	request_id, _ := ctx.Value("request_id").(string)
	headers := map[string]string{
		"request_id": request_id,
	}

	if err := e.getError(); err != nil {
		logger.Info("%+v", err)
		return 0
	}

	p1 := RedisRedisParameter{X: int32(x), Y: int32(y)}
	paramBytes, err := proto.Marshal(&p1)
	if err != nil {
		e.setError(internal.Errorf("proto.Marshal. err: %w", err))
		return 0
	}

	respBytes, err := e.app.Call(ctx, "worker-redis-redis", 2, 10, headers, paramBytes)
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
	ctx := context.Background()
	appMode := "debug"

	cfg, tp := initialize(ctx, appMode)
	defer tp.ForceFlush(ctx) // flushes any pending spans

	clients := map[string]helper.WorkerClient{}
	for k, v := range cfg.Workers {
		var err error
		clients[k], err = helper.CreateWorkerClient(ctx, k, v, otel.GetTextMapPropagator())
		if err != nil {
			panic(err)
		}
		defer clients[k].Close(ctx)
	}

	app := helper.StandardClient{Clients: clients}

	logger := internal.FromContext(ctx)
	logger.Info("Started calc-app")

	wg := sync.WaitGroup{}
	for i := 0; i < 1; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			spanCtx, span := tracer.Start(ctx, "calc-app")
			defer span.End()

			requestID, err := uuid.NewRandom()
			if err != nil {
				panic(err)
			}

			logCtx := internal.With(spanCtx, internal.Str("request_id", requestID.String()))
			logCtx = context.WithValue(logCtx, "request_id", requestID.String())
			logger := internal.FromContext(logCtx)

			expr := expr{app: &app}

			a := expr.workerRedisRedis(logCtx, 3, 5)
			b := expr.workerRedisRedis(logCtx, a, 7)

			if expr.getError() != nil {
				logger.Errorf("failed to run (3 * 5 * 7). err: %v", expr.getError())
			} else {
				logger.Infof("3 * 5 * 7= %d", b)
			}
		}()
	}
	wg.Wait()
}

func initialize(ctx context.Context, env string) (*Config, *sdktrace.TracerProvider) {
	cfg, err := LoadConfig(env)
	if err != nil {
		panic(err)
	}

	// init log
	if err := helper.InitLog(env, cfg.Log); err != nil {
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
