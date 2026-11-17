package vm

import (
	"fmt"
	"strings"
)

const regsPerLine = 8

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
	var b strings.Builder

	_, _ = fmt.Fprintf(&b, "%7d %-10s PC=%08X IR=%08X MAR=%08X MDR=%08X IDX=%d POS=%2d %s",
		r.Tick, r.Step, r.PC, r.IR, r.MAR, r.MDR, r.Idx, r.Pos, r.irq())
	if r.Mem != "" {
		_, _ = fmt.Fprintf(&b, " mem:%s", r.Mem)
	}
	if r.IO != "" {
		_, _ = fmt.Fprintf(&b, " io:%s", r.IO)
	}
	if r.Signals != "" {
		_, _ = fmt.Fprintf(&b, " | %s", r.Signals)
	}
	return b.String()
}

func (r Record) InstructionLine(source string) string {
	if source == "" {
		source = "?"
	}

	var b strings.Builder

	_, _ = fmt.Fprintf(&b, "=== %08X  %-30s tick=%d %s\n   ", r.InstrAddr, source, r.Tick, r.irq())
	for i, v := range r.Regs {
		if i == regsPerLine {
			b.WriteString("\n   ")
		}
		_, _ = fmt.Fprintf(&b, " %3s=%08X", regName(i), v)
	}
	_, _ = fmt.Fprintf(&b, "  NZVC=%d%d%d%d IE=%d", bit(r.N), bit(r.Z), bit(r.V), bit(r.C), bit(r.IE))
	return b.String()
}

func (r Record) irq() string {
	if r.IRQPending {
		return fmt.Sprintf("irq=%d pending", r.IRQDepth)
	}
	return fmt.Sprintf("irq=%d", r.IRQDepth)
}

func regName(i int) string {
	if i == 0 {
		return "rsp"
	}
	return fmt.Sprintf("r%d", i)
}

func bit(f bool) int {
	if f {
		return 1
	}
	return 0
}

func (p *Machine) snapshot(sig signals, mem memOut) Record {
	panic("TODO")
}
