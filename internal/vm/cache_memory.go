package vm

import "fmt"

const (
	cacheLines    = 16
	cacheLineSize = 16 // bytes
	wordsPerLine  = cacheLineSize / 4
	plruLevels    = 4
)

type cacheState uint8

const (
	cacheIdle cacheState = iota
	cacheFill
	cacheTag
	cacheBypassR
	cacheWrite
	cacheInv1
	cacheInv2
)

// CacheStats: one count per access, taken when the access starts.
type CacheStats struct {
	Hits, Misses, Evictions uint64
	Writes                  uint64 // write-through
	Bypasses                uint64 // unaligned reads, served by RAM
}

type tagEntry struct {
	valid bool
	tag   uint32
}

type cacheMemory struct {
	ram       *ram
	tagArray  [cacheLines]tagEntry
	dataArray [cacheLines][wordsPerLine]uint32
	plru      uint16 // 15 tree bits
	state     cacheState
	effAddr   uint32
	filled    uint8 // words filled so far
	victim    uint8
	stats     CacheStats
}

func newCache(r *ram) *cacheMemory {
	return &cacheMemory{ram: r}
}

// ramReq: ram_op depends only on the state, so a RAM access starts in the state after IDLE.
func (p *cacheMemory) ramReq(req memReq) memReq {
	switch p.state {
	case cacheFill:
		if p.filled == 0 {
			return memReq{Op: memRead, addr: p.effAddr}
		}

		return memReq{Op: memReadNext, addr: p.effAddr}
	case cacheBypassR:
		return memReq{Op: memRead, addr: req.addr}
	case cacheWrite:
		return memReq{Op: memWrite, addr: req.addr, data: req.data}
	default:
		return memReq{}
	}
}

func (p *cacheMemory) outputs(req memReq) memOut {
	ram := p.ram.outputs(p.ramReq(req))
	if ram.fault != nil {
		return ram
	}

	switch p.state {
	case cacheIdle:
		line, hit := p.lookup(req.addr)
		switch {
		case req.Op == memNone:
			return memOut{}
		case req.Op == memWrite:
			return memOut{event: "write"}
		case !aligned(req.addr):
			return memOut{event: "unaligned read"}
		case hit:
			data := p.dataArray[line][wordIndex(req.addr)]

			return memOut{ready: true, data: data, event: fmt.Sprintf("hit line %d", line)}
		default:
			return memOut{event: "miss"}
		}
	case cacheFill:
		return memOut{event: "fill " + ram.event}
	case cacheTag:
		return memOut{ready: true, data: p.dataArray[p.victim][wordIndex(req.addr)], event: "tag"}
	case cacheBypassR:
		return memOut{ready: ram.ready, data: ram.data, event: "bypass " + ram.event}
	case cacheWrite:
		return memOut{ready: ram.ready && aligned(req.addr), event: "write " + ram.event}
	case cacheInv1:
		return memOut{ready: !spansTwoLines(req.addr), event: "invalidate"}
	default: // cacheInv2
		return memOut{ready: true, event: "invalidate"}
	}
}

func (p *cacheMemory) clock(req memReq) {
	ramReq := p.ramReq(req)
	ram := p.ram.outputs(ramReq)
	p.ram.clock(ramReq)

	if ram.fault != nil {
		p.state = cacheIdle

		return
	}

	switch p.state {
	case cacheIdle:
		p.idle(req)
	case cacheFill:
		if ram.ready {
			p.dataArray[p.victim][wordIndex(p.effAddr)] = ram.data
			p.effAddr += 4

			p.filled++
			if p.filled == wordsPerLine {
				p.state = cacheTag
			}
		}
	case cacheTag:
		p.tagArray[p.victim] = tagEntry{valid: true, tag: tagOf(req.addr)}
		p.plru = plruTouch(p.plru, p.victim)
		p.state = cacheIdle
	case cacheBypassR:
		if ram.ready {
			p.state = cacheIdle
		}
	case cacheWrite:
		if ram.ready {
			p.writeDone(req)
		}
	case cacheInv1:
		p.invalidate(req.addr)

		p.state = cacheIdle
		if spansTwoLines(req.addr) {
			p.state = cacheInv2
		}
	case cacheInv2:
		p.invalidate(req.addr + 4)
		p.state = cacheIdle
	}
}

func (p *cacheMemory) idle(req memReq) {
	line, hit := p.lookup(req.addr)

	switch {
	case req.Op == memNone:
	case req.Op == memWrite:
		p.stats.Writes++
		p.state = cacheWrite
	case !aligned(req.addr):
		p.stats.Bypasses++
		p.state = cacheBypassR
	case hit:
		p.stats.Hits++
		p.plru = plruTouch(p.plru, line)
	default: // miss: fill the line from its first word
		p.stats.Misses++
		p.effAddr = req.addr &^ (cacheLineSize - 1)
		p.victim, p.filled, p.state = p.chooseVictim(), 0, cacheFill

		if p.tagArray[p.victim].valid {
			p.stats.Evictions++
		}
	}
}

func (p *cacheMemory) writeDone(req memReq) {
	if !aligned(req.addr) {
		p.state = cacheInv1

		return
	}

	if line, hit := p.lookup(req.addr); hit {
		p.dataArray[line][wordIndex(req.addr)] = req.data
	}

	p.state = cacheIdle
}

// cmp scheme - all 16 at once.
func (p *cacheMemory) lookup(addr uint32) (uint8, bool) {
	for i, e := range p.tagArray {
		if e.valid && e.tag == tagOf(addr) {
			return uint8(i), true
		}
	}

	return 0, false
}

func (p *cacheMemory) invalidate(addr uint32) {
	if line, hit := p.lookup(addr); hit {
		p.tagArray[line] = tagEntry{}
	}
}

// invalid, then plru.
func (p *cacheMemory) chooseVictim() uint8 {
	for i, e := range p.tagArray {
		if !e.valid {
			return uint8(i)
		}
	}

	return plruVictim(p.plru)
}

func tagOf(addr uint32) uint32 {
	return addr / cacheLineSize
}

func wordIndex(addr uint32) uint32 {
	return (addr / 4) % wordsPerLine
}

func aligned(addr uint32) bool {
	return addr%4 == 0
}

func spansTwoLines(addr uint32) bool {
	return addr%cacheLineSize > cacheLineSize-4
}

// plruTouch points every node on the line's path away from it.
func plruTouch(tree uint16, line uint8) uint16 {
	node := 1

	for level := plruLevels - 1; level >= 0; level-- {
		right := int(line>>level) & 1
		if right == 1 {
			tree &^= 1 << node
		} else {
			tree |= 1 << node
		}

		node = 2*node + right
	}

	return tree
}

// follows the bits from the root to a leaf.
func plruVictim(tree uint16) uint8 {
	node := 1
	for range plruLevels {
		node = 2*node + int(tree>>node)&1
	}

	return uint8(node - cacheLines)
}
