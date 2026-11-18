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
	if req.Op == memNone {
		return memOut{}
	}

	if uint64(req.addr)+4 > uint64(len(p.bytes)) {
		return memOut{ready: true, fault: fmt.Errorf("%w: %#x", ErrBadAddress, req.addr), event: "fault"}
	}

	latency := uint8(ramLatency)
	if req.Op == memReadNext {
		latency = ramBurstLatency
	}

	if p.busy+1 < latency {
		return memOut{event: "wait"}
	}

	out := memOut{ready: true, event: "done"}
	if req.Op != memWrite {
		out.data = binary.LittleEndian.Uint32(p.bytes[req.addr:])
	}

	return out
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
