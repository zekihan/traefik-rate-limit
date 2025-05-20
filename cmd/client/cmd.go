package client

import "github.com/zekihan/traefik-rate-limit/internal/client"

func Run(socketPath string) {
	client.RunClient(socketPath)
}
