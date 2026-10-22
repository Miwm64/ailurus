# Instruction set

## Operand modes
| # | Mode               | Syntax                  | Notes                                                                          |
|---|--------------------|-------------------------|--------------------------------------------------------------------------------|
| 0 | register           | `r1`                    | operand is the register itself                                                 |
| 1 | direct memory      | `(0x120)`, `(label)`    | access at the absolute address stored in the instruction; labels are relocated |
| 2 | PC-relative memory | `(@label)`              | access at `PC_next + offset`; assembler computes `offset = label − PC_next`    |
| 3 | register indirect  | `(r1)`                  | access at `r1`                                                                 |
| 4 | postincrement      | `(r1)+`                 | access at `r1`, then `r1 ← r1 + 4`                                             |
| 5 | predecrement       | `-(r1)`                 | `r1 ← r1 − 4`, then access at `r1`                                             |
| 6 | immediate          | `120`, `0x120`, `label` | value stored in the instruction; source only; a bare label is its address      |
| 7 | register relative  | `(r1+10)`               | access at `r1 + offset`;                                                       |

offset is a signed 12-bit integer
PC_next — program counter, address of the next instruction.
All memory accesses are 32-bit words.
TODO: evaluation order when both operands have side effects, e.g. `mov (r1)+, (r1)+`.
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
| asl dst, count | dst ← dst << count (zeros fill from the right) |
| asr dst, count | dst ← dst >> count (sign-filled)               |
| lsr dst, count | dst ← dst >> count (zero-filled)               |

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

[//]: # (## Encoding)
