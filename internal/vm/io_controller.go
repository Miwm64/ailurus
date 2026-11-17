package vm

const (
	portIn  = 0
	portOut = 1
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
}

func newIO(schedule []InputEvent) ioController {
	return ioController{
		schedule: schedule,
	}
}

func (p *ioController) request() bool {
	return p.requestInterrupt
}

func (p *ioController) outputs(sig signals) ioOut {
	panic("TODO")
}

func (p *ioController) clock(tick uint64, sig signals, data uint32) {
	panic("TODO")
}
