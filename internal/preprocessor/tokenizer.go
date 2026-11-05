package preprocessor

import (
	"errors"
	"fmt"
	"strings"
)

const (
	PREPROCESSOR_DIRECTIVE = iota
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

var (
	ErrUnexpectedChar     = errors.New("unexpected character")
	ErrUnterminatedString = errors.New("unterminated string")
	ErrInvalidEscape      = errors.New("invalid escape sequence")
	ErrInvalidNumber      = errors.New("invalid number")
)

var symbols = map[byte]int{
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

type token struct {
	kind  int
	value string
}

func Tokenize(input string) (error, []token) {
	tokens := make([]token, 0)

	for i := 0; i < len(input); i++ {
		j := i
		switch {
		case input[j] == '#':
			j++
			for j < len(input) && isLetter(input[j]) {
				j++
			}
			tokens = append(tokens, token{PREPROCESSOR_DIRECTIVE, input[i:j]})
			i = j - 1

		case input[j] == '.':
			j++
			for j < len(input) && isLetter(input[j]) {
				j++
			}
			tokens = append(tokens, token{ASSEMBLER_DIRECTIVE, input[i:j]})
			i = j - 1

		case isLetter(input[i]) || input[i] == '_':
			j++
			for j < len(input) && (isLetter(input[j]) || isDigit(input[j]) || input[j] == '_') {
				j++
			}
			tokens = append(tokens, token{NAME, input[i:j]})
			i = j - 1

		case isDigit(input[j]):
			j++

			hex := input[i] == '0' && j < len(input) && input[j] == 'x'
			if hex {
				j++
			}

			digitsStart := j
			for j < len(input) && (isDigit(input[j]) || (hex && isHex(input[j]))) {
				j++
			}

			// "0x", "12ab"
			if (hex && j == digitsStart) || (j < len(input) && (isLetter(input[j]) || input[j] == '_')) {
				return fmt.Errorf("%w at column %d", ErrInvalidNumber, i+1), nil
			}

			tokens = append(tokens, token{NUMBER, input[i:j]})
			i = j - 1
		case input[j] == '"':
			j++

			closed := false
			for j < len(input) {
				ch := input[j]

				if ch == '"' {
					closed = true

					break
				}

				if ch == '\\' {
					// check if real escape symbol
					if j+1 >= len(input) || strings.IndexByte(`nt\"0`, input[j+1]) < 0 {
						return fmt.Errorf("%w at column %d", ErrInvalidEscape, j+1), nil
					}

					j += 2

					continue
				}

				if ch < 0x20 || ch > 0x7E {
					return fmt.Errorf("%w %q at column %d", ErrUnexpectedChar, ch, j+1), nil
				}

				j++
			}

			if !closed {
				return fmt.Errorf("%w at column %d", ErrUnterminatedString, i+1), nil
			}

			tokens = append(tokens, token{STRING, input[i+1 : j]})
			i = j
		case input[j] == ' ' || input[j] == '\t':
		default:
			kind, ok := symbols[input[j]]
			if !ok {
				return fmt.Errorf("%w %q at column %d", ErrUnexpectedChar, input[j], j+1), nil
			}

			tokens = append(tokens, token{kind, input[j : j+1]})
		}
	}

	return nil, tokens
}

func isLetter(ch byte) bool {
	return ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z'
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

func isHex(ch byte) bool {
	return ch >= '0' && ch <= '9' || ch >= 'A' && ch <= 'F'
}
