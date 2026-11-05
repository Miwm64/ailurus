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
	ErrExpectedName     = errors.New("expected a name")
	ErrUnexpectedArgs   = errors.New("unexpected arguments")
	ErrElseWithoutIf    = errors.New("#else without #ifdef/#ifndef")
	ErrDuplicateElse    = errors.New("duplicate #else")
	ErrEndifWithoutIf   = errors.New("#endif without #ifdef/#ifndef")
	ErrUnclosedIf       = errors.New("unclosed #ifdef/#ifndef")
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
	directive := tokens[0].value

	switch directive {
	case "#ifdef", "#ifndef":
		if len(tokens) < 2 || tokens[1].kind != NAME {
			return "", skips, fmt.Errorf("%w after %s", ErrExpectedName, directive)
		}

		if len(tokens) > 2 {
			return "", skips, fmt.Errorf("%w after %s %s", ErrUnexpectedArgs, directive, tokens[1].value)
		}

		_, defined := defines[tokens[1].value]
		skip := !defined
		if directive == "#ifndef" {
			skip = defined
		}

		skips = append(skips, conditionFrame{skip: skip})

	case "#else":
		if len(tokens) != 1 {
			return "", skips, fmt.Errorf("%w after %s", ErrUnexpectedArgs, directive)
		}

		if len(skips) == 0 {
			return "", skips, ErrElseWithoutIf
		}

		top := &skips[len(skips)-1]
		if top.elseUsed {
			return "", skips, ErrDuplicateElse
		}

		top.skip = !top.skip
		top.elseUsed = true

	case "#endif":
		if len(tokens) != 1 {
			return "", skips, fmt.Errorf("%w after %s", ErrUnexpectedArgs, directive)
		}

		if len(skips) == 0 {
			return "", skips, ErrEndifWithoutIf
		}

		skips = skips[:len(skips)-1]

	default:
		return "", skips, fmt.Errorf("%w: %s", ErrUnknownDirective, directive)
	}

	return "", skips, nil
}
