; Ailurus standard library
;
; Calling convention: arguments in r1-r4, result in r1 (second result in r2).
; r1-r7 are caller-saved, r8-r15 and rsp are callee-saved.
; Strings are pstr: a length word, then one word per character.
;
; Input comes through the interrupt: _handler puts each character into a ring
; buffer, readChar / readLine take characters from it.
; Before reading input, the program's _start must set rsp and execute ei.
; A program with its own _handler is built with --no-stdlib.

.export inputPush
.export printChar
.export print
.export readChar
.export readLine
.export strEq
.export strConcat
.export strContains
.export intToStr
.export strToInt

#define _PORT_IN 0
#define _PORT_OUT 1
#define _WORD 4
#define _WORD_SHIFT 2
#define _BASE 10
#define _ERR -1
; ring buffer of 16 words; head and tail are byte offsets, mask = 16 * _WORD - 1
#define _IN_MASK 63

.section text

; inputPush: input character -> ring buffer, dropped if full. Preserves all registers.
inputPush:
    pushm r1, r2, r3
    in r1, _PORT_IN
    mov r2, (@inTail)
    mov r3, r2
    add r3, _WORD
    and r3, _IN_MASK
    cmp r3, (@inHead)
    jz inputPushDone
    add r2, inBuf
    mov (r2), r1
    mov (@inTail), r3       ; only after the store
inputPushDone:
    popm r3, r2, r1
    ret

; printChar: r1 = character
printChar:
    out _PORT_OUT, r1
    ret

; print: r1 = pstr
print:
    mov r2, (r1)+
    jz printDone
printLoop:
    out _PORT_OUT, (r1)+
    dec r2
    jnz printLoop
printDone:
    ret

; readChar: -> r1 = character; waits for input
readChar:
    mov r2, (@inHead)
readCharWait:
    cmp r2, (@inTail)
    jz readCharWait
    mov r1, r2
    add r1, inBuf
    mov r1, (r1)
    add r2, _WORD
    and r2, _IN_MASK
    mov (@inHead), r2
    ret

; readLine: r1 = buffer, r2 = capacity -> r1 = length, or -1 if the line is longer.
; Reads up to "\n", which is not stored.
readLine:
    pushm r8, r9, r10, r11
    mov r8, r1
    mov r9, r2
    clr r10
    mov r11, r1
    add r11, _WORD
readLineLoop:
    call @readChar
    cmp r1, "\n"
    jz readLineDone
    cmp r10, r9
    jae readLineOverflow
    mov (r11)+, r1
    inc r10
    jump @readLineLoop
readLineOverflow:
    mov r1, _ERR
    jump @readLineRet
readLineDone:
    mov (r8), r10
    mov r1, r10
readLineRet:
    popm r11, r10, r9, r8
    ret

; strEq: r1 = a, r2 = b -> r1 = 1 if equal, else 0
strEq:
    mov r3, (r1)+
    cmp r3, (r2)+
    jnz strEqNo
strEqLoop:
    cmp r3, 0
    jz strEqYes
    mov r4, (r1)+
    cmp r4, (r2)+
    jnz strEqNo
    dec r3
    jump @strEqLoop
strEqYes:
    mov r1, 1
    ret
strEqNo:
    clr r1
    ret

; strConcat: r1 = dst, r2 = a, r3 = b; dst <- a + b. dst may be a, but not b.
strConcat:
    mov r4, (r2)+
    mov r5, (r3)+
    mov r6, r4
    add r6, r5
    mov (r1)+, r6
strConcatA:
    cmp r4, 0
    jz strConcatB
    mov r6, (r2)+
    mov (r1)+, r6
    dec r4
    jump @strConcatA
strConcatB:
    cmp r5, 0
    jz strConcatDone
    mov r6, (r3)+
    mov (r1)+, r6
    dec r5
    jump @strConcatB
strConcatDone:
    ret

; strContains: r1 = s, r2 = sub -> r1 = 1 if sub occurs in s, else 0
strContains:
    push r8
    mov r3, (r1)+
    mov r4, (r2)+
    sub r3, r4
    jl strContainsNo
    inc r3
strContainsStart:
    mov r5, r1
    mov r6, r2
    mov r7, r4
strContainsCmp:
    cmp r7, 0
    jz strContainsYes
    mov r8, (r5)+
    cmp r8, (r6)+
    jnz strContainsNext
    dec r7
    jump @strContainsCmp
strContainsNext:
    add r1, _WORD
    dec r3
    jnz strContainsStart
strContainsNo:
    clr r1
    jump @strContainsRet
strContainsYes:
    mov r1, 1
strContainsRet:
    pop r8
    ret

; intToStr: r1 = number, r2 = buffer (12 words) -> r1 = length
intToStr:
    mov r3, r2
    add r3, _WORD
    clr r4
    cmp r1, 0
    jl intToStrSign
    neg r1                  ; work with -|n|: -2^31 has no positive pair
    jump @intToStrDigits
intToStrSign:
    mov (r3)+, "-"
intToStrDigits:
    mov r5, r1
    mod r5, _BASE
    neg r5
    add r5, "0"
    push r5
    inc r4
    div r1, _BASE
    jnz intToStrDigits
intToStrPop:
    pop r5
    mov (r3)+, r5
    dec r4
    jnz intToStrPop
    mov r1, r3
    sub r1, r2
    lsr r1, _WORD_SHIFT
    dec r1
    mov (r2), r1
    ret

; strToInt: r1 = pstr -> r1 = number, r2 = 0;
; r2 = -1 if not [+|-]digits or out of 32-bit range
strToInt:
    mov r3, (r1)+
    jz strToIntBad
    clr r4
    clr r5
    mov r6, (r1)
    cmp r6, "-"
    jnz strToIntPlus
    mov r5, 1
    jump @strToIntSkipSign
strToIntPlus:
    cmp r6, "+"
    jnz strToIntDigit
strToIntSkipSign:
    add r1, _WORD
    dec r3
    jz strToIntBad
strToIntDigit:
    mov r6, (r1)+
    sub r6, "0"
    cmp r6, _BASE
    jae strToIntBad
    mov r7, r4
    umulh r7, _BASE
    jnz strToIntBad
    mull r4, _BASE
    add r4, r6
    jb strToIntBad
    dec r3
    jnz strToIntDigit
    cmp r5, 0
    jz strToIntPositive
    neg r4
    cmp r4, 0
    jg strToIntBad
    jump @strToIntOk
strToIntPositive:
    cmp r4, 0
    jl strToIntBad
strToIntOk:
    mov r1, r4
    clr r2
    ret
strToIntBad:
    mov r2, _ERR
    ret

.section data
inHead: .word 0
inTail: .word 0
inBuf:  .word 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0
