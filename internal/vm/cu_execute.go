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

func (p *controlUnit) execute(sig signals) signals {
	if f, ok := p.format(); ok && f == isa.F0 {
		return p.execF0(sig)
	}
	return sig
}

// execF0: ei / di set IE; ret pops PC; iret pops PS, then PC. Stack: rsp = reg #0.
func (p *controlUnit) execF0(sig signals) signals {
	switch op := p.opcode(); {
	case op == opEI, op == opDI:
		// IE ← bit 2 of data: the constant 4 for ei, 0 for di
		sig.selDataMux = dataZero
		if op == opEI {
			sig.selDataMux = dataFour
		}
		sig.selALUB, sig.aluOp = aluBData, aluPassB
		sig.selPS, sig.latchPS = psResult, maskIE
	case op == opHalt:

	// ret, iret
	case p.step == stExec0: // MAR ← rsp
		sig.selALUA, sig.aluOp = aluAOutA, aluPassA
		sig.selMAR, sig.latchMAR = marResult, true
	case p.step == stExec1, p.step == stExec4: // MDR ← mem[MAR]
		sig.memOp, sig.latchMDR = memRead, true
	case op == opIRet && p.step == stExec2: // PS ← MDR
		sig.selALUA, sig.aluOp = aluAMDR, aluPassA
		sig.selPS, sig.latchPS = psResult, maskN|maskZ|maskV|maskC|maskIE
	default:
		sig.selALUA, sig.selALUB, sig.selDataMux, sig.aluOp = aluAOutA, aluBData, dataFour, aluAdd
		sig.latchReg = true
		if p.step == stExec3 {
			sig.selMAR, sig.latchMAR = marResult, true
		} else {
			sig.selPCMux, sig.latchPC = pcFromMDR, true
		}
	}
	return sig
}

// nextF0: ret takes 3 execute steps, iret 6, the rest 1.
func (p *controlUnit) nextF0() cuStep {
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

func (p *controlUnit) decode() error {
	op := p.opcode()
	f, ok := p.format()
	if !ok {
		return fmt.Errorf("%w: 0x%02X at PC %#x", ErrBadOpcode, op, p.pc-4)
	}
	if f != isa.F0 {
		return fmt.Errorf("%w: %s", errNotImplemented, isa.Mnemonics[op])
	}
	return nil
}

// format: the instruction format the opcode decodes to; false for an invalid opcode.
func (p *controlUnit) format() (isa.Format, bool) {
	info, ok := isa.Opcodes[isa.Mnemonics[p.opcode()]]
	return info.Format, ok
}
