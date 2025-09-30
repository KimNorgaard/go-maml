package maml

import (
	"testing"

	"github.com/KimNorgaard/go-maml/token"
	"github.com/stretchr/testify/require"
)

func TestNextToken(t *testing.T) {
	input := `{
	  "key": "value",
	  "multi": """hello
	world""",
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
		{token.NEWLINE, "\n", 2, 19},
		{token.STRING, "multi", 3, 4},
		{token.COLON, ":", 3, 11},
		{token.STRING, "hello\n\tworld", 3, 13},
		{token.COMMA, ",", 4, 10},
		{token.NEWLINE, "\n", 4, 11},
		{token.RBRACE, "}", 5, 2},
		{token.EOF, "", 5, 3},
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
