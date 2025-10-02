package formatter_test

import (
	"bytes"
	"testing"

	"github.com/KimNorgaard/go-maml/internal/ast"
	"github.com/KimNorgaard/go-maml/internal/formatter"
	"github.com/KimNorgaard/go-maml/internal/token"
	"github.com/stretchr/testify/require"
)

func TestFormatter_Format(t *testing.T) {
	testCases := []struct {
		name     string
		node     ast.Node
		expected string
	}{
		{
			name:     "String Literal",
			node:     &ast.StringLiteral{Token: token.Token{Type: token.STRING, Literal: "hello world"}, Value: "hello world"},
			expected: `"hello world"`,
		},
		{
			name:     "Integer Literal",
			node:     &ast.IntegerLiteral{Token: token.Token{Type: token.INT, Literal: "123"}, Value: 123},
			expected: "123",
		},
		{
			name:     "Float Literal",
			node:     &ast.FloatLiteral{Token: token.Token{Type: token.FLOAT, Literal: "3.14"}, Value: 3.14},
			expected: "3.14",
		},
		{
			name:     "Boolean Literal True",
			node:     &ast.BooleanLiteral{Token: token.Token{Type: token.TRUE, Literal: "true"}, Value: true},
			expected: "true",
		},
		{
			name:     "Boolean Literal False",
			node:     &ast.BooleanLiteral{Token: token.Token{Type: token.FALSE, Literal: "false"}, Value: false},
			expected: "false",
		},
		{
			name:     "Null Literal",
			node:     &ast.NullLiteral{Token: token.Token{Type: token.NULL, Literal: "null"}},
			expected: "null",
		},
		{
			name:     "Empty Array",
			node:     &ast.ArrayLiteral{Elements: []ast.Expression{}},
			expected: "[]",
		},
		{
			name: "Array with scalars",
			node: &ast.ArrayLiteral{
				Elements: []ast.Expression{
					&ast.IntegerLiteral{Token: token.Token{Literal: "1"}},
					&ast.StringLiteral{Value: "two"},
					&ast.BooleanLiteral{Token: token.Token{Literal: "false"}},
				},
			},
			expected: `[1, "two", false]`,
		},
		{
			name:     "Empty Object",
			node:     &ast.ObjectLiteral{Pairs: []*ast.KeyValueExpression{}},
			expected: "{}",
		},
		{
			name: "Object with pairs",
			node: &ast.ObjectLiteral{
				Pairs: []*ast.KeyValueExpression{
					{
						Key:   &ast.Identifier{Value: "key1"},
						Value: &ast.StringLiteral{Value: "value1"},
					},
					{
						Key:   &ast.Identifier{Value: "key2"},
						Value: &ast.IntegerLiteral{Token: token.Token{Literal: "123"}},
					},
				},
			},
			expected: `{ key1: "value1", key2: 123 }`,
		},
		{
			name: "Nested Object and Array",
			node: &ast.ObjectLiteral{
				Pairs: []*ast.KeyValueExpression{
					{
						Key: &ast.Identifier{Value: "data"},
						Value: &ast.ArrayLiteral{
							Elements: []ast.Expression{
								&ast.ObjectLiteral{
									Pairs: []*ast.KeyValueExpression{
										{
											Key:   &ast.Identifier{Value: "id"},
											Value: &ast.IntegerLiteral{Token: token.Token{Literal: "1"}},
										},
									},
								},
								&ast.IntegerLiteral{Token: token.Token{Literal: "2"}},
							},
						},
					},
				},
			},
			expected: `{ data: [{ id: 1 }, 2] }`,
		},
		{
			name: "Document with one statement",
			node: &ast.Document{
				Statements: []ast.Statement{
					&ast.ExpressionStatement{
						Expression: &ast.StringLiteral{Value: "top-level"},
					},
				},
			},
			expected: `"top-level"`,
		},
		{
			name: "Document with multiple statements",
			node: &ast.Document{
				Statements: []ast.Statement{
					&ast.ExpressionStatement{
						Expression: &ast.IntegerLiteral{Token: token.Token{Literal: "1"}},
					},
					&ast.ExpressionStatement{
						Expression: &ast.IntegerLiteral{Token: token.Token{Literal: "2"}},
					},
				},
			},
			expected: "1\n2",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			f := formatter.New(&buf)

			err := f.Format(tc.node)
			require.NoError(t, err)
			require.Equal(t, tc.expected, buf.String())
		})
	}
}
