package preprocessor

import (
	"maps"
	"slices"
	"strings"
)

func Preprocess(text string, globalDefines map[string]string) (res string, err error) {
	lines := strings.SplitSeq(text, "\n")
	defines := make(map[string]string)
	maps.Copy(defines, globalDefines)
	skips := make([]int, 0)

	for line := range lines {
		var l string
		l, skips, err = handleLine(line, skips, defines)
		if err != nil {
			return "", err
		}
		res += l
	}
	//res = res[:len(res)-1]
	if len(skips) > 0 {
		return "", ErrUnexpectedChar
	}
	return
}

var allowed = map[string]struct{}{
	"#ifdef":  {},
	"#ifndef": {},
	"#else":   {},
	"#endif":  {},
}

// 1 - skip, 0 - keep
func handleLine(line string, skips []int, defines map[string]string) (string, []int, error) {
	line = strings.TrimSpace(line)
	if len(line) == 0 {
		return "", skips, nil
	}

	if _, ok := allowed[strings.Fields(line)[0]]; slices.Contains(skips, 1) && !ok {
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
