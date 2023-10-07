package main

import (
	"context"
	"errors"
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
	"github.com/pecolynx/bamboo/sloghelper"
)

var tracer = otel.Tracer("github.com/pecolynx/bamboo/example/worker-redis-redis")
var appName string

func main() {
	ctx := context.Background()
	appMode := "debug"

	cfg, tp := initialize(ctx, appMode)
	defer tp.ForceFlush(ctx) // flushes any pending spans

	appName = cfg.App.Name

	debugHandler := &sloghelper.BambooHandler{Handler: slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})}

	sloghelper.BambooLoggers[cfg.App.Name] = slog.New(debugHandler)
	sloghelper.BambooLoggers[sloghelper.BambooWorkerLoggerKey] = slog.New(debugHandler)
	sloghelper.BambooLoggers[sloghelper.BambooWorkerJobLoggerKey] = slog.New(debugHandler)
	sloghelper.BambooLoggers[sloghelper.BambooWorkerClientLoggerKey] = slog.New(debugHandler)
	sloghelper.BambooLoggers[sloghelper.BambooRequestProducerLoggerKey] = slog.New(debugHandler)
	sloghelper.BambooLoggers[sloghelper.BambooRequestConsumerLoggerKey] = slog.New(debugHandler)
	sloghelper.BambooLoggers[sloghelper.BambooResultPublisherLoggerKey] = slog.New(debugHandler)
	sloghelper.BambooLoggers[sloghelper.BambooResultSubscriberLoggerKey] = slog.New(debugHandler)
	sloghelper.Init(ctx)

	logger := sloghelper.FromContext(ctx, appName)
	ctx = context.WithValue(ctx, sloghelper.LoggerNameKey, cfg.App.Name)

	factory := helper.NewBambooFactory()
	worker, err := factory.CreateBambooWorker(cfg.Worker, workerFunc)
	if err != nil {
		panic(err)
	}

	logger.InfoContext(ctx, fmt.Sprintf("Started %s", appName))

	result := run(ctx, worker)

	time.Sleep(time.Second)

	logger.InfoContext(ctx, "exited")
	os.Exit(result)
}

func run(ctx context.Context, worker bamboo.BambooWorker) int {
	eg, ctx := errgroup.WithContext(ctx)
	logger := sloghelper.FromContext(ctx, appName)

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

func initialize(ctx context.Context, mode string) (*Config, *sdktrace.TracerProvider) {
	cfg, err := LoadConfig(mode)
	if err != nil {
		panic(err)
	}

	// init log
	if err := helper.InitLog(mode, cfg.Log); err != nil {
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

	ctx, span := tracer.Start(ctx, "workerFunc")
	defer span.End()

	req := RedisRedisParameter{}
	if err := proto.Unmarshal(reqBytes, &req); err != nil {
		return nil, internal.Errorf("proto.Unmarshal. err: %w", err)
	}

	time.Sleep(time.Second * 1)

	answer := req.X * req.Y
	logger.InfoContext(ctx, fmt.Sprintf("answer: %d", answer))

	resp := RedisRedisResponse{Value: answer}
	respBytes, err := proto.Marshal(&resp)
	if err != nil {
		return nil, internal.Errorf("proto.Marshal. err: %w", err)
	}

	return respBytes, nil
}
