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

	return nil
}

func handleConditions(tokens []token, skips []int, defines map[string]string) (string, []int, error) {
	switch tokens[0].value {
	case "#ifdef":
		if len(tokens) != 2 || tokens[1].kind != NAME {
			return "", skips, ErrUnexpectedChar
		}
		if _, ok := defines[tokens[1].value]; ok {
			skips = append(skips, 0)
		} else {
			skips = append(skips, 1)
		}

	case "#ifndef":
		if len(tokens) != 2 || tokens[1].kind != NAME {
			return "", skips, ErrUnexpectedChar
		}
		if _, ok := defines[tokens[1].value]; ok {
			skips = append(skips, 1)
		} else {
			skips = append(skips, 0)
		}

	case "#else":
		if len(tokens) != 1 {
			return "", skips, ErrUnexpectedChar
		}
		if len(skips) < 1 {
			return "", skips, ErrUnexpectedChar
		}
		skips[len(skips)-1] = (skips[len(skips)-1] + 1) % 2 // invert from 0 to 1 and vice versa

	case "#endif":
		if len(tokens) != 1 {
			return "", skips, ErrUnexpectedChar
		}
		if len(skips) < 1 {
			return "", skips, ErrUnexpectedChar
		}
		skips = skips[0 : len(skips)-1]
	default:
		return "", skips, ErrUnknownDirective

	}

	return "", skips, nil

}
