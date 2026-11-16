package assembler

import "github.com/Miwm64/ailurus/internal/isa"

func Assemble(source string, file string, lineMap map[int]int) (*isa.ObjectFile, error) {
	assembler := newAssembler(source, file, lineMap)
	return assembler.assemble()
}
