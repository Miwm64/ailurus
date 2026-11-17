package vm

import "github.com/Miwm64/ailurus/internal/isa"

const (
	half16      = 1 << 16
	relRegShift = 16 // register-relative: reg# in [19:16], offset in [15:0]
)

type fieldIn struct {
	ir, ext  uint32
	idx, pos uint8
}

type fieldOut struct {
	value   uint32 // sign-extended
	reg     uint8  // register number
	mode    isa.Mode
	fits    bool
	last    bool
	posNext uint8
}

// fieldExt is combinational. The operand layout follows from IR (format and
// modes), so the current operand is the first one that starts at (IDX, POS)
// or later. fits = false: it starts in a later word, the CU fetches the next
// word into EXT first. Not called past the last operand or for F0.
func fieldExt(in fieldIn) fieldOut {
	modes, positions, err := isa.Layout(in.ir)
	if err != nil {
		return fieldOut{} // bad opcode: the CU faults on it
	}

	for k, at := range positions {
		if at.Word < int(in.idx) || at.Word == int(in.idx) && at.Offset < int(in.pos) {
			continue
		}

		out := fieldOut{
			mode:    modes[k],
			fits:    at.Word == int(in.idx),
			last:    k == len(positions)-1,
			posNext: uint8(at.Offset + at.Width),
		}
		if !out.fits {
			return out
		}

		word := in.ir
		if in.idx > 0 {
			word = in.ext
		}
		bits := word >> at.Offset & uint32(uint64(1)<<at.Width-1)
		out.value, out.reg = decodeField(out.mode, bits)
		return out
	}
	return fieldOut{}
}

func decodeField(m isa.Mode, bits uint32) (uint32, uint8) {
	switch m {
	case isa.Register, isa.RegisterIndirect, isa.PostIncrement, isa.PreDecrement:
		return 0, uint8(bits)
	case isa.Immediate, isa.PCRelative:
		return signExtend16(bits), 0
	case isa.RegisterRelative:
		return signExtend16(bits % half16), uint8(bits >> relRegShift)
	default: // isa.DirectMemory
		return bits, 0
	}
}

func signExtend16(v uint32) uint32 {
	return uint32(int32(int16(uint16(v))))
}
