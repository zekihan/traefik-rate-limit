package main

import (
	"context"
	"fmt"
	"github.com/zekihan/traefik-rate-limit/internal/client"
	"github.com/zekihan/traefik-rate-limit/internal/server"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"
)

func Test(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource:   false,
		Level:       slog.LevelDebug,
		ReplaceAttr: nil,
	}))
	logger = logger.With(slog.String("app", "server"))
	slog.SetDefault(logger)

	// Use a unique socket path for each benchmark run
	socketPath := fmt.Sprintf("./tmp/traefik-rate-limit-%d.sock", time.Now().UnixNano())
	defer os.Remove(socketPath) // Clean up socket file afterwards

	// Context to control the server lifetime
	serverCtx, serverCancel := context.WithCancel(context.Background())
	defer serverCancel() // Ensure server is signalled to stop

	// Start the server in a separate goroutine
	go func() {
		server.RunServer(serverCtx, socketPath)
	}()

	// Wait a short moment for the server to start and create the socket
	// A more robust check would try to connect in a loop
	time.Sleep(100 * time.Millisecond)

	// Check if server started correctly (socket exists)
	if _, err := os.Stat(socketPath); err != nil {
		t.Fatalf("Server socket not found: %v", err)
	}

	wg := &sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			newClient, err := client.NewClient(socketPath)
			if err != nil {
				t.Fatalf("Failed to connect to socket: %v", err)
			}
			defer newClient.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			res, err := newClient.Ping(ctx)
			if err != nil {
				cancel()
				t.Fatalf("Ping failed: %v", err)
			}
			slog.Info("Ping succeeded", slog.Any("response", res))
			cancel()
		}()
	}
	wg.Wait()
}
