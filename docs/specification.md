# Instruction set

## Operand modes

| # | Mode               | Syntax                  | Operand size (bits)  | Notes                                                                          |
|---|--------------------|-------------------------|----------------------|--------------------------------------------------------------------------------|
| 0 | register           | `r1`                    | 4 (reg#)             | operand is the register itself                                                 |
| 1 | direct memory      | `(0x120)`, `(label)`    | 32 (address)         | access at the absolute address stored in the instruction; labels are relocated |
| 2 | PC-relative memory | `(@label)`              | 16 (offset)          | access at `PC_next + offset`; assembler computes `offset = label − PC_next`    |
| 3 | register indirect  | `(r1)`                  | 4 (reg#)             | access at `r1`                                                                 |
| 4 | postincrement      | `(r1)+`                 | 4 (reg#)             | access at `r1`, then `r1 ← r1 + 4`                                             |
| 5 | predecrement       | `-(r1)`                 | 4 (reg#)             | `r1 ← r1 − 4`, then access at `r1`                                             |
| 6 | immediate          | `120`, `0x120`, `label` | 16 (value)           | value stored in the instruction; source only; a bare label is its address      |
| 7 | register relative  | `(r1+10)`               | 20 (reg# + offset)   | access at `r1 + offset`                                                        |

- All offsets are signed 16-bit.
- Immediates are signed 16-bit.
- `PC_next` — address of the next instruction.
- All data accesses are 32-bit words. Postincrement / predecrement step is ±4.
- Both operands with increment/decrement → assembler error.
- Immediate as destination → assembler error.

## Commands

`Format` — see [Instruction formats](#instruction-formats).

### Data movement
| Instruction  | Meaning           | Format | Flags |
|--------------|-------------------|--------|-------|
| mov dst, src | move: dst ← src   | F2     | NZ    |

### Logical
| Instruction  | Meaning                 | Format | Flags |
|--------------|-------------------------|--------|-------|
| clr dst      | clear: dst ← 0          | F1     |       |
| not dst      | bitwise not: dst ← ~dst | F1     |       |
| and dst, src | dst ← dst & src         | F2     |       |
| or dst, src  | dst ← dst \| src        | F2     |       |
| xor dst, src | dst ← dst ^ src         | F2     |       |

### Arithmetic
| Instruction                | Meaning                                | Format | Flags |
|----------------------------|----------------------------------------|--------|-------|
| inc dst                    | dst ← dst + 1                          | F1     |       |
| dec dst                    | dst ← dst − 1                          | F1     |       |
| neg dst                    | dst ← −dst                             | F1     |       |
| add dst, src               | dst ← dst + src                        | F2     |       |
| addc dst, src              | dst ← dst + src + C                    | F2     |       |
| sub dst, src               | dst ← dst − src                        | F2     |       |
| subc dst, src              | dst ← dst − src − C                    | F2     |       |
| mulh dst, src              | dst ← (dst × src) bits 63:32 (signed)  | F2     |       |
| umulh dst, src             | dst ← (dst × src) bits 63:32 (unsigned)| F2     |       |
| mull dst, src              | dst ← (dst × src) bits 31:0            | F2     |       |
| div dst, src               | dst ← dst / src (signed)               | F2     |       |
| mod dst, src               | dst ← dst % src (signed). -7 % 2 = 1   | F2     |       |
| cmp a, b                   | flags ← a − b                          | F2     |       |
| addm dst, src1, ..., srcN  | dst ← dst + src1 + ... + srcN          | FA     |       |

### Shifts
| Instruction    | Meaning                                        | Format | Flags |
|----------------|------------------------------------------------|--------|-------|
| lsl dst, count | dst ← dst << count (zeros fill from the right) | F2     |       |
| asl dst, count | alias for lsl, as they behave the same way     | F2     |       |
| lsr dst, count | dst ← dst >> count (zero-filled)               | F2     |       |
| asr dst, count | dst ← dst >> count (sign-filled)               | F2     |       |

### Stack
| Instruction         | Meaning                                         | Format | Flags |
|---------------------|-------------------------------------------------|--------|-------|
| push operand        | alias of `mov -(rsp), operand`                  | F2     |       |
| pushm r2, r1, r13   | push registers in list order                    | FP     |       |
| pop operand         | alias of `mov operand, (rsp)+`                  | F2     |       |
| popm r13, r1, r2    | pop into registers in list order                | FP     |       |
| call label          | push return address, PC ← label                 | ?      |       |
| ret                 | pop return address into PC                      | F0     |       |

Stack grows down.

`pushm` and `popm` both process the list in order, so restoring requires writing the list reversed:
```asm
pushm r1, r2   ; push r1, then r2
popm  r2, r1   ; pop into r2, then r1
```

### Control flow
| Instruction    | Semantics                                           | Format | Flags |
|----------------|-----------------------------------------------------|--------|-------|
| jump label     | PC ← label (absolute)                               | F1     |       |
| jump @label    | PC ← PC_next + offset                               | F1     |       |
| jz  label      | if ZF == 1:              PC ← PC_next + offset      | FB     |       |
| jnz label      | if ZF == 0:              PC ← PC_next + offset      | FB     |       |
| ja  label      | if CF == 0 && ZF == 0:   PC ← PC_next + offset      | FB     |       |
| jae label      | if CF == 0:              PC ← PC_next + offset      | FB     |       |
| jb  label      | if CF == 1:              PC ← PC_next + offset      | FB     |       |
| jbe label      | if CF == 1 \|\| ZF == 1: PC ← PC_next + offset      | FB     |       |
| jl  label      | if SF != OF:             PC ← PC_next + offset      | FB     |       |
| jle label      | if ZF == 1 \|\| SF != OF: PC ← PC_next + offset     | FB     |       |
| jg  label      | if ZF == 0 && SF == OF:  PC ← PC_next + offset      | FB     |       |
| jge label      | if SF == OF:             PC ← PC_next + offset      | FB     |       |
| halt           | stops the execution                                 | F0     |       |

Conditional jumps are always PC-relative. `jump` allows only absolute and PC-relative modes.
`offset` — signed 16-bit, computed by the translator: `addr(label) − PC_next`.

## Machine model

- Word size: 4 bytes.
- Memory: byte-addressed, single-port, one access per cycle. Shared by code and data (von Neumann).
- Registers: r0–r15, 4 bytes each.
- Special registers: PC, PS (interrupt state + N, Z, V, C).
- PC holds a byte address. Instructions are whole words; PC advances by 4 per fetched word.
- Strings: one character per machine word.
- **Code and data SHOULD be aligned to 4.** Unaligned accesses (including `.org` to an unaligned address in code) are allowed but slower: an access that spans two cache lines bypasses the cache and goes to memory.
- Cache: write-through. An unaligned write goes to memory and invalidates both affected lines.
- Memory model assumption: RAM can read/write any 4 consecutive bytes in one access.

## Errors

- Invalid source code → assembler error.
- Invalid encoding at runtime → CPU halts with an error.

## Instruction formats

All padding and unused bits are filled with 0 by the translator.
Operands are placed **from bit 0 upward**: each one takes the lowest free bits of its area.
Signed fields (immediate, PC-relative offset, register-relative offset) are sign-extended to 32 bits by the CU.
Register-relative field: `reg#` in its bits [19:16], `offset` in its bits [15:0].

### F0 — 0 operands
```
[31:26 OPCODE][25:0 0]
```
Size: 1 word.

### F1 — 1 operand
```
word1: [31:26 OPCODE][25:23 MODE][22:0 operand, from bit 0]
ext:   [31:0 address]                      ; absolute mode only
```
23 free bits: every mode except absolute fits inline.
Size: 1–2 words.

### F2 — 2 operands
```
word1: [31:26 OPCODE][25:23 MODE1][22:20 MODE2][19:0 inline operands, from bit 0]
ext:   [31:0 operand, right-aligned]       ; 0–2 words
```
OPCODE + 2 × MODE = 12 bits, so 20 bits of word 1 are free for inline operands.

Placement is greedy:
1. Try to put operand 1 inline; then try operand 2 in the remaining bits.
2. Operands that did not fit go to extension words, in operand order.
3. Absolute address never fits inline.

So operand 2 may be inline while operand 1 is in an extension word:
```
word1: [OPCODE | MODE1 | MODE2 | OPERAND2]
word2: [OPERAND1]
```

Sizes:
| Operand sizes (bits)                 | Words |
|--------------------------------------|-------|
| 4 + 4, 4 + 16                        | 1     |
| 4 + 20, 16 + 16, 16 + 20, 20 + 20    | 2     |
| absolute + any non-absolute          | 2     |
| absolute + absolute                  | 3     |

### FB — conditional branch
```
[31:26 OPCODE][25:16 0][15:0 offset, signed]
```
No MODE field. Size: 1 word.

### FP — `pushm` / `popm`
```
word1: [31:26 OPCODE][25:22 COUNT][21:20 0][19:0 reg0..reg4, 4 bits each, reg0 at [3:0]]
ext:   [31:0 next 8 registers, 4 bits each, from bit 0]
```
COUNT = number of registers − 1 (1–16 registers).
| Registers | Words |
|-----------|-------|
| 1–5       | 1     |
| 6–13      | 2     |
| 14–16     | 3     |

### FA — `addm`
```
word1:    [31:26 OPCODE][25:18 COUNT][17:0 0]
operand:  [31:29 MODE][28:0 operand, from bit 0]
absolute: [31:29 MODE][28:0 0]
          [31:0 address]
```
- First operand word(s) — destination, encoded like any other operand.
- COUNT = number of operands − 1, 8 bits.
- Each operand takes 1 word; absolute takes 2 (mode word + address word).

Example `addm r1, (r2+8), (0x2000)`:
```
word1: [OPCODE | COUNT=2]
word2: [MODE=register | r1]               ; dst, fits in 1 word
word3: [MODE=reg-relative | r2, 8]         ; fits in 1 word
word4: [MODE=absolute | 0]                 ; split: mode word
word5: [0x2000]                            ;        address word
```

## Field sizes

| Field                                                      | Size                                  |
|------------------------------------------------------------|---------------------------------------|
| OPCODE                                                     | 6 bits                                |
| MODE                                                       | 3 bits                                |
| register (register, indirect, postincrement, predecrement) | 4 bits                                |
| immediate                                                  | 16 bits, signed                       |
| PC-relative offset                                         | 16 bits, signed, in bytes             |
| register-relative                                          | 4 + 16 bits = 20 bits (reg# + offset) |
| absolute address                                           | 32 bits                               |
| branch offset                                              | 16 bits, signed, in bytes             |
| COUNT (`pushm`/`popm`)                                     | 4 bits                                |
| COUNT (`addm`)                                             | 8 bits                                |

## Immediates

- Always 16 bits, signed. There are no 32-bit immediates.
- Larger constants are either stored in the data section and loaded from memory, or built with shift/or.
- `mov r1, label` loads the address of `label` into `r1`. If the address does not fit into the immediate field → error.