.import print
.export _start
.section text
_start:
    mov r1, (@msg)
    call print
loop:
    jz loop
.section data
msg: .pstr "hi"