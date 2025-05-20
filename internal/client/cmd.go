package client

import (
	"context"
	"github.com/zekihan/traefik-rate-limit/internal/comm"
	"log/slog"
	"sync"
	"time"
)

func RunClient(socketPath string) {
	client, err := NewClient(socketPath)
	if err != nil {
		panic(err)
	}
	defer client.Close()
	slog.Info("connected to server", slog.String("socket", client.SocketPath))

	// Make 5 concurrent requests
	var wg sync.WaitGroup

	//wg.Add(1)
	//go func() {
	//	defer wg.Done()
	//	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	//	defer cancel()
	//
	//	_, err := client.Ping(ctx)
	//	if err != nil {
	//		slog.Info("failed to send request", slog.Any("error", err))
	//		return
	//	}
	//}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		res, err := client.RateLimit(ctx, &comm.RateLimitRequestData{
			Rate:   100,
			Burst:  100,
			Period: time.Hour,
			Key:    "testing",
		})
		if err != nil {
			slog.Info("failed to send request", slog.Any("error", err))
			return
		}
		slog.Debug("received response", slog.Any("response", res))
	}()

	wg.Wait()
}
