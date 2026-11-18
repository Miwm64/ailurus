.export _start
.import printStr
.import greeting
.import counter

#define TIMES 2

#macro say, 1
{
    mov r1, %0
    call printStr
}

_start:
    mov rsp, 0x1000
    mov r8, TIMES
again:
    say greeting
    inc (counter)
    dec r8
    jnz again
#ifdef VERBOSE
    say verbose
#else
    say quiet
#endif
    say counted
    mov r1, (counter)
    add r1, "0"
    out 1, r1
    out 1, "\n"
    halt

.section data
quiet:   .pstr "built without VERBOSE\n"
verbose: .pstr "built with -D VERBOSE\n"
counted: .pstr "counter (lib.asm) = "
