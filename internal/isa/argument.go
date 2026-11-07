package isa

type Mode uint8

const (
	Register Mode = iota
	DirectMemory
	PCRelative
	RegisterIndirect
	PostIncrement
	PreDecrement
	Immediate
	RegisterRelative
)

type Argument struct {
	Mode   Mode
	Value  int
	Offset int
}
