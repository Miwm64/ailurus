package vm

// machine state right after a tick.
type Record struct {
	Tick       uint64
	Step       string
	PC, IR     uint32
	MAR, MDR   uint32
	Idx, Pos   uint8
	Regs       [16]uint32
	N, Z, V, C bool
	IE         bool
	IRQPending bool
	IRQDepth   int
	Mem        string
	IO         string
	Signals    string

	InstrDone bool
	InstrAddr uint32
}

func (r Record) TickLine() string {
	panic("TODO")
}

func (r Record) InstrLine(source string) string {
	panic("TODO")
}

func (p *Machine) snapshot(sig signals, mem memOut) Record {
	panic("TODO")
}
