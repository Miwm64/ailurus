package assembler

import "encoding/hex"

type Section string

const (
	Data Section = "data"
	Text Section = "text"
)

type Binding string

const (
	Import Binding = "import"
	Export Binding = "export"
	Local  Binding = "local"
)

type RelocationKind string

const (
	Abs32   RelocationKind = "abs32"
	Imm16   RelocationKind = "imm16"
	PCRel16 RelocationKind = "pcrel16"
)

type ObjectFile struct {
	Version     int                   `json:"version"`
	Source      string                `json:"source"`
	Sections    map[Section]*HexBytes `json:"sections"`
	Symbols     []Symbol              `json:"symbols"`
	Relocations []Relocation          `json:"relocations"`
	Lines       []Line                `json:"lines"`
}

type Symbol struct {
	Name    string  `json:"name"`
	Binding Binding `json:"binding"`
	Section Section `json:"section"`
	Offset  int     `json:"offset"`
}

type Relocation struct {
	Section Section        `json:"section"`
	Offset  int            `json:"offset"`
	Bit     int            `json:"bit"`
	Width   int            `json:"width"`
	Kind    RelocationKind `json:"kind"`
	Symbol  string         `json:"symbol"`
	PCNext  int            `json:"pc_next"`
}

type Line struct {
	Section Section `json:"section"`
	Offset  int     `json:"offset"`
	Line    int     `json:"line"`
	Text    string  `json:"text"`
}

type HexBytes []byte

func (p *HexBytes) writeAt(offset int, data []byte) {
	if end := offset + len(data); end > len(*p) {
		*p = append(*p, make([]byte, end-len(*p))...)
	}

	copy((*p)[offset:], data)
}

func newObjectFile(source string) *ObjectFile {
	return &ObjectFile{
		Version: 1,
		Source:  source,
		Sections: map[Section]*HexBytes{
			Data: &HexBytes{},
			Text: &HexBytes{},
		},
	}
}

func (h HexBytes) MarshalText() ([]byte, error) {
	res := make([]byte, len(h)*2)
	hex.Encode(res, h)
	return res, nil
}

func (h *HexBytes) UnmarshalText(text []byte) error {
	*h = make([]byte, len(text)/2)
	_, err := hex.Decode(*h, text)
	return err
}
