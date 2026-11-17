package vm

import (
	"errors"
	"fmt"

	"github.com/Miwm64/ailurus/internal/isa"
)

var errNotImplemented = errors.New("instruction not implemented yet")

var (
	opHalt = mustOpcode("halt")
	opRet  = mustOpcode("ret")
	opIRet = mustOpcode("iret")
	opEI   = mustOpcode("ei")
	opDI   = mustOpcode("di")
)

// mustOpcode: the opcode of a mnemonic; a typo panics at init.
func mustOpcode(name string) isa.Opcode {
	info, ok := isa.Opcodes[name]
	if !ok {
		panic(fmt.Sprintf("vm: unknown mnemonic %q", name))
	}
	return info.Opcode
}

type aluCtl struct {
	op    aluOp
	flags flagMask
}

const (
	flagsNZV  = maskN | maskZ | maskV
	flagsNZVC = flagsNZV | maskC
)

var aluControl = map[isa.Opcode]aluCtl{
	mustOpcode("clr"): {aluPassB, flagsNZVC}, // B = data 0
	mustOpcode("not"): {aluNot, flagsNZVC},
	mustOpcode("inc"): {aluInc, flagsNZV},
	mustOpcode("dec"): {aluDec, flagsNZV},
	mustOpcode("neg"): {aluNeg, flagsNZVC},
}

func (p *controlUnit) execute(sig signals) signals {
	switch f, ok := p.format(); {
	case ok && f == isa.F0:
		return p.execF0(sig)
	case ok && f == isa.F1:
		return p.execF1(sig)
	}
	return sig
}

// nextExec: next step after fetch
func (p *controlUnit) nextExec() cuStep {
	switch f, _ := p.format(); f {
	case isa.F1:
		return p.nextF1()
	default: // F0
		return p.nextF0()
	}
}

// execF0: ei / di set IE; ret pops PC; iret pops PS, then PC. Stack: rsp = reg #0.
func (p *controlUnit) execF0(sig signals) signals {
	switch op := p.opcode(); {
	case p.step == stOFDst0, op == opHalt: // of_dst0: decode only; halt stops in exec0
	case op == opEI, op == opDI:
		// IE ← bit 2 of data: the constant 4 for ei, 0 for di
		sig.selDataMux = dataZero
		if op == opEI {
			sig.selDataMux = dataFour
		}
		sig.selALUB, sig.aluOp = aluBData, aluPassB
		sig.selPS, sig.latchPS = psResult, maskIE

	// ret, iret
	case p.step == stExec0: // MAR ← rsp
		sig.selRegA, sig.selALUA, sig.aluOp = regRSP, aluAOutA, aluPassA
		sig.selMAR, sig.latchMAR = marResult, true
	case p.step == stExec1, p.step == stExec4: // MDR ← mem[MAR]
		sig.memOp, sig.latchMDR = memRead, true
	case op == opIRet && p.step == stExec2: // PS ← MDR
		sig.selALUA, sig.aluOp = aluAMDR, aluPassA
		sig.selPS, sig.latchPS = psResult, maskN|maskZ|maskV|maskC|maskIE
	default:
		sig.selRegA, sig.selALUA, sig.selALUB, sig.selDataMux, sig.aluOp = regRSP, aluAOutA, aluBData, dataFour, aluAdd
		sig.selRegW, sig.latchReg = regRSP, true
		if p.step == stExec3 {
			sig.selMAR, sig.latchMAR = marResult, true
		} else {
			sig.selPCMux, sig.latchPC = pcFromMDR, true
		}
	}
	return sig
}

// nextF0: of_dst0, then ret takes 3 execute steps, iret 6, the rest 1.
func (p *controlUnit) nextF0() cuStep {
	if p.step == stOFDst0 {
		return stExec0
	}
	last := stExec0
	switch p.opcode() {
	case opRet:
		last = stExec2
	case opIRet:
		last = stExec5
	}
	if p.step < last {
		return p.step + 1
	}
	return stFetch0
}

// execF1: register destination.
// of_dst0: DST_REG ← reg; exec0: reg[DST_REG] ← op(reg[DST_REG]), flags.
func (p *controlUnit) execF1(sig signals) signals {
	if p.step == stOFDst0 {
		sig.latchDstReg = true
		return sig
	}
	ctl := aluControl[p.opcode()]
	sig.selRegA, sig.selALUA, sig.selALUB, sig.selDataMux, sig.aluOp = regDst, aluAOutA, aluBData, dataZero, ctl.op
	sig.selRegW, sig.latchReg = regDst, true
	sig.selPS, sig.latchPS = psFlags, ctl.flags
	return sig
}

// nextF1: of_dst0, exec0.
func (p *controlUnit) nextF1() cuStep {
	if p.step == stOFDst0 {
		return stExec0
	}
	return stFetch0
}

func (p *controlUnit) decode(fe fieldOut) error {
	op := p.opcode()
	f, ok := p.format()
	switch {
	case !ok:
		return fmt.Errorf("%w: 0x%02X at PC %#x", ErrBadOpcode, op, p.pc-4)
	case f == isa.F0, f == isa.F1 && fe.mode == isa.Register:
		return nil
	default:
		return fmt.Errorf("%w: %s", errNotImplemented, isa.Mnemonics[op])
	}
}

func (p *controlUnit) format() (isa.Format, bool) {
	info, ok := isa.Opcodes[isa.Mnemonics[p.opcode()]]
	return info.Format, ok
}
