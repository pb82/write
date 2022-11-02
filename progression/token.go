package progression

const (
	TokenTypeValue = iota
	TokenTypePlusMinus
	TokenTypeX
	TokenTypeUnderscore
)

type TokenType int

type Token struct {
	TokenType TokenType
	StringVal string
	FloatVal  float64
}

type TokenList []Token

func (in TokenList) at(index int) *Token {
	if index < len(in) {
		return &in[index]
	}
	return nil
}
