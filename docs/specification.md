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
- Immediate as destination → assembler error, except for io port numbers.
- `cmp/test a, b`: `a` cannot be an immediate → assembler error.
- `jump` / `call` do not use operand modes, see [Jump targets](#jump-targets).
-  Syntax rule: bare — address, parentheses — memory.

## Commands

`Format` — see [Instruction formats](#instruction-formats).

`Flags` — flags set from the result; flags not listed are unchanged. `X ← 0` — flag is cleared.
- N — result bit 31; Z — result is 0; V — signed overflow; C — unsigned carry out (for `sub`/`subc`/`cmp`/`neg` — borrow).
- `addm`: flags reflect the final result only (N, Z)
- Shifts: C — last bit shifted out.

### Data movement
| Instruction  | Meaning           | Format | Flags |
|--------------|-------------------|--------|-------|
| mov dst, src | move: dst ← src   | F2     | NZ    |

### Logical
| Instruction  | Meaning                 | Format | Flags       |
|--------------|-------------------------|--------|-------------|
| clr dst      | clear: dst ← 0          | F1     | NZ; V,C ← 0 |
| not dst      | bitwise not: dst ← ~dst | F1     | NZ; V,C ← 0 |
| and dst, src | dst ← dst & src         | F2     | NZ; V,C ← 0 |
| or dst, src  | dst ← dst \| src        | F2     | NZ; V,C ← 0 |
| xor dst, src | dst ← dst ^ src         | F2     | NZ; V,C ← 0 |

### Arithmetic
| Instruction               | Meaning                                 | Format | Flags       |
|---------------------------|-----------------------------------------|--------|-------------|
| inc dst                   | dst ← dst + 1                           | F1     | NZV         |
| dec dst                   | dst ← dst − 1                           | F1     | NZV         |
| neg dst                   | dst ← −dst                              | F1     | NZVC        |
| add dst, src              | dst ← dst + src                         | F2     | NZVC        |
| addc dst, src             | dst ← dst + src + C                     | F2     | NZVC        |
| sub dst, src              | dst ← dst − src                         | F2     | NZVC        |
| subc dst, src             | dst ← dst − src − C                     | F2     | NZVC        |
| mulh dst, src             | dst ← (dst × src) bits 63:32 (signed)   | F2     | NZ          |
| umulh dst, src            | dst ← (dst × src) bits 63:32 (unsigned) | F2     | NZ          |
| mull dst, src             | dst ← (dst × src) bits 31:0             | F2     | NZ          |
| div dst, src              | dst ← dst / src (signed, truncated)     | F2     | NZ          |
| mod dst, src              | dst ← dst % src (signed). -7 % 2 = -1   | F2     | NZ          |
| cmp a, b                  | flags ← a − b                           | F2     | NZVC        |
| test a, b                 | flags ← a & b                           | F2     | NZ; V,C ← 0 |
| addm dst, src1, ..., srcN | dst ← dst + src1 + ... + srcN           | FA     | NZ          |

Division by 0 → CPU halts with an error.

### Shifts
| Instruction    | Meaning                                        | Format | Flags |
|----------------|------------------------------------------------|--------|-------|
| lsl dst, count | dst ← dst << count (zeros fill from the right) | F2     | NZC   |
| asl dst, count | alias for lsl, as they behave the same way     | F2     | NZC   |
| lsr dst, count | dst ← dst >> count (zero-filled)               | F2     | NZC   |
| asr dst, count | dst ← dst >> count (sign-filled)               | F2     | NZC   |

`count mod 32` is used.

### Stack
| Instruction       | Meaning                                           | Format | Flags |
|-------------------|---------------------------------------------------|--------|-------|
| push operand      | alias of `mov -(rsp), operand`                    | F2     | NZ    |
| pushm r2, r1, r13 | push registers in list order                      | FP     | —     |
| pop operand       | alias of `mov operand, (rsp)+`                    | F2     | NZ    |
| popm r13, r1, r2  | pop into registers in list order                  | FP     | —     |
| call label        | push PC_next, PC ← label                          | FJ     | —     |
| call @label       | push PC_next, PC ← PC_next + offset               | FJ     | —     |
| ret               | pop return address into PC                        | F0     | —     |
| iret              | pop return address into PC, restore program state | F0     | —     |

Stack grows down.

`pushm` and `popm` both process the list in order, so restoring requires writing the list reversed:
```asm
pushm r1, r2   ; push r1, then r2
popm  r2, r1   ; pop into r2, then r1
```

### IO
| Instruction   | Semantics      | Format | Flags       |
|---------------|----------------|--------|-------------|
| in dst, port  | dst ← IO[port] | F2     | NZ; V,C ← 0 |
| out port, src | IO[port] ← src | F2     | —           |

- `port` — port number, any source mode (usually an immediate: `in r1, 0`).
- Port numbers are a separate address space, not memory addresses.
- Ports:
    - 0 — input: holds the last character from the input schedule; its arrival raises the interrupt.
    - 1 — output: writing appends a character to the output buffer.
- Reading/writing a nonexistent port → CPU halts with an error.

### Control flow
| Instruction | Semantics                                     | Format | Flags |
|-------------|-----------------------------------------------|--------|-------|
| jump label  | PC ← label                                    | FJ     | —     |
| jump @label | PC ← PC_next + offset                         | FJ     | —     |
| jz  label   | if Z == 1:              PC ← PC_next + offset | FB     | —     |
| jnz label   | if Z == 0:              PC ← PC_next + offset | FB     | —     |
| ja  label   | if C == 0 && Z == 0:   PC ← PC_next + offset  | FB     | —     |
| jae label   | if C == 0:              PC ← PC_next + offset | FB     | —     |
| jb  label   | if C == 1:              PC ← PC_next + offset | FB     | —     |
| jbe label   | if C == 1 \|\| Z == 1: PC ← PC_next + offset  | FB     | —     |
| jl  label   | if N != V:             PC ← PC_next + offset  | FB     | —     |
| jle label   | if Z == 1 \|\| N != V: PC ← PC_next + offset  | FB     | —     |
| jg  label   | if Z == 0 && N == V:  PC ← PC_next + offset   | FB     | —     |
| jge label   | if N == V:             PC ← PC_next + offset  | FB     | —     |
| ei          | enables interrupts                            | F0     | IE    |
| di          | disables interrupts                           | F0     | IE    |
| halt        | stops the execution                           | F0     | —     |

Conditional jumps are always PC-relative.
`offset` — signed 16-bit, computed by the translator: `addr(label) − PC_next`.

### Jump targets

`jump` and `call` have their own target forms:

| Syntax   | Kind         | Size                      |
|----------|--------------|---------------------------|
| `label`  | absolute     | 32-bit address, ext word  |
| `@label` | PC-relative  | 16-bit signed offset      |

## Interrupts

One interrupt source: input (port 0).

### Memory layout
| Address | Contents                                       |
|---------|------------------------------------------------|
| 0x0     | reset vector — address of the program start     |
| 0x4     | interrupt vector — address of the handler       |

Both vectors are filled by the linker: `mem[0x0] ← addr(_start)`, `mem[0x4] ← addr(_handler)`.
- 0x0–0x7 are reserved; code and data start at 0x8. `.org` into this range → assembler error.
- No `_start` → link error.
- No `_handler` → vector = 0; an interrupt taken with `IE = 1` → CPU halts with an error.

### Reset
- `PC ← mem[0x0]`, `IE ← 0`, `irq_pending ← 0`.
- `rsp` is not initialised.

### Request
- When a character from the input schedule arrives: `IO[0] ← char`, `irq_pending ← 1`.
- `irq_pending` is a single bit — requests are not queued.
- If a character arrives while `irq_pending = 1`, `IO[0]` is overwritten: the previous character is lost, still one pending request.

### Entry
Checked after each instruction completes (never in the middle of one):
```
if irq_pending == 1 and IE == 1:
    irq_pending ← 0
    push PC
    push PS
    IE ← 0
    PC ← mem[0x4]
```

### Return
`iret`: pop PS (restores IE and NZVC), then pop PC.

### Interrupt during interrupt
- The handler runs with `IE = 0`. A request arriving meanwhile is latched in `irq_pending` and serviced right after `iret`.
- The handler may execute `ei` to allow nesting: PC and PS are on the stack, so nested entries are safe.

## Machine model

- Word size: 4 bytes.
- Memory: byte-addressed, single-port, one access per cycle. Shared by code and data (von Neumann).
- Registers: rsp (reg #0), r1–r15, 4 bytes each. All 16 are addressable by the 4-bit register field.
- Special registers: PC, PS (IE + N, Z, V, C).
- PC holds a byte address. Instructions are whole words; PC advances by 4 per fetched word.
- Strings: one character per machine word.
- **The code and data SHOULD be aligned to 4.** Otherwise, code works but slower - access that spans 2 cache lines,
  goes directly to memory, bypassing cache.
- RAM can read/write any 4 consecutive bytes in one access.
- Physical memory is set in config.
- Labels used as immediates must be < 0x8000, since immediate value is 16 bits signed integer.

## Cache
- Write-through. An unaligned write goes to memory and invalidates both affected lines.
- Holds 16 lines, with each line of 16 bits. 
- The replacement algorithm is tree-based pseudo-LRU.

### Cache access cases

| Access              | Action                                                                                             | Ticks | Tick notes                                   |
|---------------------|----------------------------------------------------------------------------------------------------|-------|----------------------------------------------|
| Aligned read, hit   | Return word from `data_array` (picked by `addr[3:2]`)                                              | 1     | cache access                                 |
| Aligned read, miss  | Pick victim (invalid first, then tree-PLRU), burst-fill line, write tag + valid, return word       | 14    | 10 first word + 3×1 burst + 1 cache read     |
| Unaligned read      | Bypass: single RAM read, cache untouched                                                           | 10    | one RAM access                               |
| Aligned write, hit  | Write word to `data_array` and RAM (write-through)                                                 | 10    | RAM write; cache write in parallel           |
| Aligned write, miss | Write to RAM only (no-allocate)                                                                    | 10    | one RAM access                               |
| Unaligned write     | Write to RAM, invalidate every cached line it touches (1 or 2)                                     | 10–12 | RAM write + 1 tick per invalidated line      |

Reset clears all valid bits.

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
[31:26 OPCODE][25:0 padding]
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
[31:26 OPCODE][25:16 padding][15:0 offset, signed]
```
Always relative to PC. Size: 1 word.

### FJ — `jump` / `call`
```
word1: [31:26 OPCODE][25 K][24:16 padding][15:0 offset, signed]
ext:   [31:0 address]                      ; K = 0 only
```
- K = 1 — PC-relative: `PC ← PC_next + offset`. Size: 1 word.
- K = 0 — absolute: offset field is 0, address is in the extension word. Size: 2 words.

`PC_next` — address of the next instruction.

### FP — `pushm` / `popm`
```
word1: [31:26 OPCODE][25:22 COUNT][21:20 padding][19:0 reg0..reg4, 4 bits each, reg0 at [3:0]]
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
word1:  [31:26 OPCODE][25:23 COUNT][22:21 padding][20:0 MODE1..MODE7, 3 bits each, MODE1 at [2:0]]
value:  [31:0 operand values, packed from bit 0]      ; 1+ words
```
- COUNT = number of operands − 1, including the destination. 1–7 operands; COUNT = 7 → CPU halts with an error.
- Unused mode slots are 0.
- Operand 1 is the destination.
- Values are packed **in operand order**, each from the lowest free bit of the current value word (next-fit):
    - if a value fits in the rest of the current word, it goes there;
    - otherwise it starts the next word; the rest of the current word is padding.
- A value never crosses a word boundary.
- Word 1 holds only modes, so every `addm` spans at least 2 words.
  Examples:
```
addm r1, r2, r3, r4                 ; 2 words
word1: [OPCODE | COUNT=3 | modes: reg, reg, reg, reg]
word2: [r4 | r3 | r2 | r1]        
 
addm r1, (0x2000), (0x3000)         ; 4 words
word1: [OPCODE | COUNT=2 | modes: reg, abs, abs]
word2: [r1]                                   
word3: [0x2000]
word4: [0x3000]
 
addm r1, (r2+8), 100                ; 3 words
word1: [OPCODE | COUNT=2 | modes: reg, reg-rel, imm]
word2: [r2, 8 | r1]                           
word3: [100]                                  
```

## Opcodes

| Opcode | Binary | Instruction | Format |
|--------|--------|-------------|--------|
| 0x00   | 000000 | — (reserved, invalid) | —      |
| 0x01   | 000001 | halt        | F0     |
| 0x02   | 000010 | ret         | F0     |
| 0x03   | 000011 | iret        | F0     |
| 0x04   | 000100 | ei          | F0     |
| 0x05   | 000101 | di          | F0     |
| 0x06   | 000100 | pass_b      | F0     |
| 0x08   | 001000 | clr         | F1     |
| 0x09   | 001001 | not         | F1     |
| 0x0A   | 001010 | inc         | F1     |
| 0x0B   | 001011 | dec         | F1     |
| 0x0C   | 001100 | neg         | F1     |
| 0x0E   | 001110 | jump        | FJ     |
| 0x0F   | 001111 | call        | FJ     |
| 0x10   | 010000 | pushm       | FP     |
| 0x11   | 010001 | popm        | FP     |
| 0x12   | 010010 | addm        | FA     |
| 0x18   | 011000 | jz          | FB     |
| 0x19   | 011001 | jnz         | FB     |
| 0x1A   | 011010 | ja          | FB     |
| 0x1B   | 011011 | jae         | FB     |
| 0x1C   | 011100 | jb          | FB     |
| 0x1D   | 011101 | jbe         | FB     |
| 0x1E   | 011110 | jl          | FB     |
| 0x1F   | 011111 | jle         | FB     |
| 0x20   | 100000 | jg          | FB     |
| 0x21   | 100001 | jge         | FB     |
| 0x28   | 101000 | mov         | F2     |
| 0x29   | 101001 | and         | F2     |
| 0x2A   | 101010 | or          | F2     |
| 0x2B   | 101011 | xor         | F2     |
| 0x2C   | 101100 | add         | F2     |
| 0x2D   | 101101 | addc        | F2     |
| 0x2E   | 101110 | sub         | F2     |
| 0x2F   | 101111 | subc        | F2     |
| 0x30   | 110000 | mulh        | F2     |
| 0x31   | 110001 | umulh       | F2     |
| 0x32   | 110010 | mull        | F2     |
| 0x33   | 110011 | div         | F2     |
| 0x34   | 110100 | mod         | F2     |
| 0x35   | 110101 | cmp         | F2     |
| 0x36   | 110110 | test        | F2     |
| 0x37   | 110111 | lsl         | F2     |
| 0x38   | 111000 | lsr         | F2     |
| 0x39   | 111001 | asr         | F2     |
| 0x3A   | 111010 | in          | F2     |
| 0x3B   | 111011 | out         | F2     |

- Opcodes are grouped by format with gaps, so each format occupies its own range.
- Unlisted opcodes → CPU halts with an error. 0x00 is invalid on purpose: running into zeroed memory stops with an error.
- Aliases have no opcode: `asl` = `lsl`, `push` = `mov -(rsp), …`, `pop` = `mov …, (rsp)+`.
- The opcode alone determines the format.

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
| branch / jump offset                                       | 16 bits, signed, in bytes             |
| K (`jump`/`call` kind)                                     | 1 bit                                 |
| COUNT (`pushm`/`popm`)                                     | 4 bits                                |
| COUNT (`addm`)                                             | 3 bits                                |

## Immediates

- Always 16 bits, signed. There are no 32-bit immediates.
- Larger constants are either stored in the data section and loaded from memory, or built with shift/or.
- `mov r1, label` loads the address of `label` into `r1`. If the address does not fit into the immediate field → error.
