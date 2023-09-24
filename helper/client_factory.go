package helper

import (
	"context"

	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel/propagation"

	"github.com/pecolynx/bamboo"
)

func CreateWorkerClient(ctx context.Context, workerName string, cfg *WorkerClientConfig, propagator propagation.TextMapPropagator) WorkerClient {
	var rp bamboo.BambooRequestProducer
	var rs bamboo.BambooResultSubscriber

	if cfg.RequestProducer.Type == "kafka" {
		rp = bamboo.NewKafkaBambooRequestProducer(ctx, workerName, &kafka.Writer{
			Addr:     kafka.TCP(cfg.RequestProducer.Kafka.Addr),
			Topic:    cfg.RequestProducer.Kafka.Topic,
			Balancer: &kafka.LeastBytes{},
		}, propagator)
	} else if cfg.RequestProducer.Type == "redis" {
		rp = bamboo.NewRedisBambooRequestProducer(ctx, workerName, redis.UniversalOptions{
			Addrs: cfg.RequestProducer.Redis.Addrs,
		}, cfg.RequestProducer.Redis.Channel, propagator)
	}
	if cfg.ResultSubscriber.Type == "redis" {
		rs = bamboo.NewRedisResultSubscriber(ctx, workerName, &redis.UniversalOptions{
			Addrs:    cfg.ResultSubscriber.Redis.Addrs,
			Password: cfg.ResultSubscriber.Redis.Password,
		})
	}

	return NewWorkerClient(rp, rs)
}
