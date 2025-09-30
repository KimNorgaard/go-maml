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
	}`

	expectedTokens := []struct {
		expectedType    token.TokenType
		expectedLiteral string
		expectedLine    int
		expectedColumn  int
	}{
		{token.LBRACE, "{", 1, 1},
		{token.NEWLINE, "\n", 1, 2},
		{token.STRING, "key", 2, 4},
		{token.COLON, ":", 2, 9},
		{token.STRING, "value", 2, 11},
		{token.COMMA, ",", 2, 18},
		{token.COMMENT, "# a comment", 2, 20},
		{token.NEWLINE, "\n", 2, 31},
		{token.STRING, "number", 3, 4},
		{token.COLON, ":", 3, 12},
		{token.INT, "123", 3, 14},
		{token.COMMA, ",", 3, 17},
		{token.NEWLINE, "\n", 3, 18},
		{token.RBRACE, "}", 4, 2},
		{token.EOF, "", 4, 3},
	}

	l := NewLexer([]byte(input))

	for i, tt := range expectedTokens {
		tok := l.NextToken()
		require.Equal(t, tt.expectedType, tok.Type, "test[%d] - wrong token type", i)
		require.Equal(t, tt.expectedLiteral, tok.Literal, "test[%d] - wrong literal", i)
		require.Equal(t, tt.expectedLine, tok.Line, "test[%d] - wrong line", i)
		require.Equal(t, tt.expectedColumn, tok.Column, "test[%d] - wrong column", i)
	}
}
