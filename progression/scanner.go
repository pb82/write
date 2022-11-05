package progression

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type Scanner struct {
}

func NewProgressionScanner() *Scanner {
	return &Scanner{}
}

func (*Scanner) Scan(data string) (TokenList, error) {
	var tokens TokenList
	runes := []rune(data)
	index := 0

	next := func() rune {
		current := runes[index]
		index = index + 1
		return current
	}

	currentNumber := strings.Builder{}
	currentFunction := strings.Builder{}

	consumeNumber := func() error {
		if currentNumber.Len() == 0 {
			return nil
		}
		i, err := strconv.ParseFloat(currentNumber.String(), 64)
		if err != nil {
			return err
		}
		tokens = append(tokens, Token{
			TokenType: TokenTypeValue,
			FloatVal:  i,
		})
		currentNumber.Reset()
		return nil
	}

	for index < len(runes) {
		r := next()

		// ignore whitespace
		if unicode.IsSpace(r) {
			err := consumeNumber()
			if err != nil {
				return nil, err
			}
			continue
		}

		switch r {
		case '+', '-':
			err := consumeNumber()
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, Token{
				TokenType: TokenTypePlusMinus,
				StringVal: string(r),
			})
		case 'x':
			err := consumeNumber()
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, Token{
				TokenType: TokenTypeX,
				StringVal: string(r),
			})
		case '_':
			err := consumeNumber()
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, Token{
				TokenType: TokenTypeUnderscore,
				StringVal: string(r),
			})
		case '(':
			err := consumeNumber()
			if err != nil {
				return nil, err
			}
			for {
				if index >= len(runes) {
					return nil, errors.New("unexpected end of input")
				}
				nextRune := next()
				if nextRune == ')' {
					tokens = append(tokens, Token{
						TokenType: TokenTypeFn,
						StringVal: currentFunction.String(),
					})
					currentFunction.Reset()
					break
				} else if unicode.IsLetter(nextRune) {
					currentFunction.WriteRune(nextRune)
				} else {
					return nil, errors.New(fmt.Sprintf("invalid character in function name: %v", nextRune))
				}
			}
		default:
			currentNumber.WriteRune(r)
			continue
		}
	}

	err := consumeNumber()
	if err != nil {
		return nil, err
	}

	return tokens, nil
}
