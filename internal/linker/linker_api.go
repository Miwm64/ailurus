package linker

import "github.com/Miwm64/ailurus/internal/assembler"

type Linker interface {
	addObjectFile(*assembler.ObjectFile)
	link() ([]byte, []ListingEntry, error)
}

type linker struct {
	objectFiles []*assembler.ObjectFile
}

func (l *linker) addObjectFile(objectFile *assembler.ObjectFile) {

}

func (l *linker) link() ([]byte, []ListingEntry, error) {
	return nil, nil, nil
}

type ListingEntry struct {
}
