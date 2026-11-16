package linker

import "github.com/Miwm64/ailurus/internal/isa"

type Linker interface {
	AddObjectFile(*isa.ObjectFile)
	Link() ([]byte, []ListingEntry, error)
}
