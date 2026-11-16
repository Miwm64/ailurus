.import print
.export _start
.section text
.org 0x10
_start :
mov r1 , ( @ msg )
call print
loop :
jz loop
.section data
.org 0x4
msg : .pstr "hi"
msg_ref : .ref msg