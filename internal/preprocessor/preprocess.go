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

func Preprocess(text string, globalDefines map[string]string) (res string, err error) {
	defines := make(map[string]string)
	maps.Copy(defines, globalDefines)
	skips := make([]conditionFrame, 0)

	lineNum := 0
	for line := range strings.SplitSeq(text, "\n") {
		lineNum++

		var l string
		l, skips, err = handleLine(line, skips, defines)
		if err != nil {
			return "", fmt.Errorf("line %d: %w", lineNum, err)
		}

		res += l
	}

	if len(skips) > 0 {
		return "", ErrUnclosedIf
	}

	if len(res) > 0 {
		res = res[:len(res)-1]
	}

	return res, nil
}

var allowed = map[string]struct{}{
	"#ifdef":  {},
	"#ifndef": {},
	"#else":   {},
	"#endif":  {},
}

func handleLine(line string, skips []conditionFrame, defines map[string]string) (string, []conditionFrame, error) {
	line = strings.TrimSpace(line)
	if len(line) == 0 {
		return "", skips, nil
	}
	skipping := slices.ContainsFunc(skips, func(f conditionFrame) bool { return f.skip })

	if _, ok := allowed[strings.Fields(line)[0]]; skipping && !ok {
		return "", skips, nil
	}

	// #define directive can not be tokenized properly, as they are crude copy
	if strings.Fields(line)[0] == "#define" {
		err := handleDefine(line, defines)
		if err != nil {
			return "", skips, err
		}
		return "", skips, nil
	}

	tokens, err := tokenize(line)
	if err != nil || len(tokens) == 0 {
		return "", skips, err
	}

	if tokens[0].kind == PREPROCESSOR_DIRECTIVE && tokens[0].value == "#macro" {
		return "", skips, nil
	}

	if tokens[0].kind == PREPROCESSOR_DIRECTIVE {
		return handleConditions(tokens, skips, defines)
	}

	return line + "\n", skips, nil
}

func deleteComment(str string) string {
	return str
}
