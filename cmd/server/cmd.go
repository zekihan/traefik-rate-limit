package server

import (
	"context"
	"github.com/zekihan/traefik-rate-limit/internal/server"
)

func Run(socketPath string) {
	server.RunServer(context.Background(), socketPath)
}
