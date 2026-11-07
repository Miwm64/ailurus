package isa

type Instruction struct {
	Opcode    OpInfo // [31:26]
	Arguments []Argument
}

type EncodedPosition struct {
	word   int
	offset int
	width  int
}

func (p *Instruction) encode() ([]byte, []EncodedPosition, error) {

}
