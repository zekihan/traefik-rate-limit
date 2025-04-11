package main

import (
	"context"
	"fmt"
	"github.com/zekihan/traefik-rate-limit/internal/client"
	"github.com/zekihan/traefik-rate-limit/internal/server"
	"log/slog"
	"os"
	"testing"
	"time"
)

func init() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource:   false,
		Level:       slog.LevelError,
		ReplaceAttr: nil,
	}))
	logger = logger.With(slog.String("app", "server"))
	slog.SetDefault(logger)
}

func BenchmarkSendRequest(b *testing.B) {
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
		b.Fatalf("Server socket not found: %v", err)
	}

	newClient, err := client.NewClient(socketPath)
	if err != nil {
		serverCancel() // Stop server if client can't connect
		b.Fatalf("Failed to connect to socket: %v", err)
	}
	defer newClient.Close()

	// Warm-up (optional but good practice)
	for i := 0; i < 10; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		_, err := newClient.Ping(ctx)
		cancel()
		if err != nil {
			serverCancel()
			b.Fatalf("Warmup failed: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Use a shorter timeout for benchmark requests
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		_, err := newClient.Ping(ctx)
		cancel()
		if err != nil {
			// Don't necessarily stop the whole benchmark on single request failure
			b.Logf("Request %d failed: %v", i, err)
			// Consider b.Fail() or b.Error() if failures are critical
		}
	}
	b.StopTimer()

	// Cleanly shutdown server
	serverCancel()
	newClient.Close() // Close client connection
}
