package vm

type cuStep uint8

const (
	// reset: PC ← mem[0x0]
	stReset0 cuStep = iota

	// interrupt entry: push PC, push PS, IE ← 0, PC ← mem[0x4]
	stIRQ0

	// instruction fetch: IR ← mem[PC], PC ← PC + 4
	stFetch0

	// operand fetch: destination, then sources
	stOFDst0
	stOFSrc0

	// execute, writeback
	stExec0
	stWB0
)

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
	memOp      memOp
	ioRd, ioWr bool
	port       uint16

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
	panic("TODO")
}

func (p *controlUnit) clock(sig signals, regs dpRegs, comb dpOut, mem memOut, in ioOut, irqReq bool) {
	panic("TODO")
}
