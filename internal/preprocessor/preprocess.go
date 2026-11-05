package preprocessor

import (
	"errors"
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
	line = strings.TrimLeft(line, " \t\n\r\b")
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
	return line + "\n", nil
}

func handleDirective(line string, defines map[string]string) error {
	tokens := strings.Fields(line)

	switch tokens[0] {
	case "#define":
		{
			if len(tokens) < 2 {
				return errors.New("directive " + tokens[0] + " was used without arguments")
			}

			if !validateName(tokens[1]) {
				return errors.New("directive name " + tokens[1] + " does not follow naming rules")
			}

			if len(tokens) == 2 {
				defines[tokens[1]] = ""
				return nil
			}

			offset := strings.Index(line, tokens[1])
			defines[tokens[1]] = strings.TrimLeft(line[offset+len(tokens[1]):], " \t\n\r\b")
		}
	default:
		return errors.New("unknown directive: " + tokens[0])
	}

	return nil
}

func deleteComment(str string) string {
	return str
}
