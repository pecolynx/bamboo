package helper

import (
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"

	"github.com/pecolynx/bamboo"
)

func CreateBambooWorker(cfg *WorkerConfig, workerFunc bamboo.WorkerFunc, quit chan interface{}) (bamboo.BambooWorker, error) {
	var resultPublisher bamboo.BambooResultPublisher
	var heartbeatPublisher bamboo.BambooHeartbeatPublisher

	if cfg.Publisher.Type == "redis" {
		publisherOptions := &redis.UniversalOptions{
			Addrs:    cfg.Publisher.Redis.Addrs,
			Password: cfg.Publisher.Redis.Password,
		}
		resultPublisher = bamboo.NewRedisBambooResultPublisher()
		heartbeatPublisher = bamboo.NewRedisBambooHeartbeatPublisher(publisherOptions)
	} else {
		return nil, fmt.Errorf("invalid publisher type: %s", cfg.Publisher.Type)
	}

	if cfg.Consumer.Type == "redis" {
		consumerOptions := &redis.UniversalOptions{
			Addrs:    cfg.Consumer.Redis.Addrs,
			Password: cfg.Consumer.Redis.Password,
		}

		return bamboo.NewRedisBambooWorker(consumerOptions, cfg.Consumer.Redis.Channel, time.Duration(cfg.Consumer.Redis.RequestWaitTimeoutSec)*time.Second, resultPublisher, heartbeatPublisher, workerFunc, cfg.NumWorkers, LogConfigFunc), nil
	} else if cfg.Consumer.Type == "kafka" {
		consumerOptions := kafka.ReaderConfig{
			Brokers:  cfg.Consumer.Kafka.Brokers,
			GroupID:  cfg.Consumer.Kafka.GroupID,
			Topic:    cfg.Consumer.Kafka.Topic,
			MaxBytes: 10e6, // 10MB
		}

		return bamboo.NewKafkaBambooWorker(consumerOptions, time.Duration(cfg.Consumer.Redis.RequestWaitTimeoutSec)*time.Second, resultPublisher, heartbeatPublisher, workerFunc, cfg.NumWorkers, LogConfigFunc), nil
	} else {
		return nil, fmt.Errorf("invalid consumer type: %s", cfg.Consumer.Type)
	}
}
