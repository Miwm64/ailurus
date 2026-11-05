package preprocessor

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

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

func Preprocess(text string, globalDefines map[string]string) (res string, err error) {
	defines := make(map[string]string)
	maps.Copy(defines, globalDefines)
	macros := make(map[string]*Macro)
	skips := make([]conditionFrame, 0)

	var currentMacro *Macro

	lineNum := 0
	for line := range strings.SplitSeq(text, "\n") {
		lineNum++

		var l string

		l, skips, currentMacro, err = handleLine(line, skips, currentMacro, defines, macros)
		if err != nil {
			return "", fmt.Errorf("line %d: %w", lineNum, err)
		}

		res += l
	}

	if currentMacro != nil {
		return "", fmt.Errorf("%w: %s", ErrUnclosedMacro, currentMacro.Name)
	}

	if len(skips) > 0 {
		return "", ErrUnclosedIf
	}

	return strings.TrimSuffix(res, "\n"), nil
}

func handleLine(line string, skips []conditionFrame, currentMacro *Macro,
	defines map[string]string, macros map[string]*Macro) (string, []conditionFrame, *Macro, error) {
	line = strings.TrimSpace(line)
	if len(line) == 0 {
		return "", skips, currentMacro, nil
	}

	if currentMacro != nil {
		m, err := handleMacroLine(line, currentMacro, macros)

		return "", skips, m, err
	}

	first := strings.Fields(line)[0]

	skipping := slices.ContainsFunc(skips, func(f conditionFrame) bool { return f.skip })
	if _, ok := allowed[first]; skipping && !ok {
		return "", skips, currentMacro, nil
	}

	// #define values are raw text and can not be tokenized
	if first == "#define" {
		return "", skips, currentMacro, handleDefine(line, defines)
	}

	tokens, err := tokenize(line)
	if err != nil || len(tokens) == 0 {
		return "", skips, currentMacro, err
	}

	if tokens[0].kind == PREPROCESSOR_DIRECTIVE && tokens[0].value == "#macro" {
		m, err := handleMacroHeader(tokens, macros)

		return "", skips, m, err
	}

	if tokens[0].kind == PREPROCESSOR_DIRECTIVE {
		l, frames, err := handleConditions(tokens, skips, defines)

		return l, frames, currentMacro, err
	}

	return line + "\n", skips, currentMacro, nil
}

//nolint:nilnil no open macro
func handleMacroLine(line string, currentMacro *Macro, macros map[string]*Macro) (*Macro, error) {
	if !currentMacro.initialized {
		if line[0] == '{' {
			return currentMacro, fmt.Errorf("%w: %s", ErrExpectedBrace, currentMacro.Name)
		}

		currentMacro.initialized = true

		return currentMacro, nil
	}

	if line[0] == '}' {
		macros[currentMacro.Name] = currentMacro

		return nil, nil
	}

	if strings.Fields(line)[0] == "#macro" {
		return currentMacro, ErrNestedMacro
	}

	currentMacro.AddRawLine(line)

	return currentMacro, nil
}
