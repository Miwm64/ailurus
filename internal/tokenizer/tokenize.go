package tokenizer

import (
	"errors"
	"fmt"
	"strings"
)

type Kind int

const (
	PREPROCESSOR_DIRECTIVE Kind = iota
	ASSEMBLER_DIRECTIVE
	NAME
	NUMBER
	STRING
	COLON
	COMMA
	PLUS
	MINUS
	AT
	LPAREN
	RPAREN
	LBRACE
	RBRACE
)

var kindNames = [...]string{
	PREPROCESSOR_DIRECTIVE: "preprocessor directive",
	ASSEMBLER_DIRECTIVE:    "assembler directive",
	NAME:                   "name",
	NUMBER:                 "number",
	STRING:                 "string",
	COLON:                  "':'",
	COMMA:                  "','",
	PLUS:                   "'+'",
	MINUS:                  "'-'",
	AT:                     "'@'",
	LPAREN:                 "'('",
	RPAREN:                 "')'",
	LBRACE:                 "'{'",
	RBRACE:                 "'}'",
}

func (k Kind) String() string {
	if k < 0 || int(k) >= len(kindNames) {
		return fmt.Sprintf("Kind(%d)", int(k))
	}
	return kindNames[k]
}

var (
	ErrUnexpectedChar     = errors.New("unexpected character")
	ErrUnterminatedString = errors.New("unterminated string")
	ErrInvalidEscape      = errors.New("invalid escape sequence")
	ErrInvalidNumber      = errors.New("invalid number")
)

var symbols = map[byte]Kind{
	':': COLON,
	',': COMMA,
	'+': PLUS,
	'-': MINUS,
	'@': AT,
	'(': LPAREN,
	')': RPAREN,
	'{': LBRACE,
	'}': RBRACE,
}

type Token struct {
	Kind  Kind
	Value string
}

func Tokenize(input string) ([]Token, error) {
	tokens := make([]Token, 0)

	for i := 0; i < len(input); {
		switch input[i] {
		case ' ', '\t':
			i++

			continue
		case ';':
			return tokens, nil
		}

		tok, end, err := scanToken(input, i)
		if err != nil {
			return nil, err
		}

		tokens = append(tokens, tok)
		i = end
	}

	return tokens, nil
}

func scanToken(input string, i int) (Token, int, error) {
	ch := input[i]

	switch {
	case ch == '#':
		return scanDirective(input, i, PREPROCESSOR_DIRECTIVE)
	case ch == '.':
		return scanDirective(input, i, ASSEMBLER_DIRECTIVE)
	case isLetter(ch) || ch == '_':
		return scanName(input, i)
	case isDigit(ch):
		return scanNumber(input, i)
	case ch == '"':
		return scanString(input, i)
	default:
		return scanSymbol(input, i)
	}
}

func scanDirective(input string, start int, kind Kind) (Token, int, error) {
	j := start + 1
	for j < len(input) && isLetter(input[j]) {
		j++
	}

	return Token{kind, input[start:j]}, j, nil
}

func scanName(input string, start int) (Token, int, error) {
	j := start + 1
	for j < len(input) && isNameChar(input[j]) {
		j++
	}

	return Token{NAME, input[start:j]}, j, nil
}

func scanNumber(input string, start int) (Token, int, error) {
	j := start + 1

	hex := input[start] == '0' && j < len(input) && input[j] == 'x'
	if hex {
		j++
	}

	digitsStart := j
	for j < len(input) && (isDigit(input[j]) || (hex && isHex(input[j]))) {
		j++
	}

	if (hex && j == digitsStart) || (j < len(input) && isNameChar(input[j])) {
		return Token{}, 0, fmt.Errorf("%w at column %d", ErrInvalidNumber, start+1)
	}

	return Token{NUMBER, input[start:j]}, j, nil
}

func scanString(input string, start int) (Token, int, error) {
	j := start + 1

	for j < len(input) {
		ch := input[j]

		switch {
		case ch == '"':
			return Token{STRING, input[start+1 : j]}, j + 1, nil
		case ch == '\\':
			if j+1 >= len(input) || strings.IndexByte(`nt\"0`, input[j+1]) < 0 {
				return Token{}, 0, fmt.Errorf("%w at column %d", ErrInvalidEscape, j+1)
			}

			j += 2
		case ch < 0x20 || ch > 0x7E:
			return Token{}, 0, fmt.Errorf("%w %q at column %d", ErrUnexpectedChar, ch, j+1)
		default:
			j++
		}
	}

	return Token{}, 0, fmt.Errorf("%w at column %d", ErrUnterminatedString, start+1)
}

func scanSymbol(input string, start int) (Token, int, error) {
	kind, ok := symbols[input[start]]
	if !ok {
		return Token{}, 0, fmt.Errorf("%w %q at column %d", ErrUnexpectedChar, input[start], start+1)
	}

	return Token{kind, input[start : start+1]}, start + 1, nil
}

func isLetter(ch byte) bool {
	return ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z'
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

func isHex(ch byte) bool {
	return isDigit(ch) || ch >= 'a' && ch <= 'f' || ch >= 'A' && ch <= 'F'
}

func isNameChar(ch byte) bool {
	return isLetter(ch) || isDigit(ch) || ch == '_'
}
