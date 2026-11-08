package isa

import (
	"errors"
	"fmt"
)

var (
	ErrArgumentCount     = errors.New("wrong number of arguments")
	ErrFormatUnsupported = errors.New("format not supported")
)

const opcodeShift = 26
const mode1Shift = 23
const mode2Shift = 20

type Instruction struct {
	OpInfo    OpInfo // [31:26]
	Arguments []Argument
}

type EncodedPosition struct {
	word   int
	offset int
	width  int
}

func (p *Instruction) Encode() ([]uint32, []EncodedPosition, error) {
	switch p.OpInfo.Format {
	case F0:
		return p.encodeF0()
	case F1:
		return p.encodeF1()
	case F2:
		return p.encodeF2()
	default:
		return nil, nil, fmt.Errorf("%w: %d", ErrFormatUnsupported, p.OpInfo.Format)
	}
}

func (p *Instruction) encodeF0() ([]uint32, []EncodedPosition, error) {
	if len(p.Arguments) != 0 {
		return nil, nil, fmt.Errorf("%w: F0 expects 0, got %d", ErrArgumentCount, len(p.Arguments))
	}

	word := uint32(p.OpInfo.Opcode) << opcodeShift
	return []uint32{word}, nil, nil
}

func (p *Instruction) encodeF1() ([]uint32, []EncodedPosition, error) {
	if len(p.Arguments) != 1 {
		return nil, nil, fmt.Errorf("%w: F0 expects 0, got %d", ErrArgumentCount, len(p.Arguments))
	}

	argument := p.Arguments[0]
	mainWord := uint32(p.OpInfo.Opcode) << opcodeShift
	mainWord |= uint32(argument.Mode) << mode1Shift

	if argument.Mode == DirectMemory {
		return []uint32{mainWord, uint32(argument.Value)}, nil, nil
	} else if argument.Mode == RegisterRelative {

	}

	return []uint32{mainWord}, nil, nil
}

func (p *Instruction) encodeF2() ([]uint32, []EncodedPosition, error) {
	if len(p.Arguments) != 2 {
		return nil, nil, fmt.Errorf("%w: F0 expects 0, got %d", ErrArgumentCount, len(p.Arguments))
	}

	word := uint32(p.OpInfo.Opcode) << opcodeShift
	return []uint32{word}, nil, nil
}
