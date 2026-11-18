package vm

import "errors"

var (
	ErrImageTooBig = errors.New("image does not fit in memory")
	ErrTickLimit   = errors.New("tick limit reached")
	ErrBadOpcode   = errors.New("invalid opcode")
	ErrDivZero     = errors.New("division by zero")
	ErrBadAddress  = errors.New("address out of memory")
	ErrBadPort     = errors.New("invalid port")
)

// Machine wires the components together and owns the clock.
type Machine struct {
	mem memory
	dp  dataPath
	cu  controlUnit
	io  ioController

	tick    uint64
	limit   uint64
	journal []Record
	memWait uint64

	// journal annotations
	irqDepth  int
	instrAddr uint32
	prevStep  cuStep
}

// Stats: memory counters for the run summary.
type Stats struct {
	MemWait uint64      // ticks with a memory request that is not ready yet
	Cache   *CacheStats // nil when the cache is off
}

// New builds a machine with image loaded at address 0.
func New(image []byte, cfg Config) (*Machine, error) {
	r, err := newRAM(cfg.MemorySize, image)
	if err != nil {
		return nil, err
	}

	var mem memory = r
	if cfg.Cache {
		mem = newCache(r)
	}

	return &Machine{
		mem:   mem,
		io:    newIO(cfg.Input),
		limit: cfg.TickLimit,
	}, nil
}

// Run runs until halt or tick limit.
func (p *Machine) Run() error {
	for !p.cu.halted {
		if p.tick >= p.limit {
			return ErrTickLimit
		}

		p.Tick()
	}

	return p.cu.fault
}

func (p *Machine) Tick() {
	// 1. latched outputs
	regs := p.dp.registers()
	irqReq := p.io.isRequestInterrupt()

	// 2. control logic: the part that doesn't depend on ready / div_zero
	sig := p.cu.outputs(regs)

	// 3. combinational, order: IO data → ALU → result → memory
	in := p.io.outputs(sig)
	out := p.dp.outputs(sig, in.data)
	req := memReq{Op: sig.memOp, addr: regs.mar, data: out.result}
	mem := p.mem.outputs(req)

	if req.Op != memNone && !mem.ready {
		p.memWait++
	}

	// 4. clock edge: everyone latches
	p.mem.clock(req)
	p.io.clock(p.tick, sig, out.result)
	p.dp.clock(sig, out, mem)
	p.cu.clock(sig, regs, out, mem, in, irqReq)

	p.journal = append(p.journal, p.snapshot(sig, mem))
	p.tick++
}

func (p *Machine) Halted() bool {
	return p.cu.halted
}

func (p *Machine) Ticks() uint64 {
	return p.tick
}

func (p *Machine) Output() []byte {
	return p.io.output
}

func (p *Machine) Journal() []Record {
	return p.journal
}

func (p *Machine) Stats() Stats {
	stats := Stats{MemWait: p.memWait}
	if cache, ok := p.mem.(*cacheMemory); ok {
		cacheStats := cache.stats
		stats.Cache = &cacheStats
	}

	return stats
}
