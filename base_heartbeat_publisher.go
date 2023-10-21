package bamboo

import (
	"context"
	"time"
)

type baseHeartbeatPublisher struct {
}

func (h *baseHeartbeatPublisher) Run(ctx context.Context, heartbeatIntervalMSec int, done <-chan interface{}, aborted <-chan interface{}, fn func()) {
	logger := GetLoggerFromContext(ctx, BambooHeartbeatPublisherLoggerContextKey)
	ctx = WithLoggerName(ctx, BambooHeartbeatPublisherLoggerContextKey)

	if heartbeatIntervalMSec == 0 {
		logger.DebugContext(ctx, "heartbeat is disabled because heartbeatIntervalMSec is zero.")
		return
	}

	heartbeatInterval := time.Duration(heartbeatIntervalMSec) * time.Millisecond
	pulse := time.NewTicker(heartbeatInterval)

	go func() {
		defer func() {
			logger.DebugContext(ctx, "stop heartbeat loop")
			pulse.Stop()
		}()

		for {
			select {
			case <-ctx.Done():
				logger.DebugContext(ctx, "ctx.Done(). stop heartbeat loop")
				return
			case <-done:
				logger.DebugContext(ctx, "done. stop heartbeat loop")
				return
			case <-aborted:
				logger.DebugContext(ctx, "aborted. stop heartbeat loop")
				return
			case <-pulse.C:
				logger.DebugContext(ctx, "pulse")
				fn()
			}
		}
	}()

}
