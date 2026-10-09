# Instruction set

## Operand modes
| Mode                 | Syntax    | Notes                                                        |
|----------------------|-----------|--------------------------------------------------------------|
| register             | `r1`      |                                                              |
| direct memory        | `(0x120)` |                                                              |
| indirect memory      | `(r1)`    |                                                              |
| postincrement memory | `(r1)+`   | access at `r1`, then `r1 ← r1 + size`                        |
| predecrement memory  | `-(r1)`   | `r1 ← r1 − size`, then access at `r1`                        |
| immediate value      | `120`     | source only; a bare label is its address                     |
| pc-relative          | `(@label)`| encoded as register relative with base `pc`; assembler computes `label − pc` |
| register relative    | `(r1+10)` |                                                              |

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
| Instruction      | Meaning                          |
|------------------|----------------------------------|
| inc dst        | dst ← dst + 1                    |
| dec dst        | dst ← dst − 1                    |
| neg dst        | dst ← −dst                       |
| add dst, src   | dst ← dst + src                  |
| addc dst, src   | dst ← dst + src + C              |
| sub dst, src   | dst ← dst − src                  |
| subc dst, src  | dst ← dst − src − C              |
| mulh dst, src  | dst ← (dst × src) bits 63:32(signed)         |
| mull dst, src  | dst ← (dst × src) bits 31:0(signed)          |
| div dst, src   | dst ← dst / src (signed)         |
| mod dst, a, b  | dst ← a % b (signed)             |
| cmp a, b       | flags ← a − b.                   |

### Shifts
| Instruction    | Meaning |
|----------------|---------|
| asl dst, count | dst ← dst << count (zeros fill from the right) |
| asr dst, count | dst ← dst >> count (sign-filled) |
| lsr dst, count | dst ← dst >> count (zero-filled) |

### Stack
| Instruction   | Meaning |
|---------------|---------|
| push operand  | alias of mov -(rsp),  operand|
| pop operand   | alias of mov operand, (rsp)+        |
| call label    | pushes return address to stack, changes pc        |
| ret           | pops return address from stack and sets in pc|
