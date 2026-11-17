package vm

// sel_alu_a
const (
	aluAOutA = iota
	aluAPC
	aluAMDR
)

// sel_alu_b
const (
	aluBOutB = iota
	aluBTMP
	aluBData
	aluBPS
	aluBIO
)

// sel_mar
const (
	marResult = iota
	marPC
	marMDR
	marMARDst
)

// sel_tmp
const (
	tmpMDR = iota
	tmpResult
)

// sel_ps_mux
const (
	psFlags = iota
	psResult
)

type psReg struct {
	ie bool
	flags
}

// packs PS into a 32-bit word
func (p psReg) word() uint32 {
	panic("TODO")
}

// unpacks a 32-bit word into PS
func psFromWord(w uint32) psReg {
	panic("TODO")
}

// outputs that leave the DataPath
type dpRegs struct {
	mar, mdr uint32
	ps       psReg
}

type dpOut struct {
	result  uint32 // the only write bus
	flags   flags
	divZero bool
}

type dataPath struct {
	regs                  [16]uint32
	mar, mdr, marDst, tmp uint32
	ps                    psReg
}

func (p *dataPath) registers() dpRegs {
	panic("TODO")
}

func (p *dataPath) outputs(sig signals, ioData uint32) dpOut {
	panic("TODO")
}

func (p *dataPath) clock(sig signals, out dpOut, mem memOut) {
	panic("TODO")
}
