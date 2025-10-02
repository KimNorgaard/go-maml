package maml

import (
	"bytes"
	"testing"

	"github.com/KimNorgaard/go-maml/internal/ast"
	"github.com/KimNorgaard/go-maml/internal/token"
	"github.com/stretchr/testify/require"
)

func TestFormatter(t *testing.T) {
	// Define a sample AST structure to test against
	sampleAST := &ast.Document{
		Statements: []ast.Statement{
			&ast.ExpressionStatement{
				Expression: &ast.ObjectLiteral{
					Token: token.Token{Type: token.LBRACE, Literal: "{"},
					Pairs: []*ast.KeyValueExpression{
						{
							Key:   &ast.Identifier{Token: token.Token{Type: token.IDENT, Literal: "string"}, Value: "string"},
							Value: &ast.StringLiteral{Token: token.Token{Type: token.STRING, Literal: `"hello"`}, Value: "hello"},
						},
						{
							Key: &ast.Identifier{Token: token.Token{Type: token.IDENT, Literal: "nested"}, Value: "nested"},
							Value: &ast.ObjectLiteral{
								Pairs: []*ast.KeyValueExpression{
									{
										Key:   &ast.Identifier{Token: token.Token{Type: token.IDENT, Literal: "a"}, Value: "a"},
										Value: &ast.IntegerLiteral{Token: token.Token{Type: token.INT, Literal: "1"}, Value: 1},
									},
								},
							},
						},
						{
							Key: &ast.Identifier{Token: token.Token{Type: token.IDENT, Literal: "array"}, Value: "array"},
							Value: &ast.ArrayLiteral{
								Elements: []ast.Expression{
									&ast.BooleanLiteral{Token: token.Token{Type: token.TRUE, Literal: "true"}, Value: true},
									&ast.NullLiteral{Token: token.Token{Type: token.NULL, Literal: "null"}},
								},
							},
						},
					},
				},
			},
		},
	}

	testCases := []struct {
		name     string
		node     ast.Node
		opts     []Option
		expected string
	}{
		{
			name:     "Compact Mode",
			node:     sampleAST,
			opts:     []Option{Indent(0)},
			expected: `{string:"hello",nested:{a:1},array:[true,null]}`,
		},
		{
			name:     "Default Indent (2 spaces)",
			node:     sampleAST,
			opts:     []Option{Indent(2)},
			expected: "{\n  string: \"hello\",\n  nested: {\n    a: 1\n  },\n  array: [\n    true,\n    null\n  ]\n}",
		},
		{
			name:     "Custom Indent (4 spaces)",
			node:     sampleAST,
			opts:     []Option{Indent(4)},
			expected: "{\n    string: \"hello\",\n    nested: {\n        a: 1\n    },\n    array: [\n        true,\n        null\n    ]\n}",
		},
		{
			name:     "Empty Object",
			node:     &ast.ObjectLiteral{},
			opts:     []Option{Indent(2)},
			expected: "{}",
		},
		{
			name:     "Empty Array",
			node:     &ast.ArrayLiteral{},
			opts:     []Option{Indent(2)},
			expected: "[]",
		},
		{
			name: "String with quotes",
			node: &ast.StringLiteral{Value: `a "quote"`},
			opts: []Option{},
			// Note: The String() method on ast.StringLiteral adds quotes and escapes.
			expected: `"a \"quote\""`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			o := options{}
			for _, opt := range tc.opts {
				opt(&o)
			}

			f := newFormatter(&buf, &o)
			err := f.format(tc.node)

			require.NoError(t, err)
			require.Equal(t, tc.expected, buf.String())
		})
	}
}
