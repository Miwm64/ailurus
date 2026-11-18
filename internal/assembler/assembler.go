package assembler

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/Miwm64/ailurus/internal/isa"
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
	ErrSectionTooLarge      = errors.New("section too large")
)

const (
	minNumber = math.MinInt32  // -2^31
	maxNumber = math.MaxUint32 // 2^32 - 1

	maxSectionSize = 1 << 20
)

const (
	aliasPush = "push"
	aliasPop  = "pop"
)

type fixup struct {
	isa.Relocation

	lineIndex int
}

type assembler struct {
	lineMap    map[int]int
	lines      []string
	objectFile *isa.ObjectFile

	symbolIndex map[string]int // name -> index in objectFile.Symbols

	dataCounter    int
	textCounter    int
	currentSection isa.Section
	currentLabel   string

	lineIndex int
	labelLine int

	fixups []fixup
}

func newAssembler(source string, file string, lineMap map[int]int) *assembler {
	return &assembler{
		lineMap:        lineMap,
		lines:          strings.Split(file, "\n"),
		objectFile:     isa.NewObjectFile(source),
		symbolIndex:    make(map[string]int),
		currentSection: isa.Text,
	}
}

func (p *assembler) assemble() (*isa.ObjectFile, error) {
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
		if s.Binding == isa.Export && s.Section == "" {
			undefined = append(undefined, s.Name)
		}
	}

	if len(undefined) > 0 {
		return nil, fmt.Errorf("%w: %s", ErrUndefinedExport, strings.Join(undefined, ", "))
	}

	for _, f := range p.fixups {
		idx, ok := p.symbolIndex[f.Symbol]
		if !ok {
			return nil, fmt.Errorf("line %d: %w: %s is neither defined nor imported",
				p.lineMap[f.lineIndex], ErrUndefinedSymbol, f.Symbol)
		}

		symbol := p.objectFile.Symbols[idx]
		if f.Kind != isa.PCRel16 || symbol.Binding == isa.Import || symbol.Section != f.Section {
			p.objectFile.Relocations = append(p.objectFile.Relocations, f.Relocation)

			continue
		}

		value := symbol.Offset - f.PCNext
		if value < math.MinInt16 || value > math.MaxInt16 {
			return nil, fmt.Errorf("line %d: %w: %s is %d bytes away, does not fit in 16 bits",
				p.lineMap[f.lineIndex], ErrNumberRange, f.Symbol, value)
		}

		section := *p.objectFile.Sections[f.Section]
		word := binary.LittleEndian.Uint32(section[f.Offset:])
		binary.LittleEndian.PutUint32(section[f.Offset:], isa.PatchField(word, f.Bit, f.Width, uint32(value)))
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

	if len(tokens) >= 2 && tokens[0].Kind == tokenizer.Name && tokens[1].Kind == tokenizer.Colon {
		err := p.handleLabel(tokens[0].Value)
		if err != nil {
			return err
		}

		tokens = tokens[2:]
		hasLabel = true
	}

	if len(tokens) >= 2 && tokens[0].Kind == tokenizer.Name && tokens[1].Kind == tokenizer.Colon {
		return fmt.Errorf("%w: %s defined on the same line as %s", ErrMultipleLabels, tokens[0].Value, p.currentLabel)
	}

	if len(tokens) == 0 {
		return nil
	}

	// main part
	switch tokens[0].Kind {
	case tokenizer.AssemblerDirective:
		switch tokens[0].Value {
		case ".word", ".pstr", ".ref":
			err := p.handleData(tokens)
			if err != nil {
				return err
			}
		default:
			if hasLabel {
				return fmt.Errorf(
					"%w: %s cannot follow label %s",
					ErrLabelBeforeDirective,
					tokens[0].Value,
					p.currentLabel,
				)
			}

			err := p.handleDirective(tokens)
			if err != nil {
				return err
			}
		}
	case tokenizer.Name:
		err := p.handleInstruction(tokens)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("%w: %s %q at start of line", ErrUnexpectedToken, tokens[0].Kind, tokens[0].Value)
	}

	return nil
}

func parseValue(tokens []tokenizer.Token) (int64, int, error) {
	switch tokens[0].Kind {
	case tokenizer.Number:
		val, err := parseNumber(tokens[0].Value)

		return val, 1, err
	case tokenizer.Minus:
		if len(tokens) < 2 || tokens[1].Kind != tokenizer.Number {
			return 0, 0, fmt.Errorf("%w: %s must be followed by %s", ErrArgumentKind, tokenizer.Minus, tokenizer.Number)
		}

		val, err := parseNumber("-" + tokens[1].Value)

		return val, 2, err
	case tokenizer.String:
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
	for i := range in {
		out[i*4] = in[i]
		out[i*4+1] = 0
		out[i*4+2] = 0
		out[i*4+3] = 0
	}

	return out
}

func stringToBytes(str string) []byte {
	var res []byte

	for i := 0; i < len(str); i++ {
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

var aliases = map[string]string{
	"asl":     "lsl",
	aliasPush: "mov",
	aliasPop:  "mov",
}

func rewriteAlias(mnemonic string, operands []operand) ([]operand, error) {
	const rsp = 0

	switch mnemonic {
	case aliasPush, aliasPop:
		if len(operands) != 1 {
			return nil, fmt.Errorf("%w: %s expects 1, got %d", isa.ErrArgumentCount, mnemonic, len(operands))
		}
	}

	switch mnemonic {
	case aliasPush:
		dst := operand{arg: isa.Argument{Mode: isa.PreDecrement, Value: rsp}}

		return []operand{dst, operands[0]}, nil
	case aliasPop:
		src := operand{arg: isa.Argument{Mode: isa.PostIncrement, Value: rsp}}

		return []operand{operands[0], src}, nil
	}

	return operands, nil
}

func mapOperands(format isa.Format, operands []operand) error {
	for i := range operands {
		op := &operands[i]
		bareName := op.label != "" && op.arg.Mode == isa.Immediate

		switch format {
		case isa.FB:
			if op.bareAt {
				return fmt.Errorf("%w: argument %d: branches take a bare label, not @", ErrInvalidOperand, i+1)
			}

			if bareName {
				op.arg.Mode = isa.PCRelative
			}

		case isa.FJ:
			if bareName {
				op.arg.Mode = isa.DirectMemory
			}

		default:
			if op.bareAt {
				return fmt.Errorf("%w: argument %d: @name is only allowed in jump and call", ErrInvalidOperand, i+1)
			}
		}
	}

	return nil
}

// converts words to bytes as little-endian.
func wordsToBytes(words []uint32) []byte {
	res := make([]byte, 0, 4*len(words))
	for _, w := range words {
		res = binary.LittleEndian.AppendUint32(res, w)
	}

	return res
}

func (p *assembler) addLineToListing(offset int) {
	p.objectFile.Lines = append(p.objectFile.Lines, isa.Line{
		Section: p.currentSection,
		Offset:  offset,
		Line:    p.lineMap[p.lineIndex],
		Text:    strings.TrimSpace(p.lines[p.lineIndex]),
	})
}

func (p *assembler) resolveLabel() {
	if p.currentLabel == "" {
		return
	}

	labelIndex := p.symbolIndex[p.currentLabel]
	if p.currentSection == isa.Text {
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
		case isa.Import:
			return fmt.Errorf("%w: %s is imported, cannot be defined", ErrSymbolConflict, labelName)
		case isa.Local:
			return fmt.Errorf("%w: %s is already defined", ErrSymbolConflict, labelName)
		case isa.Export:
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
	p.objectFile.Symbols = append(p.objectFile.Symbols, isa.Symbol{
		Name:    labelName,
		Binding: isa.Local,
	})

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

func (p *assembler) handleInstruction(tokens []tokenizer.Token) error {
	if p.currentSection != isa.Text {
		return fmt.Errorf("%w: instructions are only allowed in %s, current section is %s",
			ErrWrongSection, isa.Text, p.currentSection)
	}

	if len(tokens) < 1 {
		return nil
	}

	if tokens[0].Kind != tokenizer.Name {
		return fmt.Errorf("%w: expected name, got %s %q", ErrUnexpectedToken, tokens[0].Kind, tokens[0].Value)
	}

	mnemonic := tokens[0].Value

	name := mnemonic
	if realMnemonic, ok := aliases[mnemonic]; ok {
		name = realMnemonic
	}

	opcode, ok := isa.Opcodes[name]
	if !ok {
		return fmt.Errorf("%w: expected opcode, got %s %q", ErrUnexpectedToken, tokens[0].Kind, tokens[0].Value)
	}

	operands, err := parseOperands(mnemonic, tokens[1:])
	if err != nil {
		return err
	}

	operands, err = rewriteAlias(mnemonic, operands)
	if err != nil {
		return err
	}

	if err := mapOperands(opcode.Format, operands); err != nil {
		return fmt.Errorf("%s: %w", mnemonic, err)
	}

	instruction := isa.Instruction{OpInfo: opcode}
	for _, op := range operands {
		instruction.Arguments = append(instruction.Arguments, op.arg)
	}

	words, positions, err := instruction.Encode()
	if err != nil {
		return fmt.Errorf("%s: %w", mnemonic, err)
	}

	if p.textCounter+4*len(words) > maxSectionSize {
		return fmt.Errorf("%w: %s exceeds %d bytes", ErrSectionTooLarge, isa.Text, maxSectionSize)
	}

	p.objectFile.Sections[isa.Text].WriteAt(p.textCounter, wordsToBytes(words))
	p.resolveLabel()
	p.addLineToListing(p.textCounter)
	p.addFixups(operands, positions)
	p.textCounter += 4 * len(words)

	return nil
}

// parseOperands: the comma-separated operands after the mnemonic.
func parseOperands(mnemonic string, tokens []tokenizer.Token) ([]operand, error) {
	operands := make([]operand, 0)
	add := func(t []tokenizer.Token) error {
		op, err := parseOperand(t)
		if err != nil {
			return fmt.Errorf("%s: argument %d: %w", mnemonic, len(operands)+1, err)
		}

		operands = append(operands, op)

		return nil
	}

	prev := 0

	for i, tok := range tokens {
		if tok.Kind != tokenizer.Comma {
			continue
		}

		if err := add(tokens[prev:i]); err != nil {
			return nil, err
		}

		prev = i + 1
	}

	if prev < len(tokens) {
		if err := add(tokens[prev:]); err != nil {
			return nil, err
		}
	}

	return operands, nil
}

// addFixups: a fixup for every operand that names a label.
func (p *assembler) addFixups(operands []operand, positions []isa.EncodedPosition) {
	for i, op := range operands {
		if op.label == "" {
			continue
		}

		pos := positions[i]
		kind := isa.ModeToRelocationKind(op.arg.Mode)

		pcNext := 0
		if kind == isa.PCRel16 {
			pcNext = p.textCounter + 4*(pos.Word+1)
		}

		p.fixups = append(p.fixups, fixup{
			Section:   isa.Text,
			Offset:    p.textCounter + 4*pos.Word,
			Bit:       pos.Offset,
			Width:     pos.Width,
			Kind:      isa.ModeToRelocationKind(op.arg.Mode),
			Symbol:    op.label,
			PCNext:    pcNext,
			lineIndex: p.lineIndex,
		})
	}
}

func (p *assembler) handleDirective(tokens []tokenizer.Token) error {
	directive := tokens[0]
	if len(tokens) != 2 {
		return fmt.Errorf("%w: %s expects 1 argument, got %d", ErrArgumentCount, directive.Value, len(tokens)-1)
	}

	argument := tokens[1]

	switch directive.Value {
	case ".section":
		return p.directiveSection(directive, argument)
	case ".org":
		return p.directiveOrg(directive, argument)
	case ".import":
		return p.directiveImport(directive, argument)
	case ".export":
		return p.directiveExport(directive, argument)
	default:
		return fmt.Errorf("%w: %q", ErrUnknownDirective, directive.Value)
	}
}

// expectKind: the directive's argument is a token of the given kind.
func expectKind(directive, argument tokenizer.Token, kind tokenizer.Kind) error {
	if argument.Kind != kind {
		return fmt.Errorf("%w: %s expects %s, got %s", ErrArgumentKind, directive.Value, kind, argument.Kind)
	}

	return nil
}

func (p *assembler) directiveSection(directive, argument tokenizer.Token) error {
	if err := expectKind(directive, argument, tokenizer.Name); err != nil {
		return err
	}

	if argument.Value != "data" && argument.Value != "text" {
		return fmt.Errorf("%w: expected 'data' or 'text', got %s", ErrUnknownSection, argument.Value)
	}

	p.currentSection = isa.Section(argument.Value)

	return nil
}

func (p *assembler) directiveOrg(directive, argument tokenizer.Token) error {
	if err := expectKind(directive, argument, tokenizer.Number); err != nil {
		return err
	}

	val, err := parseNumber(argument.Value)
	if err != nil {
		return fmt.Errorf("%s: %w", directive.Value, err)
	}

	if val > maxSectionSize {
		return fmt.Errorf("%w: %s %d exceeds %d", ErrNumberRange, directive.Value, val, maxSectionSize)
	}

	if p.currentSection == isa.Text {
		p.textCounter = int(val)
	} else {
		p.dataCounter = int(val)
	}

	return nil
}

func (p *assembler) directiveImport(directive, argument tokenizer.Token) error {
	if err := expectKind(directive, argument, tokenizer.Name); err != nil {
		return err
	}

	if argument.Value == "_start" || argument.Value == "_handler" {
		return fmt.Errorf("%w: %s cannot be imported", ErrReservedSymbol, argument.Value)
	}

	if labelIndex, ok := p.symbolIndex[argument.Value]; ok {
		if p.objectFile.Symbols[labelIndex].Binding != isa.Import {
			return fmt.Errorf("%w: %s is defined in this file, cannot be imported", ErrSymbolConflict, argument.Value)
		}

		return nil // repeated import: allowed
	}

	p.symbolIndex[argument.Value] = len(p.objectFile.Symbols)
	p.objectFile.Symbols = append(p.objectFile.Symbols, isa.Symbol{
		Name:    argument.Value,
		Binding: isa.Import,
	})

	return nil
}

func (p *assembler) directiveExport(directive, argument tokenizer.Token) error {
	if err := expectKind(directive, argument, tokenizer.Name); err != nil {
		return err
	}

	if tokenizer.IsReserved(argument.Value) {
		return fmt.Errorf("%w: %s cannot be exported", ErrReservedSymbol, argument.Value)
	}

	if labelIndex, ok := p.symbolIndex[argument.Value]; ok {
		binding := &p.objectFile.Symbols[labelIndex].Binding
		if *binding == isa.Import {
			return fmt.Errorf("%w: %s is imported in this file, cannot be exported", ErrSymbolConflict, argument.Value)
		}

		if *binding == isa.Local {
			*binding = isa.Export
		}

		return nil // repeated export: allowed
	}

	p.symbolIndex[argument.Value] = len(p.objectFile.Symbols)
	p.objectFile.Symbols = append(p.objectFile.Symbols, isa.Symbol{
		Name:    argument.Value,
		Binding: isa.Export,
	})

	return nil
}

func (p *assembler) handleData(tokens []tokenizer.Token) error {
	directive := tokens[0]

	if p.currentSection != isa.Data {
		return fmt.Errorf("%w: %s is only allowed in %s, current section is %s",
			ErrWrongSection, directive.Value, isa.Data, p.currentSection)
	}

	var (
		res []byte
		err error
	)

	switch directive.Value {
	case ".pstr":
		res, err = dataPstr(tokens)
	case ".ref":
		res, err = p.dataRef(tokens)
	case ".word":
		res, err = dataWord(tokens)
	}

	if err != nil {
		return err
	}

	if p.dataCounter+len(res) > maxSectionSize {
		return fmt.Errorf("%w: %s exceeds %d bytes", ErrSectionTooLarge, isa.Data, maxSectionSize)
	}
	p.objectFile.Sections[isa.Data].WriteAt(p.dataCounter, res)

	p.resolveLabel()
	p.addLineToListing(p.dataCounter)
	p.dataCounter += len(res)

	return nil
}

// oneArgument: the directive has exactly one argument, of the given kind.
func oneArgument(tokens []tokenizer.Token, kind tokenizer.Kind) error {
	if len(tokens) != 2 {
		return fmt.Errorf("%w: %s expects 1 argument, got %d", ErrArgumentCount, tokens[0].Value, len(tokens)-1)
	}

	return expectKind(tokens[0], tokens[1], kind)
}

func dataPstr(tokens []tokenizer.Token) ([]byte, error) {
	if err := oneArgument(tokens, tokenizer.String); err != nil {
		return nil, err
	}

	chars := stringToBytes(tokens[1].Value)
	res := binary.LittleEndian.AppendUint32(nil, uint32(len(chars)))

	return append(res, bytesToWords(chars)...), nil
}

func (p *assembler) dataRef(tokens []tokenizer.Token) ([]byte, error) {
	if err := oneArgument(tokens, tokenizer.Name); err != nil {
		return nil, err
	}

	if tokenizer.IsReserved(tokens[1].Value) {
		return nil, fmt.Errorf("%w: %s cannot be used as a reference", ErrReservedSymbol, tokens[1].Value)
	}

	p.fixups = append(p.fixups, fixup{
		Section:   isa.Data,
		Offset:    p.dataCounter,
		Bit:       0,
		Width:     32,
		Kind:      isa.Abs32,
		Symbol:    tokens[1].Value,
		lineIndex: p.lineIndex,
	})

	return make([]byte, 4), nil
}

func dataWord(tokens []tokenizer.Token) ([]byte, error) {
	directive, args := tokens[0], tokens[1:]
	if len(args) == 0 {
		return nil, fmt.Errorf("%w: %s expects at least 1 argument", ErrArgumentCount, directive.Value)
	}

	var res []byte

	for i := 0; i < len(args); {
		val, n, err := parseValue(args[i:])
		if err != nil {
			return nil, fmt.Errorf("%s: %w", directive.Value, err)
		}

		res = binary.LittleEndian.AppendUint32(res, uint32(val))
		i += n

		if i == len(args) {
			break
		}

		if args[i].Kind != tokenizer.Comma {
			return nil, fmt.Errorf("%w: %s expects %s between values, got %s %q",
				ErrUnexpectedToken, directive.Value, tokenizer.Comma, args[i].Kind, args[i].Value)
		}

		i++
		if i == len(args) {
			return nil, fmt.Errorf("%w: %s ends with %s", ErrUnexpectedToken, directive.Value, tokenizer.Comma)
		}
	}

	return res, nil
}
