package preprocessor

import (
	"strings"
)

func Preprocess(text string, globalDefines map[string]string) (res string) {
	lines := strings.SplitSeq(text, "\n")

	for line := range lines {
		res += handleLine(line, globalDefines)
	}
	res = res[:len(res)-1]

	return
}

func handleLine(line string, globalDefines map[string]string) (res string) {
	line = strings.TrimLeft(line, " \t\n\r\b")
	if len(line) == 0 {
		return ""
	}

	if strings.HasPrefix(line, "#") {
		handleDirective(line, globalDefines)
	} else {

	}
	return line + "\n"
}

func handleDirective(line string, globalDefines map[string]string) (res string) {
	return line
}

func deleteComment(str string) string {
	return str
}
