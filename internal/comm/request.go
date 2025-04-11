package comm

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"sync"
)

type Request struct {
	*Header
	Type RequestType
	Data any
}

type RequestType uint8

const (
	RequestTypeUnknown RequestType = iota
	RequestTypePing
	RequestTypeRateLimit
)

func (r *Request) GetPingData() string {
	if r.Type != RequestTypePing {
		panic("not a ping request")
	}
	return r.Data.(string)
}

func (r *Request) GetRateLimitData() *RateLimitRequestData {
	if r.Type != RequestTypeRateLimit {
		panic("not a rate limit request")
	}
	return r.Data.(*RateLimitRequestData)
}

// buffer pool for marshaling request data part
var requestDataPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

// Marshal encodes the request into the writer.
func (r *Request) Marshal(w io.Writer) error {
	payloadBuf := requestDataPool.Get().(*bytes.Buffer)
	payloadBuf.Reset()
	defer requestDataPool.Put(payloadBuf)

	switch r.Type {
	case RequestTypePing:
		payloadBuf.WriteByte(byte(RequestTypePing))
		if r.Data != nil {
			switch v := r.Data.(type) {
			case string:
				payloadBuf.WriteString(v)
			case []byte:
				payloadBuf.Write(v)
			default:
				return fmt.Errorf("unsupported data type %T for request type %d", v, r.Type)
			}
		}
	case RequestTypeRateLimit:
		payloadBuf.WriteByte(byte(RequestTypeRateLimit))
		if r.Data != nil {
			payloadBuf.Write(r.Data.(*RateLimitRequestData).Marshall())
		}
	default:
		return fmt.Errorf("unknown request type: %d", r.Type)
	}

	headerBytes := make([]byte, 12)
	binary.BigEndian.PutUint32(headerBytes, r.Header.RequestID)
	binary.BigEndian.PutUint32(headerBytes[4:], VERSION)
	binary.BigEndian.PutUint32(headerBytes[8:], uint32(payloadBuf.Len()))

	if _, err := w.Write(headerBytes); err != nil {
		return fmt.Errorf("failed to write request header: %w", err)
	}
	if _, err := io.Copy(w, payloadBuf); err != nil {
		return fmt.Errorf("failed to write request payload: %w", err)
	}
	return nil
}

// Unmarshal decodes the request from header and data.
func (r *Request) Unmarshal(header *Header, data []byte) error {
	if len(data) < 1 {
		return fmt.Errorf("request data too short: got %d bytes, expected at least 1", len(data))
	}
	r.Header = header
	switch data[0] {
	case byte(RequestTypePing):
		r.Type = RequestTypePing
		r.Data = string(data[1:])
	case byte(RequestTypeRateLimit):
		r.Type = RequestTypeRateLimit
		r.Data = &RateLimitRequestData{}
		if err := r.Data.(*RateLimitRequestData).Unmarshal(data[1:]); err != nil {
			return fmt.Errorf("failed to unmarshal rate limit data: %w", err)
		}
	default:
		r.Type = RequestTypeUnknown
		r.Data = data[1:]
	}
	return nil
}
