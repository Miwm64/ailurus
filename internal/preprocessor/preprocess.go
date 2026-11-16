package preprocessor

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/Miwm64/ailurus/internal/tokenizer"
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

	for i := range origins {
		origins[i] = i + 1
	}

	defines := make(map[string]string)
	maps.Copy(defines, globalDefines)
	macros := make(map[string]*Macro)
	skips := make([]conditionFrame, 0)

	var currentMacro *Macro

	var out []string
	lineMap = map[int]int{}

	for i := 0; i < len(lines); i++ {
		var l []string

		l, skips, currentMacro, err = handleLine(lines[i], fromMacro[i], skips, currentMacro, defines, macros)
		if err != nil {
			return "", nil, fmt.Errorf("preprocess: line %d: %w", origins[i], err)
		}

		if len(l) == 0 {
			continue
		}

		if l[0] != "" {
			lineMap[len(out)] = origins[i]                    // index this line WILL have
			out = append(out, strings.TrimSuffix(l[0], "\n")) // normalise: no trailing "\n"
		}

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

	return strings.Join(out, "\n"), lineMap, nil
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

	tokens, err := tokenizer.Tokenize(line)
	if err != nil || len(tokens) == 0 {
		return nil, skips, currentMacro, err
	}

	if tokens[0].Kind == tokenizer.PREPROCESSOR_DIRECTIVE && tokens[0].Value == "#macro" {
		m, err := handleMacroHeader(tokens, macros)

		return nil, skips, m, err
	}

	if tokens[0].Kind == tokenizer.PREPROCESSOR_DIRECTIVE {
		l, frames, err := handleConditions(tokens, skips, defines)

		return []string{l}, frames, currentMacro, err
	}

	lines, err := handleNormalLine(tokens, fromMacro, defines, macros)

	return lines, skips, currentMacro, err
}

func handleNormalLine(tokens []tokenizer.Token, fromMacro bool, defines map[string]string,
	macros map[string]*Macro) ([]string, error) {

	label, call := "", 0
	if len(tokens) >= 2 && tokens[0].Kind == tokenizer.NAME && tokens[1].Kind == tokenizer.COLON {
		label, call = tokens[0].Value+":\n", 2
	}

	if call < len(tokens) && tokens[call].Kind == tokenizer.NAME {
		if macro, ok := macros[tokens[call].Value]; ok {
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

		switch tok.Kind {
		case tokenizer.PREPROCESSOR_DIRECTIVE, tokenizer.LBRACE, tokenizer.RBRACE:
			return nil, fmt.Errorf("%w: %s", tokenizer.ErrUnexpectedChar, tok.Value)
		case tokenizer.NAME:
			if value, ok := defines[tok.Value]; ok {
				res.WriteString(value)

				break
			}

			res.WriteString(tok.Value)
		default:
			res.WriteString(tokenText(tok))
		}
	}

	res.WriteString("\n")

	return []string{res.String()[1:]}, nil
}

// splitArguments joins the tokens between commas into one argument each.
func splitArguments(tokens []tokenizer.Token) []string {
	if len(tokens) == 0 {
		return nil
	}

	args := make([]string, 0)
	parts := make([]string, 0)

	for _, tok := range tokens {
		if tok.Kind == tokenizer.COMMA {
			args = append(args, strings.Join(parts, " "))
			parts = parts[:0]

			continue
		}

		parts = append(parts, tokenText(tok))
	}

	return append(args, strings.Join(parts, " "))
}

func tokenText(tok tokenizer.Token) string {
	if tok.Kind == tokenizer.STRING {
		return "\"" + tok.Value + "\""
	}

	return tok.Value
}

func isBraceLine(line string, brace byte) bool {
	if line[0] != brace {
		return false
	}

	rest := strings.TrimSpace(line[1:])

	return rest == "" || rest[0] == ';'
}
