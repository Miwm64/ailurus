package isa

import (
	"errors"
	"fmt"
)

var (
	ErrArgumentCount        = errors.New("wrong number of arguments")
	ErrFormatUnsupported    = errors.New("format not supported")
	ErrImmediateDestination = errors.New("immediate value is not allowed as destination")
	ErrImmediateOperand     = errors.New("immediate not allowed as first operand")
	ErrPortMode             = errors.New("port must be an immediate")
	ErrBranchMode           = errors.New("branch target must be PC-relative")
	ErrJumpMode             = errors.New("jump target must be PC-relative or absolute")
)

const (
	opcodeShift    = 26
	capacity       = 32
	mode1Shift     = 23
	mode2Shift     = 20
	f1AreaSize     = 23
	f2AreaSize     = 20
	kShift         = 25
	branchAreaSize = 16
)

var (
	opIn   = Opcodes["in"].Opcode
	opOut  = Opcodes["out"].Opcode
	opCmp  = Opcodes["cmp"].Opcode
	opTest = Opcodes["test"].Opcode
)

type Instruction struct {
	OpInfo    OpInfo // [31:26]
	Arguments []Argument
}

type EncodedPosition struct {
	Word   int
	Offset int
	Width  int
}

func (p *Instruction) Encode() ([]uint32, []EncodedPosition, error) {
	switch p.OpInfo.Format {
	case F0:
		return p.encodeF0()
	case F1:
		return p.encodeF1()
	case F2:
		return p.encodeF2()
	case FB:
		return p.encodeFB()
	case FJ:
		return p.encodeFJ()
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

func nextFitPackArguments(arguments []Argument, mainWord uint32, mainWordSize int) ([]uint32, []EncodedPosition, error) {
	words := []uint32{mainWord}
	positions := make([]EncodedPosition, 0, len(arguments))

	current := 0
	used := 0
	limit := mainWordSize

	for i, a := range arguments {
		bits, err := a.Bits()
		if err != nil {
			return nil, nil, fmt.Errorf("argument %d: %w", i+1, err)
		}
		width := a.Mode.Width()

		if used+width > limit {
			words = append(words, 0)
			current = len(words) - 1
			used = 0
			limit = capacity
		}

		words[current] |= bits << used
		positions = append(positions, EncodedPosition{
			Word:   current,
			Offset: used,
			Width:  width,
		})
		used += width
	}

	return words, positions, nil
}

func (p *Instruction) encodeF1() ([]uint32, []EncodedPosition, error) {
	if len(p.Arguments) != 1 {
		return nil, nil, fmt.Errorf("%w: F1 expects 1, got %d", ErrArgumentCount, len(p.Arguments))
	}

	argument := p.Arguments[0]
	if argument.Mode == Immediate {
		return nil, nil, fmt.Errorf("%w: got %d", ErrImmediateDestination, argument.Value)
	}

	mainWord := uint32(p.OpInfo.Opcode) << opcodeShift
	mainWord |= uint32(argument.Mode) << mode1Shift

	return nextFitPackArguments(p.Arguments, mainWord, f1AreaSize)
}

func (p *Instruction) encodeF2() ([]uint32, []EncodedPosition, error) {
	if len(p.Arguments) != 2 {
		return nil, nil, fmt.Errorf("%w: F2 expects 2, got %d", ErrArgumentCount, len(p.Arguments))
	}
	if err := p.checkF2Modes(); err != nil {
		return nil, nil, err
	}

	mainWord := uint32(p.OpInfo.Opcode) << opcodeShift
	mainWord |= uint32(p.Arguments[0].Mode) << mode1Shift
	mainWord |= uint32(p.Arguments[1].Mode) << mode2Shift

	return nextFitPackArguments(p.Arguments, mainWord, f2AreaSize)
}

func (p *Instruction) checkF2Modes() error {
	first, second := p.Arguments[0], p.Arguments[1]

	switch p.OpInfo.Opcode {
	case opOut:
		if first.Mode != Immediate {
			return fmt.Errorf("%w: out, got mode %d", ErrPortMode, first.Mode)
		}
	case opIn:
		if second.Mode != Immediate {
			return fmt.Errorf("%w: in, got mode %d", ErrPortMode, second.Mode)
		}
		if first.Mode == Immediate {
			return fmt.Errorf("%w: got %d", ErrImmediateDestination, first.Value)
		}
	case opCmp, opTest:
		if first.Mode == Immediate {
			return fmt.Errorf("%w: got %d", ErrImmediateOperand, first.Value)
		}
	default:
		if first.Mode == Immediate {
			return fmt.Errorf("%w: got %d", ErrImmediateDestination, first.Value)
		}
	}
	return nil
}

func (p *Instruction) encodeFB() ([]uint32, []EncodedPosition, error) {
	if len(p.Arguments) != 1 {
		return nil, nil, fmt.Errorf("%w: FB expects 1, got %d", ErrArgumentCount, len(p.Arguments))
	}
	if mode := p.Arguments[0].Mode; mode != PCRelative {
		return nil, nil, fmt.Errorf("%w: got mode %d", ErrBranchMode, mode)
	}

	mainWord := uint32(p.OpInfo.Opcode) << opcodeShift
	return nextFitPackArguments(p.Arguments, mainWord, branchAreaSize)
}

func (p *Instruction) encodeFJ() ([]uint32, []EncodedPosition, error) {
	if len(p.Arguments) != 1 {
		return nil, nil, fmt.Errorf("%w: FJ expects 1, got %d", ErrArgumentCount, len(p.Arguments))
	}

	mainWord := uint32(p.OpInfo.Opcode) << opcodeShift
	switch mode := p.Arguments[0].Mode; mode {
	case PCRelative:
		mainWord |= 1 << kShift // K = 1
	case DirectMemory:
		// K = 0
	default:
		return nil, nil, fmt.Errorf("%w: got mode %d", ErrJumpMode, mode)
	}

	return nextFitPackArguments(p.Arguments, mainWord, branchAreaSize)
}
