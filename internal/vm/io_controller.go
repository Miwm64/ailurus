package vm

import "fmt"

const (
	portIn  = 0
	portOut = 1
)

// ioOp: io_op from the CU (2 bits).
type ioOp uint8

const (
	ioNone ioOp = iota
	ioRead
	ioWrite
)

type ioOut struct {
	data  uint32
	fault error
}

// port 0 — input, port 1 — output.
type ioController struct {
	schedule         []InputEvent
	next             int // next schedule entry
	port0            byte
	requestInterrupt bool
	output           []byte
	event            string // journal: what happened at the last edge
}

func newIO(schedule []InputEvent) ioController {
	return ioController{
		schedule: schedule,
	}
}

func (p *ioController) isRequestInterrupt() bool {
	return p.requestInterrupt
}

func (p *ioController) outputs(sig signals) ioOut {
	switch {
	case sig.ioOp == ioRead && sig.port == portIn:
		return ioOut{data: uint32(p.port0)}
	case sig.ioOp == ioWrite && sig.port == portOut:
		return ioOut{}
	case sig.ioOp == ioNone:
		return ioOut{}
	default:
		return ioOut{fault: fmt.Errorf("%w: %d", ErrBadPort, sig.port)}
	}
}

func (p *ioController) clock(tick uint64, sig signals, data uint32) {
	p.event = ""
	p.requestInterrupt = false

	if sig.ioOp == ioWrite && sig.port == portOut {
		p.output = append(p.output, byte(data))
		p.event = fmt.Sprintf("out %q", rune(byte(data)))
	}

	if p.next < len(p.schedule) && p.schedule[p.next].Tick <= tick {
		p.port0 = p.schedule[p.next].Char
		p.requestInterrupt = true
		p.next++
		p.event = joinEvents(p.event, fmt.Sprintf("in %q", rune(p.port0)))
	}
}

func joinEvents(a, b string) string {
	if a == "" {
		return b
	}

	return a + ", " + b
}
