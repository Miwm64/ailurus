# Instruction set

## Operand modes
| # | Mode               | Syntax                  | Notes                                                                          |
|---|--------------------|-------------------------|--------------------------------------------------------------------------------|
| 0 | register           | `r1`                    | operand is the register itself                                                 |
| 1 | direct memory      | `(0x120)`, `(label)`    | access at the absolute address stored in the instruction; labels are relocated |
| 2 | PC-relative memory | `(@label)`              | access at `PC_next + offset`; assembler computes `offset = label − PC_next`    |
| 3 | register indirect  | `(r1)`                  | access at `r1`                                                                 |
| 4 | postincrement      | `(r1)+`                 | access at `r1`, then `r1 ← r1 + instruction_size`                              |
| 5 | predecrement       | `-(r1)`                 | `r1 ← r1 − instruction_size`, then access at `r1`                              |
| 6 | immediate          | `120`, `0x120`, `label` | value stored in the instruction; source only; a bare label is its address      |
| 7 | register relative  | `(r1+10)`               | access at `r1 + offset`;                                                       |z

offset is a signed 12-bit integer
PC_next — program counter, address of the next instruction.
All memory accesses are 32-bit words.
When both operands have increments/decrements - compilation error.
## Commands

### Data movement
| Instruction  | Meaning           |
|--------------|-------------------|
| mov dst, src | move: dst ← src   |

### Logical
| Instruction  | Meaning                 |
|--------------|-------------------------|
| clr dst      | clear: dst ← 0          |
| not dst      | bitwise not: dst ← ~dst |
| and dst, src | dst ← dst & src         |
| or dst, src  | dst ← dst \| src        |
| xor dst, src | dst ← dst ^ src         |

### Arithmetic
| Instruction    | Meaning                                |
|----------------|----------------------------------------|
| inc dst        | dst ← dst + 1                          |
| dec dst        | dst ← dst − 1                          |
| neg dst        | dst ← −dst                             |
| add dst, src   | dst ← dst + src                        |
| addc dst, src  | dst ← dst + src + C                    |
| sub dst, src   | dst ← dst − src                        |
| subc dst, src  | dst ← dst − src − C                    |
| mulh dst, src  | dst ← (dst × src) bits 63:32(signed)   |
| umulh dst, src | dst ← (dst × src) bits 63:32(unsigned) |
| mull dst, src  | dst ← (dst × src) bits 31:0            |
| div dst, src   | dst ← dst / src (signed)               |
| mod dst, src   | dst ← dst % src (signed). -7 % 2 = 1   |
| cmp a, b       | flags ← a − b.                         |

### Shifts
| Instruction    | Meaning                                        |
|----------------|------------------------------------------------|
| lsl dst, count | dst ← dst << count (zeros fill from the right) |
| asl dst, count | alias for lsl, as they behave the same way     |
| lsr dst, count | dst ← dst >> count (zero-filled)               |
| asr dst, count | dst ← dst >> count (sign-filled)               |

### Stack
| Instruction       | Meaning                                         |
|-------------------|-------------------------------------------------|
| push operand      | alias of mov -(rsp),  operand                   |
| pushm r2, r1, r13 | pushes a variable amount of registries          |
| pop operand       | alias of mov operand, (rsp)+                    |
| popm operand      | pop a variable amount of values into registries |
| call label        | pushes return address to stack, changes pc      |
| ret               | pops return address from stack and sets in pc   |

Stack is grown down

### Control flow
| Instruction   | Semantics                                           |
|---------------|-----------------------------------------------------|
| jump  operand | PC <- operand (see [operand modes](#operand-modes)) |
| jz  label     | if ZF == 1:              PC <- PC_next + offset     |
| jnz label     | if ZF == 0:              PC <- PC_next + offset     |
| ja  label     | if CF == 0 && ZF == 0:   PC <- PC_next + offset     |
| jae label     | if CF == 0:              PC <- PC_next + offset     |
| jb  label     | if CF == 1:              PC <- PC_next + offset     |
| jbe label     | if CF == 1 \|\| ZF == 1: PC <- PC_next + offset     |
| jl  label     | if SF != OF:             PC <- PC_next + offset     |
| jle label     | if ZF == 1 \|\| SF != OF: PC <- PC_next + offset    |
| jg  label     | if ZF == 0 && SF == OF:  PC <- PC_next + offset     |
| jge label     | if SF == OF:             PC <- PC_next + offset     |
| halt          | stops the execution                                 |

PC_next — address of the instruction following the jump.
offset — signed N-bit, computed by translator: addr(label) - PC_next.

## Encoding
### Machine model
- Word size: 4 bytes
- Memory: byte-addressed, single-port, one access per cycle. Word-aligned instruction fetch.
- Registers: r0-r15, 4 bytes each.
- PC: word pointer, always +- bytes. If PC is not aligned - error.

### Instruction classes
- Fixed-arity, 0 operands: 
[OPCODE | padding]
size = 1 word.

- Fixed-arity, 1–2 operands: 
word1: [OPCODE | MODE1 | MODE2 | OPERAND1 | OPERAND2]
If any operand fits in the first word - it is put there. Otherwise, operands are put in the 1-2 next machine words.
size = 1-3 words.

- Variable-arity(pushm/popm): 
word1: [OPCODE | COUNT | registries] 
if registries did not fit into first word, they are put into second.
size > 1 word.

## Encoding

### Machine model
- Word size: 4 bytes.
- Memory: byte-addressed, single-port, one access per cycle. Instruction fetch is word-aligned.
- Registers: r0–r15, 4 bytes each.
- PC: word pointer, always ± bytes. Unaligned PC → error.

### Instruction classes

**Fixed arity, 0 operands**

    [OPCODE | padding]

Size: 1 word.

**Fixed arity, 1–2 operands**

    word1: [OPCODE | MODE1 | MODE2 | inline operands]

Operands that do not fit into word 1 go into extension words.
Size: 1–3 words.

**Variable arity (`pushm` / `popm`)**

    word1: [OPCODE | COUNT | registers]

If the registers do not fit into word 1, they go into a second word.

### Field sizes

| Field                                                   | Size              |
|---------------------------------------------------------|-------------------|
| OPCODE                                                  | 6 bits            |
| MODE                                                    | 3 bits            |
| register (register, indirect, postincrement, predecrement) | 4 bits         |
| immediate                                               | 16 bits, signed   |
| PC-relative offset                                      | 16 bits, signed, in bytes |
| register-relative                                       | 4 + 16 bits = 20 bits (reg# + offset) |
| absolute address                                        | 32 bits           |

OPCODE + 2 × MODE = 12 bits, so 20 bits of word 1 are free for inline operands.

### Operand packing
- Operands are placed in order: operand 1, then operand 2. If they do not fit in that order, the order is changed (the only such case: absolute + register-relative).
- An absolute address is always placed in an extension word.
- Absolute + register-relative: register-relative goes in word 1, absolute in word 2.
- Instruction sizes:
    - absolute + absolute: 3 words
    - register-relative + register-relative: 2 words
    - all other combinations: at most 2 words

### Immediates
- Always 16 bits, signed. There are no 32-bit immediates.
- Larger constants are either stored in the data section and loaded from memory, or built with shift/or.
- `mov r1, label` loads the address of `label` into `r1`. If the address does not fit into the immediate field → error.

### Unresolved (encoding)
- [ ] PC line in the machine model is unclear ("word pointer, always ± bytes").
- [ ] The operand modes section says the offset is 12-bit, and conditional jumps say "N-bit"; this section says 16.
- [ ] Is "change order" needed, or does greedy per-operand placement already cover absolute + register-relative?
- [ ] Order of extension words when both operands go out.
- [ ] Layout of a 16-bit or 20-bit operand inside a 32-bit extension word (bit positions, sign extension).
- [ ] Relocation of external labels: how instruction size is fixed before linking, and what the linker needs to patch a field.
- [ ] Signed 16-bit immediate means addresses ≤ 0x7FFF; memory size is not specified.
- [ ] Which stage reports "label address does not fit" (assembler or linker).
- [ ] 1-operand instructions and conditional jumps: what goes in MODE2, and do jumps carry a MODE field at all.
- [ ] Invalid encodings (immediate as destination, junk in padding): error or undefined.
- [ ] `pushm`/`popm`: register mask or list, COUNT width, size "> 1 word" vs "≥ 1 word".
- [ ] Bit-position diagrams for each instruction class.