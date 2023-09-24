package helper

// client
type WorkerClientConfig struct {
	RequestProducer  *RequestProducerConfig  `yaml:"requestProducer" validate:"required"`
	ResultSubscriber *ResultSubscriberConfig `yaml:"resultSubscriber" validate:"required"`
}

// request
type RequestProducerConfig struct {
	Type  string                      `yaml:"type" validate:"required"`
	Kafka *KafkaRequestProducerConfig `yaml:"kafka"`
	Redis *RedisRequestProducerConfig `yaml:"redis"`
}

type KafkaRequestProducerConfig struct {
	Addr  string `yaml:"addr" validate:"required"`
	Topic string `yaml:"topic" validate:"required"`
}

type RedisRequestProducerConfig struct {
	Addrs    []string `yaml:"addrs" validate:"required"`
	Password string   `yaml:"password"`
	Channel  string   `yaml:"channel" validate:"required"`
}

// result
type RedisResultSubscriberConfig struct {
	Addrs    []string `yaml:"addrs" validate:"required"`
	Password string   `yaml:"password"`
}

type ResultSubscriberConfig struct {
	Type  string                       `yaml:"type" validate:"required"`
	Redis *RedisResultSubscriberConfig `yaml:"redis"`
}

// worker
type WorkerConfig struct {
	Consumer   *ConsumerConfig  `yaml:"consumer"`
	Publisher  *PublisherConfig `yaml:"publisher"`
	NumWorkers int              `yaml:"numWorkers"`
}

type ConsumerConfig struct {
	Type  string               `yaml:"type" validate:"required"`
	Kafka *KafkaConsumerConfig `yaml:"kafka"`
	Redis *RedisConsumerConfig `yaml:"redis"`
}

type KafkaConsumerConfig struct {
	Brokers []string `yaml:"brokers" validate:"required"`
	GroupID string   `yaml:"groupId" validate:"required"`
	Topic   string   `yaml:"topic" validate:"required"`
}

type RedisConsumerConfig struct {
	Addrs    []string `yaml:"addrs" validate:"required"`
	Password string   `yaml:"password"`
	Channel  string   `yaml:"channel" validate:"required"`
}

type PublisherConfig struct {
	Type  string                `yaml:"type" validate:"required"`
	Redis *RedisPublisherConfig `yaml:"redis"`
}

type RedisPublisherConfig struct {
	Addrs    []string `yaml:"addrs" validate:"required"`
	Password string   `yaml:"password"`
}
