package formatter_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/KimNorgaard/go-maml/internal/ast"
	"github.com/KimNorgaard/go-maml/internal/formatter"
	"github.com/KimNorgaard/go-maml/internal/token"
	"github.com/stretchr/testify/require"
)

// Centralized test cases to be used across different format settings.
var testCases = []struct {
	name             string
	node             ast.Node
	expectedCompact  string
	expectedIndented string // 2 spaces, may reflect buggy output
}{
	{
		name:             "String Literal",
		node:             &ast.StringLiteral{Value: "hello world"},
		expectedCompact:  `"hello world"`,
		expectedIndented: `"hello world"`,
	},
	{
		name:             "Integer Literal",
		node:             &ast.IntegerLiteral{Token: token.Token{Literal: "123"}},
		expectedCompact:  "123",
		expectedIndented: "123",
	},
	{
		name:             "Empty Array",
		node:             &ast.ArrayLiteral{Elements: []ast.Expression{}},
		expectedCompact:  "[]",
		expectedIndented: "[]",
	},
	{
		name: "Array with scalars",
		node: &ast.ArrayLiteral{
			Elements: []ast.Expression{
				&ast.IntegerLiteral{Token: token.Token{Literal: "1"}},
				&ast.StringLiteral{Value: "two"},
			},
		},
		expectedCompact:  `[1, "two"]`,
		expectedIndented: "[\n  1,\n  \"two\"\n]",
	},
	{
		name:             "Empty Object",
		node:             &ast.ObjectLiteral{Pairs: []*ast.KeyValueExpression{}},
		expectedCompact:  "{}",
		expectedIndented: "{}",
	},
	{
		name: "Object with pairs",
		node: &ast.ObjectLiteral{
			Pairs: []*ast.KeyValueExpression{
				{Key: &ast.Identifier{Value: "key1"}, Value: &ast.StringLiteral{Value: "value1"}},
				{Key: &ast.Identifier{Value: "key2"}, Value: &ast.IntegerLiteral{Token: token.Token{Literal: "123"}}},
			},
		},
		expectedCompact:  `{ key1: "value1", key2: 123 }`,
		expectedIndented: "{\n  key1: \"value1\",\n  key2: 123\n}",
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
									{Key: &ast.Identifier{Value: "id"}, Value: &ast.IntegerLiteral{Token: token.Token{Literal: "1"}}},
									{Key: &ast.Identifier{Value: "status"}, Value: &ast.StringLiteral{Value: "ok"}},
								},
							},
							&ast.IntegerLiteral{Token: token.Token{Literal: "2"}},
						},
					},
				},
			},
		},
		expectedCompact:  `{ data: [{ id: 1, status: "ok" }, 2] }`,
		expectedIndented: "{\n  data: [\n    {\n      id: 1,\n      status: \"ok\"\n    },\n    2\n  ]\n}",
	},
}

func TestFormatter_Indentation(t *testing.T) {
	t.Run("Default Indent (2 spaces)", func(t *testing.T) {
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				var buf bytes.Buffer
				f := formatter.New(&buf, nil)
				err := f.Format(tc.node)
				require.NoError(t, err)
				require.Equal(t, tc.expectedIndented, buf.String())
			})
		}
	})

	t.Run("Compact Output (indent 0)", func(t *testing.T) {
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				var buf bytes.Buffer
				zero := 0
				f := formatter.New(&buf, &zero)
				err := f.Format(tc.node)
				require.NoError(t, err)
				require.Equal(t, tc.expectedCompact, buf.String())
			})
		}
	})

	t.Run("Custom Indent (4 spaces)", func(t *testing.T) {
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				var buf bytes.Buffer
				four := 4
				f := formatter.New(&buf, &four)
				expected := strings.ReplaceAll(tc.expectedIndented, "  ", "    ")
				err := f.Format(tc.node)
				require.NoError(t, err)
				require.Equal(t, expected, buf.String())
			})
		}
	})
}
