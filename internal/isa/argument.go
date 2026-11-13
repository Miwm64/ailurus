package isa

import (
	"errors"
	"fmt"
)

type Mode uint8

const (
	Register Mode = iota
	DirectMemory
	PCRelative
	RegisterIndirect
	PostIncrement
	PreDecrement
	Immediate
	RegisterRelative
)

type Argument struct {
	Mode   Mode
	Value  int
	Offset int
}

var modeWidths = [...]int{
	Register:         4,
	DirectMemory:     32,
	PCRelative:       16,
	RegisterIndirect: 4,
	PostIncrement:    4,
	PreDecrement:     4,
	Immediate:        16,
	RegisterRelative: 20,
}

func (m Mode) Width() int {
	return modeWidths[m]
}

var ErrValueRange = errors.New("value out of range")

const (
	minSigned16 = -1 << 15  // -32768
	maxSigned16 = 1<<15 - 1 // 32767
	maxRegister = 15
	maxAddress  = 1<<32 - 1
)

// returns bits ready to be shifted into instruction
func (a Argument) Bits() (uint32, error) {
	switch a.Mode {
	case Register, RegisterIndirect, PostIncrement, PreDecrement:
		if err := checkRange("register", a.Value, 0, maxRegister); err != nil {
			return 0, err
		}
		return uint32(a.Value), nil

	case DirectMemory:
		if err := checkRange("address", a.Value, 0, maxAddress); err != nil {
			return 0, err
		}
		return uint32(a.Value), nil

	case PCRelative, Immediate:
		if err := checkRange("16-bit value", a.Value, minSigned16, maxSigned16); err != nil {
			return 0, err
		}
		return uint32(a.Value) & 0xFFFF, nil

	case RegisterRelative:
		if err := checkRange("register", a.Value, 0, maxRegister); err != nil {
			return 0, err
		}
		if err := checkRange("offset", a.Offset, minSigned16, maxSigned16); err != nil {
			return 0, err
		}
		return uint32(a.Value)<<16 | uint32(a.Offset)&0xFFFF, nil

	default:
		panic(fmt.Sprintf("isa: invalid mode %d", a.Mode))
	}
}

func checkRange(what string, v, lo, hi int) error {
	if v < lo || v > hi {
		return fmt.Errorf("%w: %s %d not in [%d, %d]", ErrValueRange, what, v, lo, hi)
	}
	return nil
}
