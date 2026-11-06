package assembler

import (
	"fmt"
	"strings"

	"github.com/Miwm64/ailurus/internal/tokenizer"
)

type assembler struct {
	lineMap    map[int]int
	lines      []string
	objectFile *ObjectFile
}

func newAssembler(source string, file string, lineMap map[int]int) *assembler {
	return &assembler{
		lineMap:    lineMap,
		lines:      strings.Split(file, "\n"),
		objectFile: newObjectFile(source),
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

	return p.objectFile, nil
}

func (p *assembler) handleLine(line string) error {
	_, err := tokenizer.Tokenize(line)
	if err != nil {
		return err
	}

	return nil
}

func (p *assembler) handleDirective(tokens []tokenizer.Token) error {
	switch tokens[0].Value {
	case ".section":
	case ".org":
	case ".import":
	case ".export":

	}
	return nil
}
