package bamboo

import "github.com/prometheus/client_golang/prometheus"

type MetricsEventHandler interface {
	OnReceiveRequest()
	OnSuccessJob()
	OnInternalErrorJob()
	OnInvalidArgumentJob()
	OnIncrNumRunningWorkers()
	OnDecrNumRunningWorkers()
}

type emptyEventHandler struct {
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

func (e *emptyEventHandler) OnIncrNumRunningWorkers() {
}

func (e *emptyEventHandler) OnDecrNumRunningWorkers() {
}

type prometheusEventHandler struct {
	receiveRequestCounter     prometheus.Counter
	successJobCounter         prometheus.Counter
	internalErrorJobCounter   prometheus.Counter
	invalidArgumentJobCounter prometheus.Counter
	numRunningWorkersGauge    prometheus.Gauge
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
	numRunningWorkersGauge := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "bamboo_num_running_workers",
		})

	if err := prometheus.Register(receiveRequestCounter); err != nil {
		panic(err)
	}
	if err := prometheus.Register(successJobCounter); err != nil {
		panic(err)
	}
	if err := prometheus.Register(internalErrorJobCounter); err != nil {
		panic(err)
	}
	if err := prometheus.Register(invalidArgumentJobCounter); err != nil {
		panic(err)
	}
	if err := prometheus.Register(numRunningWorkersGauge); err != nil {
		panic(err)
	}

	return &prometheusEventHandler{
		receiveRequestCounter:     receiveRequestCounter,
		successJobCounter:         successJobCounter,
		internalErrorJobCounter:   internalErrorJobCounter,
		invalidArgumentJobCounter: invalidArgumentJobCounter,
		numRunningWorkersGauge:    numRunningWorkersGauge,
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

func (e *prometheusEventHandler) OnIncrNumRunningWorkers() {
	go func() { e.numRunningWorkersGauge.Inc() }()
}

func (e *prometheusEventHandler) OnDecrNumRunningWorkers() {
	go func() { e.numRunningWorkersGauge.Dec() }()
}
