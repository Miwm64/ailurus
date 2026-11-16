package linker

import "github.com/Miwm64/ailurus/internal/isa"

type bases struct {
	text uint32
	data uint32
}

type symRef struct {
	file int
	sec  isa.Section
	off  uint32
}

type linker struct {
	objectFiles []*isa.ObjectFile
	base        []bases // per file, final after pass 1
	globals     map[string]symRef
	size        uint32
}

func (l *linker) AddObjectFile(objectFile *isa.ObjectFile) {

}

func (l *linker) Link() ([]byte, []ListingEntry, error) {
	return nil, nil, nil
}
