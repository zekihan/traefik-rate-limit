package client

import (
	"context"
	"fmt"
	"github.com/zekihan/traefik-rate-limit/internal/comm"
	"log/slog"
	"math/rand/v2"
	"net"
	"os"
	"sync"
	"time"
)

type Client struct {
	SocketPath string
	conn       net.Conn
	pending    sync.Map
}

func NewClient(socketPath string) (*Client, error) {
	startTime := time.Now()
	maxWait := 5 * time.Second
	for {
		if _, err := os.Stat(socketPath); err == nil {
			break
		}
		if time.Since(startTime) > maxWait {
			slog.Error("timed out waiting for socket file", slog.String("socket", socketPath))
			return nil, fmt.Errorf("timed out waiting for socket file: %s", socketPath)
		}
		time.Sleep(100 * time.Millisecond)
	}
	slog.Info("found socket file", slog.String("socket", socketPath))

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		slog.Error("failed to dial server", slog.Any("error", err), slog.String("socket", socketPath))
		return nil, err
	}

	newClient := &Client{
		SocketPath: socketPath,
		conn:       conn,
	}

	go newClient.ReadResponses(conn)

	return newClient, nil
}

func (c *Client) Close() {
	if c.conn != nil {
		err := c.conn.Close()
		if err != nil {
			slog.Error("failed to close connection", slog.Any("error", err))
		}
		c.conn = nil
	}
	slog.Info("client closed")
}

func (c *Client) Ping(ctx context.Context) (string, error) {
	req := &comm.Request{}
	req.Header = &comm.Header{}
	req.RequestID = rand.Uint32()
	req.Version = comm.VERSION

	req.Type = comm.RequestTypePing
	req.Data = "ping"
	res, err := c.SendRequest(ctx, c.conn, req)
	if err != nil {
		slog.Error("failed to send request", slog.Any("error", err))
		return "", err
	}
	return res.(string), nil
}

func (c *Client) RateLimit(ctx context.Context, payload *comm.RateLimitRequestData) (*comm.RateLimitResponseData, error) {
	req := &comm.Request{}
	req.Header = &comm.Header{}
	req.RequestID = rand.Uint32()
	req.Version = comm.VERSION

	req.Type = comm.RequestTypeRateLimit
	req.Data = payload
	res, err := c.SendRequest(ctx, c.conn, req)
	if err != nil {
		slog.Error("failed to send request", slog.Any("error", err))
		return nil, err
	}
	data, ok := res.(*comm.RateLimitResponseData)
	if !ok {
		slog.Error("unexpected response data type", slog.Any("data", res))
		return nil, nil
	}
	if data == nil {
		slog.Error("unexpected nil response data")
		return nil, nil
	}
	slog.Debug("received response", slog.Any("data", data))
	return data, nil
}
