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
	ErrWrongArgumentNumber = errors.New("macro source wrong argument number")
)

var macroArg = regexp.MustCompile("%[0-9]+")

type Macro struct {
	Name          string
	RawLines      []string
	ArgumentCount int
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
		return nil, ErrWrongArgumentAmount
	}

	lines := slices.Clone(p.RawLines)

	for i, line := range lines {
		var err error
		lines[i] = macroArg.ReplaceAllStringFunc(line, func(s string) string {
			var num int
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
