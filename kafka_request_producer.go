package bamboo

import (
	"context"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/protobuf/proto"

	"github.com/pecolynx/bamboo/internal"
	pb "github.com/pecolynx/bamboo/proto"
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

func (p *kafkaBambooRequestProducer) Produce(ctx context.Context, resultChannel string, heartbeatIntervalSec int, jobTimeoutSec int, headers map[string]string, data []byte) error {
	carrier := propagation.MapCarrier{}

	spanCtx, span := tracer.Start(ctx, p.workerName)
	defer span.End()

	p.propagator.Inject(spanCtx, carrier)

	messageID, err := uuid.NewRandom()
	if err != nil {
		return internal.Errorf("uuid.NewRandom. err: %w", err)
	}

	req := pb.WorkerParameter{
		Carrier:       carrier,
		Headers:       headers,
		ResultChannel: resultChannel,
		Data:          data,
	}
	reqBytes, err := proto.Marshal(&req)
	if err != nil {
		return internal.Errorf("proto.Marshal. err: %w", err)
	}

	msg := kafka.Message{
		Key:   []byte(messageID.String()),
		Value: reqBytes,
	}

	if err := p.kafkaWriter.WriteMessages(spanCtx, msg); err != nil {
		return internal.Errorf("kafkaWriter.WriteMessages. err: %w", err)
	}

	return nil
}

func (p *kafkaBambooRequestProducer) Ping(ctx context.Context) error {
	return nil
}

func (p *kafkaBambooRequestProducer) Close(ctx context.Context) error {
	return p.kafkaWriter.Close()
}
