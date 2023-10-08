package bamboo

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"

	"github.com/pecolynx/bamboo/internal"
	pb "github.com/pecolynx/bamboo/proto"
	"github.com/pecolynx/bamboo/sloghelper"
)

type kafkaBambooRequestConsumer struct {
	consumerOptions    kafka.ReaderConfig
	consumer           *kafka.Reader
	requestWaitTimeout time.Duration
}

func NewKafkaBambooRequestConsumer(consumerOptions kafka.ReaderConfig, requestWaitTimeout time.Duration) BambooRequestConsumer {
	return &kafkaBambooRequestConsumer{
		consumerOptions:    consumerOptions,
		consumer:           kafka.NewReader(consumerOptions),
		requestWaitTimeout: requestWaitTimeout,
	}
}

func (c *kafkaBambooRequestConsumer) Consume(ctx context.Context) (*pb.WorkerParameter, error) {
	logger := sloghelper.FromContext(ctx, sloghelper.BambooRequestConsumerLoggerContextKey)

	for {
		select {
		case <-ctx.Done():
			return nil, ErrContextCanceled
		default:
			m, err := c.consumer.ReadMessage(ctx)
			if err != nil {
				return nil, internal.Errorf("kafka.ReadMessage. err: %w", err)
			}

			if len(m.Key) == 0 && len(m.Value) == 0 {
				return nil, errors.New("size of kafkaMessage received is invalid")
			}

			// fmt.Printf("message at offset %d: %s = %s\n", m.Offset, string(m.Key), string(m.Value))

			req := pb.WorkerParameter{}
			if err := proto.Unmarshal(m.Value, &req); err != nil {
				logger.WarnContext(ctx, "invalid parameter. failed to proto.Unmarshal.", slog.Any("err", err))
				continue
			}
			return &req, nil
		}
	}
}

func (c *kafkaBambooRequestConsumer) Ping(ctx context.Context) error {
	if len(c.consumerOptions.Brokers) == 0 {
		return errors.New("broker size is 0")
	}

	conn, err := kafka.Dial("tcp", c.consumerOptions.Brokers[0])
	if err != nil {
		return internal.Errorf("kafka.Dial. err: %w", err)
	}
	defer conn.Close()

	if _, err := conn.ReadPartitions(); err != nil {
		return internal.Errorf("conn.ReadPartitions. err: %w", err)
	}

	return nil
}

func (c *kafkaBambooRequestConsumer) Close(ctx context.Context) error {
	return c.consumer.Close()
}
