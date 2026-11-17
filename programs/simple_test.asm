.export _start
.import print

.section data
a: .pstr "Hello, World!"

.section text
_start:
    mov rsp, 0x100
    mov r1, a
    call print
    halt