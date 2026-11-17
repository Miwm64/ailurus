.import print
.export _start

.section text
.org 0x10
_start :
    mov rsp, 0x100
    mov r1 , ( @ msg_ref )
    call print
    halt

.section data
.org 0x4
msg : .pstr "hi"
msg_ref : .ref msg
