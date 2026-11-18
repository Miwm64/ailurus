.export printStr
.export greeting
.export counter

; printStr: r1 = pstr
printStr:
    mov r2, (r1)+
    jz printStrDone
printStrLoop:
    out 1, (r1)+
    dec r2
    jnz printStrLoop
printStrDone:
    ret

.section data
counter:  .word 0
.org 0x10                   ; greeting starts 16 bytes into this file's data
greeting: .pstr "hello from lib.asm\n"
