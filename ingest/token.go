package ingest

const (
	TokenTypeLBrace = iota
	TokenTypeRBrace
	TokenTypeQuote
	TokenTypeName
	TokenTypeEquals
	TokenTypeComma
)

var TokenMapping = map[TokenType]string{
	TokenTypeLBrace: "{",
	TokenTypeRBrace: "}",
	TokenTypeQuote:  "\"",
	TokenTypeName:   "<name>",
	TokenTypeEquals: "=",
	TokenTypeComma:  ",",
}

type TokenType int

type Token struct {
	TokenType TokenType
	StringVal string
	Line      int
}

type TokenList []Token

func (in TokenList) at(index int) *Token {
	if index < len(in) {
		return &in[index]
	}
	return nil
}
