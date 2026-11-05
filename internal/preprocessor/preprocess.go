package preprocessor

import (
	"maps"
	"strings"
)

func Preprocess(text string, globalDefines map[string]string) (res string, err error) {
	lines := strings.SplitSeq(text, "\n")
	defines := make(map[string]string)
	maps.Copy(defines, globalDefines)

	for line := range lines {
		l, err := handleLine(line, defines)
		if err != nil {
			return "", err
		}
		res += l
	}
	res = res[:len(res)-1]

	return
}

func handleLine(line string, defines map[string]string) (string, error) {
	line = strings.TrimSpace(line)
	if len(line) == 0 {
		return "", nil
	}

	if strings.HasPrefix(line, "#") {
		err := handleDirective(line, defines)
		if err != nil {
			return "", err
		}
		return "", nil
	}
	return deleteComment(line) + "\n", nil
}

func deleteComment(str string) string {
	return str
}
