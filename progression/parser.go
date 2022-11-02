package progression

import (
	"errors"
	"time"
)

type ProgressionParser struct {
	index  int
	tokens TokenList
}

func NewProgressionParser(tokens TokenList) *ProgressionParser {
	return &ProgressionParser{
		index:  0,
		tokens: tokens,
	}
}

func (p *ProgressionParser) hasTokens() bool {
	return p.index < len(p.tokens)
}

func (p *ProgressionParser) next() (*Token, error) {
	if !p.hasTokens() {
		return nil, errors.New("unexpected end of stream")
	}
	current := p.index
	p.index = p.index + 1
	return p.tokens.at(current), nil
}

func (p *ProgressionParser) peek() (*Token, error) {
	if !p.hasTokens() {
		return nil, errors.New("unexpected end of stream")
	}
	return p.tokens.at(p.index), nil
}

func (p *ProgressionParser) expect(t TokenType) (*Token, error) {
	token, err := p.next()
	if err != nil {
		return nil, err
	}

	if token.TokenType == t {
		return token, nil
	}

	return nil, errors.New("unexpected token: " + token.StringVal)
}

func (p *ProgressionParser) nodata() (*Progression, error) {
	progression := Progression{
		NoData: true,
	}

	_, err := p.expect(TokenTypeUnderscore)
	if err != nil {
		return nil, err
	}

	_, err = p.expect(TokenTypeX)
	if err != nil {
		return nil, err
	}

	token, err := p.expect(TokenTypeValue)
	if err != nil {
		return nil, err
	}
	progression.Times = token.FloatVal
	return &progression, nil
}

func (p *ProgressionParser) progression() (*Progression, error) {
	progression := Progression{
		NoData: false,
	}
	token, err := p.expect(TokenTypeValue)
	if err != nil {
		return nil, err
	}
	progression.Initial = token.FloatVal

	incrementType, err := p.expect(TokenTypePlusMinus)
	if err != nil {
		return nil, err
	}

	incrementValue, err := p.expect(TokenTypeValue)
	if err != nil {
		return nil, err
	}

	if incrementType.StringVal == "+" {
		progression.Increment = incrementValue.FloatVal
	} else {
		progression.Increment = -incrementValue.FloatVal
	}

	p.expect(TokenTypeX)
	iterations, err := p.expect(TokenTypeValue)
	if err != nil {
		return nil, err
	}

	progression.Times = iterations.FloatVal
	return &progression, nil
}

func (p *ProgressionParser) Parse(interval time.Duration) (*ProgressionList, error) {
	list := &ProgressionList{
		interval:   interval,
		iterations: 0,
	}
	for p.hasTokens() {
		nextToken, err := p.peek()
		if err != nil {
			return nil, err
		}

		if nextToken.TokenType == TokenTypeUnderscore {
			progression, err := p.nodata()
			if err != nil {
				return nil, err
			}
			list.progressions = append(list.progressions, progression)
		} else {
			progression, err := p.progression()
			if err != nil {
				return nil, err
			}
			list.progressions = append(list.progressions, progression)
		}
	}
	list.startTimestamp = time.Now().UnixMilli() - (list.count() * list.interval.Milliseconds())
	return list, nil
}
