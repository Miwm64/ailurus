package assembler

import (
	"fmt"
	"strings"

	"github.com/Miwm64/ailurus/internal/tokenizer"
)

func Assemble(file string, lineMap map[int]int) (*ObjectFile, error) {
	objectFile := &ObjectFile{}

	lines := strings.Split(file, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		err := handleLine(line, lineMap, objectFile)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineMap[i], err)
		}
	}

	return objectFile, nil
}

func handleLine(line string, lineMap map[int]int, objectFile *ObjectFile) error {
	_, err := tokenizer.Tokenize(line)
	if err != nil {
		return err
	}
	return nil
}
