package server

import (
	"context"
	"log/slog"
	"time"
)

func RunServer(ctx context.Context, socketPath string) {
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- runServer(ctx, socketPath)
	}()

	// Listen for the interrupt signal or server error.
	select {
	case err := <-serverErr:
		if err != nil {
			slog.Error("server error", slog.Any("error", err))
		}
	case <-ctx.Done():
		slog.Info("shutting down gracefully, press Ctrl+C again to force")
		// Optionally add a timeout for graceful shutdown
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Wait for server goroutine to finish (it should return when context is cancelled)
		select {
		case err := <-serverErr:
			if err != nil {
				slog.Error("server shutdown error", slog.Any("error", err))
			}
		case <-shutdownCtx.Done():
			slog.Error("server shutdown timed out")
		}
	}
}
