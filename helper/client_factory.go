package helper

import (
	"context"

	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel/propagation"

	"github.com/pecolynx/bamboo"
	"github.com/pecolynx/bamboo/internal"
)

func CreateWorkerClient(ctx context.Context, workerName string, cfg *WorkerClientConfig, propagator propagation.TextMapPropagator) (WorkerClient, error) {
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
	} else {
		return nil, internal.Errorf("invalid requestproducer type: %s", cfg.RequestProducer.Type)
	}

	if cfg.ResultSubscriber.Type == "redis" {
		rs = bamboo.NewRedisBambooResultSubscriber(ctx, workerName, &redis.UniversalOptions{
			Addrs:    cfg.ResultSubscriber.Redis.Addrs,
			Password: cfg.ResultSubscriber.Redis.Password,
		})
	} else {
		return nil, internal.Errorf("invalid requestproducer type: %s", cfg.RequestProducer.Type)
	}

	return NewWorkerClient(rp, rs), nil
}
