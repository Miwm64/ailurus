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

func handleConditions(tokens []token, skips []conditionFrame, defines map[string]string) (string, []conditionFrame, error) {
	switch tokens[0].value {
	case "#ifdef":
		if len(tokens) != 2 || tokens[1].kind != NAME {
			return "", skips, ErrUnexpectedChar
		}
		if _, ok := defines[tokens[1].value]; ok {
			skips = append(skips, conditionFrame{
				skip:     false,
				elseUsed: false,
			})
		} else {
			skips = append(skips, conditionFrame{
				skip:     true,
				elseUsed: false,
			})
		}

	case "#ifndef":
		if len(tokens) != 2 || tokens[1].kind != NAME {
			return "", skips, ErrUnexpectedChar
		}
		if _, ok := defines[tokens[1].value]; ok {
			skips = append(skips, conditionFrame{
				skip:     true,
				elseUsed: false,
			})
		} else {
			skips = append(skips, conditionFrame{
				skip:     false,
				elseUsed: false,
			})
		}

	case "#else":
		if len(tokens) != 1 {
			return "", skips, ErrUnexpectedChar
		}
		if len(skips) < 1 {
			return "", skips, ErrUnexpectedChar
		}
		if skips[len(skips)-1].elseUsed {
			return "", skips, ErrUnexpectedChar
		}
		skips[len(skips)-1].skip = !skips[len(skips)-1].skip
		skips[len(skips)-1].elseUsed = true

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
