package linker

import (
	"github.com/Miwm64/ailurus/internal/isa"
)

type Linker interface {
	addObjectFile(*isa.ObjectFile)
	link() ([]byte, []ListingEntry, error)
}

type linker struct {
	objectFiles []*isa.ObjectFile
}

func (l *linker) addObjectFile(objectFile *isa.ObjectFile) {

}

func (l *linker) link() ([]byte, []ListingEntry, error) {
	return nil, nil, nil
}

type ListingEntry struct {
	Address  uint32
	Word     uint32
	Mnemonic string
}
