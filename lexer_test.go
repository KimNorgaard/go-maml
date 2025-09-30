package maml

import (
	"testing"

	"github.com/KimNorgaard/go-maml/token"
	"github.com/stretchr/testify/require"
)

func TestNextToken(t *testing.T) {
	input := `{
	  "key": "value", # a comment
	  "number": 123,
	  "float": -123.45,
	  "bool_true": true,
	  "bool_false": false,
	  "is_null": null,
	  "array": [1, "two"],
	}`

	expectedTokens := []struct {
		expectedType    token.TokenType
		expectedLiteral string
	}{
		{token.LBRACE, "{"},
		{token.NEWLINE, "\n"},
		{token.STRING, "key"},
		{token.COLON, ":"},
		{token.STRING, "value"},
		{token.COMMA, ","},
		{token.COMMENT, "# a comment"},
		{token.NEWLINE, "\n"},
		{token.STRING, "number"},
		{token.COLON, ":"},
		{token.INT, "123"},
		{token.COMMA, ","},
		{token.NEWLINE, "\n"},
		{token.STRING, "float"},
		{token.COLON, ":"},
		{token.FLOAT, "-123.45"},
		{token.COMMA, ","},
		{token.NEWLINE, "\n"},
		{token.STRING, "bool_true"},
		{token.COLON, ":"},
		{token.TRUE, "true"},
		{token.COMMA, ","},
		{token.NEWLINE, "\n"},
		{token.STRING, "bool_false"},
		{token.COLON, ":"},
		{token.FALSE, "false"},
		{token.COMMA, ","},
		{token.NEWLINE, "\n"},
		{token.STRING, "is_null"},
		{token.COLON, ":"},
		{token.NULL, "null"},
		{token.COMMA, ","},
		{token.NEWLINE, "\n"},
		{token.STRING, "array"},
		{token.COLON, ":"},
		{token.LBRACK, "["},
		{token.INT, "1"},
		{token.COMMA, ","},
		{token.STRING, "two"},
		{token.RBRACK, "]"},
		{token.COMMA, ","},
		{token.NEWLINE, "\n"},
		{token.RBRACE, "}"},
		{token.EOF, ""},
	}

	l := NewLexer([]byte(input))

	for i, tt := range expectedTokens {
		tok := l.NextToken()
		require.Equal(t, tt.expectedType, tok.Type, "test[%d] - wrong token type", i)
		require.Equal(t, tt.expectedLiteral, tok.Literal, "test[%d] - wrong literal", i)
	}
}
