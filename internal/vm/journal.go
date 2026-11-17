package vm

import (
	"fmt"
	"strings"
)

const regsPerLine = 8

const (
	irqWidth = len("irq=1 pending")
	memWidth = len("unaligned read")
	ioWidth  = len("out 'a', in 'b'")
)

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
	return fmt.Sprintf("%7d %-10s PC=%08X IR=%08X MAR=%08X MDR=%08X IDX=%d POS=%2d %-*s mem:%-*s io:%-*s | %s",
		r.Tick, r.Step, r.PC, r.IR, r.MAR, r.MDR, r.Idx, r.Pos,
		irqWidth, r.irq(), memWidth, orDash(r.Mem), ioWidth, orDash(r.IO), orDash(r.Signals))
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

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
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
	prev, now := p.prevStep, p.cu.step
	p.prevStep = now

	if prev == stFetch0 {
		p.instrAddr = p.cu.pc
	}
	done := p.cu.halted || instrStep(prev) && (now == stFetch0 || now == stIRQ0)
	if now == stIRQ0 && prev != stIRQ0 {
		p.irqDepth++
	}
	if done && !p.cu.halted && p.cu.opcode() == opIRet {
		p.irqDepth--
	}

	ps := p.dp.ps
	return Record{
		Tick: p.tick, Step: prev.String(),
		PC: p.cu.pc, IR: p.cu.ir, MAR: p.dp.mar, MDR: p.dp.mdr,
		Idx: p.cu.idx, Pos: p.cu.pos,
		Regs: p.dp.regs,
		N:    ps.n, Z: ps.z, V: ps.v, C: ps.c, IE: ps.ie,
		IRQPending: p.cu.irqPending, IRQDepth: p.irqDepth,
		Mem: mem.event, IO: p.io.event, Signals: activeSignals(sig),
		InstrDone: done, InstrAddr: p.instrAddr,
	}
}

func instrStep(s cuStep) bool {
	switch s {
	case stReset0, stReset1, stIRQ0, stFetch0, stFetch1, stFetch2:
		return false
	default:
		return true
	}
}

func activeSignals(sig signals) string {
	signalNames := []struct {
		on   bool
		name string
	}{
		{sig.latchReg, "latch_reg_w"}, {sig.latchMAR, "latch_mar"}, {sig.latchMARDst, "latch_mar_dst"},
		{sig.latchMDR, "latch_mdr"}, {sig.latchTMP, "latch_tmp"}, {sig.latchPS != 0, "latch_ps"},
		{sig.latchIR, "latch_ir"}, {sig.latchPC, "latch_pc"}, {sig.latchIdx, "latch_idx"},
		{sig.latchPos, "latch_pos"}, {sig.clearIRQ, "clear_irq"},
		{sig.memOp == memRead, "mem_op=read"}, {sig.memOp == memWrite, "mem_op=write"},
		{sig.ioOp == ioRead, "io_op=read"}, {sig.ioOp == ioWrite, "io_op=write"},
	}

	var names []string
	for _, s := range signalNames {
		if s.on {
			names = append(names, s.name)
		}
	}
	return strings.Join(names, " ")
}
