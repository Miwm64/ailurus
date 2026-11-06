package assembler

func Assemble(file string, lineMap map[int]int) (*ObjectFile, error) {
	assembler := newAssembler(file, lineMap)
	return assembler.assemble()
}
