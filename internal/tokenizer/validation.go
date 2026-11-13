package tokenizer

func ValidateName(str string) bool {
	if str == "" {
		return false
	}

	for i := 0; i < len(str); i++ {
		c := str[i]
		isLetter := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
		isDigit := c >= '0' && c <= '9'

		if !isLetter && c != '_' && (i == 0 || !isDigit) {
			return false
		}
	}

	return true
}

var reserved = map[string]struct{}{
	// registers
	"rsp": {}, "r1": {}, "r2": {}, "r3": {}, "r4": {}, "r5": {}, "r6": {}, "r7": {},
	"r8": {}, "r9": {}, "r10": {}, "r11": {}, "r12": {}, "r13": {}, "r14": {}, "r15": {},

	// data and logic
	"mov": {}, "clr": {}, "not": {}, "and": {}, "or": {}, "xor": {},

	// arithmetic
	"inc": {}, "dec": {}, "neg": {}, "add": {}, "addc": {}, "sub": {}, "subc": {},
	"mulh": {}, "umulh": {}, "mull": {}, "div": {}, "mod": {}, "cmp": {}, "test": {}, "addm": {},

	// shifts
	"lsl": {}, "asl": {}, "lsr": {}, "asr": {},

	// stack
	"push": {}, "pop": {}, "pushm": {}, "popm": {}, "call": {}, "ret": {}, "iret": {},

	// io
	"in": {}, "out": {},

	// control flow
	"jump": {}, "jz": {}, "jnz": {}, "ja": {}, "jae": {}, "jb": {}, "jbe": {},
	"jl": {}, "jle": {}, "jg": {}, "jge": {}, "ei": {}, "di": {}, "halt": {},

	// section names
	"data": {}, "text": {},
}

func IsReserved(str string) bool {
	_, ok := reserved[str]

	return ok
}
