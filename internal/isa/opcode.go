package isa

import "fmt"

type Format uint8

const (
	F0 Format = iota // no operands
	F1               // 1 operand
	F2               // 2 operands
	FB               // conditional branch
	FJ               // jump / call
	FP               // pushm / popm
	FA               // addm
)

type Opcode uint8

type OpInfo struct {
	Opcode Opcode
	Format Format
}

var Opcodes = map[string]OpInfo{
	// F0
	"halt": {0x01, F0},
	"ret":  {0x02, F0},
	"iret": {0x03, F0},
	"ei":   {0x04, F0},
	"di":   {0x05, F0},

	// F1
	"clr": {0x08, F1},
	"not": {0x09, F1},
	"inc": {0x0A, F1},
	"dec": {0x0B, F1},
	"neg": {0x0C, F1},

	// FJ
	"jump": {0x0E, FJ},
	"call": {0x0F, FJ},

	// FP
	"pushm": {0x10, FP},
	"popm":  {0x11, FP},

	// FA
	"addm": {0x12, FA},

	// FB
	"jz":  {0x18, FB},
	"jnz": {0x19, FB},
	"ja":  {0x1A, FB},
	"jae": {0x1B, FB},
	"jb":  {0x1C, FB},
	"jbe": {0x1D, FB},
	"jl":  {0x1E, FB},
	"jle": {0x1F, FB},
	"jg":  {0x20, FB},
	"jge": {0x21, FB},

	// F2
	"mov":   {0x28, F2},
	"and":   {0x29, F2},
	"or":    {0x2A, F2},
	"xor":   {0x2B, F2},
	"add":   {0x2C, F2},
	"addc":  {0x2D, F2},
	"sub":   {0x2E, F2},
	"subc":  {0x2F, F2},
	"mulh":  {0x30, F2},
	"umulh": {0x31, F2},
	"mull":  {0x32, F2},
	"div":   {0x33, F2},
	"mod":   {0x34, F2},
	"cmp":   {0x35, F2},
	"test":  {0x36, F2},
	"lsl":   {0x37, F2},
	"lsr":   {0x38, F2},
	"asr":   {0x39, F2},
	"in":    {0x3A, F2},
	"out":   {0x3B, F2},
}

// reverse table
var Mnemonics = buildMnemonics()

func buildMnemonics() map[Opcode]string {
	m := make(map[Opcode]string, len(Opcodes))
	for name, info := range Opcodes {
		if other, ok := m[info.Opcode]; ok {
			panic(fmt.Sprintf("isa: opcode 0x%02X used by both %q and %q", info.Opcode, other, name))
		}
		if info.Opcode > 0x3F {
			panic(fmt.Sprintf("isa: opcode 0x%02X of %q does not fit in 6 bits", info.Opcode, name))
		}
		m[info.Opcode] = name
	}
	return m
}
