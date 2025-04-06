package redis

import "sync"

var (
	clientInstance *Client
	clientOnce     sync.Once
)

func GetClient(options *Options) *Client {
	clientOnce.Do(func() {
		clientInstance = NewClient(options)
	})
	return clientInstance
}
