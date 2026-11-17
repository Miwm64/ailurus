package vm

// Simulator configuration, passed from cli.
type Config struct {
	MemorySize uint32
	Cache      bool
	TickLimit  uint64
	Input      []InputEvent
}

type InputEvent struct {
	Tick uint64
	Char byte
}
