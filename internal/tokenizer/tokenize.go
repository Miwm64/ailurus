package tokenizer

import (
	"errors"
	"fmt"
	"strings"
)

type Kind int

const (
	PreprocessorDirective Kind = iota
	AssemblerDirective
	Name
	Number
	String
	Colon
	Comma
	Plus
	Minus
	At
	LParen
	RParen
	LBrace
	RBrace
)

var kindNames = [...]string{
	PreprocessorDirective: "preprocessor directive",
	AssemblerDirective:    "assembler directive",
	Name:                  "name",
	Number:                "number",
	String:                "string",
	Colon:                 "':'",
	Comma:                 "','",
	Plus:                  "'+'",
	Minus:                 "'-'",
	At:                    "'@'",
	LParen:                "'('",
	RParen:                "')'",
	LBrace:                "'{'",
	RBrace:                "'}'",
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
	':': Colon,
	',': Comma,
	'+': Plus,
	'-': Minus,
	'@': At,
	'(': LParen,
	')': RParen,
	'{': LBrace,
	'}': RBrace,
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
		return scanDirective(input, i, PreprocessorDirective)
	case ch == '.':
		return scanDirective(input, i, AssemblerDirective)
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

	return Token{Name, input[start:j]}, j, nil
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

	return Token{Number, input[start:j]}, j, nil
}

func scanString(input string, start int) (Token, int, error) {
	j := start + 1

	for j < len(input) {
		ch := input[j]

		switch {
		case ch == '"':
			return Token{String, input[start+1 : j]}, j + 1, nil
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
