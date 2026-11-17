package vm

import (
	"errors"
	"fmt"

	"github.com/Miwm64/ailurus/internal/isa"
)

const irOpcodeShift = 26 // IR[31:26]

var errNotImplemented = errors.New("instruction not implemented yet")

var opIRet = isa.Opcodes["iret"].Opcode

type cuStep uint8

const (
	// reset: PC ← mem[0x0]
	stReset0 cuStep = iota // MDR ← mem[MAR]; reset leaves MAR = 0
	stReset1               // PC ← MDR

	// interrupt entry: push PC, push PS, IE ← 0, PC ← mem[0x4]
	stIRQ0

	// instruction fetch: IR ← mem[PC], PC ← PC + 4
	stFetch0 // MAR ← PC
	stFetch1 // MDR ← mem[MAR]
	stFetch2 // IR ← MDR, PC ← PC + 4, IDX ← 0, POS ← 0

	// operand fetch: destination, then sources
	stOFDst0
	stOFSrc0

	// execute, writeback
	stExec0 // first execute step, by opcode
	stWB0
)

var stepNames = [...]string{
	stReset0: "reset0",
	stReset1: "reset1",
	stIRQ0:   "irq0",
	stFetch0: "fetch0",
	stFetch1: "fetch1",
	stFetch2: "fetch2",
	stOFDst0: "of_dst0",
	stOFSrc0: "of_src0",
	stExec0:  "exec0",
	stWB0:    "wb0",
}

func (s cuStep) String() string {
	if int(s) < len(stepNames) && stepNames[s] != "" {
		return stepNames[s]
	}
	return fmt.Sprintf("step%d", s)
}

// sel_add_mux
const (
	addFour = iota
	addOffset
)

// sel_pc_mux
const (
	pcFromAdd = iota
	pcFromMDR
)

// sel_pos
const (
	posZero = iota
	posNext
)

// sel_idx
const (
	idxZero = iota
	idxInc
)

// sel_data_mux
const (
	dataImm = iota
	dataFour
	dataZero
)

type signals struct {
	// register file
	regA, regB uint8
	regW       uint8
	latchReg   bool
	forceRSP   bool

	// dataPath muxes
	selALUA uint8
	selALUB uint8
	selMAR  uint8
	selTMP  uint8
	selPS   uint8
	aluOp   aluOp

	// dataPath latches
	latchMAR, latchMARDst, latchMDR, latchTMP bool
	latchPS                                   flagMask // latch_n, _z, _v, _c, _ie

	// memory and IO
	memOp memOp
	ioOp  ioOp
	port  uint16

	// CU internal
	latchIR    bool
	latchPC    bool
	latchPos   bool
	latchIdx   bool
	clearIRQ   bool
	selAddMux  uint8
	selPCMux   uint8
	selPos     uint8
	selIdx     uint8
	selDataMux uint8

	// to Datapath
	data uint32 // data_mux output
	pc   uint32
}

type controlUnit struct {
	pc, ir     uint32
	step       cuStep
	idx, pos   uint8
	irqPending bool
	halted     bool
	fault      error
}

func (p *controlUnit) outputs(regs dpRegs) signals {
	sig := signals{pc: p.pc}

	switch p.step {
	case stReset0, stFetch1:
		sig.memOp, sig.latchMDR = memRead, true
	case stReset1:
		sig.selPCMux, sig.latchPC = pcFromMDR, true
	case stFetch0:
		sig.selMAR, sig.latchMAR = marPC, true
	case stFetch2:
		sig.latchIR = true
		sig.selAddMux, sig.selPCMux, sig.latchPC = addFour, pcFromAdd, true
		sig.selIdx, sig.latchIdx = idxZero, true
		sig.selPos, sig.latchPos = posZero, true
	case stExec0:
		sig = p.execute(sig)
	}

	sig.data = p.dataMux(sig.selDataMux, regs)
	return sig
}

func (p *controlUnit) execute(sig signals) signals {
	return sig
}

func (p *controlUnit) clock(sig signals, regs dpRegs, out dpOut, mem memOut, in ioOut, irqReq bool) {
	if irqReq {
		p.irqPending = true
	}

	switch {
	case p.halted:
		return
	case mem.fault != nil:
		p.stop(mem.fault)
		return
	case in.fault != nil:
		p.stop(in.fault)
		return
	case sig.memOp != memNone && !mem.ready:
		return // waiting for memory
	case p.step == stExec0:
		if err := p.decode(); err != nil {
			p.stop(err)
			return
		}
	}

	fe := p.field(regs)
	if sig.latchIR {
		p.ir = regs.mdr
	}
	if sig.latchPC {
		p.pc = p.pcMux(sig, regs, fe)
	}
	if sig.latchIdx {
		p.idx = p.idxMux(sig.selIdx)
	}
	if sig.latchPos {
		p.pos = posMux(sig.selPos, fe)
	}
	p.step = p.next()
}

// next_step
func (p *controlUnit) next() cuStep {
	switch p.step {
	case stReset0:
		return stReset1
	case stFetch0:
		return stFetch1
	case stFetch1:
		return stFetch2
	case stFetch2:
		return stExec0
	default: // stReset1, end of an instruction
		return stFetch0
	}
}

func (p *controlUnit) decode() error {
	op := p.opcode()
	if _, ok := isa.Mnemonics[op]; !ok {
		return fmt.Errorf("%w: 0x%02X at PC %#x", ErrBadOpcode, op, p.pc-4)
	}
	return fmt.Errorf("%w: %s", errNotImplemented, isa.Mnemonics[op])
}

func (p *controlUnit) opcode() isa.Opcode {
	return isa.Opcode(p.ir >> irOpcodeShift)
}

func (p *controlUnit) field(regs dpRegs) fieldOut {
	return fieldExt(fieldIn{ir: p.ir, mdr: regs.mdr, idx: p.idx, pos: p.pos})
}

func (p *controlUnit) stop(err error) {
	p.halted, p.fault = true, err
}

func (p *controlUnit) dataMux(sel uint8, regs dpRegs) uint32 {
	switch sel {
	case dataImm:
		return p.field(regs).value
	case dataFour:
		return 4
	default: // dataZero
		return 0
	}
}

func (p *controlUnit) pcMux(sig signals, regs dpRegs, fe fieldOut) uint32 {
	if sig.selPCMux == pcFromMDR {
		return regs.mdr
	}
	if sig.selAddMux == addOffset {
		return p.pc + fe.value
	}
	return p.pc + 4
}

func (p *controlUnit) idxMux(sel uint8) uint8 {
	if sel == idxInc {
		return p.idx + 1
	}
	return 0
}

func posMux(sel uint8, fe fieldOut) uint8 {
	if sel == posNext {
		return fe.posNext
	}
	return 0
}
