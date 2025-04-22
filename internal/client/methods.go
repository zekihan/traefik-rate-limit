package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"

	"github.com/zekihan/traefik-rate-limit/internal/comm"
)

var bufPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

func (c *Client) SendRequest(ctx context.Context, conn net.Conn, req *comm.Request) (any, error) {
	if conn == nil {
		return "", fmt.Errorf("connection is nil")
	}

	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)

	if err := req.Marshal(buf); err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	ch := make(chan *comm.Response, 1)
	c.pending.Store(req.RequestID, ch)
	defer c.pending.Delete(req.RequestID)

	_, err := conn.Write(buf.Bytes())
	if err != nil {
		return "", fmt.Errorf("write request: %w", err)
	}

	select {
	case resp := <-ch:
		return c.handleResponse(req.Type, resp)
	case <-ctx.Done():
		slog.Info("context done", slog.Any("error", ctx.Err()))
		return "", ctx.Err()
	}
}

func (c *Client) handleResponse(reqType comm.RequestType, resp *comm.Response) (any, error) {
	if resp.Status == comm.ResponseStatusError {
		return "", fmt.Errorf("server error: %s", resp.Error)
	}

	switch reqType {
	case comm.RequestTypePing:
		data, ok := resp.Data.(string)
		if !ok {
			return "", fmt.Errorf("unexpected response type: %T", resp.Data)
		}
		return data, nil
	case comm.RequestTypeRateLimit:
		data, ok := resp.Data.(*comm.RateLimitResponseData)
		if !ok {
			return "", fmt.Errorf("unexpected response type: %T", resp.Data)
		}
		return data, nil
	default:
		return resp.Data, nil
	}
}

func (c *Client) ReadResponses(conn net.Conn) {
	defer func() { slog.Debug("response reader stopped") }()
	for {
		header, err := readHeader(conn)
		if err != nil {
			if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) {
				slog.Debug("connection closed while reading header")
			} else {
				slog.Error("read header error", slog.Any("error", err))
			}
			return
		}
		if header.ContentLength > 10*1024*1024 {
			slog.Error("response payload too large", slog.Uint64("length", uint64(header.ContentLength)))
			conn.Close()
			return
		}
		payload := make([]byte, header.ContentLength)
		if err := readFull(conn, payload); err != nil {
			slog.Debug("read payload error", slog.Any("error", err))
			return
		}
		slog.Debug("read payload", slog.Int("bytes", int(header.ContentLength)))

		resp := &comm.Response{}
		if err := resp.Unmarshal(header, payload); err != nil {
			slog.Info("unmarshal response error", slog.Any("error", err))
			continue
		}
		if chVal, ok := c.pending.Load(header.RequestID); ok {
			if ch, castOK := chVal.(chan *comm.Response); castOK {
				select {
				case ch <- resp:
				default:
					slog.Warn("could not send response to channel", slog.Uint64("request_id", uint64(header.RequestID)))
				}
			} else {
				slog.Error("invalid type in pending map", slog.Uint64("request_id", uint64(header.RequestID)))
				c.pending.Delete(header.RequestID)
			}
		} else {
			slog.Warn("unknown request ID", slog.Uint64("request_id", uint64(header.RequestID)))
		}
	}
}

func readHeader(conn net.Conn) (*comm.Header, error) {
	header := make([]byte, 12)
	if err := readFull(conn, header); err != nil {
		return nil, err
	}
	return comm.UnmarshalHeader(header), nil
}

func readFull(r io.Reader, buf []byte) error {
	_, err := io.ReadFull(r, buf)
	return err
}
