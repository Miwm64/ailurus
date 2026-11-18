.export _start
.import print
_start:
    mov rsp, 0x1000
    mov r1, msg
    call print
    halt
.section data
msg: .pstr "Hello, world!\n"
