package maml

import (
	"bytes"
	"errors"
	"testing"

	"github.com/KimNorgaard/go-maml/internal/ast"
	"github.com/KimNorgaard/go-maml/internal/token"
	"github.com/stretchr/testify/require"
)

// errorWriter is a helper that implements io.Writer but always returns an error.
type errorWriter struct{}

func (ew *errorWriter) Write(p []byte) (n int, err error) {
	return 0, errors.New("write error")
}

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
			name:     "String with internal quotes and escapes (should always be standard quoted)",
			node:     &ast.StringLiteral{Value: `a "quote" and \n newline`},
			opts:     []Option{},
			expected: `"a \"quote\" and \\n newline"`,
		},
		{
			name:     "InlineArrays enabled (no separate fields)",
			node:     sampleAST.Statements[0].(*ast.ExpressionStatement).Expression.(*ast.ObjectLiteral).Pairs[4].Value, // arrayField
			opts:     []Option{InlineArrays()},
			expected: `[10,"foo",null]`,
		},
		{
			name:     "InlineArrays enabled (with separate fields)",
			node:     sampleAST.Statements[0].(*ast.ExpressionStatement).Expression.(*ast.ObjectLiteral).Pairs[4].Value, // arrayField
			opts:     []Option{InlineArrays(), UseFieldCommas()},
			expected: `[10,"foo",null]`,
		},
		{
			name: "Multiline string default (InlineStrings: false)",
			node: &ast.StringLiteral{Value: "first line\nsecond line\nthird line"},
			opts: []Option{Indent(2)},
			expected: `"""
first line
second line
third line"""`,
		},
		{
			name: "Multiline string with leading newline (InlineStrings: false)",
			node: &ast.StringLiteral{Value: "\nfirst line\nsecond line"},
			opts: []Option{Indent(2)},
			expected: `"""

first line
second line"""`,
		},
		{
			name:     "Compact Mode - InlineStrings explicitly true",
			node:     &ast.StringLiteral{Value: "line1\nline2"},
			opts:     []Option{Indent(0), InlineStrings()},
			expected: `"line1\nline2"`,
		},
		{
			name:     "Multiline string forced inline (InlineStrings: true)",
			node:     &ast.StringLiteral{Value: "line1\nline2"},
			opts:     []Option{Indent(2), InlineStrings()},
			expected: `"line1\nline2"`,
		},
		{
			name:     "String with triple quotes (InlineStrings: false, should fallback to standard)",
			node:     &ast.StringLiteral{Value: `This string has """ triple quotes`},
			opts:     []Option{Indent(2)},
			expected: `"This string has \"\"\" triple quotes"`,
		},
		{
			name:     "String with triple quotes (InlineStrings: true, should be standard anyway)",
			node:     &ast.StringLiteral{Value: `This string has """ triple quotes`},
			opts:     []Option{Indent(2), InlineStrings()},
			expected: `"This string has \"\"\" triple quotes"`,
		},
		{
			name:     "Single line string with newlines forced inline (InlineStrings: true)",
			node:     &ast.StringLiteral{Value: "single line\nwith newline"},
			opts:     []Option{Indent(2), InlineStrings()},
			expected: `"single line\nwith newline"`,
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
		{
			name: "Object with all comment types",
			node: &ast.ObjectLiteral{
				Token: token.Token{Type: token.LBRACE, Literal: "{"},
				Pairs: []*ast.KeyValueExpression{
					{
						HeadComments: []*ast.Comment{
							{Value: "Head comment line 1"},
							{Value: "Head comment line 2"},
						},
						Key:         &ast.Identifier{Token: token.Token{Type: token.IDENT, Literal: "key"}, Value: "key"},
						Value:       &ast.StringLiteral{Token: token.Token{Type: token.STRING, Literal: `"value"`}, Value: "value"},
						LineComment: &ast.Comment{Value: "Line comment"},
						FootComments: []*ast.Comment{
							{Value: "Foot comment"},
						},
					},
				},
			},
			opts: []Option{Indent(2), UseFieldCommas()},
			expected: `{
  # Head comment line 1
  # Head comment line 2
  key: "value" # Line comment
  # Foot comment
}`,
		},
		{
			name: "Object with multiple pairs and comments",
			node: &ast.ObjectLiteral{
				Token: token.Token{Type: token.LBRACE, Literal: "{"},
				Pairs: []*ast.KeyValueExpression{
					{
						HeadComments: []*ast.Comment{
							{Value: "Head for key1"},
						},
						Key:         &ast.Identifier{Token: token.Token{Type: token.IDENT, Literal: "key1"}, Value: "key1"},
						Value:       &ast.IntegerLiteral{Token: token.Token{Type: token.INT, Literal: "1"}, Value: 1},
						LineComment: &ast.Comment{Value: "Line for key1"},
					},
					{
						HeadComments: []*ast.Comment{
							{Value: "Head for key2"},
						},
						Key:   &ast.Identifier{Token: token.Token{Type: token.IDENT, Literal: "key2"}, Value: "key2"},
						Value: &ast.IntegerLiteral{Token: token.Token{Type: token.INT, Literal: "2"}, Value: 2},
						FootComments: []*ast.Comment{
							{Value: "Foot for key2"},
						},
					},
				},
			},
			opts: []Option{Indent(2), UseFieldCommas()},
			expected: `{
  # Head for key1
  key1: 1, # Line for key1
  # Head for key2
  key2: 2
  # Foot for key2
}`,
		},
		{
			name: "Complex types with line comments",
			node: &ast.ObjectLiteral{
				Token: token.Token{Type: token.LBRACE, Literal: "{"},
				Pairs: []*ast.KeyValueExpression{
					{
						Key: &ast.Identifier{Token: token.Token{Type: token.IDENT, Literal: "innerObject"}, Value: "innerObject"},
						Value: &ast.ObjectLiteral{
							Token: token.Token{Type: token.LBRACE, Literal: "{"},
							Pairs: []*ast.KeyValueExpression{
								{
									Key:   &ast.Identifier{Token: token.Token{Type: token.IDENT, Literal: "a"}, Value: "a"},
									Value: &ast.IntegerLiteral{Token: token.Token{Type: token.INT, Literal: "1"}, Value: 1},
								},
							},
						},
						LineComment: &ast.Comment{Value: "comment on object"},
					},
					{
						Key: &ast.Identifier{Token: token.Token{Type: token.IDENT, Literal: "innerArray"}, Value: "innerArray"},
						Value: &ast.ArrayLiteral{
							Token: token.Token{Type: token.LBRACK, Literal: "["},
							Elements: []ast.Expression{
								&ast.IntegerLiteral{Token: token.Token{Type: token.INT, Literal: "1"}, Value: 1},
							},
						},
						LineComment: &ast.Comment{Value: "comment on array"},
					},
				},
			},
			opts: []Option{Indent(2), UseFieldCommas()},
			expected: `{
  innerObject: {
    a: 1
  }, # comment on object
  innerArray: [
    1
  ] # comment on array
}`,
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

func TestFormatter_WriteErrors(t *testing.T) {
	// This test ensures that error handling for io.Writer operations is working.
	// We use a custom writer that always fails.
	doc := &ast.Document{
		Statements: []ast.Statement{
			&ast.ExpressionStatement{
				Expression: &ast.ObjectLiteral{
					Pairs: []*ast.KeyValueExpression{
						{
							Key:   &ast.Identifier{Value: "key"},
							Value: &ast.StringLiteral{Value: "value"},
						},
					},
				},
			},
		},
	}

	// Use the errorWriter to ensure that all write operations are tested for error handling.
	f := newFormatter(&errorWriter{}, &options{})
	err := f.format(doc)
	require.Error(t, err)
	require.Contains(t, err.Error(), "write error")
}
