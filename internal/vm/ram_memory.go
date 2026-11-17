package vm

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
	panic("TODO")
}

func (p *ram) outputs(req memReq) memOut {
	panic("TODO")
}

func (p *ram) clock(req memReq) {
	panic("TODO")
}
