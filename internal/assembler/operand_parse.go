package assembler

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Miwm64/ailurus/internal/isa"
	"github.com/Miwm64/ailurus/internal/tokenizer"
)

var ErrInvalidOperand = errors.New("invalid operand")

type operand struct {
	arg    isa.Argument
	label  string
	bareAt bool
}

func parseOperand(tokens []tokenizer.Token) (operand, error) {
	if len(tokens) == 0 {
		return operand{}, fmt.Errorf("%w: empty", ErrInvalidOperand)
	}

	// -(rX)
	if match(tokens, tokenizer.MINUS, tokenizer.LPAREN, tokenizer.NAME, tokenizer.RPAREN) {
		if reg, ok := parseRegister(tokens[2].Value); ok {
			return operand{arg: isa.Argument{Mode: isa.PreDecrement, Value: reg}}, nil
		}
	}

	// @name
	if match(tokens, tokenizer.AT, tokenizer.NAME) {
		return labelOperand(isa.PCRelative, tokens[1].Value, true)
	}

	// rX or name
	if match(tokens, tokenizer.NAME) {
		if reg, ok := parseRegister(tokens[0].Value); ok {
			return operand{arg: isa.Argument{Mode: isa.Register, Value: reg}}, nil
		}
		return labelOperand(isa.Immediate, tokens[0].Value, false)
	}

	// number, -number, "c"
	switch tokens[0].Kind {
	case tokenizer.NUMBER, tokenizer.MINUS, tokenizer.STRING:
		val, used, err := parseValue(tokens)
		if err != nil {
			return operand{}, err
		}
		if used != len(tokens) {
			return operand{}, fmt.Errorf("%w: unexpected %q after value", ErrInvalidOperand, tokens[used].Value)
		}
		return operand{arg: isa.Argument{Mode: isa.Immediate, Value: int(val)}}, nil
	}

	if tokens[0].Kind == tokenizer.LPAREN {
		return parseMemoryOperand(tokens)
	}

	return operand{}, fmt.Errorf("%w: unexpected %s %q", ErrInvalidOperand, tokens[0].Kind, tokens[0].Value)
}

// starts with "("
func parseMemoryOperand(tokens []tokenizer.Token) (operand, error) {
	switch {
	// (@name)
	case match(tokens, tokenizer.LPAREN, tokenizer.AT, tokenizer.NAME, tokenizer.RPAREN):
		return labelOperand(isa.PCRelative, tokens[2].Value, false)

	// (number)
	case match(tokens, tokenizer.LPAREN, tokenizer.NUMBER, tokenizer.RPAREN):
		val, err := parseNumber(tokens[1].Value)
		if err != nil {
			return operand{}, err
		}
		return operand{arg: isa.Argument{Mode: isa.DirectMemory, Value: int(val)}}, nil

	// (rX), (name)
	case match(tokens, tokenizer.LPAREN, tokenizer.NAME, tokenizer.RPAREN):
		if reg, ok := parseRegister(tokens[1].Value); ok {
			return operand{arg: isa.Argument{Mode: isa.RegisterIndirect, Value: reg}}, nil
		}
		return labelOperand(isa.DirectMemory, tokens[1].Value, false)

	// (rX)+
	case match(tokens, tokenizer.LPAREN, tokenizer.NAME, tokenizer.RPAREN, tokenizer.PLUS):
		if reg, ok := parseRegister(tokens[1].Value); ok {
			return operand{arg: isa.Argument{Mode: isa.PostIncrement, Value: reg}}, nil
		}

	// (rX+n), (rX-n)
	case match(tokens, tokenizer.LPAREN, tokenizer.NAME, tokenizer.PLUS, tokenizer.NUMBER, tokenizer.RPAREN),
		match(tokens, tokenizer.LPAREN, tokenizer.NAME, tokenizer.MINUS, tokenizer.NUMBER, tokenizer.RPAREN):
		reg, ok := parseRegister(tokens[1].Value)
		if !ok {
			break
		}
		off, err := parseNumber(tokens[3].Value)
		if err != nil {
			return operand{}, err
		}
		if tokens[2].Kind == tokenizer.MINUS {
			off = -off
		}
		return operand{arg: isa.Argument{Mode: isa.RegisterRelative, Value: reg, Offset: int(off)}}, nil
	}

	return operand{}, fmt.Errorf("%w: %s", ErrInvalidOperand, joinTokens(tokens))
}

func labelOperand(mode isa.Mode, name string, bareAt bool) (operand, error) {
	if tokenizer.IsReserved(name) {
		return operand{}, fmt.Errorf("%w: %s cannot be used as a label", ErrReservedSymbol, name)
	}
	return operand{arg: isa.Argument{Mode: mode}, label: name, bareAt: bareAt}, nil
}

// maps rsp and r1-r15
func parseRegister(name string) (int, bool) {
	if name == "rsp" {
		return 0, true
	}
	digits, ok := strings.CutPrefix(name, "r")
	if !ok {
		return 0, false
	}
	n, err := strconv.Atoi(digits)
	if err != nil || n < 1 || n > 15 || digits != strconv.Itoa(n) {
		return 0, false
	}
	return n, true
}

func match(tokens []tokenizer.Token, kinds ...tokenizer.Kind) bool {
	if len(tokens) != len(kinds) {
		return false
	}
	for i, k := range kinds {
		if tokens[i].Kind != k {
			return false
		}
	}
	return true
}

func joinTokens(tokens []tokenizer.Token) string {
	parts := make([]string, len(tokens))
	for i, t := range tokens {
		parts[i] = t.Value
	}
	return strings.Join(parts, " ")
}
