package assembler

import "github.com/Miwm64/ailurus/internal/isa"

func Assemble(name string, text string, lineMap map[int]int) (*isa.ObjectFile, error) {
	assembler := newAssembler(name, text, lineMap)

	return assembler.assemble()
}
