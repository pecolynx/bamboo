package bamboo

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const readHeaderTimeout = time.Duration(30) * time.Second

func MetricsServerProcess(ctx context.Context, port int, gracefulShutdownTimeSec int) error {
	logger := GetLoggerFromContext(ctx, "")
	router := gin.New()
	router.Use(gin.Recovery())

	httpServer := http.Server{
		Addr:              ":" + strconv.Itoa(port),
		Handler:           router,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	router.GET("/healthcheck", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	logger.InfoContext(ctx, fmt.Sprintf("metrics server listening at %v", httpServer.Addr))

	errCh := make(chan error)
	go func() {
		defer close(errCh)
		if err := httpServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			logger.InfoContext(ctx, "httpServer.ListenAndServe", slog.Any("err", err))
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		gracefulShutdownTime1 := time.Duration(gracefulShutdownTimeSec) * time.Second
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), gracefulShutdownTime1)
		defer shutdownCancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			logger.InfoContext(ctx, "httpServer.Shutdown", slog.Any("err", err))
			return err
		}
		return nil
	case err := <-errCh:
		return err
	}
}
