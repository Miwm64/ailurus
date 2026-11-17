package vm

import "fmt"

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

// PS word layout
const (
	psIE = 1 << 2
	psC  = 1 << 4
	psV  = 1 << 5
	psZ  = 1 << 6
	psN  = 1 << 7
)

type psReg struct {
	ie bool
	flags
}

// packs PS into a 32-bit word
func (p psReg) word() uint32 {
	var w uint32
	if p.ie {
		w |= psIE
	}
	if p.c {
		w |= psC
	}
	if p.v {
		w |= psV
	}
	if p.z {
		w |= psZ
	}
	if p.n {
		w |= psN
	}
	return w
}

// unpacks a 32-bit word into PS
func psFromWord(w uint32) psReg {
	return psReg{
		ie: w&psIE != 0,
		n:  w&psN != 0, z: w&psZ != 0, v: w&psV != 0, c: w&psC != 0,
	}
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
	return dpRegs{mar: p.mar, mdr: p.mdr, ps: p.ps}
}

func (p *dataPath) outputs(sig signals, ioData uint32) dpOut {
	panic("TODO")
}

func (p *dataPath) clock(sig signals, out dpOut, mem memOut) {
	panic("TODO")
}

func (p *dataPath) marMux(sig signals, result uint32) uint32 {
	switch sig.selMAR {
	case marResult:
		return result
	case marPC:
		return sig.pc
	case marMDR:
		return p.mdr
	case marMARDst:
		return p.marDst
	default:
		panic(fmt.Sprintf("datapath: bad sel_mar %d", sig.selMAR))
	}
}

func (p *dataPath) tmpMux(sel uint8, result uint32) uint32 {
	switch sel {
	case tmpMDR:
		return p.mdr
	case tmpResult:
		return result
	default:
		panic(fmt.Sprintf("datapath: bad sel_tmp %d", sel))
	}
}

func (p *dataPath) psMux(sel uint8, out dpOut) psReg {
	switch sel {
	case psFlags:
		return psReg{ie: p.ps.ie, flags: out.flags}
	case psResult:
		return psFromWord(out.result)
	default:
		panic(fmt.Sprintf("datapath: bad sel_ps_mux %d", sel))
	}
}

func latchPS(ps, src psReg, mask flagMask) psReg {
	if mask&maskN != 0 {
		ps.n = src.n
	}
	if mask&maskZ != 0 {
		ps.z = src.z
	}
	if mask&maskV != 0 {
		ps.v = src.v
	}
	if mask&maskC != 0 {
		ps.c = src.c
	}
	if mask&maskIE != 0 {
		ps.ie = src.ie
	}
	return ps
}
