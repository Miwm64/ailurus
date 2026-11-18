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
