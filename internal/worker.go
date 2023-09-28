package internal

import (
	"context"
	"fmt"
)

type Job interface {
	Run(ctx context.Context) error
}

type emptyJob struct {
}

func NewEmptyJob() Job {
	return &emptyJob{}
}

func (j *emptyJob) Run(ctx context.Context) error { return nil }

type Worker struct {
	id         int
	workerPool chan chan Job // used to communicate between dispatcher and workers
	jobQueue   chan Job
	quit       chan bool
}

func NewWorker(id int, workerPool chan chan Job) Worker {
	return Worker{
		id:         id,
		workerPool: workerPool,
		jobQueue:   make(chan Job),
		quit:       make(chan bool),
	}
}

func (w *Worker) Start(ctx context.Context) {
	logger := FromContext(ctx)

	go func() {
		for {
			w.workerPool <- w.jobQueue

			select {
			case job := <-w.jobQueue:
				logger.Debugf("worker[%d].Start", w.id)
				if err := job.Run(context.Background()); err != nil {
					fmt.Println(err)
				}

			case <-w.quit:
				return
			}
		}
	}()
}

func (w *Worker) Stop(ctx context.Context) {
	w.quit <- true
}
