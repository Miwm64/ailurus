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
	opCmp  = mustOpcode("cmp")
	opTest = mustOpcode("test")
	opIn   = mustOpcode("in")
	opOut  = mustOpcode("out")
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
	flagsNZ   = maskN | maskZ
	flagsNZC  = flagsNZ | maskC
	flagsNZV  = flagsNZ | maskV
	flagsNZVC = flagsNZV | maskC
)

var aluControl = map[isa.Opcode]aluCtl{
	// F1: B = data 0
	mustOpcode("clr"): {aluPassB, flagsNZVC},
	mustOpcode("not"): {aluNot, flagsNZVC},
	mustOpcode("inc"): {aluInc, flagsNZV},
	mustOpcode("dec"): {aluDec, flagsNZV},
	mustOpcode("neg"): {aluNeg, flagsNZVC},

	// F2: A = destination, B = source
	mustOpcode("mov"):   {aluPassB, flagsNZ},
	mustOpcode("and"):   {aluAnd, flagsNZVC},
	mustOpcode("or"):    {aluOr, flagsNZVC},
	mustOpcode("xor"):   {aluXor, flagsNZVC},
	mustOpcode("add"):   {aluAdd, flagsNZVC},
	mustOpcode("addc"):  {aluAddC, flagsNZVC},
	mustOpcode("sub"):   {aluSub, flagsNZVC},
	mustOpcode("subc"):  {aluSubC, flagsNZVC},
	mustOpcode("mulh"):  {aluMulH, flagsNZ},
	mustOpcode("umulh"): {aluUMulH, flagsNZ},
	mustOpcode("mull"):  {aluMulL, flagsNZ},
	mustOpcode("div"):   {aluDiv, flagsNZ},
	mustOpcode("mod"):   {aluMod, flagsNZ},
	mustOpcode("cmp"):   {aluSub, flagsNZVC},
	mustOpcode("test"):  {aluAnd, flagsNZVC},
	mustOpcode("lsl"):   {aluLsl, flagsNZC},
	mustOpcode("lsr"):   {aluLsr, flagsNZC},
	mustOpcode("asr"):   {aluAsr, flagsNZC},
}

// shape: the decoder's view of an F1 / F2 instruction, from IR.
type shape struct {
	dstMem  bool
	hasSrc  bool
	srcMem  bool
	srcExt  bool // source in a later word than the destination
	srcTMP  bool // of_src_end puts the source into TMP
	readDst bool // all but mov, clr
	write   bool // all but cmp, test
}

func (p *controlUnit) shape() shape {
	modes, positions, _ := isa.Layout(p.ir)
	op := p.opcode()
	s := shape{
		dstMem: isMemory(modes[0]),
		write:  op != opCmp && op != opTest,
	}
	s.readDst = s.dstMem && aluControl[op].op != aluPassB
	if len(modes) == 2 {
		s.hasSrc, s.srcMem, s.srcExt = true, isMemory(modes[1]), positions[1].Word > positions[0].Word
		s.srcTMP = s.srcMem || s.srcExt && s.dstMem
	}
	return s
}

func isMemory(m isa.Mode) bool {
	return m != isa.Register && m != isa.Immediate
}

var addrSteps = [...]cuStep{
	isa.DirectMemory:     1, // MAR ← MDR
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
		return sig // faults in clock
	}
	switch f, ok := p.format(); {
	case ok && f == isa.F0:
		return p.execF0(sig)
	case ok && (f == isa.F1 || f == isa.F2):
		return p.execALU(sig, fe)
	}
	return sig
}

// nextExec: next step after fetch
func (p *controlUnit) nextExec(fe fieldOut) cuStep {
	switch f, _ := p.format(); f {
	case isa.F1, isa.F2:
		return p.nextALU(fe)
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

// execALU: F1 / F2. A = destination, B = source (F1: data 0).
func (p *controlUnit) execALU(sig signals, fe fieldOut) signals {
	s := p.shape()

	switch p.step {
	case stOFDst0, stOFDst1, stOFDst2: // register: DST_REG ← r; memory: address → MAR
		i := p.step - stOFDst0
		if s.dstMem {
			sig = memOperand(sig, fe, i)
		} else {
			sig.latchDstReg = true
		}
		if fe.fits && (!s.dstMem || i == addrSteps[fe.mode]-1) { // last destination step: POS → source
			sig.selPos, sig.latchPos = posNext, true
		}
	case stOFSrc0, stOFSrc1, stOFSrc2: // memory source → MDR
		i := p.step - stOFSrc0
		sig = memOperand(sig, fe, i)
		sig.latchMARDst = s.dstMem && (!fe.fits || !s.srcExt && i == 0) // first source step only
	case stOFSrcEnd: // TMP ← source, MAR ← MAR_DST
		sig.selTMP, sig.latchTMP = tmpMDR, true
		if !s.srcMem { // register / immediate from the next word: TMP ← pass_b
			sig = sourceB(sig, fe)
			sig.aluOp, sig.selTMP = aluPassB, tmpResult
		}
		if s.dstMem {
			sig.selMAR, sig.latchMAR = marMARDst, true
		}
	case stOFDstRead: // MDR ← mem[MAR]
		sig.memOp, sig.latchMDR = memRead, true
	case stExec0: // DST_REG register / TMP ← A op B, flags
		ctl := aluControl[p.opcode()]
		sig.aluOp = ctl.op
		sig.selPS, sig.latchPS = psFlags, ctl.flags

		if s.dstMem {
			sig.selALUA = aluAMDR
		} else {
			sig.selRegA, sig.selALUA = regDst, aluAOutA
		}

		switch {
		case !s.hasSrc:
			sig.selALUB, sig.selDataMux = aluBData, dataZero
		case s.srcTMP:
			sig.selALUB = aluBTMP
		default:
			sig = sourceB(sig, fe)
		}

		switch {
		case !s.write:
		case s.dstMem:
			sig.selTMP, sig.latchTMP = tmpResult, true
		default:
			sig.selRegW, sig.latchReg = regDst, true
		}
	case stWB0: // mem[MAR] ← TMP
		sig.selALUB, sig.aluOp, sig.memOp = aluBTMP, aluPassB, memWrite
	}
	return sig
}

// sourceB: register / immediate source → ALU B.
func sourceB(sig signals, fe fieldOut) signals {
	if fe.mode == isa.Immediate {
		sig.selALUB, sig.selDataMux = aluBData, dataImm
	} else { // reg_b straight from FIELD_EXT
		sig.selALUB = aluBOutB
	}
	return sig
}

func (p *controlUnit) nextALU(fe fieldOut) cuStep {
	s := p.shape()
	operandsDone := stExec0
	if s.readDst {
		operandsDone = stOFDstRead
	}

	switch p.step {
	case stOFDst0, stOFDst1, stOFDst2:
		switch {
		case !fe.fits:
			return stOFDstExt
		case s.dstMem && p.step-stOFDst0 < addrSteps[fe.mode]-1:
			return p.step + 1
		case s.srcMem || s.srcExt:
			return stOFSrc0
		default:
			return operandsDone
		}
	case stOFSrc0, stOFSrc1, stOFSrc2:
		switch {
		case !fe.fits:
			return stOFSrcExt
		case p.step-stOFSrc0 < addrSteps[fe.mode]:
			return p.step + 1
		default:
			return stOFSrcEnd
		}
	case stOFSrcExt:
		switch {
		case s.srcMem:
			return stOFSrc0
		case s.dstMem:
			return stOFSrcEnd
		default:
			return stExec0
		}
	case stOFSrcEnd:
		return operandsDone
	case stOFDstRead:
		return stExec0
	case stExec0:
		if s.dstMem && s.write {
			return stWB0
		}
		return stFetch0
	default: // stWB0
		return stFetch0
	}
}

func (p *controlUnit) decode(fe fieldOut) error {
	op := p.opcode()
	f, ok := p.format()
	switch {
	case !ok:
		return fmt.Errorf("%w: 0x%02X at PC %#x", ErrBadOpcode, op, p.pc-4)
	case op == opIn, op == opOut:
		return fmt.Errorf("%w: %s", errNotImplemented, isa.Mnemonics[op])
	case (f == isa.F1 || f == isa.F2) && fe.mode == isa.Immediate:
		return fmt.Errorf("%w: immediate destination at PC %#x", ErrBadOpcode, p.pc-4)
	case f == isa.F0, f == isa.F1, f == isa.F2:
		return nil
	default:
		return fmt.Errorf("%w: %s", errNotImplemented, isa.Mnemonics[op])
	}
}

func (p *controlUnit) format() (isa.Format, bool) {
	info, ok := isa.Opcodes[isa.Mnemonics[p.opcode()]]
	return info.Format, ok
}
