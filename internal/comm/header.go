package comm

import (
	"encoding/binary"
)

const VERSION = uint32(1)

type Header struct {
	RequestID     uint32
	Version       uint32
	ContentLength uint32
}

func UnmarshalHeader(header []byte) *Header {
	return &Header{
		RequestID:     binary.BigEndian.Uint32(header[0:4]),
		Version:       binary.BigEndian.Uint32(header[4:8]),
		ContentLength: binary.BigEndian.Uint32(header[8:12]),
	}
}
