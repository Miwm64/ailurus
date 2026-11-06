package assembler

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/Miwm64/ailurus/internal/tokenizer"
)

var (
	ErrArgumentCount    = errors.New("wrong number of arguments")
	ErrArgumentKind     = errors.New("wrong argument kind")
	ErrUnknownSection   = errors.New("unknown section")
	ErrUnknownDirective = errors.New("unknown assembler directive")
	ErrInvalidNumber    = errors.New("invalid number")
	ErrNumberRange      = errors.New("number out of range")
	ErrSymbolConflict   = errors.New("symbol conflict")
	ErrReservedSymbol   = errors.New("reserved symbol")
	ErrMultipleLabels   = errors.New("multiple labels for one address")
)

const (
	minNumber = math.MinInt32  // -2^31
	maxNumber = math.MaxUint32 // 2^32 - 1
)

type assembler struct {
	lineMap    map[int]int
	lines      []string
	objectFile *ObjectFile

	symbolIndex map[string]int // name -> index in objectFile.Symbols

	dataCounter    int
	textCounter    int
	currentSection Section
	currentLabel   string
}

func newAssembler(source string, file string, lineMap map[int]int) *assembler {
	return &assembler{
		lineMap:        lineMap,
		lines:          strings.Split(file, "\n"),
		objectFile:     newObjectFile(source),
		symbolIndex:    make(map[string]int),
		currentSection: Text,
	}
}

func (p *assembler) assemble() (*ObjectFile, error) {
	for i, line := range p.lines {
		line = strings.TrimSpace(line)
		err := p.handleLine(line)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", p.lineMap[i], err)
		}
	}

	if p.currentLabel != "" {
		return nil, fmt.Errorf("%w: %s", ErrArgumentKind, p.currentLabel)
	}

	return p.objectFile, nil
}

func (p *assembler) handleLine(line string) error {
	tokens, err := tokenizer.Tokenize(line)
	if err != nil {
		return err
	}

	switch tokens[0].Kind {
	case tokenizer.ASSEMBLER_DIRECTIVE:
		err := p.handleDirective(tokens)
		if err != nil {
			return err
		}
	default:
		err := p.handleNormalLine(tokens)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *assembler) handleNormalLine(tokens []tokenizer.Token) error {
	tokenBase := 0
	if len(tokens) >= 2 && tokens[0].Kind == tokenizer.NAME && tokens[1].Kind == tokenizer.COLON {
		tokenBase = 2
		err := p.handleLabel(tokens[0].Value)
		if err != nil {
			return err
		}
	}
	if len(tokens) == tokenBase {
		return nil
	}
	tokens = tokens[tokenBase:]

	return nil
}

func (p *assembler) handleLabel(labelName string) error {
	if tokenizer.IsReserved(labelName) {
		return fmt.Errorf("%w: %s cannot be used as a label", ErrReservedSymbol, labelName)
	}

	if p.currentLabel != "" {
		return fmt.Errorf("%w: %s defined while %s has no content yet", ErrMultipleLabels, labelName, p.currentLabel)
	}

	if i, ok := p.symbolIndex[labelName]; ok {
		switch p.objectFile.Symbols[i].Binding {
		case Import:
			return fmt.Errorf("%w: %s is imported, cannot be defined", ErrSymbolConflict, labelName)
		case Local:
			return fmt.Errorf("%w: %s is already defined", ErrSymbolConflict, labelName)
		}

		if p.objectFile.Symbols[i].Section != "" {
			return fmt.Errorf("%w: %s is already defined", ErrSymbolConflict, labelName)
		}

		p.currentLabel = labelName
		return nil
	}

	p.currentLabel = labelName
	p.symbolIndex[labelName] = len(p.objectFile.Symbols)
	p.objectFile.Symbols = append(p.objectFile.Symbols, Symbol{
		Name:    labelName,
		Binding: Local,
	})
	return nil
}

func (p *assembler) handleDirective(tokens []tokenizer.Token) error {
	directive := tokens[0]
	if len(tokens) != 2 {
		return fmt.Errorf("%w: %s expects 1 argument, got %d", ErrArgumentCount, directive.Value, len(tokens)-1)
	}
	argument := tokens[1]

	switch directive.Value {
	case ".section":
		if argument.Kind != tokenizer.NAME {
			return fmt.Errorf("%w: %s expects name, got %s", ErrArgumentKind, directive.Value, argument.Kind)
		}

		if argument.Value != "data" && argument.Value != "text" {
			return fmt.Errorf("%w: expected 'data' or 'text', got %s", ErrUnknownSection, argument.Value)
		}
		p.currentSection = Section(argument.Value)
	case ".org":
		if argument.Kind != tokenizer.NUMBER {
			return fmt.Errorf("%w: %s expects number, got %s", ErrArgumentKind, directive.Value, argument.Kind)
		}

		val, err := parseNumber(argument.Value)
		if err != nil {
			return fmt.Errorf("%s: %w", directive.Value, err)
		}

		if p.currentSection == Text {
			p.textCounter = int(val)
		} else {
			p.dataCounter = int(val)
		}

	case ".import":
		if argument.Kind != tokenizer.NAME {
			return fmt.Errorf("%w: %s expects name, got %s", ErrArgumentKind, directive.Value, argument.Kind)
		}
		if argument.Value == "_start" || argument.Value == "_handler" {
			return fmt.Errorf("%w: %s cannot be imported", ErrReservedSymbol, argument.Value)
		}

		if i, ok := p.symbolIndex[argument.Value]; ok {
			if p.objectFile.Symbols[i].Binding != Import {
				return fmt.Errorf("%w: %s is defined in this file, cannot be imported", ErrSymbolConflict, argument.Value)
			}
			return nil // repeated import: allowed
		}

		p.symbolIndex[argument.Value] = len(p.objectFile.Symbols)
		p.objectFile.Symbols = append(p.objectFile.Symbols, Symbol{
			Name:    argument.Value,
			Binding: Import,
		})
	case ".export":
	default:
		return fmt.Errorf("%w: %q", ErrUnknownDirective, directive.Value)
	}
	return nil
}

func parseNumber(s string) (int64, error) {
	digits, neg := strings.CutPrefix(s, "-")

	base := 10
	if hex, ok := strings.CutPrefix(digits, "0x"); ok {
		digits, base = hex, 16
	}

	val, err := strconv.ParseInt(digits, base, 64)
	if err != nil {
		return 0, fmt.Errorf("%w %q: %w", ErrInvalidNumber, s, err)
	}
	if neg {
		val = -val
	}

	if val < minNumber || val > maxNumber {
		return 0, fmt.Errorf("%w: %q not in [%d, %d]", ErrNumberRange, s, minNumber, maxNumber)
	}
	return val, nil
}
