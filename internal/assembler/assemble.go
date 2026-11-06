package assembler

import "strings"

func Assemble(file string, lineMap map[int]int) (*ObjectFile, error) {
	objectFile := &ObjectFile{}

	lines := strings.Split(file, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		err := handleLine(line, lineMap, objectFile)
		if err != nil {
			return nil, err
		}
	}

	return objectFile, nil
}

func handleLine(line string, lineMap map[int]int, objectFile *ObjectFile) error {

	return nil
}
