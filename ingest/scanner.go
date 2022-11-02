package ingest

import (
	"strings"
	"unicode"
)

type Scanner struct {
}

func NewTimeseriesScanner() *Scanner {
	return &Scanner{}
}

func (*Scanner) Scan(data string) (TokenList, error) {
	var tokens TokenList
	runes := []rune(data)
	index := 0
	line := 0
	quoted := false

	next := func() rune {
		current := runes[index]
		index = index + 1
		return current
	}

	peek := func() rune {
		return runes[index]
	}

	name := func(t rune) Token {
		sb := strings.Builder{}
		sb.WriteRune(t)
		for index < len(runes) {
			r := peek()
			if quoted {
				if r == '"' {
					break
				}
			} else {
				if r == '{' || r == '=' || r == ',' || unicode.IsSpace(r) {
					break
				}
			}

			sb.WriteRune(next())
		}

		return Token{
			TokenType: TokenTypeName,
			StringVal: sb.String(),
			Line:      line,
		}
	}

	comment := func() {
		for index < len(runes) {
			if next() == '\n' {
				line = line + 1
				break
			}
		}
	}

	for index < len(runes) {
		r := next()

		if r == '\n' {
			line = line + 1
			continue
		}

		// ignore whitespace
		if unicode.IsSpace(r) {
			continue
		}

		switch r {
		case '#':
			comment()
		case '{':
			tokens = append(tokens, Token{
				TokenType: TokenTypeLBrace,
				Line:      line,
			})
		case '}':
			tokens = append(tokens, Token{
				TokenType: TokenTypeRBrace,
				Line:      line,
			})
		case '"':
			quoted = !quoted
			tokens = append(tokens, Token{
				TokenType: TokenTypeQuote,
				Line:      line,
			})
		case '=':
			tokens = append(tokens, Token{
				TokenType: TokenTypeEquals,
				Line:      line,
			})
		case ',':
			tokens = append(tokens, Token{
				TokenType: TokenTypeComma,
				Line:      line,
			})
		default:
			tokens = append(tokens, name(r))
		}
	}

	return tokens, nil
}
