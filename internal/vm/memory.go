package vm

type memOp uint8

const (
	memNone memOp = iota
	memRead
	memWrite
	memReadNext // burst mode - 1 tick
)

// interface for datapath or control unit to interact with
// 2 implementations: cacheMemory and direct.
type memory interface {
	outputs(req memReq) memOut
	clock(req memReq)
}

type memReq struct {
	Op         memOp
	Addr, Data uint32
}

type memOut struct {
	data  uint32
	ready bool
	fault error
	event string // for the journal
}
