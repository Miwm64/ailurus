package isa

import (
	"errors"
	"fmt"
)

var ErrUnknownOpcode = errors.New("unknown opcode")

const (
	modeMask    = 1<<modeBits - 1
	countMask   = 0xF
	faCountMask = 0x7
)

func Layout(word uint32) ([]Mode, []EncodedPosition, error) {
	op := Opcode(word >> opcodeShift)
	name, ok := Mnemonics[op]
	if !ok {
		return nil, nil, fmt.Errorf("%w: 0x%02X", ErrUnknownOpcode, op)
	}

	var (
		modes []Mode
		area  int
	)
	switch Opcodes[name].Format {
	case F0:
		return nil, nil, nil
	case F1:
		modes, area = []Mode{mode(word, mode1Shift)}, f1AreaSize
	case F2:
		modes, area = []Mode{mode(word, mode1Shift), mode(word, mode2Shift)}, f2AreaSize
	case FB:
		modes, area = []Mode{PCRelative}, branchAreaSize
	case FJ:
		if word>>kShift&1 == 1 {
			modes = []Mode{PCRelative}
		} else {
			modes = []Mode{DirectMemory}
		}
		area = branchAreaSize
	case FP:
		modes, area = make([]Mode, int(word>>countShift&countMask)+1), pushmAreaSize // all Register
	case FA:
		modes, area = make([]Mode, int(word>>faCountShift&faCountMask)+1), faAreaSize
		for i := range modes {
			modes[i] = mode(word, modeBits*i)
		}
	}
	return modes, layout(area, modes), nil
}

func layout(area int, modes []Mode) []EncodedPosition {
	positions := make([]EncodedPosition, 0, len(modes))
	word, used, limit := 0, 0, area
	for _, m := range modes {
		width := m.Width()
		if used+width > limit {
			word, used, limit = word+1, 0, capacity
		}
		positions = append(positions, EncodedPosition{Word: word, Offset: used, Width: width})
		used += width
	}
	return positions
}

func mode(word uint32, shift int) Mode {
	return Mode(word >> shift & modeMask)
}
