package main

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/pecolynx/bamboo"
	"github.com/pecolynx/bamboo/helper"
)

var appNameContextKey bamboo.ContextKey
var channel = "helloworld-app"

func main() {
	ctx := context.Background()
	bamboo.InitLogger(ctx)
	if err := helper.InitLog(bamboo.ContextKey("helloworld-app"), &helper.LogConfig{
		Level: "debug",
	}); err != nil {
		panic(err)
	}
	factory := helper.NewBambooFactory()
	if os.Getenv("WORKER") == "TRUE" {
		worker(ctx, factory)
	} else {
		workerClient(ctx, factory)
	}
}

func worker(ctx context.Context, factory helper.BambooFactory) {
	worker, err := factory.CreateBambooWorker(&helper.WorkerConfig{
		Consumer: &helper.ConsumerConfig{
			Type: "redis",
			Redis: &helper.RedisConsumerConfig{
				Addrs:                  []string{"localhost:6379"},
				Password:               "",
				Channel:                channel,
				RequestWaitTimeoutMSec: 5000,
			},
		},
		Publisher: &helper.PublisherConfig{
			Type: "redis",
			Redis: &helper.RedisPublisherConfig{
				Addrs:    []string{"localhost:6379"},
				Password: "",
			},
		},
		NumWorkers: 1,
	}, func(ctx context.Context, headers map[string]string, reqBytes []byte, aborted <-chan interface{}) ([]byte, error) {
		return []byte(fmt.Sprintf("Hello, %s", string(reqBytes))), nil
	})
	if err != nil {
		panic(err)
	}

	if err := worker.Run(ctx); err != nil {
		panic(err)
	}
}

func workerClient(ctx context.Context, factory helper.BambooFactory) {
	workerClient, err := factory.CreateBambooWorkerClient(ctx, "worker-helloworld", &helper.WorkerClientConfig{
		RequestProducer: &helper.RequestProducerConfig{
			Type: "redis",
			Redis: &helper.RedisRequestProducerConfig{
				Addrs:    []string{"localhost:6379"},
				Password: "",
				Channel:  channel,
			},
		},
		ResultSubscriber: &helper.ResultSubscriberConfig{
			Type: "redis",
			Redis: &helper.RedisResultSubscriberConfig{
				Addrs:    []string{"localhost:6379"},
				Password: "",
			},
		},
	})
	if err != nil {
		panic(err)
	}
	defer workerClient.Close(ctx)

	fmt.Println("Started")

	headers := map[string]string{
		bamboo.RequestIDKey: uuid.New().String(),
	}
	heartbeatIntervalMSec := 0
	connectTimeoutMSec := 0
	jobTimeoutMSec := 1000

	respBytes, err := workerClient.Call(ctx, heartbeatIntervalMSec, connectTimeoutMSec, jobTimeoutMSec, headers, []byte("John"))
	if err != nil {
		panic(err)
	}

	fmt.Printf("Resp: %s\n", string(respBytes))
}
