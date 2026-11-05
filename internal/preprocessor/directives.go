package preprocessor

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

var (
	ErrNoArguments      = errors.New("directive used without arguments")
	ErrInvalidName      = errors.New("name does not follow naming rules")
	ErrUnknownDirective = errors.New("unknown directive")
)

func handleDefine(line string, defines map[string]string) error {
	line = strings.TrimSpace(line)
	tokens := strings.Fields(line)

	switch tokens[0] {
	case "#define":
		if len(tokens) < 2 {
			return fmt.Errorf("%w: %s", ErrNoArguments, tokens[0])
		}

		if !validateName(tokens[1]) {
			return fmt.Errorf("%w: %q", ErrInvalidName, tokens[1])
		}

		if len(tokens) == 2 {
			defines[tokens[1]] = ""

			return nil
		}

		line = strings.TrimLeftFunc(line[len(tokens[0]):], unicode.IsSpace)
		defines[tokens[1]] = strings.TrimLeftFunc(line[len(tokens[1]):], unicode.IsSpace)

	default:
		return fmt.Errorf("%w: %s", ErrUnknownDirective, tokens[0])
	}

	return nil
}
