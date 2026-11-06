package assembler

func Assemble(source string, file string, lineMap map[int]int) (*ObjectFile, error) {
	assembler := newAssembler(source, file, lineMap)
	return assembler.assemble()
}
