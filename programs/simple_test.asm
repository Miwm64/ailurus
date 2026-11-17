.export _start

.section data
a: .word 0x123
b: .word -0x123

.section text
_start:
    out 1, (a)
    halt