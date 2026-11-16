package linker

import "github.com/Miwm64/ailurus/internal/isa"

func Link(objs []*isa.ObjectFile) ([]byte, []ListingEntry, error) {
	l := linker{
		objectFiles: objs,
	}

	err := l.firstPass()
	if err != nil {
		return nil, nil, err
	}

	return l.secondPass()
}
