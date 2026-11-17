package vm

import (
	"math/bits"
)

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

const (
	wordBits = 32
	signBit  = wordBits - 1
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

func arithm(op aluOp, a, b uint32, carry bool) aluOut {
	var cin uint32
	if carry {
		cin = 1
	}

	switch op {
	case aluAdd:
		return add(a, b, 0)
	case aluAddC:
		return add(a, b, cin)
	case aluInc:
		return add(a, 1, 0)
	case aluSub:
		return sub(a, b, 0)
	case aluSubC:
		return sub(a, b, cin)
	case aluDec:
		return sub(a, 1, 0)
	default: // aluNeg
		return sub(0, a, 0)
	}
}

func add(a, b, cin uint32) aluOut {
	r, c := bits.Add32(a, b, cin)
	v := ((a ^ r) & (b ^ r)) >> signBit
	return aluOut{result: r, flags: flags{v: v == 1, c: c == 1}}
}

func sub(a, b, bin uint32) aluOut {
	r, borrow := bits.Sub32(a, b, bin)
	v := ((a ^ b) & (a ^ r)) >> signBit
	return aluOut{result: r, flags: flags{v: v == 1, c: borrow == 1}}
}

func logic(op aluOp, a, b uint32) uint32 {
	switch op {
	case aluPassA:
		return a
	case aluPassB:
		return b
	case aluAnd:
		return a & b
	case aluOr:
		return a | b
	case aluXor:
		return a ^ b
	default: // aluNot
		return ^a
	}
}
