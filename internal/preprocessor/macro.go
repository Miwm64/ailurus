package preprocessor

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
)

var (
	ErrWrongArgumentAmount = errors.New("macro argument count mismatch")
	ErrWrongArgumentNumber = errors.New("macro argument index out of range")
	ErrUnclosedMacro       = errors.New("unclosed macro definition")
	ErrExpectedBrace       = errors.New("expected { after macro header")
	ErrBadMacroHeader      = errors.New("invalid macro header, expected #macro name, count")
	ErrNestedMacro         = errors.New("macro definition inside a macro body")
	ErrMacroRedefined      = errors.New("macro already defined")
)

var macroArg = regexp.MustCompile("%[0-9]+")

type Macro struct {
	Name          string
	RawLines      []string
	ArgumentCount int
	initialized   bool
}

func NewMacro(name string, rawLines []string, argumentCount int) *Macro {
	return &Macro{
		Name:          name,
		RawLines:      rawLines,
		ArgumentCount: argumentCount,
	}
}

func (p *Macro) Apply(arguments []string) ([]string, error) {
	if len(arguments) != p.ArgumentCount {
		return nil, fmt.Errorf("%w: %s expects %d, got %d",
			ErrWrongArgumentAmount, p.Name, p.ArgumentCount, len(arguments))
	}

	lines := slices.Clone(p.RawLines)

	for i, line := range lines {
		var err error

		lines[i] = macroArg.ReplaceAllStringFunc(line, func(s string) string {
			num, convErr := strconv.Atoi(s[1:])
			if convErr != nil || num >= len(arguments) {
				if err == nil {
					err = fmt.Errorf("%w: %s in macro %s", ErrWrongArgumentNumber, s, p.Name)
				}

				return ""
			}

			return arguments[num]
		})

		if err != nil {
			return nil, err
		}
	}

	return lines, nil
}

func (p *Macro) AddRawLine(line string) {
	p.RawLines = append(p.RawLines, line)
}

func handleMacroHeader(tokens []token, macros map[string]*Macro) (*Macro, error) {
	brace := len(tokens) == 5 && tokens[4].kind == LBRACE

	if (len(tokens) != 4 && !brace) ||
		tokens[1].kind != NAME || tokens[2].kind != COMMA || tokens[3].kind != NUMBER {
		return nil, ErrBadMacroHeader
	}

	name := tokens[1].value
	if isReserved(name) {
		return nil, fmt.Errorf("%w: %s", ErrInvalidName, name)
	}

	if _, ok := macros[name]; ok {
		return nil, fmt.Errorf("%w: %s", ErrMacroRedefined, name)
	}

	count, err := strconv.ParseUint(tokens[3].value, 0, 8)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrBadMacroHeader, tokens[3].value)
	}

	m := NewMacro(name, nil, int(count))
	m.initialized = brace

	return m, nil
}
