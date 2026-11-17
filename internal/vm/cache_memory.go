package vm

const (
	cacheLines    = 16
	cacheLineSize = 16 // bytes
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

type tagEntry struct {
	valid bool
	tag   uint32
}

type cacheSignals struct {
	ramOp memOp

	// tag_array
	taWr      bool
	taAddress uint8
	taData    tagEntry

	// data_array
	dataArrWr bool
	line      uint8
	selDArr   uint8

	selCmp     uint8
	selAddr    uint8
	selEffAddr uint8
	selIndArr  uint8
	selDOut    uint8

	latchEffAddr bool
	ready        bool
}

// cacheMemory: fully associative, tree-PLRU, write-through, no-allocate
type cacheMemory struct {
	ram       *ram
	tagArray  [cacheLines]tagEntry
	dataArray [cacheLines][cacheLineSize / 4]uint32
	plru      uint16 // 15 tree bits
	state     cacheState
	effAddr   uint32
	filled    uint8 // words filled so far
	victim    uint8
}

func newCache(r *ram) *cacheMemory {
	return &cacheMemory{ram: r}
}

func (p *cacheMemory) control(req memReq, hit bool, lineNumber uint8, ramDone bool) cacheSignals {
	panic("TODO")
}

func (p *cacheMemory) outputs(req memReq) memOut {
	panic("TODO")
}

func (p *cacheMemory) clock(req memReq) {
	panic("TODO")
}
