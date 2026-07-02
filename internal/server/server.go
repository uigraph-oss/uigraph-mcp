package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"
)

func Run(port string, handler http.Handler) error {
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 0, // SSE streams stay open
		IdleTimeout:  120 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("uigraph-mcp listening", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}
	return nil
}
