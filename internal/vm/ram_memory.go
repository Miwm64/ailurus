package vm

import (
	"encoding/binary"
	"fmt"
)

const (
	ramLatency      = 10 // ticks per memRead / memWrite
	ramBurstLatency = 1  // ticks per memReadNext
)

// ram: byte-addressed, little-endian, any 4 consecutive bytes per access.
type ram struct {
	bytes []byte
	busy  uint8 // current tick number
}

func newRAM(size uint32, image []byte) (*ram, error) {
	if uint64(len(image)) > uint64(size) {
		return nil, fmt.Errorf("%w: %d > %d bytes", ErrImageTooBig, len(image), size)
	}
	bytes := make([]byte, size)
	copy(bytes, image)
	return &ram{bytes: bytes}, nil
}

func (p *ram) outputs(req memReq) memOut {
	panic("TODO")
}

func (p *ram) clock(req memReq) {
	out := p.outputs(req)

	switch {
	case req.Op == memNone, out.fault != nil:
		p.busy = 0
	case out.ready:
		if req.Op == memWrite {
			binary.LittleEndian.PutUint32(p.bytes[req.addr:], req.data)
		}
		p.busy = 0
	default:
		p.busy++
	}
}
