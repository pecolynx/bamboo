package main

import (
	"context"
)

type Dispatcher interface {
	Run(ctx context.Context, numWorkers int) error
	Add(ctx context.Context, job Job)
}

type dispatcher struct {
	workerPool chan chan Job
	jobQueue   chan Job
}

func NewDispatcher() Dispatcher {
	return &dispatcher{
		workerPool: make(chan chan Job),
		jobQueue:   make(chan Job),
	}
}

func (d *dispatcher) Run(ctx context.Context, numWorkers int) error {
	workers := make([]Worker, numWorkers)
	for i := 0; i < numWorkers; i++ {
		workers[i] = NewWorker(i, d.workerPool)
		workers[i].Start(ctx)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				for _, w := range workers {
					w.Stop(ctx)
				}
				return
			case job := <-d.jobQueue:
				worker := <-d.workerPool // wait for available channel
				worker <- job            // dispatch work to worker
			}
		}
	}()

	return nil
}

func (d *dispatcher) Add(ctx context.Context, job Job) {
	d.jobQueue <- job
}
