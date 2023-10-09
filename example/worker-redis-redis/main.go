package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"

	"github.com/pecolynx/bamboo"
	"github.com/pecolynx/bamboo/helper"
	"github.com/pecolynx/bamboo/internal"
)

var tracer = otel.Tracer("github.com/pecolynx/bamboo/example/worker-redis-redis")
var appNameContextKey bamboo.ContextKey

func getValue(values ...string) string {
	for _, v := range values {
		if len(v) != 0 {
			return v
		}
	}
	return ""
}

func main() {
	ctx := context.Background()
	appModeParam := flag.String("app_mode", "", "")
	flag.Parse()
	appMode := getValue(*appModeParam, os.Getenv("APP_MODE"), "debug")

	cfg, tp := initialize(ctx, appMode)
	defer tp.ForceFlush(ctx) // flushes any pending spans

	bamboo.InitLogger(ctx)

	appNameContextKey = bamboo.ContextKey(cfg.App.Name)
	logger := bamboo.GetLoggerFromContext(ctx, appNameContextKey)
	ctx = bamboo.WithValue(ctx, bamboo.LoggerNameContextKey, cfg.App.Name)

	factory := helper.NewBambooFactory()
	worker, err := factory.CreateBambooWorker(cfg.Worker, workerFunc)
	if err != nil {
		panic(err)
	}

	logger.InfoContext(ctx, fmt.Sprintf("Started %s", appNameContextKey))

	result := run(ctx, worker)

	time.Sleep(time.Second)

	logger.InfoContext(ctx, "exited")
	os.Exit(result)
}

func run(ctx context.Context, worker bamboo.BambooWorker) int {
	eg, ctx := errgroup.WithContext(ctx)
	logger := bamboo.GetLoggerFromContext(ctx, appNameContextKey)

	eg.Go(func() error {
		return worker.Run(ctx)
	})
	eg.Go(func() error {
		return bamboo.MetricsServerProcess(ctx, 8081, 1)
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

func workerFunc(ctx context.Context, headers map[string]string, reqBytes []byte, aborted <-chan interface{}) ([]byte, error) {
	logger := bamboo.GetLoggerFromContext(ctx, appNameContextKey)

	ctx, span := tracer.Start(ctx, "workerFunc")
	defer span.End()

	req := RedisRedisParameter{}
	if err := proto.Unmarshal(reqBytes, &req); err != nil {
		return nil, internal.Errorf("proto.Unmarshal. err: %w", err)
	}

	time.Sleep(time.Duration(req.JobSec) * time.Second)

	answer := req.X * req.Y
	logger.InfoContext(ctx, fmt.Sprintf("answer: %d", answer))

	resp := RedisRedisResponse{Value: answer}
	respBytes, err := proto.Marshal(&resp)
	if err != nil {
		return nil, internal.Errorf("proto.Marshal. err: %w", err)
	}

	return respBytes, nil
}
