package linker

import (
	"cmp"
	"encoding/binary"
	"errors"
	"slices"

	"github.com/Miwm64/ailurus/internal/isa"
)

// secondPass builds the image
// requires base, globals and size
func (p *linker) secondPass() ([]byte, []ListingEntry, error) {
	image := make([]byte, p.size)
	p.writeVectors(image)
	listing := vectorEntries(image)
	var errs []error

	for i, o := range p.objectFiles {
		copy(image[p.base[i].text:], sectionBytes(o, isa.Text))
		copy(image[p.base[i].data:], sectionBytes(o, isa.Data))
	}

	if err := errors.Join(errs...); err != nil {
		return nil, nil, err
	}

	slices.SortFunc(listing, func(a, b ListingEntry) int {
		return cmp.Compare(a.Address, b.Address)
	})
	return image, listing, nil
}


func (p *linker) secBase(i int, sec isa.Section) uint32 {
	if sec == isa.Data {
		return p.base[i].data
	}
	return p.base[i].text
}

func (p *linker) addr(r symRef) uint32 {
	return p.secBase(r.file, r.sec) + r.off
}

func sectionBytes(o *isa.ObjectFile, sec isa.Section) []byte {
	if b := o.Sections[sec]; b != nil {
		return *b
	}
	return nil
}

func (p *linker) writeVectors(image []byte) {
	binary.LittleEndian.PutUint32(image[0x0:], p.addr(p.globals["_start"]))
	if h, ok := p.globals["_handler"]; ok {
		binary.LittleEndian.PutUint32(image[0x4:], p.addr(h))
	} // no _handler = 0
}

func vectorEntries(image []byte) []ListingEntry {
	return []ListingEntry{
		{Address: 0x0, Word: binary.LittleEndian.Uint32(image[0x0:]), Mnemonic: "_start address"},
		{Address: 0x4, Word: binary.LittleEndian.Uint32(image[0x4:]), Mnemonic: "_handler address"},
	}
}
