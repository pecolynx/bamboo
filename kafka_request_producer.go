package bamboo

import (
	"context"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"github.com/pecolynx/bamboo/internal"
)

type kafkaBambooRequestProducer struct {
	workerName  string
	kafkaWriter *kafka.Writer
	propagator  propagation.TextMapPropagator
}

func NewKafkaBambooRequestProducer(ctx context.Context, workerName string, kafkaWriter *kafka.Writer) BambooRequestProducer {
	return &kafkaBambooRequestProducer{
		workerName:  workerName,
		kafkaWriter: kafkaWriter,
		propagator:  otel.GetTextMapPropagator(),
	}
}

func (p *kafkaBambooRequestProducer) Produce(ctx context.Context, resultChannel string, heartbeatIntervalMSec int, jobTimeoutMSec int, headers map[string]string, data []byte) error {
	ctx = WithLoggerName(ctx, BambooWorkerClientLoggerContextKey)

	baseBambooRequestProducer := baseBambooRequestProducer{}
	return baseBambooRequestProducer.Produce(ctx, resultChannel, heartbeatIntervalMSec, jobTimeoutMSec, headers, data, p.workerName, p.propagator, func(ctx context.Context, reqBytes []byte) error {

		messageID, err := uuid.NewRandom()
		if err != nil {
			return internal.Errorf("uuid.NewRandom. err: %w", err)
		}

		msg := kafka.Message{
			Key:   []byte(messageID.String()),
			Value: reqBytes,
		}

		if err := p.kafkaWriter.WriteMessages(ctx, msg); err != nil {
			return internal.Errorf("kafkaWriter.WriteMessages. err: %w", err)
		}

		return nil
	})
}

func (p *kafkaBambooRequestProducer) Ping(ctx context.Context) error {
	return nil
}

func (p *kafkaBambooRequestProducer) Close(ctx context.Context) error {
	return p.kafkaWriter.Close()
}
