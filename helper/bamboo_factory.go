package helper

import (
	"context"
	"fmt"
	"time"

	"github.com/pecolynx/bamboo"
	"github.com/pecolynx/bamboo/internal"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
)

type BambooFactory interface {
	CreateBambooWorkerClient(ctx context.Context, workerName string, cfg *WorkerClientConfig) (bamboo.BambooWorkerClient, error)
	CreateBambooWorker(cfg *WorkerConfig, workerFunc bamboo.WorkerFunc) (bamboo.BambooWorker, error)
}

type bambooFactory struct {
	queueMap  map[string]chan []byte
	pubsubMap bamboo.GoroutineBambooPubSubMap
}

func NewBambooFactory() BambooFactory {
	return &bambooFactory{
		queueMap:  make(map[string]chan []byte),
		pubsubMap: bamboo.NewGoroutineBambooPubSubMap(),
	}
}

func (f *bambooFactory) CreateBambooWorkerClient(ctx context.Context, workerName string, cfg *WorkerClientConfig) (bamboo.BambooWorkerClient, error) {
	var rp bamboo.BambooRequestProducer
	var rs bamboo.BambooResultSubscriber

	if cfg.RequestProducer.Type == "kafka" {
		rp = bamboo.NewKafkaBambooRequestProducer(ctx, workerName, &kafka.Writer{
			Addr:     kafka.TCP(cfg.RequestProducer.Kafka.Addr),
			Topic:    cfg.RequestProducer.Kafka.Topic,
			Balancer: &kafka.LeastBytes{},
		})
	} else if cfg.RequestProducer.Type == "redis" {
		rp = bamboo.NewRedisBambooRequestProducer(ctx, workerName, redis.UniversalOptions{
			Addrs: cfg.RequestProducer.Redis.Addrs,
		}, cfg.RequestProducer.Redis.Channel)
	} else if cfg.RequestProducer.Type == "goroutine" {
		if _, ok := f.queueMap[cfg.RequestProducer.Goroutine.Channel]; !ok {
			f.queueMap[cfg.RequestProducer.Goroutine.Channel] = make(chan []byte)
		}
		queue := f.queueMap[cfg.RequestProducer.Goroutine.Channel]
		rp = bamboo.NewGoroutineBambooRequestProducer(ctx, workerName, queue)
	} else {
		return nil, internal.Errorf("invalid type of request producer: %s", cfg.RequestProducer.Type)
	}

	if cfg.ResultSubscriber.Type == "redis" {
		rs = bamboo.NewRedisBambooResultSubscriber(ctx, workerName, &redis.UniversalOptions{
			Addrs:    cfg.ResultSubscriber.Redis.Addrs,
			Password: cfg.ResultSubscriber.Redis.Password,
		})
	} else if cfg.ResultSubscriber.Type == "goroutine" {
		rs = bamboo.NewGoroutineBambooResultSubscriber(ctx, workerName, f.pubsubMap)
	} else {
		return nil, internal.Errorf("invalid type of request producer: %s", cfg.RequestProducer.Type)
	}

	return bamboo.NewBambooWorkerClient(rp, rs), nil
}

func (f *bambooFactory) CreateBambooWorker(cfg *WorkerConfig, workerFunc bamboo.WorkerFunc) (bamboo.BambooWorker, error) {
	var resultPublisher bamboo.BambooResultPublisher
	var heartbeatPublisher bamboo.BambooHeartbeatPublisher

	if cfg.Publisher.Type == "redis" {
		publisherOptions := &redis.UniversalOptions{
			Addrs:    cfg.Publisher.Redis.Addrs,
			Password: cfg.Publisher.Redis.Password,
		}
		resultPublisher = bamboo.NewRedisBambooResultPublisher(publisherOptions)
		heartbeatPublisher = bamboo.NewRedisBambooHeartbeatPublisher(publisherOptions)
	} else if cfg.Publisher.Type == "goroutine" {
		resultPublisher = bamboo.NewGoroutineBambooResultPublisher(f.pubsubMap)
		heartbeatPublisher = bamboo.NewGoroutineBambooHeartbeatPublisher(f.pubsubMap)
	} else {
		return nil, fmt.Errorf("invalid publisher type: %s", cfg.Publisher.Type)
	}

	var createBambooRequestConsumerFunc bamboo.CreateBambooRequestConsumerFunc
	if cfg.Consumer.Type == "redis" {
		consumerOptions := &redis.UniversalOptions{
			Addrs:    cfg.Consumer.Redis.Addrs,
			Password: cfg.Consumer.Redis.Password,
		}

		createBambooRequestConsumerFunc = func(ctx context.Context) bamboo.BambooRequestConsumer {
			return bamboo.NewRedisBambooRequestConsumer(consumerOptions, cfg.Consumer.Redis.Channel, time.Duration(cfg.Consumer.Redis.RequestWaitTimeoutMSec)*time.Millisecond)
		}
	} else if cfg.Consumer.Type == "kafka" {
		consumerOptions := kafka.ReaderConfig{
			Brokers:  cfg.Consumer.Kafka.Brokers,
			GroupID:  cfg.Consumer.Kafka.GroupID,
			Topic:    cfg.Consumer.Kafka.Topic,
			MaxBytes: 10e6, // 10MB
		}

		createBambooRequestConsumerFunc = func(ctx context.Context) bamboo.BambooRequestConsumer {
			return bamboo.NewKafkaBambooRequestConsumer(consumerOptions, time.Duration(cfg.Consumer.Redis.RequestWaitTimeoutMSec)*time.Millisecond)
		}
	} else if cfg.Consumer.Type == "goroutine" {
		if _, ok := f.queueMap[cfg.Consumer.Goroutine.Channel]; !ok {
			f.queueMap[cfg.Consumer.Goroutine.Channel] = make(chan []byte)
		}
		queue := f.queueMap[cfg.Consumer.Goroutine.Channel]

		createBambooRequestConsumerFunc = func(ctx context.Context) bamboo.BambooRequestConsumer {
			return bamboo.NewGoroutineBambooRequestConsumer(queue)
		}
	} else {
		return nil, fmt.Errorf("invalid consumer type: %s", cfg.Consumer.Type)
	}

	return bamboo.NewBambooWorker(createBambooRequestConsumerFunc, resultPublisher, heartbeatPublisher, workerFunc, cfg.NumWorkers, LogConfigFunc)
}
