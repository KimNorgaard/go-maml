package lexer_test

import (
	_ "embed"
	"testing"

	"github.com/KimNorgaard/go-maml/lexer"
	"github.com/KimNorgaard/go-maml/token"
	"github.com/stretchr/testify/require"
)

func TestNextToken(t *testing.T) {
	input := `
# Top-level comment
{
  # An inline comment
  key-1: "value with \"escapes\""
  123: 12345
  negative: -100
  float_val: 6.626e-34
  is_true: true
  is_false: false
  is_null: null
  arr: [1, "two",]
  multi: """
    hello
    world
  """
}
`
	expectedTokens := []struct {
		expectedType    token.TokenType
		expectedLiteral string
		expectedLine    int
		expectedColumn  int
	}{
		{token.NEWLINE, "\n", 1, 1},
		{token.COMMENT, "Top-level comment", 2, 1},
		{token.NEWLINE, "\n", 2, 20},
		{token.LBRACE, "{", 3, 1},
		{token.NEWLINE, "\n", 3, 2},
		{token.COMMENT, "An inline comment", 4, 3},
		{token.NEWLINE, "\n", 4, 22},
		{token.IDENT, "key-1", 5, 3},
		{token.COLON, ":", 5, 8},
		{token.STRING, "value with \"escapes\"", 5, 10},
		{token.NEWLINE, "\n", 5, 34},
		{token.INT, "123", 6, 3},
		{token.COLON, ":", 6, 6},
		{token.INT, "12345", 6, 8},
		{token.NEWLINE, "\n", 6, 13},
		{token.IDENT, "negative", 7, 3},
		{token.COLON, ":", 7, 11},
		{token.INT, "-100", 7, 13},
		{token.NEWLINE, "\n", 7, 17},
		{token.IDENT, "float_val", 8, 3},
		{token.COLON, ":", 8, 12},
		{token.FLOAT, "6.626e-34", 8, 14},
		{token.NEWLINE, "\n", 8, 23},
		{token.IDENT, "is_true", 9, 3},
		{token.COLON, ":", 9, 10},
		{token.TRUE, "true", 9, 12},
		{token.NEWLINE, "\n", 9, 16},
		{token.IDENT, "is_false", 10, 3},
		{token.COLON, ":", 10, 11},
		{token.FALSE, "false", 10, 13},
		{token.NEWLINE, "\n", 10, 18},
		{token.IDENT, "is_null", 11, 3},
		{token.COLON, ":", 11, 10},
		{token.NULL, "null", 11, 12},
		{token.NEWLINE, "\n", 11, 16},
		{token.IDENT, "arr", 12, 3},
		{token.COLON, ":", 12, 6},
		{token.LBRACK, "[", 12, 8},
		{token.INT, "1", 12, 9},
		{token.COMMA, ",", 12, 10},
		{token.STRING, "two", 12, 12},
		{token.COMMA, ",", 12, 17},
		{token.RBRACK, "]", 12, 18},
		{token.NEWLINE, "\n", 12, 19},
		{token.IDENT, "multi", 13, 3},
		{token.COLON, ":", 13, 8},
		{token.STRING, "    hello\n    world\n  ", 13, 10},
		{token.NEWLINE, "\n", 16, 6},
		{token.RBRACE, "}", 17, 1},
		{token.NEWLINE, "\n", 17, 2},
		{token.EOF, "", 18, 1},
	}

	l := lexer.New([]byte(input))

	for i, tt := range expectedTokens {
		tok := l.NextToken()
		require.Equal(t, tt.expectedType, tok.Type, "test[%d] - wrong token type. expected=%q, got=%q", i, tt.expectedType, tok.Type)
		require.Equal(t, tt.expectedLiteral, tok.Literal, "test[%d] - wrong literal. expected=%q, got=%q", i, tt.expectedLiteral, tok.Literal)
		require.Equal(t, tt.expectedLine, tok.Line, "test[%d] - wrong line. expected=%d, got=%d", i, tt.expectedLine, tok.Line)
		require.Equal(t, tt.expectedColumn, tok.Column, "test[%d] - wrong column. expected=%d, got=%d", i, tt.expectedColumn, tok.Column)
	}
}

func TestStringEscapes(t *testing.T) {
	tests := []struct {
		input     string
		expected  string
		isIllegal bool
	}{
		{`""`, "", false},
		{`"\""`, `"`, false},
		{`"\\"`, `\`, false},
		{`"\/"`, `/`, false},
		{`"\b"`, "\b", false},
		{`"\f"`, "\f", false},
		{`"\n"`, "\n", false},
		{`"\r"`, "\r", false},
		{`"\t"`, "\t", false},
		{`"\u0022"`, `"`, false},
		{`"\uD83D\uDE00"`, "invalid unicode scalar value (surrogate pair)", true},
		{`"\u12G"`, "invalid unicode escape", true},
		{`"\x"`, "invalid escape sequence \\x", true},
		{`"a\qc"`, "invalid escape sequence \\q", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			l := lexer.New([]byte(tt.input))
			tok := l.NextToken()
			if tt.isIllegal {
				require.Equal(t, token.ILLEGAL, tok.Type)
			} else {
				require.Equal(t, token.STRING, tok.Type)
			}
			require.Equal(t, tt.expected, tok.Literal)
		})
	}
}

func TestIllegalTokens(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		expectedLiteral string
	}{
		{
			name:            "Unterminated string",
			input:           `"hello`,
			expectedLiteral: "unterminated string",
		},
		{
			name:            "Unterminated multiline string",
			input:           `"""hello`,
			expectedLiteral: "unterminated multiline string",
		},
		{
			name:            "Invalid character",
			input:           `^`,
			expectedLiteral: `^`,
		},
		{
			name:            "Invalid utf8 sequence",
			input:           string([]byte{0xff}),
			expectedLiteral: "invalid utf-8",
		},
		{
			name:            "Unterminated string after multi-byte char",
			input:           `"é`,
			expectedLiteral: "unterminated string",
		},
		{
			name:            "Invalid utf8 sequence in string",
			input:           "\"\xff\"",
			expectedLiteral: "invalid utf-8 sequence in string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New([]byte(tt.input))
			tok := l.NextToken()
			require.Equal(t, token.ILLEGAL, tok.Type)
			require.Equal(t, tt.expectedLiteral, tok.Literal)
		})
	}
}

func TestIdentifierAndNumberParsing(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedType token.TokenType
		expectedLit  string
	}{
		// These should be IDENTs
		{"identifier with hyphen", "abc-123", token.IDENT, "abc-123"},
		{"identifier with leading digits", "123-abc", token.IDENT, "123-abc"},
		{"identifier with leading zero", "0123", token.IDENT, "0123"}, // Not a valid INT
		{"invalid number is identifier", "1.2.3", token.IDENT, "1.2.3"},
		{"invalid number is identifier 2", "5e-", token.IDENT, "5e-"},

		// These should be valid numbers
		{"integer", "12345", token.INT, "12345"},
		{"negative integer", "-100", token.INT, "-100"},
		{"zero", "0", token.INT, "0"},
		{"float", "123.45", token.FLOAT, "123.45"},
		{"float with exponent", "6.626e-34", token.FLOAT, "6.626e-34"},
		{"negative float", "-0.01", token.FLOAT, "-0.01"},
		// This case tests an edge case of the number/identifier logic,
		// though it does not cover the empty string case for parseAsNumber
		// as that path appears to be unreachable from the lexer's public API.
		{"just a hyphen is an identifier", "-", token.IDENT, "-"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New([]byte(tt.input))
			tok := l.NextToken()

			require.Equal(t, tt.expectedType, tok.Type, "wrong token type")
			require.Equal(t, tt.expectedLit, tok.Literal, "wrong literal")

			// Make sure we consumed the whole input
			eof := l.NextToken()
			require.Equal(t, token.EOF, eof.Type, "should be EOF")
		})
	}
}

func TestControlCharacterValidation(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectIllegal bool
		expectedType  token.TokenType
	}{
		// Strings
		{
			name:          "valid string with tab",
			input:         "\"\t\"",
			expectIllegal: false,
			expectedType:  token.STRING,
		},
		{
			name:          "string with forbidden char",
			input:         "\"\x01\"",
			expectIllegal: true,
		},
		// Multiline Strings
		{
			name:          "valid multiline with tab",
			input:         "\"\"\"\t\"\"\"",
			expectIllegal: false,
			expectedType:  token.STRING,
		},
		{
			name:          "multiline with forbidden char",
			input:         "\"\"\"\x07\"\"\"",
			expectIllegal: true,
		},
		// Comments
		{
			name:          "valid comment",
			input:         "# hello world",
			expectIllegal: false,
			expectedType:  token.COMMENT,
		},
		{
			name:          "comment with forbidden char",
			input:         "# hello\x0fworld",
			expectIllegal: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New([]byte(tt.input))
			tok := l.NextToken()

			if tt.expectIllegal {
				require.Equal(t, token.ILLEGAL, tok.Type, "expected token to be ILLEGAL")
			} else {
				require.Equal(t, tt.expectedType, tok.Type, "wrong token type")
			}
		})
	}
}

func TestMultilineStringEdgeCases(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		expectedLiteral string
	}{
		{
			name:            "Empty multiline string",
			input:           `""""""`,
			expectedLiteral: "",
		},
		{
			name:            "Leading newline is ignored",
			input:           "\"\"\"\nhello\"\"\"",
			expectedLiteral: "hello",
		},
		{
			name:            "No leading newline to ignore",
			input:           `"""hello"""`,
			expectedLiteral: "hello",
		},
		{
			name:            "Contains single and double quotes",
			input:           `"""a " b "" c"""`,
			expectedLiteral: `a " b "" c`,
		},
		{
			name:            "String with only a newline",
			input:           "\"\"\"\n\n\"\"\"",
			expectedLiteral: "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New([]byte(tt.input))
			tok := l.NextToken()
			require.Equal(t, token.STRING, tok.Type)
			require.Equal(t, tt.expectedLiteral, tok.Literal)
			require.Equal(t, token.EOF, l.NextToken().Type)
		})
	}
}

func TestNewlineHandling(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedType token.TokenType
		expectedLit  string
	}{
		{"LF newline", "\n", token.NEWLINE, "\n"},
		{"CRLF newline", "\r\n", token.NEWLINE, "\r\n"},
		{"Standalone CR is illegal", "\r", token.ILLEGAL, "\r"},
		{"CR not followed by LF is illegal", "a\rb", token.ILLEGAL, "\r"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New([]byte(tt.input))
			// Skip the 'a' token for the last test case
			if tt.name == "CR not followed by LF is illegal" {
				l.NextToken()
			}
			tok := l.NextToken()
			require.Equal(t, tt.expectedType, tok.Type)
			require.Equal(t, tt.expectedLit, tok.Literal)
		})
	}
}

//go:embed testdata/large.maml
var benchmarkInput []byte

func BenchmarkNextToken(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		l := lexer.New(benchmarkInput)
		for {
			tok := l.NextToken()
			if tok.Type == token.EOF {
				break
			}
		}
	}
}
