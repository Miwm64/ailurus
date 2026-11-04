# Things i shall not forget - mainly specs and docs
ISA (programmer-visible):

PS layout: IE = bit 0, C/V/Z/N = bits 1–4, with your exact order. State that bits 5–31 of a pushed PS are 0, and what iret does with them if the stack value has them set: ignore them, or halt?
Reset: you specify IE ← 0, but what are NZVC after reset?
Request tie-break: if a character arrives on the tick the CU takes the interrupt, clear wins. The handler reads the new character once, and the previous one is lost.
Malformed schedule: two characters on the same tick is rejected when the simulator loads it.

Report (DataPath section):
5. cu_data: it carries an immediate, PC, or a CU-built PS word. Explain why one line is enough: they’re never needed on the same tick.
6. ALU ops: list them, including PASS_B.
7. New signals and blocks: sel_ps_mux, latch_ie, request_interrupt, INPUT_DEVICE (the schedule plus data), OUTPUT_DEVICE.
8. Where things live: irq_pending and PC arithmetic are in the CU, not the datapath.

Still to ask: what the teacher means by "внутренние прерывания".
