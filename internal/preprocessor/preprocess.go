package preprocessor

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
)

var ErrNestedMacroCall = errors.New("macro call inside a macro expansion")

type conditionFrame struct {
	skip     bool
	elseUsed bool
}

var allowed = map[string]struct{}{
	"#ifdef":  {},
	"#ifndef": {},
	"#else":   {},
	"#endif":  {},
}

func Preprocess(text string, globalDefines map[string]string) (res string, lineMap map[int]int, err error) {
	lines := strings.Split(text, "\n")
	origins := make([]int, len(lines))
	fromMacro := make([]bool, len(lines))

	lineMap = make(map[int]int)
	outLine := 0

	for i := range origins {
		origins[i] = i + 1
	}

	defines := make(map[string]string)
	maps.Copy(defines, globalDefines)
	macros := make(map[string]*Macro)
	skips := make([]conditionFrame, 0)

	var currentMacro *Macro

	for i := 0; i < len(lines); i++ {
		var l []string

		l, skips, currentMacro, err = handleLine(lines[i], fromMacro[i], skips, currentMacro, defines, macros)
		if err != nil {
			return "", nil, fmt.Errorf("line %d: %w", origins[i], err)
		}

		if len(l) == 0 {
			continue
		}

		if l[0] != "" {
			outLine++
			lineMap[outLine] = origins[i]
		}

		res += l[0]

		res += l[0]

		if len(l) == 1 {
			continue
		}

		n := len(l) - 1
		lines = slices.Insert(lines, i+1, l[1:]...)
		origins = slices.Insert(origins, i+1, slices.Repeat([]int{origins[i]}, n)...)
		fromMacro = slices.Insert(fromMacro, i+1, slices.Repeat([]bool{true}, n)...)
	}

	if currentMacro != nil {
		return "", nil, fmt.Errorf("%w: %s", ErrUnclosedMacro, currentMacro.Name)
	}

	if len(skips) > 0 {
		return "", nil, ErrUnclosedIf
	}

	return strings.TrimSuffix(res, "\n"), lineMap, nil
}

func handleLine(line string, fromMacro bool, skips []conditionFrame, currentMacro *Macro,
	defines map[string]string, macros map[string]*Macro) ([]string, []conditionFrame, *Macro, error) {
	line = strings.TrimSpace(line)
	if len(line) == 0 {
		return nil, skips, currentMacro, nil
	}

	if currentMacro != nil {
		m, err := handleMacroLine(line, currentMacro, macros)

		return nil, skips, m, err
	}

	first := strings.Fields(line)[0]

	skipping := slices.ContainsFunc(skips, func(f conditionFrame) bool { return f.skip })
	if _, ok := allowed[first]; skipping && !ok {
		return nil, skips, currentMacro, nil
	}

	// #define values are raw text and can not be tokenized
	if first == "#define" {
		return nil, skips, currentMacro, handleDefine(line, defines)
	}

	tokens, err := tokenize(line)
	if err != nil || len(tokens) == 0 {
		return nil, skips, currentMacro, err
	}

	if tokens[0].kind == PREPROCESSOR_DIRECTIVE && tokens[0].value == "#macro" {
		m, err := handleMacroHeader(tokens, macros)

		return nil, skips, m, err
	}

	if tokens[0].kind == PREPROCESSOR_DIRECTIVE {
		l, frames, err := handleConditions(tokens, skips, defines)

		return []string{l}, frames, currentMacro, err
	}

	lines, err := handleNormalLine(tokens, fromMacro, defines, macros)

	return lines, skips, currentMacro, err
}

func handleNormalLine(tokens []token, fromMacro bool, defines map[string]string,
	macros map[string]*Macro) ([]string, error) {

	label, call := "", 0
	if len(tokens) >= 2 && tokens[0].kind == NAME && tokens[1].kind == COLON {
		label, call = tokens[0].value+":\n", 2
	}

	if call < len(tokens) && tokens[call].kind == NAME {
		if macro, ok := macros[tokens[call].value]; ok {
			if fromMacro {
				return nil, fmt.Errorf("%w: %s", ErrNestedMacroCall, macro.Name)
			}

			expanded, err := macro.Apply(splitArguments(tokens[call+1:]))
			if err != nil {
				return nil, err
			}

			return append([]string{label}, expanded...), nil
		}
	}

	var res strings.Builder

	for _, tok := range tokens {
		res.WriteString(" ")

		switch tok.kind {
		case PREPROCESSOR_DIRECTIVE, LBRACE, RBRACE:
			return nil, fmt.Errorf("%w: %s", ErrUnexpectedChar, tok.value)
		case NAME:
			if value, ok := defines[tok.value]; ok {
				res.WriteString(value)

				break
			}

			res.WriteString(tok.value)
		default:
			res.WriteString(tokenText(tok))
		}
	}

	res.WriteString("\n")

	return []string{res.String()[1:]}, nil
}

// splitArguments joins the tokens between commas into one argument each.
func splitArguments(tokens []token) []string {
	if len(tokens) == 0 {
		return nil
	}

	args := make([]string, 0)
	parts := make([]string, 0)

	for _, tok := range tokens {
		if tok.kind == COMMA {
			args = append(args, strings.Join(parts, " "))
			parts = parts[:0]

			continue
		}

		parts = append(parts, tokenText(tok))
	}

	return append(args, strings.Join(parts, " "))
}

func tokenText(tok token) string {
	if tok.kind == STRING {
		return "\"" + tok.value + "\""
	}

	return tok.value
}

func isBraceLine(line string, brace byte) bool {
	if line[0] != brace {
		return false
	}

	rest := strings.TrimSpace(line[1:])

	return rest == "" || rest[0] == ';'
}
