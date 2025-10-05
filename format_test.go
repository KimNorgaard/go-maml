package maml

import (
	"bytes"
	"testing"

	"github.com/KimNorgaard/go-maml/internal/ast"
	"github.com/KimNorgaard/go-maml/internal/token"
	"github.com/stretchr/testify/require"
)

func TestFormatter(t *testing.T) {
	// Define a comprehensive sample AST structure to test against
	sampleAST := &ast.Document{
		Statements: []ast.Statement{
			&ast.ExpressionStatement{
				Expression: &ast.ObjectLiteral{
					Token: token.Token{Type: token.LBRACE, Literal: "{"},
					Pairs: []*ast.KeyValueExpression{
						{
							Key:   &ast.Identifier{Token: token.Token{Type: token.IDENT, Literal: "stringField"}, Value: "stringField"},
							Value: &ast.StringLiteral{Token: token.Token{Type: token.STRING, Literal: `"hello world"`}, Value: "hello world"},
						},
						{
							Key: &ast.Identifier{Token: token.Token{Type: token.IDENT, Literal: "multilineString"}, Value: "multilineString"},
							// Raw value with actual newlines
							Value: &ast.StringLiteral{Token: token.Token{Type: token.STRING, Literal: `line one\nline two`}, Value: "line one\nline two"},
						},
						{
							Key:   &ast.Identifier{Token: token.Token{Type: token.IDENT, Literal: "floatField"}, Value: "floatField"},
							Value: &ast.FloatLiteral{Token: token.Token{Type: token.FLOAT, Literal: "3.14"}, Value: 3.14},
						},
						{
							Key: &ast.Identifier{Token: token.Token{Type: token.IDENT, Literal: "nestedObject"}, Value: "nestedObject"},
							Value: &ast.ObjectLiteral{
								Token: token.Token{Type: token.LBRACE, Literal: "{"},
								Pairs: []*ast.KeyValueExpression{
									{
										Key:   &ast.Identifier{Token: token.Token{Type: token.IDENT, Literal: "a"}, Value: "a"},
										Value: &ast.IntegerLiteral{Token: token.Token{Type: token.INT, Literal: "1"}, Value: 1},
									},
									{
										Key:   &ast.Identifier{Token: token.Token{Type: token.IDENT, Literal: "b"}, Value: "b"},
										Value: &ast.BooleanLiteral{Token: token.Token{Type: token.TRUE, Literal: "true"}, Value: true},
									},
								},
							},
						},
						{
							Key: &ast.Identifier{Token: token.Token{Type: token.IDENT, Literal: "arrayField"}, Value: "arrayField"},
							Value: &ast.ArrayLiteral{
								Token: token.Token{Type: token.LBRACK, Literal: "["},
								Elements: []ast.Expression{
									&ast.IntegerLiteral{Token: token.Token{Type: token.INT, Literal: "10"}, Value: 10},
									&ast.StringLiteral{Token: token.Token{Type: token.STRING, Literal: `"foo"`}, Value: "foo"},
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
			expected: `{stringField:"hello world",multilineString:"line one\nline two",floatField:3.14,nestedObject:{a:1,b:true},arrayField:[10,"foo",null]}`,
		},
		{
			name: "Default Indent (2 spaces) - No SeparateFields (default)",
			node: sampleAST,
			opts: []Option{Indent(2)},
			expected: `{
  stringField: "hello world"
  multilineString: """
line one
line two"""
  floatField: 3.14
  nestedObject: {
    a: 1
    b: true
  }
  arrayField: [
    10
    "foo"
    null
  ]
}`,
		},
		{
			name: "Default Indent (2 spaces) - With SeparateFields",
			node: sampleAST,
			opts: []Option{Indent(2), UseFieldCommas()},
			expected: `{
  stringField: "hello world",
  multilineString: """
line one
line two""",
  floatField: 3.14,
  nestedObject: {
    a: 1,
    b: true
  },
  arrayField: [
    10,
    "foo",
    null
  ]
}`,
		},
		{
			name: "Custom Indent (4 spaces) - With SeparateFields",
			node: sampleAST,
			opts: []Option{Indent(4), UseFieldCommas()},
			expected: `{
    stringField: "hello world",
    multilineString: """
line one
line two""",
    floatField: 3.14,
    nestedObject: {
        a: 1,
        b: true
    },
    arrayField: [
        10,
        "foo",
        null
    ]
}`,
		},
		{
			name:     "Empty Object",
			node:     &ast.ObjectLiteral{},
			opts:     []Option{Indent(2)},
			expected: `{}`,
		},
		{
			name:     "Empty Array",
			node:     &ast.ArrayLiteral{},
			opts:     []Option{Indent(2)},
			expected: `[]`,
		},
		{
			name: "String with internal quotes and escapes (should always be standard quoted)",
			node: &ast.StringLiteral{Value: `a "quote" and \n newline`},
			opts: []Option{},
			// Even if multiline strings are preferred, this cannot be triple-quoted due to explicit quotes.
			expected: "\"a \\\"quote\\\" and \\\\n newline\"",
		},
		{
			name:     "InlineArrays enabled (no separate fields)",
			node:     sampleAST.Statements[0].(*ast.ExpressionStatement).Expression.(*ast.ObjectLiteral).Pairs[4].Value, // arrayField
			opts:     []Option{InlineArrays()},
			expected: `[10,"foo",null]`, // Inline arrays always use commas regardless of SeparateFields
		},
		{
			name:     "InlineArrays enabled (with separate fields)",
			node:     sampleAST.Statements[0].(*ast.ExpressionStatement).Expression.(*ast.ObjectLiteral).Pairs[4].Value, // arrayField
			opts:     []Option{InlineArrays(), UseFieldCommas()},
			expected: `[10,"foo",null]`, // Inline arrays always use commas regardless of SeparateFields
		},
		{
			name: "Multiline string default (InlineStrings: false)",
			node: &ast.StringLiteral{Value: "first line\nsecond line\nthird line"},
			opts: []Option{},
			expected: `"""
first line
second line
third line"""`,
		},
		{
			name: "Multiline string with leading newline (InlineStrings: false)",
			node: &ast.StringLiteral{Value: "\nfirst line\nsecond line"},
			opts: []Option{},
			expected: `"""

first line
second line"""`,
		},
		{
			name:     "Compact Mode - InlineStrings explicitly true",
			node:     &ast.StringLiteral{Value: "line1\nline2"},
			opts:     []Option{Indent(0), InlineStrings()},
			expected: `"line1\nline2"`, // Newlines escaped in standard string
		},
		{
			name:     "Multiline string forced inline (InlineStrings: true)",
			node:     &ast.StringLiteral{Value: "line1\nline2"},
			opts:     []Option{InlineStrings()},
			expected: "\"line1\\nline2\"",
		},
		{
			name:     "String with triple quotes (InlineStrings: false, should fallback to standard)",
			node:     &ast.StringLiteral{Value: `This string has """ triple quotes`},
			opts:     []Option{InlineStrings()},
			expected: "\"This string has \\\"\\\"\\\" triple quotes\"",
		},
		{
			name:     "String with triple quotes (InlineStrings: true, should be standard anyway)",
			node:     &ast.StringLiteral{Value: `This string has """ triple quotes`},
			opts:     []Option{InlineStrings()},
			expected: "\"This string has \\\"\\\"\\\" triple quotes\"",
		},
		{
			name:     "Single line string with newlines forced inline (InlineStrings: true)",
			node:     &ast.StringLiteral{Value: "single line\nwith newline"},
			opts:     []Option{InlineStrings()},
			expected: "\"single line\\nwith newline\"",
		},
		{
			name: "Default Indent (2 spaces) - With UseFieldCommas and UseTrailingCommas (Object)",
			node: sampleAST,
			opts: []Option{Indent(2), UseFieldCommas(), UseTrailingCommas()},
			expected: `{
  stringField: "hello world",
  multilineString: """
line one
line two""",
  floatField: 3.14,
  nestedObject: {
    a: 1,
    b: true,
  },
  arrayField: [
    10,
    "foo",
    null,
  ],
}`,
		},
		{
			name: "Default Indent (2 spaces) - With UseFieldCommas and UseTrailingCommas (Array)",
			node: sampleAST.Statements[0].(*ast.ExpressionStatement).Expression.(*ast.ObjectLiteral).Pairs[4].Value, // arrayField
			opts: []Option{Indent(2), UseFieldCommas(), UseTrailingCommas()},
			expected: `[
  10,
  "foo",
  null,
]`,
		},
		{
			name: "Default Indent (2 spaces) - UseTrailingCommas only (Object - no field commas)",
			node: sampleAST,
			opts: []Option{Indent(2), UseTrailingCommas()},
			expected: `{
  stringField: "hello world"
  multilineString: """
line one
line two"""
  floatField: 3.14
  nestedObject: {
    a: 1
    b: true
  }
  arrayField: [
    10
    "foo"
    null
  ]
}`,
		},
		{
			name: "Default Indent (2 spaces) - UseTrailingCommas only (Array - no field commas)",
			node: sampleAST.Statements[0].(*ast.ExpressionStatement).Expression.(*ast.ObjectLiteral).Pairs[4].Value, // arrayField
			opts: []Option{Indent(2), UseTrailingCommas()},
			expected: `[
  10
  "foo"
  null
]`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			o := options{}
			for _, opt := range tc.opts {
				err := opt(&o)
				require.NoError(t, err)
			}

			f := newFormatter(&buf, &o)
			err := f.format(tc.node)

			require.NoError(t, err)
			require.Equal(t, tc.expected, buf.String())
		})
	}
}
