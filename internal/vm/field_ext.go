package vm

type fieldIn struct {
	ir, mdr  uint32
	idx, pos uint8
	operand  uint8
}

type fieldOut struct {
	value   uint32 // sign-extended
	reg     uint8  // register number
	mode    uint8
	fits    bool
	last    bool
	posNext uint8
}

func fieldExt(in fieldIn) fieldOut {
	panic("TODO")
}
