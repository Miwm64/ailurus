![CI](https://github.com/Miwm64/ailurus/actions/workflows/ci.yml/badge.svg)

# Lab 4: assembler, linker and processor model

- **Student:** Добкес Михаил Константинович, group P3206, ISU number: 398893
- **Variant:**

  ```text
  asm | cisc | neum | hw | trap | port | pstr | cache | alg1
  ```

- **Implementation language:** Go.

This README is the short report which follows rules from lab4.md. 
The full description of the project (every instruction, signal, format and corner case)
is in **[docs/specification.md](docs/specification.md)**.

| Option  | How it is implemented                                                                                    |
|---------|----------------------------------------------------------------------------------------------------------|
| `asm`   | assembler with labels, sections, `.org`, macros, `#define`, `#ifdef`; separate translation + linker      |
| `cisc`  | 8 addressing modes, memory-to-memory operations, instructions of 1–8 words, `addm` with up to 7 operands |
| `neum`  | one memory for code and data                                                                             |
| `hw`    | hardwired control unit with a step register                                                              |
| `trap`  | input comes through interrupts on a tick schedule; the handler is written in asm                         |
| `port`  | separate `in` / `out` instructions with port numbers                                                     |
| `pstr`  | a string is a length word, then one word per character                                                   |
| `cache` | 16-line fully associative cache, tree-PLRU, write-through                                                |
| `alg1`  | `is_prime`, `format_string`                                                                              |


## Programming language

### Syntax

Described in EBNF:

```text
(* --- structure --- *)
<program>   ::= { <line> | <macro> | <define> | <condition> | <directive> }
<end>       ::= [ <comment> ] <eol>
<eol>       ::= "\n" | <EOF>

(* --- preprocessor --- *)
<macro>     ::= "#macro" <name> "," <non_neg_number> { "\n" }
                "{" <end>
                { <raw_line> }
                "}" <end>
<raw_line>  ::= ? any characters except newline, or unopened "}" ? <eol>

<define>    ::= "#define" <name> [ <value> ] <eol>

<condition> ::= ( "#ifdef" | "#ifndef" ) <name> <end>
                <program>
                [ "#else" <end> <program> ]
                "#endif" <end>

(* --- directives --- *)
<directive> ::= ( ".section" ( "data" | "text" )
                | ".org" <non_neg_number>
                | ".import" <name>
                | ".export" <name>
                ) <end>

(* --- lines --- *)
<line>        ::= [ <label> ] [ <data> | <instruction> ] <end>
<label>       ::= <name> ":"

<data>        ::= ".word" <word> { "," <word> }
                | ".pstr" <string>
                | ".ref" <name>
<word>        ::= <number> | <char>

<instruction> ::= <command> [ <operand> { "," <operand> } ]
<command>     ::= ? any mnemonic from the instruction list, or a macro name ?

<operand>     ::= <register>                                        (* register *)
                | "(" ( <non_neg_number> | <name> ) ")"             (* direct *)
                | "(" "@" <name> ")"                                (* PC-relative *)
                | "(" <register> ")"                                (* indirect *)
                | "(" <register> ")" "+"                            (* postincrement *)
                | "-" "(" <register> ")"                            (* predecrement *)
                | "(" <register> ( "+" | "-" ) <non_neg_number> ")" (* register-relative *)
                | <number> | <name> | <char>                        (* immediate *)
                | "@" <name>                                        (* jump PC-relative *)
<register>    ::= "rsp" | "r1" | "r2" | ... | "r15"

(* --- tokens --- *)
<name>           ::= ( <letter> | "_" ) { <letter> | <digit> | "_" }
<number>         ::= [ "-" ] <non_neg_number>
<non_neg_number> ::= <digit> { <digit> } | "0x" <hex_digit> { <hex_digit> }
<char>           ::= '"' <str_char> '"'
<string>         ::= '"' { <str_char> } '"'
<str_char>       ::= ? printable ASCII character 0x20–0x7E, except "\" and '"' ?
                   | "\" ( "n" | "t" | "\" | '"' | "0" )
<comment>        ::= ";" { ? any UTF-8 character except newline ? }
<value>          ::= ? any characters up to the end of the line ?

<letter>, <digit>, <hex_digit> have their usual meaning: a–z A–Z; 0–9; 0–9 a–f A–F.
<EOF> is the end of input.
```

Example:

```asm
.export _start
.export _handler
.import print
.import inputPush

#define STACK_TOP 0x1000

_start:
    mov rsp, STACK_TOP
    mov r1, msg
    call print          ; stdlib: print a pstr
    halt

_handler:               ; every input character raises an interrupt
    call inputPush      ; stdlib: put the character into the input buffer
    iret

.section data
msg: .pstr "Hello, world!\n"
```

### Semantics

- **Evaluation.** Instructions run one by one, starting from `_start`. Every operand
  is computed by its addressing mode when the instruction runs.
- **Compile time.** `#define`, macros and `#ifdef` are text substitution, done by the preprocessor in one pass.
  A label used before its definition is recorded as a fixup. At the end of the file the assembler resolves
  PC-relative fixups in the same file and section; everything else (absolute addresses,
  imported labels, labels in the other section) becomes a relocation for the linker.
- **Scope.** Defines and macros are local for the file. Labels are local by default but can be used in other files by
`.export label` and `.import label`
- **Types.** There are no types: every value is a 32-bit machine word. The instruction chooses signed or unsigned
  meaning (`jl` vs `jb`, `mulh` vs `umulh`).
- **Literals.** Positive and negative decimals `42`, `-7`, hex `0x2A`, characters `"a"` (ASCII code). Strings are allowed only in
  `.pstr`. Escapes: `\n \t \\ \" \0`.
- **Sections.** `text`  holds only instructions, `data` holds only data. `.org N` sets the offset inside
  the current file's part of the section.
- **Entry points.** `_start` is required, `_handler` is optional. Both must be exported.
  `rsp` is not initialized at reset, so `_start` must set it before the first `call`, `push` or `ei`.
- **Standard library** (linked by default): `print`, `printChar`, `readChar`, `readLine`, `strEq`, `strConcat`,
  `strContains`, `intToStr`, `strToInt` and `inputPush` for the interrupt handler.

## Memory organization

- Von Neumann: one single-port memory for code and data, byte addressing, 32-bit words, little-endian.
- Every access reads or writes 4 bytes. Unaligned access works, but it bypasses the cache and should not be used.
- Size is set in the simulator config (`memory_size`, 4096 bytes by default).
- Registers for the programmer: `rsp`, `r1`–`r15` (32 bits). PC and PS (IE + NZVC) are not addressable.
- IO ports are a separate address space: port 0 — input, port 1 — output.

Code and data share one memory. The linker builds the image in this order; the stack starts at the address
`_start` puts into `rsp` and grows down toward `data`:
```text
          Memory
+-------------------------------+
| 0x0  : reset vector → _start  |
| 0x4  : irq vector → _handler  |
| 0x8  : text, file 1           |
|        text, file 2           |
|        ...                    |
|        text, stdlib           |
|        data, file 1           |
|        ...                    |
|        data, stdlib           |
|        free                   |
|        ...                    |
| top  : stack (grows down)     |
+-------------------------------+
```

How each kind of program entity is handled when the program is built and when it runs:

| What        | Compile time                                                                                                                                             | Run time                                                                                    |
|-------------|----------------------------------------------------------------------------------------------------------------------------------------------------------|---------------------------------------------------------------------------------------------|
| Literal     | fits in $[-2^{15},\ 2^{15}-1]$ → immediate inside the instruction;<br/> bigger → translator error, the programmer needs to put it in `data` with `.word` | immediate comes with the instruction fetch; a value in `data` is read with a memory operand |
| Constant    | `#define`: the preprocessor replaces it with its text, no memory                                                                                         | —                                                                                           |
| Variable    | `.word` in `data`: static, the linker gives it the final address                                                                                         | read and written with any memory mode                                                       |
| String      | `.pstr` in `data`: length word, then one word per character                                                                                              | read word by word, e.g. with `(r1)+`                                                        |
| Instruction | encoded in `text`, 1–8 words; the linker gives final addresses                                                                                           | fetched word by word, PC += 4                                                               |
| Procedure   | a label in `text`; the `call` target is resolved by the assembler or the linker                                                                          | `call` pushes `ret_addr`, `ret` pops it                                                     |
| Interrupt   | `_handler` in `text`; the linker writes its address to the vector at `0x4`                                                                               | entry pushes PC and PS, `iret` pops them                                                    |

- Values in `data` are placed in source order, word after word; each file's part follows the previous file's.
  A multi-word value goes low word first (64-bit: `addr` — bits 31:0, `addr + 4` — bits 63:32).
- There is no register allocation: the programmer decides what to keep in registers. If there are not
  enough registers, values are kept on the stack or in `data`.
- The stack holds return addresses, saved registers, PC and PS of an interrupt, arguments after the 4th and
  local values (`push`, `(rsp+n)`). It is the only dynamic memory: there is no heap.
- Addressing is by byte; the 8 operand addressing modes are listed in [Processor features](#processor-features).

## Instruction set

### Processor features

- 16 registers: `rsp` (#0) and `r1`–`r15`. PS holds the flags NZVC and the interrupt enable bit IE.
- Word: 32 bits. There are no data types: every value is a word, signed (two's complement) or unsigned
  by instruction. A character is one byte, stored in its own word.
- IO is port-mapped: `in` / `out` with an immediate port number. Port 0 holds the last input character
  (its arrival raises the interrupt), port 1 appends a character to the output.
- Control flow: `jump` / `call` take a PC-relative or an absolute target; conditional branches are always
  PC-relative and check NZVC (`jl`, `jg`, … — signed, `jb`, `ja`, … — unsigned).
- Most instructions take any addressing mode for both operands, including memory for both at once:

| # | Mode              | Syntax         | Field (bits) | Effective address / value  |
|---|-------------------|----------------|--------------|----------------------------|
| 0 | register          | `r1`           | 4            | the register itself        |
| 1 | direct            | `(0x120)`      | 32           | the absolute address       |
| 2 | PC-relative       | `(@label)`     | 16           | `operand_pc_next + offset` |
| 3 | indirect          | `(r1)`         | 4            | `r1`                       |
| 4 | postincrement     | `(r1)+`        | 4            | `r1`, then `r1 ← r1 + 4`   |
| 5 | predecrement      | `-(r1)`        | 4            | `r1 ← r1 − 4`, then `r1`   |
| 6 | immediate         | `120`, `label` | 16           | the value itself           |
| 7 | register-relative | `(r1+8)`       | 20           | `r1 + offset`              |

`operand_pc_next` is the address right after the word that holds the offset.

### Instructions

A short overview by group. The full command description and flags change is in
[Commands](docs/specification.md#commands) in the specification.

| Group        | Instructions                                                                                    | Format     |
|--------------|-------------------------------------------------------------------------------------------------|------------|
| Data         | `mov`; aliases `push x` = `mov -(rsp), x`, `pop x` = `mov x, (rsp)+`                            | F2         |
| Logic        | `clr`, `not`; `and`, `or`, `xor`, `test`                                                        | F1; F2     |
| Arithmetic   | `inc`, `dec`, `neg`; `add`, `addc`, `sub`, `subc`, `mull`, `mulh`, `umulh`, `div`, `mod`, `cmp` | F1; F2     |
| Multi-add    | `addm dst, src1, ..., src6` — `dst ← dst + src1 + ... + src6`                                   | FA         |
| Shifts       | `lsl` (`asl`), `lsr`, `asr`                                                                     | F2         |
| Stack        | `pushm`, `popm` (1–16 registers); `call`; `ret`, `iret`                                         | FP; FJ; F0 |
| IO           | `in dst, port`, `out port, src`                                                                 | F2         |
| Control flow | `jump`; `jz jnz ja jae jb jbe jl jle jg jge`; `ei`, `di`, `halt`                                | FJ; FB; F0 |

- Flags are set from the result; every instruction has its own set.
- `div` / `mod` by zero, a wrong port and an access outside memory stop the machine with a fault.

### Interrupts

- A character from the input schedule arrives: `IO[0] ← char`, `irq_pending ← 1`.
- At the last tick of each instruction, if `irq_pending` and IE are both 1: push PC, push PS, IE ← 0,
  PC ← mem[0x4]. `iret` pops PS, then PC.
- **Interrupt during interrupt:** the handler runs with IE = 0. A new request is kept in `irq_pending`
  (one bit, no queue) and is checked after `iret`; a character that comes before the previous one was read
  overwrites it. Programmer can run `ei` to allow nesting.

### Encoding

Opcode is 6 bits, the mode of an operand is 3 bits. Word 1 always holds the opcode and all modes, so the CU
knows the whole layout after the first fetch. Operand values are packed **next-fit** from bit 0: a value goes into
the rest of the current word if it fits, otherwise into the next word.

| Format | Instructions       | Word 1                                                                  | Size |
|--------|--------------------|-------------------------------------------------------------------------|------|
| F0     | `halt`, `ret`, ... | `[31:26 OPCODE][25:0 0]`                                                | 1    |
| F1     | `inc`, `clr`, ...  | `[31:26 OPCODE][25:23 MODE][22:0 operand]`                              | 1–2  |
| F2     | `mov`, `add`, ...  | `[31:26 OPCODE][25:23 MODE1][22:20 MODE2][19:0 operands]`               | 1–3  |
| FB     | `jz`, `jl`, ...    | `[31:26 OPCODE][25:16 0][15:0 offset]`                                  | 1    |
| FJ     | `jump`, `call`     | `[31:26 OPCODE][25 K][24:16 0][15:0 offset]`, K = 0 → address in word 2 | 1–2  |
| FP     | `pushm`, `popm`    | `[31:26 OPCODE][25:22 COUNT][19:0 5 registers]`, then 8 per word        | 1–3  |
| FA     | `addm`             | `[31:26 OPCODE][25:23 COUNT][20:0 7 modes]`, values from word 2         | 2–8  |

Signed fields are sign-extended; register-relative is `reg#` in [19:16] + offset in [15:0]. Opcodes of all
instructions are in [Opcodes](docs/specification.md#opcodes).

Example from the [cisc test](golden_test/testdata/cisc.yml), one instruction of 7 words:

```text
000000C8 - 4B1BFFF8 - addm r6 , ( r4 + 4 ) , ( r4 + 8 ) , ( r4 + 12 ) , ( r4 + 16 ) , ( r4 + 20 ) , 100
000000CC - 00400046
000000D0 - 00040008
000000D4 - 0004000C
000000D8 - 00040010
000000DC - 00040014
000000E0 - 00000064
```

### Ticks per instruction

An instruction takes $N$ ticks without memory, plus the cost of every memory access:

$$t = N + \sum_{\text{accesses}} T_{\text{access}}$$

| Access          | `cache: false` | `cache: true`  |
|-----------------|----------------|----------------|
| aligned read    | 10             | hit 1, miss 15 |
| unaligned read  | 10             | 11             |
| aligned write   | 10             | 11             |
| unaligned write | 10             | 12–13          |

$R$ / $W$ — reads / writes (word 1 and every extension word are reads) \
$e$ — extension words \
$a(m)$ — address steps of a mode: 0 for register and immediate, 2 for `(r)+`, 1 for the others.
$k_{\text{mem}}$ — number of memory operands of `addm`.

| Instruction                  | $N$                                   | $R$                                                       | $W$                            |
|------------------------------|---------------------------------------|-----------------------------------------------------------|--------------------------------|
| F1, F2                       | $3 + d + a(m_{src}) + e + s$          | $1 + e + [\text{src in mem}] + [\text{dst in mem, read}]$ | $[\text{dst in mem, written}]$ |
| `halt`, `ei`, `di`           | 4                                     | 1                                                         | 0                              |
| `ret` / `iret`               | 5 / 7                                 | 2 / 3                                                     | 0                              |
| branch, `jump @l` / `jump l` | 4                                     | 1 / 2                                                     | 0                              |
| `call @l` / `call l`         | 4                                     | 1 / 2                                                     | 1                              |
| `pushm` / `popm`, $n$ regs   | $3 + n + e$ / $4 + 2n + 2e$           | $1 + e$ / $1 + n + e$                                     | $n$ / 0                        |
| `addm` $m_1 \dots m_k$       | $3 + e + \sum_i \big(a(m_i) + 1\big)$ | $1 + e + k_{\text{mem}}$                                  | $[m_1 \text{ in mem}]$         |
| interrupt entry              | 4                                     | 1                                                         | 2                              |

F1/F2 terms: $d = 1$ for a register destination (and the `out` port), $a(m_{dst})$ for a memory one;
$s = 1$ if the source is in memory, or sits in a later word than a memory destination, or the instruction is `out`.
The destination is read by all except `mov`, `clr`, `in`; written by all except `cmp`, `test`, `out`.

For example, `add (r1), 7` takes $4 + 2 \cdot 10 + 1 \cdot 10 = 34$ ticks without the cache.

### CISC traits

- Memory to memory operations: `add (total), (@seed)`.
- 8 addressing modes, including postincrement, predecrement and register-relative.
- Variable length: 1–8 words. `addm` sums up to 7 operands of any mode, `pushm` / `popm` take a register list.
- The CU fetches every extension word as a separate step (`of_dst_ext`, `of_src_ext`), which is visible in the journal.

## Translator

### CLI
```text
ailurus-build [flags] FILE...
```

| Flag              | Meaning                                                  |
|-------------------|----------------------------------------------------------|
| `-o FILE`         | output binary (default `a.bin`)                          |
| `-l FILE`         | debug listing: `<address> - <HEXCODE> - <source>`        |
| `-c`              | assemble only: one `.o` (JSON object file) per input     |
| `-E`              | preprocess only                                          |
| `-D NAME[=value]` | define `NAME` in every file                              |
| `--no-stdlib`     | do not link the standard library                         |

`FILE` is an `.asm` or `.o` file.

### Pipeline

The work goes in three stages:
1. **Preprocessor** — one pass over the text: defines, macros, conditional blocks. Every line keeps its
   `file:line`, so errors and the listing point to the real source.
2. **Assembler** — one pass + fixups. The size of an instruction depends only on the mnemonic and the modes,
   so it can be encoded right away. Every label operand is encoded as 0 right away, a PC-relative reference to a label
   in the same file and section is fixed at the end of the file, all others go to the relocation table. 
   The result is an object file.
3. **Linker** — places all `text` parts, then all `data` parts, resolves imports, applies relocations and writes
   the vectors `0x0` (`_start`) and `0x4` (`_handler`). The output is a flat binary image.

### Object file

The object file (`.o`, from `-c`) is JSON, because it is an intermediate file and should be easy to read. 
The executable has binary format, as the task specified.
It holds the section bytes, the symbol table and the relocations:

```json
{
  "version": 1,
  "source": "main.asm",
  "sections": { "text": "<bytes, hex>", "data": "<bytes, hex>" },
  "symbols": [
    { "name": "print",  "binding": "import" },
    { "name": "_start", "binding": "export", "section": "text" },
    { "name": "msg",    "binding": "local",  "section": "data", "offset": 0 }
  ],
  "relocations": [
    { "section": "text", "offset": 0, "bit": 4, "width": 16, "kind": "pcrel16", "symbol": "msg", "pc_next": 4 },
    { "section": "text", "offset": 8, "bit": 0, "width": 32, "kind": "abs32",   "symbol": "print" }
  ]
}
```

A relocation says which field to fill (`offset`, `bit`, `width`) and how: `abs32` / `imm16` — the symbol
address, `pcrel16` — the address minus `pc_next`. The full format is in
[Object file format](docs/specification.md#object-file-format).

