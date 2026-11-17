package vm

import (
	"fmt"

	"github.com/Miwm64/ailurus/internal/isa"
)

const irOpcodeShift = 26 // IR[31:26]

type cuStep uint8

const (
	// reset: PC ← mem[0x0]
	stReset0 cuStep = iota // MDR ← mem[MAR]; reset leaves MAR = 0
	stReset1               // PC ← MDR

	// interrupt entry: push PC, push PS, IE ← 0, PC ← mem[0x4]
	stIRQ0 // rsp, MAR ← rsp − 4; irq_pending ← 0
	stIRQ1 // mem[MAR] ← PC
	stIRQ2 // rsp, MAR ← rsp − 4
	stIRQ3 // mem[MAR] ← PS
	stIRQ4 // MAR ← 4
	stIRQ5 // MDR ← mem[MAR], IE ← 0
	stIRQ6 // PC ← MDR

	// instruction fetch: IR ← mem[PC], PC ← PC + 4
	stFetch0 // MAR ← PC
	stFetch1 // MDR ← mem[MAR]
	stFetch2 // IR ← MDR, PC ← PC + 4, IDX ← 0, POS ← 0

	// operand fetch: destination, then sources
	stOFDst0
	stOFDst1
	stOFDst2
	stOFDst3
	stOFDstExt // MDR ← mem[MAR], PC ← PC + 4, IDX ← IDX + 1, POS ← 0
	stOFSrc0
	stOFSrc1
	stOFSrc2
	stOFSrc3
	stOFSrcExt
	stOFSrcEnd  // TMP ← source, MAR ← MAR_DST
	stOFDstRead // MDR ← mem[MAR]

	// execute, writeback
	stExec0 // first execute step, by opcode
	stExec1
	stExec2
	stExec3
	stExec4
	stExec5
	stWB0
)

var stepNames = [...]string{
	stReset0:    "reset0",
	stReset1:    "reset1",
	stIRQ0:      "irq0",
	stIRQ1:      "irq1",
	stIRQ2:      "irq2",
	stIRQ3:      "irq3",
	stIRQ4:      "irq4",
	stIRQ5:      "irq5",
	stIRQ6:      "irq6",
	stFetch0:    "fetch0",
	stFetch1:    "fetch1",
	stFetch2:    "fetch2",
	stOFDst0:    "of_dst0",
	stOFDst1:    "of_dst1",
	stOFDst2:    "of_dst2",
	stOFDst3:    "of_dst3",
	stOFDstExt:  "of_dst_ext",
	stOFSrc0:    "of_src0",
	stOFSrc1:    "of_src1",
	stOFSrc2:    "of_src2",
	stOFSrc3:    "of_src3",
	stOFSrcExt:  "of_src_ext",
	stOFSrcEnd:  "of_src_end",
	stOFDstRead: "of_dst_rd",
	stExec0:     "exec0",
	stExec1:     "exec1",
	stExec2:     "exec2",
	stExec3:     "exec3",
	stExec4:     "exec4",
	stExec5:     "exec5",
	stWB0:       "wb0",
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

// sel_reg_a_mux, sel_reg_w_mux
const (
	regField = iota // FIELD_EXT reg
	regDst          // DST_REG
	regRSP          // constant 0 = rsp
)

type signals struct {
	// register file
	regA, regB uint8 // REG_A_MUX, FIELD_EXT reg
	regW       uint8 // REG_W_MUX
	latchReg   bool

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
	latchIR     bool
	latchPC     bool
	latchPos    bool
	latchIdx    bool
	latchDstReg bool
	latchExt    bool
	clearIRQ    bool
	selAddMux   uint8
	selPCMux    uint8
	selPos      uint8
	selIdx      uint8
	selDataMux  uint8
	selRegA     uint8
	selRegW     uint8

	// to Datapath
	data uint32 // data_mux output
	pc   uint32
}

type controlUnit struct {
	pc, ir     uint32
	step       cuStep
	idx, pos   uint8
	dstReg     uint8
	ext        uint32 // EXT: FIELD_EXT's input for the words after IR
	irqPending bool
	halted     bool
	fault      error
}

func (p *controlUnit) outputs(regs dpRegs) signals {
	sig := signals{pc: p.pc}
	fe := p.field()

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
	case stIRQ0, stIRQ2:
		sig = decRSP(sig)
		sig.clearIRQ = p.step == stIRQ0
	case stIRQ1:
		sig.selALUA, sig.aluOp, sig.memOp = aluAPC, aluPassA, memWrite
	case stIRQ3:
		sig.selALUB, sig.aluOp, sig.memOp = aluBPS, aluPassB, memWrite
	case stIRQ4:
		sig.selALUB, sig.selDataMux, sig.aluOp = aluBData, dataFour, aluPassB
		sig.selMAR, sig.latchMAR = marResult, true
	case stIRQ5:
		sig.memOp, sig.latchMDR = memRead, true
		sig.selALUB, sig.selDataMux, sig.aluOp = aluBData, dataZero, aluPassB
		sig.selPS, sig.latchPS = psResult, maskIE
	case stIRQ6:
		sig.selPCMux, sig.latchPC = pcFromMDR, true
	case stOFDstExt, stOFSrcExt:
		sig.memOp, sig.latchMDR, sig.latchExt = memRead, true, true
		sig.selAddMux, sig.selPCMux, sig.latchPC = addFour, pcFromAdd, true
		sig.selIdx, sig.latchIdx = idxInc, true
		sig.selPos, sig.latchPos = posZero, true
	default: // operand fetch, execute, writeback
		sig = p.execute(sig, fe, regs)
	}

	sig.regA, sig.regB, sig.regW = p.regMux(sig.selRegA, fe), fe.reg, p.regMux(sig.selRegW, fe)
	sig.data = dataMux(sig.selDataMux, fe)
	return sig
}

func (p *controlUnit) clock(sig signals, regs dpRegs, out dpOut, mem memOut, in ioOut, irqReq bool) {
	pending := p.irqPending
	if sig.clearIRQ {
		p.irqPending = false
	}
	if irqReq {
		p.irqPending = true
	}

	fe := p.field()
	switch {
	case p.halted:
		return
	case mem.fault != nil:
		p.stop(mem.fault)
		return
	case in.fault != nil:
		p.stop(in.fault)
		return
	case out.divZero:
		p.stop(fmt.Errorf("%w: %s", ErrDivZero, isa.Mnemonics[p.opcode()]))
		return
	case sig.memOp != memNone && !mem.ready:
		return // waiting for memory
	case p.step == stOFDst0:
		if err := p.decode(fe); err != nil {
			p.stop(err)
			return
		}
	case p.step == stExec0 && p.opcode() == opHalt:
		p.halted = true
		return
	}

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
	if sig.latchDstReg {
		p.dstReg = fe.reg
	}
	if sig.latchExt {
		p.ext = mem.data
	}
	p.step = p.next(fe)
	if p.step == stFetch0 && pending && regs.ps.ie { // interrupt check: after an instruction
		p.step = stIRQ0
	}
}

// next_step
func (p *controlUnit) next(fe fieldOut) cuStep {
	switch p.step {
	case stReset0:
		return stReset1
	case stFetch0:
		return stFetch1
	case stFetch1:
		return stFetch2
	case stFetch2:
		return stOFDst0
	case stIRQ0, stIRQ1, stIRQ2, stIRQ3, stIRQ4, stIRQ5:
		return p.step + 1
	case stReset1, stIRQ6:
		return stFetch0
	default: // operand fetch, execute, writeback
		return p.nextExec(fe)
	}
}

func (p *controlUnit) opcode() isa.Opcode {
	return isa.Opcode(p.ir >> irOpcodeShift)
}

func (p *controlUnit) field() fieldOut {
	return fieldExt(fieldIn{ir: p.ir, ext: p.ext, idx: p.idx, pos: p.pos})
}

func (p *controlUnit) stop(err error) {
	p.halted, p.fault = true, err
}

func dataMux(sel uint8, fe fieldOut) uint32 {
	switch sel {
	case dataImm:
		return fe.value
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

func (p *controlUnit) regMux(sel uint8, fe fieldOut) uint8 {
	switch sel {
	case regField:
		return fe.reg
	case regDst:
		return p.dstReg
	default: // regRSP
		return 0
	}
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
