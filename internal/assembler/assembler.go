package assembler

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/Miwm64/ailurus/internal/tokenizer"
)

var (
	ErrArgumentCount        = errors.New("wrong number of arguments")
	ErrArgumentKind         = errors.New("wrong argument kind")
	ErrUnknownSection       = errors.New("unknown section")
	ErrUnknownDirective     = errors.New("unknown assembler directive")
	ErrInvalidNumber        = errors.New("invalid number")
	ErrNumberRange          = errors.New("number out of range")
	ErrSymbolConflict       = errors.New("symbol conflict")
	ErrReservedSymbol       = errors.New("reserved symbol")
	ErrMultipleLabels       = errors.New("multiple labels for one address")
	ErrPendingLabel         = errors.New("label without content")
	ErrLabelBeforeDirective = errors.New("label before directive")
	ErrUnexpectedToken      = errors.New("unexpected token")
	ErrUndefinedExport      = errors.New("exported symbol not defined")
	ErrWrongSection         = errors.New("wrong section")
	ErrUndefinedSymbol      = errors.New("undefined symbol")
)

const (
	minNumber = math.MinInt32  // -2^31
	maxNumber = math.MaxUint32 // 2^32 - 1
)

type symbolUse struct {
	name      string
	lineIndex int
}

type assembler struct {
	lineMap    map[int]int
	lines      []string
	objectFile *ObjectFile

	symbolIndex map[string]int // name -> index in objectFile.Symbols

	dataCounter    int
	textCounter    int
	currentSection Section
	currentLabel   string

	lineIndex int
	labelLine int

	uses []symbolUse
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
		p.lineIndex = i
		line = strings.TrimSpace(line)
		err := p.handleLine(line)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", p.lineMap[i], err)
		}
	}

	if p.currentLabel != "" {
		return nil, fmt.Errorf("line %d: %w: %s is not followed by data or an instruction",
			p.lineMap[p.labelLine], ErrPendingLabel, p.currentLabel)
	}

	var undefined []string
	for _, s := range p.objectFile.Symbols {
		if s.Binding == Export && s.Section == "" {
			undefined = append(undefined, s.Name)
		}
	}

	for _, u := range p.uses {
		if _, ok := p.symbolIndex[u.name]; !ok {
			return nil, fmt.Errorf("line %d: %w: %s is neither defined nor imported",
				p.lineMap[u.lineIndex], ErrUndefinedSymbol, u.name)
		}
	}

	if len(undefined) > 0 {
		return nil, fmt.Errorf("%w: %s", ErrUndefinedExport, strings.Join(undefined, ", "))
	}

	return p.objectFile, nil
}

func (p *assembler) handleLine(line string) error {
	tokens, err := tokenizer.Tokenize(line)
	if err != nil {
		return err
	}

	// label
	hasLabel := false
	if len(tokens) >= 2 && tokens[0].Kind == tokenizer.NAME && tokens[1].Kind == tokenizer.COLON {
		err := p.handleLabel(tokens[0].Value)
		if err != nil {
			return err
		}
		tokens = tokens[2:]
		hasLabel = true
	}

	if len(tokens) >= 2 && tokens[0].Kind == tokenizer.NAME && tokens[1].Kind == tokenizer.COLON {
		return fmt.Errorf("%w: %s defined on the same line as %s", ErrMultipleLabels, tokens[0].Value, p.currentLabel)
	}
	if len(tokens) == 0 {
		return nil
	}

	// main part
	switch tokens[0].Kind {
	case tokenizer.ASSEMBLER_DIRECTIVE:
		switch tokens[0].Value {
		case ".word", ".pstr", ".ref":
			err := p.handleData(tokens)
			if err != nil {
				return err
			}
		default:
			if hasLabel {
				return fmt.Errorf("%w: %s cannot follow label %s", ErrLabelBeforeDirective, tokens[0].Value, p.currentLabel)
			}
			err := p.handleDirective(tokens)
			if err != nil {
				return err
			}
		}
	case tokenizer.NAME:
		err := p.handleInstruction(tokens)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("%w: %s %q at start of line", ErrUnexpectedToken, tokens[0].Kind, tokens[0].Value)
	}

	return nil
}

func (p *assembler) handleData(tokens []tokenizer.Token) error {
	directive := tokens[0]
	dataSize := 0

	if p.currentSection != Data {
		return fmt.Errorf("%w: %s is only allowed in %s, current section is %s",
			ErrWrongSection, directive.Value, Data, p.currentSection)
	}

	switch directive.Value {
	case ".pstr":
		if len(tokens) != 2 {
			return fmt.Errorf("%w: %s expects 1 argument, got %d", ErrArgumentCount, directive.Value, len(tokens)-1)
		}
		if tokens[1].Kind != tokenizer.STRING {
			return fmt.Errorf("%w: %s expects %s, got %s", ErrArgumentKind, directive.Value, tokenizer.STRING, tokens[1].Kind)
		}

		chars := stringToBytes(tokens[1].Value)
		res := binary.LittleEndian.AppendUint32(nil, uint32(len(chars)))
		res = append(res, bytesToWords(chars)...)
		p.objectFile.Sections[Data].writeAt(p.dataCounter, res)

		dataSize = len(res)
	case ".ref":
		if len(tokens) != 2 {
			return fmt.Errorf("%w: %s expects 1 argument, got %d", ErrArgumentCount, directive.Value, len(tokens)-1)
		}
		if tokens[1].Kind != tokenizer.NAME {
			return fmt.Errorf("%w: %s expects %s, got %s", ErrArgumentKind, directive.Value, tokenizer.NAME, tokens[1].Kind)
		}

		if tokenizer.IsReserved(tokens[1].Value) {
			return fmt.Errorf("%w: %s cannot be used as a reference", ErrReservedSymbol, tokens[1].Value)
		}

		res := make([]byte, 4)
		p.objectFile.Sections[Data].writeAt(p.dataCounter, res)

		p.objectFile.Relocations = append(p.objectFile.Relocations, Relocation{
			Section: p.currentSection,
			Offset:  p.dataCounter,
			Bit:     0,
			Width:   32,
			Kind:    Abs32,
			Symbol:  tokens[1].Value,
		})
		p.uses = append(p.uses, symbolUse{
			name:      tokens[1].Value,
			lineIndex: p.lineIndex,
		})

		dataSize = len(res)
	case ".word":
		args := tokens[1:]
		if len(args) == 0 {
			return fmt.Errorf("%w: %s expects at least 1 argument", ErrArgumentCount, directive.Value)
		}

		var res []byte
		for i := 0; i < len(args); {
			val, n, err := parseValue(args[i:])
			if err != nil {
				return fmt.Errorf("%s: %w", directive.Value, err)
			}
			res = binary.LittleEndian.AppendUint32(res, uint32(val))
			i += n

			if i == len(args) {
				break
			}
			if args[i].Kind != tokenizer.COMMA {
				return fmt.Errorf("%w: %s expects %s between values, got %s %q",
					ErrUnexpectedToken, directive.Value, tokenizer.COMMA, args[i].Kind, args[i].Value)
			}
			i++
			if i == len(args) {
				return fmt.Errorf("%w: %s ends with %s", ErrUnexpectedToken, directive.Value, tokenizer.COMMA)
			}
		}

		p.objectFile.Sections[Data].writeAt(p.dataCounter, res)
		dataSize = len(res)
	}

	p.resolveLabel()
	p.dataCounter += dataSize
	return nil
}

func parseValue(tokens []tokenizer.Token) (int64, int, error) {
	switch tokens[0].Kind {
	case tokenizer.NUMBER:
		val, err := parseNumber(tokens[0].Value)
		return val, 1, err
	case tokenizer.MINUS:
		if len(tokens) < 2 || tokens[1].Kind != tokenizer.NUMBER {
			return 0, 0, fmt.Errorf("%w: %s must be followed by %s", ErrArgumentKind, tokenizer.MINUS, tokenizer.NUMBER)
		}
		val, err := parseNumber("-" + tokens[1].Value)
		return val, 2, err
	case tokenizer.STRING:
		chars := stringToBytes(tokens[0].Value)
		if len(chars) != 1 {
			return 0, 0, fmt.Errorf("%w: char literal %q must be exactly 1 character", ErrArgumentKind, tokens[0].Value)
		}
		return int64(chars[0]), 1, nil
	default:
		return 0, 0, fmt.Errorf("%w: expected value, got %s %q", ErrArgumentKind, tokens[0].Kind, tokens[0].Value)
	}
}

func bytesToWords(in []byte) (out []byte) {
	out = make([]byte, len(in)*4)
	for i := 0; i < len(in); i++ {
		out[i*4] = in[i]
		out[i*4+1] = 0
		out[i*4+2] = 0
		out[i*4+3] = 0
	}
	return out
}

func stringToBytes(str string) []byte {
	var res []byte
	for i := 0; i < len(str); i += 1 {
		if str[i] == '\\' && i < len(str)-1 {
			switch str[i+1] {
			case '"':
				res = append(res, '"')
				i++
			case '\\':
				res = append(res, '\\')
				i++
			case 'n':
				res = append(res, '\n')
				i++
			case 't':
				res = append(res, '\t')
				i++
			case '0':
				res = append(res, 0)
				i++
			default:
				res = append(res, str[i])
			}
		} else {
			res = append(res, str[i])
		}
	}
	return res
}

func (p *assembler) handleInstruction(tokens []tokenizer.Token) error {
	instructionSize := 0
	// TODO check if instruction
	p.resolveLabel()
	p.textCounter += instructionSize
	return nil
}

func (p *assembler) resolveLabel() {
	if p.currentLabel == "" {
		return
	}

	labelIndex := p.symbolIndex[p.currentLabel]
	if p.currentSection == Text {
		p.objectFile.Symbols[labelIndex].Offset = p.textCounter
	} else {
		p.objectFile.Symbols[labelIndex].Offset = p.dataCounter
	}
	p.objectFile.Symbols[labelIndex].Section = p.currentSection
	p.currentLabel = ""
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
		p.labelLine = p.lineIndex
		return nil
	}

	p.currentLabel = labelName
	p.labelLine = p.lineIndex
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

		if labelIndex, ok := p.symbolIndex[argument.Value]; ok {
			if p.objectFile.Symbols[labelIndex].Binding != Import {
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
		if argument.Kind != tokenizer.NAME {
			return fmt.Errorf("%w: %s expects name, got %s", ErrArgumentKind, directive.Value, argument.Kind)
		}

		if tokenizer.IsReserved(argument.Value) {
			return fmt.Errorf("%w: %s cannot be exported", ErrReservedSymbol, argument.Value)
		}

		if labelIndex, ok := p.symbolIndex[argument.Value]; ok {
			if p.objectFile.Symbols[labelIndex].Binding == Import {
				return fmt.Errorf("%w: %s is imported in this file, cannot be exported", ErrSymbolConflict, argument.Value)
			}

			if p.objectFile.Symbols[labelIndex].Binding == Local {
				p.objectFile.Symbols[labelIndex].Binding = Export
			}

			return nil // repeated export: allowed
		}

		p.symbolIndex[argument.Value] = len(p.objectFile.Symbols)
		p.objectFile.Symbols = append(p.objectFile.Symbols, Symbol{
			Name:    argument.Value,
			Binding: Export,
		})
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
