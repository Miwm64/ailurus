; binsearch.asm — binary search over a sorted word array
; expected: results = 3, 0, 7, -1, -1

#define STACK_TOP 0x1000
#define NOT_FOUND -1
#define WORD_SHIFT 2
#define ARRAY_LEN 8
#define KEYS_LEN 5

; dst <- (a + b) / 2, a and b non-negative
#macro midpoint, 3 {
    mov %0, %1
    add %0, %2
    lsr %0, 1
}

; dst <- base[idx]; dst must differ from base and idx
#macro load_word, 3 {
    mov %0, %2
    lsl %0, WORD_SHIFT
    add %0, %1
    mov %0, (%0)
}

; prints a char only when built with -D TRACE
#macro trace, 1 {
#ifdef TRACE
    out 1, %0
#endif
}

.export _start

.section text
_start:
    mov rsp, STACK_TOP
    mov r8, keys            ; r8 -> next key (callee-saved, survives calls)
    mov r9, results         ; r9 -> next result slot
    mov r10, KEYS_LEN
next_key:
    mov r1, array
    mov r2, ARRAY_LEN
    mov r3, (r8)+
    call @binsearch
    mov (r9)+, r1
    dec r10
    jnz next_key
    halt

; binsearch(r1 = base, r2 = count, r3 = key) -> r1 = index or NOT_FOUND
; uses r4 = lo, r5 = hi (exclusive), r6 = mid, r7 = base[mid]
binsearch:
    clr r4
    mov r5, r2
bs_loop:
    cmp r4, r5
    jge bs_missing          ; lo >= hi: range is empty
    midpoint r6, r4, r5
    load_word r7, r1, r6
    cmp r7, r3
    jz bs_found
    jl bs_go_right          ; base[mid] < key
    mov r5, r6              ; hi = mid
    jump @bs_loop
bs_go_right:
    mov r4, r6
    inc r4                  ; lo = mid + 1
    jump @bs_loop
bs_found:
    trace "+"
    mov r1, r6
    ret
bs_missing:
    trace "-"
    mov r1, NOT_FOUND
    ret

.section data
array:   .word 2, 3, 5, 7, 11, 13, 17, 19
keys:    .word 7, 2, 19, 4, 20
results: .word 0, 0, 0, 0, 0