package server

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"os"
	"sync"

	"github.com/zekihan/traefik-rate-limit/internal/comm"
	"github.com/zekihan/traefik-rate-limit/internal/rate_limit"
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, 128))
	},
}

func runServer(ctx context.Context, socketPath string) error {
	_ = os.Remove(socketPath)
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return err
	}
	defer listener.Close()
	slog.Info("server listening", slog.String("socket", socketPath))

	var wg sync.WaitGroup
	connChan := make(chan net.Conn)
	errChan := make(chan error, 1)

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					slog.Info("listener closed")
				} else {
					slog.Error("accept error", slog.Any("error", err))
					errChan <- err
				}
				close(connChan)
				return
			}
			connChan <- conn
		}
	}()

	for {
		select {
		case conn, ok := <-connChan:
			if !ok {
				goto shutdown
			}
			wg.Add(1)
			go handleConn(&wg, conn)
		case <-ctx.Done():
			slog.Info("shutdown signal received")
			_ = listener.Close()
			goto shutdown
		case err := <-errChan:
			slog.Error("accept error", slog.Any("error", err))
			goto shutdown
		}
	}

shutdown:
	wg.Wait()
	slog.Info("all connections closed")
	return nil
}

func handleConn(wg *sync.WaitGroup, conn net.Conn) {
	defer wg.Done()
	defer conn.Close()
	slog.Debug("new connection", slog.String("remote_addr", conn.RemoteAddr().String()))

	for {
		header, err := readHeader(conn)
		if err != nil {
			if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) {
				slog.Debug("connection closed", slog.Any("error", err))
			} else {
				slog.Warn("read header error", slog.Any("error", err))
			}
			return
		}
		if header == nil || header.Version != comm.VERSION || header.ContentLength > 1024*1024 {
			return
		}

		payload := make([]byte, header.ContentLength)
		if err := readFull(conn, payload); err != nil {
			slog.Debug("read payload error", slog.Any("error", err))
			return
		}

		req := &comm.Request{}
		if err := req.Unmarshal(header, payload); err != nil {
			slog.Debug("parse request error", slog.Any("error", err))
			return
		}

		resp := &comm.Response{Header: header, Type: req.Type, Status: comm.ResponseStatusOK}
		switch req.Type {
		case comm.RequestTypePing:
			resp.Data = "pong to " + req.GetPingData()
		case comm.RequestTypeRateLimit:
			data := req.GetRateLimitData()
			slog.Debug("rate limit request", slog.Any("data", data))
			result, err := rate_limit.RateLimit(data)
			if err != nil {
				resp.Status = comm.ResponseStatusError
				resp.Error = err.Error()
				break
			}
			resp.Data = &comm.RateLimitResponseData{
				Allowed:    int64(result.Allowed),
				Remaining:  int64(result.Remaining),
				RetryAfter: result.RetryAfter,
				ResetAfter: result.ResetAfter,
			}
		default:
			resp.Status = comm.ResponseStatusError
			resp.Error = "unknown request type"
		}
		respond(conn, resp)
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

func respond(conn net.Conn, resp *comm.Response) {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufferPool.Put(buf)

	if err := resp.Marshal(buf); err != nil {
		slog.Info("marshal response error", slog.Any("error", err))
		return
	}
	_, err := conn.Write(buf.Bytes())
	if err != nil {
		slog.Debug("write response error", slog.Any("error", err), slog.Int("attempted_bytes", buf.Len()))
	}
}
