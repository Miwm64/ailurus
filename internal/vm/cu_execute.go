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

// addrSteps: OF steps that put a memory operand's address into MAR, by mode; MDR ← mem[MAR] follows.
var addrSteps = [...]cuStep{
	isa.DirectMemory:     1, // MAR ← MDR (the address, read by of_dst_ext)
	isa.PCRelative:       1, // MAR ← PC + offset
	isa.RegisterIndirect: 1, // MAR ← r
	isa.PostIncrement:    2, // MAR ← r; r ← r + 4
	isa.PreDecrement:     1, // r, MAR ← r − 4
	isa.RegisterRelative: 1, // MAR ← r + offset
}

func memOperand(sig signals, fe fieldOut, i cuStep) signals {
	switch {
	case !fe.fits: // MAR ← PC
		sig.selMAR, sig.latchMAR = marPC, true
		return sig
	case i == addrSteps[fe.mode]: // MDR ← mem[MAR]
		sig.memOp, sig.latchMDR = memRead, true
		return sig
	case fe.mode == isa.DirectMemory: // MAR ← MDR
		sig.selMAR, sig.latchMAR = marMDR, true
		return sig
	}

	sig.selRegA, sig.selALUA, sig.selALUB, sig.selDataMux = regField, aluAOutA, aluBData, dataImm
	switch {
	case fe.mode == isa.PostIncrement && i == 1: // r ← r + 4
		sig.selDataMux, sig.aluOp = dataFour, aluAdd
		sig.selRegW, sig.latchReg = regField, true
		return sig
	case fe.mode == isa.PreDecrement: // r, MAR ← r − 4
		sig.selDataMux, sig.aluOp = dataFour, aluSub
		sig.selRegW, sig.latchReg = regField, true
	case fe.mode == isa.PCRelative: // MAR ← PC + offset
		sig.selALUA, sig.aluOp = aluAPC, aluAdd
	case fe.mode == isa.RegisterRelative: // MAR ← r + offset
		sig.aluOp = aluAdd
	default: // indirect, postincrement: MAR ← r
		sig.aluOp = aluPassA
	}
	sig.selMAR, sig.latchMAR = marResult, true
	return sig
}

func (p *controlUnit) execute(sig signals, fe fieldOut) signals {
	if p.step == stOFDst0 && p.decode(fe) != nil {
		return sig // no signals: the CU faults on this tick
	}
	switch f, ok := p.format(); {
	case ok && f == isa.F0:
		return p.execF0(sig)
	case ok && f == isa.F1:
		return p.execF1(sig, fe)
	}
	return sig
}

// nextExec: next step after fetch
func (p *controlUnit) nextExec(fe fieldOut) cuStep {
	switch f, _ := p.format(); f {
	case isa.F1:
		return p.nextF1(fe)
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

func (p *controlUnit) execF1(sig signals, fe fieldOut) signals {
	reg := fe.mode == isa.Register
	switch {
	case p.step == stWB0:
		sig.selALUB, sig.aluOp, sig.memOp = aluBTMP, aluPassB, memWrite
	case p.step == stExec0:
		ctl := aluControl[p.opcode()]
		sig.selALUB, sig.selDataMux, sig.aluOp = aluBData, dataZero, ctl.op
		sig.selPS, sig.latchPS = psFlags, ctl.flags
		if reg {
			sig.selRegA, sig.selALUA = regDst, aluAOutA
			sig.selRegW, sig.latchReg = regDst, true
		} else {
			sig.selALUA = aluAMDR
			sig.selTMP, sig.latchTMP = tmpResult, true
		}
	case reg: // of_dst0
		sig.latchDstReg = true
	default:
		sig = memOperand(sig, fe, p.step-stOFDst0)
	}
	return sig
}

// nextF1: of_dst, exec0, wb0.
func (p *controlUnit) nextF1(fe fieldOut) cuStep {
	reg := fe.mode == isa.Register
	switch {
	case p.step == stExec0 && !reg:
		return stWB0
	case p.step == stExec0, p.step == stWB0:
		return stFetch0
	case !fe.fits:
		return stOFDstExt
	case reg, p.step-stOFDst0 == addrSteps[fe.mode]:
		return stExec0
	default:
		return p.step + 1
	}
}

func (p *controlUnit) decode(fe fieldOut) error {
	op := p.opcode()
	f, ok := p.format()
	switch {
	case !ok:
		return fmt.Errorf("%w: 0x%02X at PC %#x", ErrBadOpcode, op, p.pc-4)
	case f == isa.F1 && fe.mode == isa.Immediate:
		return fmt.Errorf("%w: immediate destination at PC %#x", ErrBadOpcode, p.pc-4)
	case f == isa.F0, f == isa.F1:
		return nil
	default:
		return fmt.Errorf("%w: %s", errNotImplemented, isa.Mnemonics[op])
	}
}

func (p *controlUnit) format() (isa.Format, bool) {
	info, ok := isa.Opcodes[isa.Mnemonics[p.opcode()]]
	return info.Format, ok
}
