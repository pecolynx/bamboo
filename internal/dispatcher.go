package internal

import (
	"context"
)

type Dispatcher struct {
	JobQueue   chan Job
	Quit       chan bool
	workerPool chan chan Job
}

func NewDispatcher() Dispatcher {
	return Dispatcher{
		JobQueue:   make(chan Job),
		Quit:       make(chan bool),
		workerPool: make(chan chan Job),
	}
}

func (d *Dispatcher) Start(ctx context.Context, numWorkers int) {
	workers := make([]Worker, numWorkers)
	for i := 0; i < numWorkers; i++ {
		workers[i] = NewWorker(i, d.workerPool)
		workers[i].Start(ctx)
	}

	go func() {
		for {
			select {
			case <-d.Quit:
				for i := range workers {
					workers[i].Stop(ctx)
				}
				return
			case job := <-d.JobQueue:
				worker := <-d.workerPool // wait for available worker
				worker <- job            // dispatch job to worker
			}
		}
	}()
}

func (d *Dispatcher) Stop(ctx context.Context) {
	d.Quit <- true
}

func (d *Dispatcher) AddJob(job Job) {
	d.JobQueue <- job
}
