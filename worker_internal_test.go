package bamboo

import (
	"context"
	"testing"

	"github.com/pecolynx/bamboo/internal"
	mocks "github.com/pecolynx/bamboo/mocks/github.com/pecolynx/bamboo"
)

func TestBC(t *testing.T) {
	ctx := context.Background()
	consumer := mocks.BambooRequestConsumer{}
	worker := bambooWorker{}

	jobQueue := make(chan internal.Job)
	worker.consumeRequestAndDispatchJob(ctx, &consumer, jobQueue)
}
