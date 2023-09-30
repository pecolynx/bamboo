package main

import (
	"context"
	"os"
	"time"

	"github.com/sirupsen/logrus"
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

func main() {
	ctx := context.Background()
	appMode := "debug"

	cfg, tp := initialize(ctx, appMode)
	defer tp.ForceFlush(ctx) // flushes any pending spans

	factory := helper.NewBambooFactory()
	worker, err := factory.CreateBambooWorker(cfg.Worker, workerFunc)
	if err != nil {
		panic(err)
	}

	logger := internal.FromContext(ctx)
	logger.Info("Started worker-redis-redis")
	result := run(ctx, worker)

	time.Sleep(time.Second)

	logrus.Info("exited")
	os.Exit(result)
}

func run(ctx context.Context, worker bamboo.BambooWorker) int {
	eg, ctx := errgroup.WithContext(ctx)

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
		logrus.Error(err)
		return 1
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
	logger := internal.FromContext(ctx)

	req := RedisRedisParameter{}
	if err := proto.Unmarshal(reqBytes, &req); err != nil {
		logger.Errorf("proto.Unmarshal %+v", err)
		return nil, internal.Errorf("proto.Unmarshal. err: %w", err)
	}

	time.Sleep(time.Second * 5)

	answer := req.X * req.Y
	logger.Infof("answer: %d", answer)
	resp := RedisRedisResponse{Value: answer}
	respBytes, err := proto.Marshal(&resp)
	if err != nil {
		return nil, internal.Errorf("proto.Marshal. err: %w", err)
	}

	return respBytes, nil
}
