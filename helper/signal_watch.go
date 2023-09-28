package helper

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func SignalWatchProcess(ctx context.Context) error {
	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-ctx.Done():
		signal.Reset()
		return nil
	case sig := <-sigs:
		return fmt.Errorf("signal received: %v", sig.String())
	}
}
