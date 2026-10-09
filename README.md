# Ailurus
Author: Mikhail Dobkes

## Description
<picture>
  <img src="panda.png" align="right" width="300" alt="Red panda">
</picture>

Ailurus is an open source project with the goal of creating an assembler, linker and a simple VM to run the code on.

It consists of 3 parts:
- Assembler
- Linker
- Virtual machine which emulates a simple CISC architecture

## Project specifics
### Syntax
Assembly language, which supports labels, sections, user-defined macros, constants and conditional compilation, .org directive. \
The project supports separate translation. \
The linker resolves symbols across translation units using symbol tables and address relocation. 

### ISA
The VM implements a CISC instruction set with complex instructions:
- Arithmetic operations that take operands from both registers and memory in a single instruction.
- Instructions that operate on special-purpose registers.
- Variable-length instructions with a variable number of operands, encoded in a variable number of machine words.

Instruction fetch reads the instruction word by word, explicitly advancing to the next machine word for each additional operand.

### Memory
- Single port
- Byte-addressable
- Von Neumann architecture
- Cache which speeds access 10 times(1 tick vs 10)

### Control Unit
The architecture includes a hardwired CU, with a step counter to explicitly control execution flow.

### Input/Output
I/O is token-based and driven by an interrupt system.
It is also port-mapped with ports being addressable

- Input is defined by a schedule given at startup, e.g. `[(1, 'h'), (10, 'e'), (20, 'l'), (25, 'l'), (100, 'o')]`. Each entry raises an interrupt at the given tick, and the character becomes available on the input port from that tick on. Raising the interrupt and reading the data are separate steps: the handler reads the port explicitly.
- Interrupts are internal. The handler is written by the programmer.
- Nested interrupts: an interrupt arriving during handling is //TODO: write mechanism
- Output is character-by-character: each output instruction appends one character to the output buffer. All output is printed when the simulation ends.
- The execution log shows whether the processor is currently inside an interrupt handler.
- Buffers are part of the VM model; there are no hidden queues.

### Strings and characters
- Static strings are stored in data section
- Single character is a single byte
- One symbol is stored in one word
- Strings are packed with sized(pstr)

### Testing
The project uses golden testing
