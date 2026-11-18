package linker

import (
	"cmp"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"slices"

	"github.com/Miwm64/ailurus/internal/isa"
)

var (
	ErrMissingImport = errors.New("missing import")
	ErrOutOfRange    = errors.New("relocation out of range")
)

// secondPass builds the image
// requires base, globals and size.
func (p *linker) secondPass() ([]byte, []ListingEntry, error) {
	image := make([]byte, p.size)
	p.writeVectors(image)
	listing := vectorEntries(image)

	var errs []error

	for i, o := range p.objectFiles {
		entries, fileErrs := p.handleFile(image, i, o)
		listing = append(listing, entries...)
		errs = append(errs, fileErrs...)
	}

	err := errors.Join(errs...)
	if err != nil {
		return nil, nil, err
	}

	slices.SortFunc(listing, func(a, b ListingEntry) int {
		return cmp.Compare(a.Address, b.Address)
	})

	return image, listing, nil
}

// handleFile copies file into image, does relocations, returns listing.
func (p *linker) handleFile(image []byte, i int, o *isa.ObjectFile) ([]ListingEntry, []error) {
	var errs []error

	copy(image[p.base[i].text:], sectionBytes(o, isa.Text))
	copy(image[p.base[i].data:], sectionBytes(o, isa.Data))

	syms := make(map[string]uint32)

	for _, s := range o.Symbols {
		if s.Binding != isa.Import {
			syms[s.Name] = p.secBase(i, s.Section) + uint32(s.Offset)
		} else if ref, ok := p.globals[s.Name]; ok {
			syms[s.Name] = p.addr(ref)
		} else {
			errs = append(errs, fmt.Errorf("%s: %w: %s", o.Source, ErrMissingImport, s.Name))
		}
	}

	for _, r := range o.Relocations {
		s, ok := syms[r.Symbol]
		if !ok {
			continue // missing import
		}

		base := p.secBase(i, r.Section)
		v := int64(s)

		// relocate
		if r.Kind == isa.PCRel16 {
			v -= int64(base + uint32(r.PCNext))
		}

		if (r.Kind == isa.Imm16 && v > 0x7FFF) ||
			(r.Kind == isa.PCRel16 && (v < math.MinInt16 || v > math.MaxInt16)) {
			errs = append(errs, fmt.Errorf("%s: %s %s: %w", o.Source, r.Kind, r.Symbol, ErrOutOfRange))

			continue
		}

		at := base + uint32(r.Offset)
		w := binary.LittleEndian.Uint32(image[at:])
		binary.LittleEndian.PutUint32(image[at:], isa.PatchField(w, r.Bit, r.Width, uint32(v)))
	}

	// listing
	var listing []ListingEntry

	for j, l := range o.Lines {
		end := len(sectionBytes(o, l.Section))
		for _, next := range o.Lines[j+1:] {
			if next.Section == l.Section {
				end = next.Offset

				break
			}
		}

		base := p.secBase(i, l.Section)
		for off := l.Offset; off+4 <= end; off += 4 {
			a := base + uint32(off)

			e := ListingEntry{Address: a, Word: binary.LittleEndian.Uint32(image[a:])}
			if off == l.Offset {
				e.Mnemonic = l.Text
			}

			listing = append(listing, e)
		}
	}

	return listing, errs
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
