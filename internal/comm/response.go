package comm

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"sync"
)

type Response struct {
	*Header
	Status ResponseStatus
	Type   RequestType
	Data   any
	Error  string
}

type ResponseStatus uint8

const (
	ResponseStatusUnknown ResponseStatus = iota
	ResponseStatusOK
	ResponseStatusError
)

// buffer pool for marshaling response data part
var responseDataPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

// Marshal encodes the response into the writer.
func (r *Response) Marshal(w io.Writer) error {
	payloadBuf := responseDataPool.Get().(*bytes.Buffer)
	payloadBuf.Reset()
	defer responseDataPool.Put(payloadBuf)

	payloadBuf.WriteByte(byte(r.Type))
	switch r.Status {
	case ResponseStatusOK:
		payloadBuf.WriteByte(byte(ResponseStatusOK))
		if r.Data != nil {
			switch r.Type {
			case RequestTypePing:
				switch v := r.Data.(type) {
				case string:
					payloadBuf.WriteString(v)
				case []byte:
					payloadBuf.Write(v)
				default:
					return fmt.Errorf("unsupported data type %T for response type %d", v, r.Type)
				}
			case RequestTypeRateLimit:
				data, ok := r.Data.(*RateLimitResponseData)
				if !ok {
					return fmt.Errorf("unsupported data type %T for response type %d", r.Data, r.Type)
				}
				payloadBuf.Write(data.Marshall())
			default:
				return fmt.Errorf("unsupported response type for data: %d", r.Type)
			}
		}
	case ResponseStatusError:
		payloadBuf.WriteByte(byte(ResponseStatusError))
		if r.Error != "" {
			payloadBuf.WriteString(r.Error)
		}
	default:
		return fmt.Errorf("unknown response status: %d", r.Status)
	}

	headerBytes := make([]byte, 12)
	binary.BigEndian.PutUint32(headerBytes, r.Header.RequestID)
	binary.BigEndian.PutUint32(headerBytes[4:], VERSION)
	binary.BigEndian.PutUint32(headerBytes[8:], uint32(payloadBuf.Len()))

	if _, err := w.Write(headerBytes); err != nil {
		return fmt.Errorf("failed to write response header: %w", err)
	}
	if _, err := io.Copy(w, payloadBuf); err != nil {
		return fmt.Errorf("failed to write response payload: %w", err)
	}
	return nil
}

// Unmarshal decodes the response from header and data.
func (r *Response) Unmarshal(header *Header, data []byte) error {
	if len(data) < 2 {
		return fmt.Errorf("response data too short: got %d bytes, expected at least 2", len(data))
	}
	r.Header = header
	switch data[0] {
	case byte(RequestTypePing):
		r.Type = RequestTypePing
	case byte(RequestTypeRateLimit):
		r.Type = RequestTypeRateLimit
	default:
		r.Type = RequestTypeUnknown
	}
	switch data[1] {
	case byte(ResponseStatusOK):
		r.Status = ResponseStatusOK
		switch r.Type {
		case RequestTypePing:
			r.Data = string(data[2:])
		case RequestTypeRateLimit:
			dataObj := RateLimitResponseData{}
			if err := dataObj.Unmarshal(data[2:]); err != nil {
				return fmt.Errorf("failed to unmarshal RateLimitResponseData: %w", err)
			}
			r.Data = &dataObj
		default:
			r.Data = data[2:]
		}
	case byte(ResponseStatusError):
		r.Status = ResponseStatusError
		r.Error = string(data[2:])
	default:
		r.Status = ResponseStatusUnknown
		return fmt.Errorf("unknown response status byte: %d", data[1])
	}
	return nil
}
