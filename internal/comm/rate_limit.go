package comm

import (
	"encoding/binary"
	"fmt"
	"time"
)

const (
	rateLimitReqHeaderSize = 28
	rateLimitRespSize      = 32
)

type RateLimitRequestData struct {
	Rate   uint64
	Burst  uint64
	Period time.Duration // int64
	Key    string
}

// Marshall encodes RateLimitRequestData into a byte slice.
func (r *RateLimitRequestData) Marshall() []byte {
	keyLen := len(r.Key)
	data := make([]byte, rateLimitReqHeaderSize+keyLen)
	binary.BigEndian.PutUint64(data[0:], r.Rate)
	binary.BigEndian.PutUint64(data[8:], r.Burst)
	binary.BigEndian.PutUint64(data[16:], uint64(r.Period))
	binary.BigEndian.PutUint32(data[24:], uint32(keyLen))
	if keyLen > 0 {
		copy(data[rateLimitReqHeaderSize:], r.Key)
	}
	return data
}

// Unmarshal decodes RateLimitRequestData from a byte slice.
func (r *RateLimitRequestData) Unmarshal(data []byte) error {
	if len(data) < rateLimitReqHeaderSize {
		return fmt.Errorf("data too short: got %d bytes, expected at least %d", len(data), rateLimitReqHeaderSize)
	}
	r.Rate = binary.BigEndian.Uint64(data[0:])
	r.Burst = binary.BigEndian.Uint64(data[8:])
	r.Period = time.Duration(binary.BigEndian.Uint64(data[16:]))
	keyLen := binary.BigEndian.Uint32(data[24:])
	if len(data) < int(rateLimitReqHeaderSize+keyLen) {
		return fmt.Errorf("data length mismatch: expected %d, got %d", rateLimitReqHeaderSize+keyLen, len(data))
	}
	if keyLen > 0 {
		r.Key = string(data[rateLimitReqHeaderSize : rateLimitReqHeaderSize+keyLen])
	} else {
		r.Key = ""
	}
	return nil
}

type RateLimitResponseData struct {
	Allowed    int64
	Remaining  int64
	RetryAfter time.Duration // int64
	ResetAfter time.Duration // int64
}

// Marshall encodes RateLimitResponseData into a byte slice.
func (r *RateLimitResponseData) Marshall() []byte {
	data := make([]byte, rateLimitRespSize)
	binary.BigEndian.PutUint64(data[0:], uint64(r.Allowed))
	binary.BigEndian.PutUint64(data[8:], uint64(r.Remaining))
	binary.BigEndian.PutUint64(data[16:], uint64(r.RetryAfter))
	binary.BigEndian.PutUint64(data[24:], uint64(r.ResetAfter))
	return data
}

// Unmarshal decodes RateLimitResponseData from a byte slice.
func (r *RateLimitResponseData) Unmarshal(data []byte) error {
	if len(data) < rateLimitRespSize {
		return fmt.Errorf("data too short: got %d bytes, expected at least %d", len(data), rateLimitRespSize)
	}
	r.Allowed = int64(binary.BigEndian.Uint64(data[0:]))
	r.Remaining = int64(binary.BigEndian.Uint64(data[8:]))
	r.RetryAfter = time.Duration(binary.BigEndian.Uint64(data[16:]))
	r.ResetAfter = time.Duration(binary.BigEndian.Uint64(data[24:]))
	return nil
}
