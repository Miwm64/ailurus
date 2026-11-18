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
