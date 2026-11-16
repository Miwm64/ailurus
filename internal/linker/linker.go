package linker

import (
	"errors"
	"fmt"

	"github.com/Miwm64/ailurus/internal/isa"
)

var (
	ErrDuplicateExport = errors.New("duplicate export")
	ErrNoStart         = errors.New("no exported _start")
)

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

func (p *linker) firstPass() error {
	p.base = make([]bases, len(p.objectFiles))
	p.globals = make(map[string]symRef)
	var errs []error

	textCurr := uint32(0x8)
	dataRel := uint32(0)
	for i, o := range p.objectFiles {
		textCurr = align4(textCurr)
		p.base[i].text = textCurr
		textCurr += uint32(len(*o.Sections[isa.Text]))

		dataRel = align4(dataRel)
		p.base[i].data = dataRel
		dataRel += uint32(len(*o.Sections[isa.Data]))

		for _, s := range o.Symbols {
			if s.Binding != isa.Export {
				continue
			}
			if prev, ok := p.globals[s.Name]; ok {
				errs = append(errs, fmt.Errorf("%s and %s: %w: %s",
					p.objectFiles[prev.file].Source, o.Source, ErrDuplicateExport, s.Name))
				continue
			}
			p.globals[s.Name] = symRef{file: i, sec: s.Section, off: uint32(s.Offset)}
		}
	}

	dataStart := align4(textCurr)

	for i := range p.base {
		p.base[i].data += dataStart
	}
	p.size = dataStart + dataRel

	if _, ok := p.globals["_start"]; !ok {
		errs = append(errs, ErrNoStart)
	}
	return errors.Join(errs...)
}

func align4(number uint32) uint32 {
	return (number + 3) &^ 3
}
