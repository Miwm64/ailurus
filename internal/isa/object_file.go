package isa

import (
	"encoding/hex"
	"fmt"
)

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
	Section Section `json:"section,omitempty"`
	Offset  int     `json:"offset,omitempty"`
}

type Relocation struct {
	Section Section        `json:"section"`
	Offset  int            `json:"offset"`
	Bit     int            `json:"bit"`
	Width   int            `json:"width"`
	Kind    RelocationKind `json:"kind"`
	Symbol  string         `json:"symbol"`
	PCNext  int            `json:"pc_next,omitempty"`
}

type Line struct {
	Section Section `json:"section"`
	Offset  int     `json:"offset"`
	Line    int     `json:"line"`
	Text    string  `json:"text"`
}

type HexBytes []byte

func (p *HexBytes) WriteAt(offset int, data []byte) {
	if end := offset + len(data); end > len(*p) {
		*p = append(*p, make([]byte, end-len(*p))...)
	}

	copy((*p)[offset:], data)
}

func NewObjectFile(source string) *ObjectFile {
	return &ObjectFile{
		Version: 1,
		Source:  source,
		Sections: map[Section]*HexBytes{
			Data: {},
			Text: {},
		},
	}
}

func (p *HexBytes) MarshalText() ([]byte, error) {
	res := make([]byte, hex.EncodedLen(len(*p)))
	hex.Encode(res, *p)

	return res, nil
}

func (p *HexBytes) UnmarshalText(text []byte) error {
	*p = make([]byte, len(text)/2)
	_, err := hex.Decode(*p, text)

	return fmt.Errorf("unmarshall: %w", err)
}

func ModeToRelocationKind(argumentMode Mode) RelocationKind {
	switch argumentMode {
	case PCRelative:
		return PCRel16
	case DirectMemory:
		return Abs32
	case Immediate:
		return Imm16
	default:
		panic("invalid argument mode")
	}
}
