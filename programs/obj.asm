.import print
.export _start
.section text
#define tmp #macr231o
_start:
    mov tmp, (@msg)
    call print
loop:
    jz loop
.section data
msg: .pstr "hi"