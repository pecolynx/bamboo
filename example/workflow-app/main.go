package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/google/uuid"
	"github.com/pecolynx/bamboo"
	"github.com/pecolynx/bamboo/helper"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
)

var appNameContextKey bamboo.ContextKey

type job struct {
	fn func(ctx context.Context) error
}

func (j *job) Run(ctx context.Context) error {
	return j.fn(ctx)
}

func toParam(v int) []byte {
	data, err := proto.Marshal(&WorkflowAppParameter{Value: int32(v)})
	if err != nil {
		panic(err)
	}
	return data
}

func fromResp(data []byte) int {
	resp := WorkflowAppResponse{}
	if err := proto.Unmarshal(data, &resp); err != nil {
		panic(err)
	}
	return int(resp.Value)
}
func main2() {
	ctx := context.Background()
	dispatcher := NewDispatcher()
	dispatcher.Run(ctx, 20)
	wg := sync.WaitGroup{}

	start := time.Now()
	wg.Add(100)
	for i := 0; i < 100; i++ {
		job := job{
			fn: func(ctx context.Context) error {
				time.Sleep(time.Second)
				wg.Done()
				return nil
			},
		}
		dispatcher.Add(ctx, &job)
	}
	wg.Wait()
	end := time.Now()
	fmt.Printf("done, %f\n", end.Sub(start).Seconds())
}

func main() {
	ctx := context.Background()
	bamboo.InitLogger(ctx)
	if err := helper.InitLog(bamboo.ContextKey("workflow-app"), &helper.LogConfig{
		// Level: "debug",
		Level: "info",
	}); err != nil {
		panic(err)
	}

	factory := helper.NewBambooFactory()
	if os.Getenv("WORKER") == "TRUE" {

		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		eg, ctx := errgroup.WithContext(ctx)

		eg.Go(func() error {
			return baseWorker(ctx, factory, "worker_a", "worker_a", func(ctx context.Context, headers map[string]string, data []byte, aborted <-chan interface{}) ([]byte, error) {
				param := WorkflowAppParameter{}
				if err := proto.Unmarshal(data, &param); err != nil {
					return nil, err
				}
				resp := WorkflowAppResponse{Value: param.Value * 2}
				return proto.Marshal(&resp)
			})
		})
		eg.Go(func() error {
			return baseWorker(ctx, factory, "worker_b", "worker_b", func(ctx context.Context, headers map[string]string, data []byte, aborted <-chan interface{}) ([]byte, error) {
				// time.Sleep(100 * time.Millisecond)

				param := WorkflowAppParameter{}
				if err := proto.Unmarshal(data, &param); err != nil {
					return nil, err
				}
				resp := WorkflowAppResponse{Value: param.Value * 3}
				return proto.Marshal(&resp)
			})
		})
		eg.Go(func() error {
			return baseWorker(ctx, factory, "worker_c", "worker_c", func(ctx context.Context, headers map[string]string, data []byte, aborted <-chan interface{}) ([]byte, error) {
				// time.Sleep(100 * time.Millisecond)

				if 0 == r.Intn(1000) {
					return nil, errors.New("RANDOM ERROR")
				}
				param := WorkflowAppParameter{}
				if err := proto.Unmarshal(data, &param); err != nil {
					return nil, err
				}
				resp := WorkflowAppResponse{Value: param.Value + 10}
				return proto.Marshal(&resp)
			})
		})
		eg.Go(func() error {
			return bamboo.MetricsServerProcess(ctx, 8081, 1)
		})
		if err := eg.Wait(); err != nil {
			panic(err)
		}
	} else {
		workerClient(ctx, factory)
	}
}

func baseWorker(ctx context.Context, factory helper.BambooFactory, workerName string, channel string, fn bamboo.WorkerFunc) error {
	worker, err := factory.CreateBambooWorker(workerName, &helper.WorkerConfig{
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
		NumWorkers: 100,
	}, fn)
	if err != nil {
		return err
	}

	if err := worker.Run(ctx); err != nil {
		return err
	}

	return nil
}

func baseWorkerClient(ctx context.Context, workerName string, channel string, factory helper.BambooFactory) (bamboo.BambooWorkerClient, error) {
	return factory.CreateBambooWorkerClient(ctx, workerName, &helper.WorkerClientConfig{
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
}

// func workerClient0(ctx context.Context, factory helper.BambooFactory) {
// 	workerClientA, err := baseWorkerClient(ctx, "worker_a", "worker_a", factory)
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer workerClientA.Close(ctx)
// 	workerClientB, err := baseWorkerClient(ctx, "worker_b", "worker_b", factory)
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer workerClientB.Close(ctx)
// 	workerClientC, err := baseWorkerClient(ctx, "worker_c", "worker_c", factory)
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer workerClientC.Close(ctx)

// 	fmt.Println("Started")

// 	headers := map[string]string{
// 		bamboo.RequestIDKey: uuid.New().String(),
// 	}
// 	heartbeatIntervalMSec := 0
// 	connectTimeoutMSec := 0
// 	jobTimeoutMSec := 10000

// 	input := 100

// 	wg := sync.WaitGroup{}
// 	for i := 0; i < input; i++ {
// 		fmt.Println("xx")
// 		wg.Add(1)
// 		go func(i int) {
// 			respBytesB, err := workerClientB.Call(ctx, heartbeatIntervalMSec, connectTimeoutMSec, jobTimeoutMSec, headers, toParam(i))
// 			if err != nil {
// 				panic(err)
// 			}
// 			respB := fromResp(respBytesB)
// 			fmt.Printf("Resp: %d\n", respB)
// 			wg.Done()
// 		}(i + 1)
// 	}

// 	fmt.Println("b")

// 	wg.Wait()
// }

// func workerClient3(ctx context.Context, factory helper.BambooFactory) {
// 	workerClientA, err := baseWorkerClient(ctx, "worker_a", "worker_a", factory)
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer workerClientA.Close(ctx)
// 	workerClientB, err := baseWorkerClient(ctx, "worker_b", "worker_b", factory)
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer workerClientB.Close(ctx)
// 	workerClientC, err := baseWorkerClient(ctx, "worker_c", "worker_c", factory)
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer workerClientC.Close(ctx)

// 	dispatcher := NewDispatcher()
// 	dispatcher.Run(ctx, 10)

// 	fmt.Println("Started")

// 	headersA := map[string]string{
// 		bamboo.RequestIDKey: uuid.New().String(),
// 	}
// 	heartbeatIntervalMSec := 0
// 	connectTimeoutMSec := 0
// 	jobTimeoutMSec := 10000

// 	input := 10000

// 	// x * 2
// 	respBytesA, err := workerClientA.Call(ctx, heartbeatIntervalMSec, connectTimeoutMSec, jobTimeoutMSec, headersA, toParam(input))
// 	if err != nil {
// 		panic(err)
// 	}
// 	respA := fromResp(respBytesA)

// 	// x * 3
// 	wg := sync.WaitGroup{}
// 	chC := make(chan int)
// 	quit := make(chan struct{})
// 	go func() {
// 		sum := 0
// 		for i := 0; i < respA; i++ {
// 			c := <-chC
// 			sum += int(c)
// 		}

// 		fmt.Println(sum)

// 		answer := (1+input*2)*(input*2*3)/2 + (input * 20)
// 		fmt.Printf("sum: %d, answer: %d\n", sum, answer)
// 		if sum != answer {
// 			panic(fmt.Errorf("sum: %d, answer: %d", sum, answer))
// 		}
// 		quit <- struct{}{}
// 	}()
// 	for i := 0; i < respA; i++ {
// 		wg.Add(1)
// 		x := i + 1
// 		func(i int) {
// 			headersB := map[string]string{
// 				bamboo.RequestIDKey: "B" + uuid.New().String(),
// 			}
// 			backOff1 := backoff.WithContext(backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 10), ctx)
// 			respBytesB, err := workerClientB.CallWithRetry(ctx, heartbeatIntervalMSec, connectTimeoutMSec, jobTimeoutMSec, headersB, backOff1, toParam(i))
// 			if err != nil {
// 				panic(err)
// 			}
// 			// respB := fromResp(respBytesB)
// 			// fmt.Printf("RespB: %d\n", respB)

// 			headersC := map[string]string{
// 				bamboo.RequestIDKey: "C" + uuid.New().String(),
// 			}
// 			backOff2 := backoff.WithContext(backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 3), ctx)

// 			respBytesC, err := workerClientC.CallWithRetry(ctx, heartbeatIntervalMSec, connectTimeoutMSec, jobTimeoutMSec, headersC, backOff2, respBytesB)
// 			if err != nil {
// 				panic(err)
// 			}
// 			respC := fromResp(respBytesC)
// 			// fmt.Printf("RespC: %d\n", respC)

// 			chC <- respC
// 			wg.Done()
// 		}(x)
// 	}

// 	fmt.Println("a")

// 	fmt.Println("b")

// 	wg.Wait()

// 	<-quit
// }

func workerClient(ctx context.Context, factory helper.BambooFactory) {
	workerClientA, err := baseWorkerClient(ctx, "worker_a", "worker_a", factory)
	if err != nil {
		panic(err)
	}
	defer workerClientA.Close(ctx)
	workerClientB, err := baseWorkerClient(ctx, "worker_b", "worker_b", factory)
	if err != nil {
		panic(err)
	}
	defer workerClientB.Close(ctx)
	workerClientC, err := baseWorkerClient(ctx, "worker_c", "worker_c", factory)
	if err != nil {
		panic(err)
	}
	defer workerClientC.Close(ctx)

	dispatcher := NewDispatcher()
	dispatcher.Run(ctx, 1)
	// dispatcher.Run(ctx, 20)

	fmt.Println("Started")

	headersA := map[string]string{
		bamboo.RequestIDKey: uuid.New().String(),
	}
	heartbeatIntervalMSec := 0
	connectTimeoutMSec := 0
	jobTimeoutMSec := 10000

	// input := 1000
	input := 5000

	// x * 2
	respBytesA, err := workerClientA.Call(ctx, heartbeatIntervalMSec, connectTimeoutMSec, jobTimeoutMSec, headersA, toParam(input))
	if err != nil {
		panic(err)
	}
	respA := fromResp(respBytesA)

	// x * 3
	wg := sync.WaitGroup{}
	chC := make(chan int)
	quit := make(chan struct{})
	go func() {
		sum := 0
		for i := 0; i < respA; i++ {
			c := <-chC
			sum += int(c)
		}

		answer := (1+input*2)*(input*2*3)/2 + (input * 20)
		fmt.Printf("sum: %d, answer: %d\n", sum, answer)
		if sum != answer {
			panic(fmt.Errorf("sum: %d, answer: %d", sum, answer))
		}
		quit <- struct{}{}
	}()

	start := time.Now()

	fmt.Printf("respA: %d\n", respA)

	for i := 0; i < respA; i++ {
		wg.Add(1)
		x := i + 1
		go func(x int) {
			fn := func(ctx context.Context) error {
				func(i int) {
					headersB := map[string]string{
						bamboo.RequestIDKey: "B" + uuid.New().String(),
					}
					backOff1 := backoff.WithContext(backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 20), ctx)
					respBytesB, err := workerClientB.CallWithRetry(ctx, heartbeatIntervalMSec, connectTimeoutMSec, jobTimeoutMSec, headersB, backOff1, toParam(i))
					if err != nil {
						panic(err)
					}
					// respB := fromResp(respBytesB)
					// fmt.Printf("RespB: %d\n", respB)

					headersC := map[string]string{
						bamboo.RequestIDKey: "C" + uuid.New().String(),
					}
					backOff2 := backoff.WithContext(backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 20), ctx)

					respBytesC, err := workerClientC.CallWithRetry(ctx, heartbeatIntervalMSec, connectTimeoutMSec, jobTimeoutMSec, headersC, backOff2, respBytesB)
					if err != nil {
						panic(err)
					}
					respC := fromResp(respBytesC)
					// fmt.Printf("RespC: %d\n", respC)

					chC <- respC
					wg.Done()
				}(x)
				return nil
			}
			job := job{fn: fn}
			dispatcher.Add(ctx, &job)
		}(x)
	}

	wg.Wait()

	end := time.Now()
	fmt.Println(end.Sub(start).Seconds())
	<-quit
}

// func workerClient2(ctx context.Context, factory helper.BambooFactory) {
// 	workerClientA, err := baseWorkerClient(ctx, "worker_a", "worker_a", factory)
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer workerClientA.Close(ctx)
// 	workerClientB, err := baseWorkerClient(ctx, "worker_b", "worker_b", factory)
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer workerClientB.Close(ctx)
// 	workerClientC, err := baseWorkerClient(ctx, "worker_c", "worker_c", factory)
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer workerClientC.Close(ctx)

// 	dispatcher := NewDispatcher()
// 	dispatcher.Run(ctx, 10)

// 	fmt.Println("Started")

// 	headers := map[string]string{
// 		bamboo.RequestIDKey: uuid.New().String(),
// 	}
// 	heartbeatIntervalMSec := 0
// 	connectTimeoutMSec := 0
// 	jobTimeoutMSec := 10000

// 	input := 10000

// 	// x * 2
// 	respBytesA, err := workerClientA.Call(ctx, heartbeatIntervalMSec, connectTimeoutMSec, jobTimeoutMSec, headers, toParam(input))
// 	if err != nil {
// 		panic(err)
// 	}
// 	respA := fromResp(respBytesA)

// 	// x * 3
// 	wg := sync.WaitGroup{}
// 	chC := make(chan int)
// 	quit := make(chan struct{})
// 	go func() {
// 		sum := 0
// 		for i := 0; i < respA; i++ {
// 			c := <-chC
// 			sum += int(c)
// 		}

// 		fmt.Println(sum)

// 		answer := (1+input*2)*(input*2*3)/2 + (input * 20)
// 		fmt.Printf("sum: %d, answer: %d\n", sum, answer)
// 		if sum != answer {
// 			panic(fmt.Errorf("sum: %d, answer: %d", sum, answer))
// 		}
// 		quit <- struct{}{}
// 	}()
// 	for i := 0; i < respA; i++ {
// 		wg.Add(1)
// 		x := i + 1
// 		fn := func(ctx context.Context) error {
// 			// fmt.Println("aaaaaaaa")
// 			func(i int) {
// 				// fmt.Println("bbbbb")
// 				respBytesB, err := workerClientB.Call(ctx, heartbeatIntervalMSec, connectTimeoutMSec, jobTimeoutMSec, headers, toParam(i))
// 				if err != nil {
// 					panic(err)
// 				}
// 				// respB := fromResp(respBytesB)
// 				// fmt.Printf("RespB: %d\n", respB)

// 				backOff := backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 5)

// 				respBytesC, err := workerClientC.CallWithRetry(ctx, heartbeatIntervalMSec, connectTimeoutMSec, jobTimeoutMSec, headers, backOff, respBytesB)
// 				if err != nil {
// 					panic(err)
// 				}
// 				respC := fromResp(respBytesC)
// 				// fmt.Printf("RespC: %d\n", respC)

// 				chC <- respC
// 				wg.Done()
// 			}(x)
// 			// fmt.Println("cccc")
// 			return nil
// 		}
// 		job := job{fn: fn}
// 		dispatcher.Add(ctx, &job)
// 	}

// 	fmt.Println("a")

// 	fmt.Println("b")

// 	wg.Wait()

// 	<-quit
// }
