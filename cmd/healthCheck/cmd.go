package healthCheck

import (
	"context"
	"github.com/zekihan/traefik-rate-limit/internal/client"
	"log/slog"
	"os"
	"time"
)

func Run(socketPath string) {
	startTime := time.Now()
	maxWait := 1 * time.Second
	for {
		if _, err := os.Stat(socketPath); err == nil {
			break
		}
		if time.Since(startTime) > maxWait {
			slog.Error("timed out waiting for socket file", slog.String("socket", socketPath))
			os.Exit(1)
		}
		time.Sleep(100 * time.Millisecond)
	}
	slog.Debug("found socket file", slog.String("socket", socketPath))

	newClient, err := client.NewClient(socketPath)
	if err != nil {
		slog.Error("failed to dial server", slog.Any("error", err), slog.String("socket", newClient.SocketPath))
		os.Exit(1)
	}
	defer newClient.Close()
	slog.Debug("connected to server", slog.String("socket", newClient.SocketPath))

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, err = newClient.Ping(ctx)
	if err != nil {
		slog.Error("failed to send request", slog.Any("error", err))
		os.Exit(1)
	}
	slog.Info("healthcheck OK", slog.String("socket", socketPath))
}
