package vm

import (
	"fmt"
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

func alu(op aluOp, a, b uint32, carry bool) aluOut {
	var out aluOut

	switch op {
	case aluAdd, aluAddC, aluSub, aluSubC, aluInc, aluDec, aluNeg:
		out = arithm(op, a, b, carry)
	case aluPassA, aluPassB, aluAnd, aluOr, aluXor, aluNot:
		out = aluOut{result: logic(op, a, b)}
	case aluLsl, aluLsr, aluAsr:
		out = shift(op, a, b, carry)
	case aluMulL, aluMulH, aluUMulH, aluDiv, aluMod:
		out = mulDiv(op, a, b)
	default:
		panic(fmt.Sprintf("alu: unknown op %d", op))
	}

	out.flags.n = out.result>>signBit == 1
	out.flags.z = out.result == 0

	return out
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

func shift(op aluOp, a, b uint32, carry bool) aluOut {
	n := b % wordBits
	if n == 0 {
		return aluOut{result: a, flags: flags{c: carry}}
	}

	switch op {
	case aluLsl:
		return aluOut{result: a << n, flags: flags{c: (a>>(wordBits-n))&1 == 1}}
	case aluLsr:
		return aluOut{result: a >> n, flags: flags{c: (a>>(n-1))&1 == 1}}
	default: // aluAsr
		return aluOut{result: uint32(int32(a) >> n), flags: flags{c: (a>>(n-1))&1 == 1}}
	}
}

// INT_MIN / -1 = INT_MIN, INT_MIN % -1 = 0 (Go semantics).
func mulDiv(op aluOp, a, b uint32) aluOut {
	switch op {
	case aluMulL:
		return aluOut{result: a * b}
	case aluMulH:
		return aluOut{result: uint32((int64(int32(a)) * int64(int32(b))) >> wordBits)}
	case aluUMulH:
		hi, _ := bits.Mul32(a, b)

		return aluOut{result: hi}
	default:
	}

	if b == 0 {
		return aluOut{divZero: true}
	}

	if op == aluDiv {
		return aluOut{result: uint32(int32(a) / int32(b))}
	}

	return aluOut{result: uint32(int32(a) % int32(b))}
}
