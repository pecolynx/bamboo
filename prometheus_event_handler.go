package bamboo

import "github.com/prometheus/client_golang/prometheus"

type MetricsEventHandler interface {
	OnReceiveRequest()
	OnSuccessJob()
	OnInternalErrorJob()
	OnInvalidArgumentJob()
}

type emptyEventHandler struct {
	receiveRequestCounter  prometheus.Counter
	successCounter         prometheus.Counter
	internalErrorCounter   prometheus.Counter
	InvalidArgumentCounter prometheus.Counter
}

func NewEmptyEventHandler() MetricsEventHandler {
	return &emptyEventHandler{}
}

func (e *emptyEventHandler) OnReceiveRequest() {
}

func (e *emptyEventHandler) OnSuccessJob() {
}

func (e *emptyEventHandler) OnInternalErrorJob() {
}

func (e *emptyEventHandler) OnInvalidArgumentJob() {
}

type prometheusEventHandler struct {
	receiveRequestCounter     prometheus.Counter
	successJobCounter         prometheus.Counter
	internalErrorJobCounter   prometheus.Counter
	invalidArgumentJobCounter prometheus.Counter
}

func NewPrometheusEventHandler() MetricsEventHandler {
	receiveRequestCounter := prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "bamboo_requests_received",
		})
	successJobCounter := prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "bamboo_jobs_success",
		})
	internalErrorJobCounter := prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "bamboo_jobs_internal_error",
		})
	invalidArgumentJobCounter := prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "bamboo_jobs_invalid_argument",
		})

	prometheus.Register(receiveRequestCounter)
	prometheus.Register(successJobCounter)
	prometheus.Register(internalErrorJobCounter)
	prometheus.Register(invalidArgumentJobCounter)

	return &prometheusEventHandler{
		receiveRequestCounter:     receiveRequestCounter,
		successJobCounter:         successJobCounter,
		internalErrorJobCounter:   internalErrorJobCounter,
		invalidArgumentJobCounter: invalidArgumentJobCounter,
	}
}

func (e *prometheusEventHandler) OnReceiveRequest() {
	go func() { e.receiveRequestCounter.Inc() }()
}

func (e *prometheusEventHandler) OnSuccessJob() {
	go func() { e.successJobCounter.Inc() }()
}

func (e *prometheusEventHandler) OnInternalErrorJob() {
	go func() { e.internalErrorJobCounter.Inc() }()
}

func (e *prometheusEventHandler) OnInvalidArgumentJob() {
	go func() { e.invalidArgumentJobCounter.Inc() }()
}
