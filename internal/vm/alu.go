package vm

// TODO: match the scheme's ALU operation list (docs: "TODO: ALU operations").
type aluOp uint8

const (
	aluPassA aluOp = iota
	aluPassB
	aluAdd
	aluAddC
	aluSub
	aluSubC
	aluInc
	aluDec
	aluNeg
	aluAnd
	aluOr
	aluXor
	aluNot
	aluLsl
	aluLsr
	aluAsr
	aluMulL
	aluMulH
	aluUMulH
	aluDiv
	aluMod
)

type flags struct {
	n, z, v, c bool
}

type flagMask uint8

const (
	maskN flagMask = 1 << iota
	maskZ
	maskV
	maskC
	maskIE
)

type aluOut struct {
	result  uint32
	flags   flags
	divZero bool
}

func alu(op aluOp, a, b uint32, carry bool) aluOut {
	panic("TODO")
}
