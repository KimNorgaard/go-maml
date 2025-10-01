package parser_test

import (
	"testing"

	"github.com/KimNorgaard/go-maml/ast"
	"github.com/KimNorgaard/go-maml/lexer"
	"github.com/KimNorgaard/go-maml/parser"
	"github.com/stretchr/testify/require"
)

func TestLiteralExpressions(t *testing.T) {
	tests := []struct {
		input    string
		expected any
	}{
		{"5", int64(5)},
		{"true", true},
		{"false", false},
		{"foobar", "foobar"},
		{"1.23", float64(1.23)},
		{"\"hello world\"", "hello world"},
		{"null", nil},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable
		t.Run(tt.input, func(t *testing.T) {
			l := lexer.New([]byte(tt.input))
			p := parser.New(l)
			doc := p.Parse()
			require.Empty(t, p.Errors(), "parser has errors")
			require.Len(t, doc.Statements, 1, "doc.Statements does not contain 1 statement")

			stmt, ok := doc.Statements[0].(*ast.ExpressionStatement)
			require.True(t, ok, "doc.Statements[0] is not ast.ExpressionStatement")

			testLiteralExpression(t, stmt.Expression, tt.expected)
		})
	}
}

func testLiteralExpression(t *testing.T, exp ast.Expression, expected any) {
	t.Helper()

	switch v := expected.(type) {
	case int64:
		lit, ok := exp.(*ast.IntegerLiteral)
		require.True(t, ok, "exp not *ast.IntegerLiteral, got=%T", exp)
		require.Equal(t, v, lit.Value)
	case bool:
		lit, ok := exp.(*ast.BooleanLiteral)
		require.True(t, ok, "exp not *ast.BooleanLiteral, got=%T", exp)
		require.Equal(t, v, lit.Value)
	case string:
		// Could be Identifier or StringLiteral
		if ident, ok := exp.(*ast.Identifier); ok {
			require.Equal(t, v, ident.Value)
		} else if str, ok := exp.(*ast.StringLiteral); ok {
			require.Equal(t, v, str.Value)
		} else {
			t.Fatalf("exp not *ast.Identifier or *ast.StringLiteral, got=%T", exp)
		}
	case float64:
		lit, ok := exp.(*ast.FloatLiteral)
		require.True(t, ok, "exp not *ast.FloatLiteral, got=%T", exp)
		require.Equal(t, v, lit.Value)
	case nil:
		_, ok := exp.(*ast.NullLiteral)
		require.True(t, ok, "exp not *ast.NullLiteral, got=%T", exp)
	default:
		t.Fatalf("type of expected not handled: %T", expected)
	}
}

func TestArrayLiteralParsing(t *testing.T) {
	input := `[1, "two", true]`

	l := lexer.New([]byte(input))
	p := parser.New(l)
	doc := p.Parse()
	require.Empty(t, p.Errors(), "parser has errors")

	require.Len(t, doc.Statements, 1, "doc.Statements does not contain 1 statement")

	stmt, ok := doc.Statements[0].(*ast.ExpressionStatement)
	require.True(t, ok, "doc.Statements[0] is not ast.ExpressionStatement")

	array, ok := stmt.Expression.(*ast.ArrayLiteral)
	require.True(t, ok, "exp not *ast.ArrayLiteral")

	require.Len(t, array.Elements, 3, "len(array.Elements) not 3")

	// Test elements inside the array
	testLiteralExpression(t, array.Elements[0], int64(1))
	testLiteralExpression(t, array.Elements[1], "two")
	testLiteralExpression(t, array.Elements[2], true)
}

func TestObjectLiteralParsing(t *testing.T) {
	input := "{\n\t\"one\": 1,\n\ttwo: \"two\",\n\t\"three\": true\n}"

	l := lexer.New([]byte(input))
	p := parser.New(l)
	doc := p.Parse()
	require.Empty(t, p.Errors(), "parser has errors")

	stmt, ok := doc.Statements[0].(*ast.ExpressionStatement)
	require.True(t, ok, "doc.Statements[0] is not ast.ExpressionStatement")

	obj, ok := stmt.Expression.(*ast.ObjectLiteral)
	require.True(t, ok, "exp not *ast.ObjectLiteral")

	require.Len(t, obj.Pairs, 3, "obj.Pairs has wrong number of pairs")

	// Check pair 1
	testLiteralExpression(t, obj.Pairs[0].Key, "one")
	testLiteralExpression(t, obj.Pairs[0].Value, int64(1))

	// Check pair 2
	testLiteralExpression(t, obj.Pairs[1].Key, "two")
	testLiteralExpression(t, obj.Pairs[1].Value, "two")

	// Check pair 3
	testLiteralExpression(t, obj.Pairs[2].Key, "three")
	testLiteralExpression(t, obj.Pairs[2].Value, true)
}

func TestCommentWithControlCharacter(t *testing.T) {
	input := "# a comment with a \x01 control character\n{ key: \"value\" }"
	l := lexer.New([]byte(input))
	p := parser.New(l)
	p.Parse()
	require.NotEmpty(t, p.Errors(), "parser should have errors for control characters in comments")
}

func TestArrayWithOptionalCommas(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "No commas, just newlines",
			input: "[\n\"one\"\n\"two\"\n]",
		},
		{
			name:  "Mixed commas and newlines",
			input: "[\"one\",\n\"two\"\n]",
		},
		{
			name:  "Trailing comma with newline",
			input: "[\"one\", \"two\",\n]",
		},
		{
			name:  "Trailing comma on same line",
			input: "[\"one\", \"two\",]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New([]byte(tt.input))
			p := parser.New(l)
			doc := p.Parse()

			require.Empty(t, p.Errors(), "parser has errors")
			require.NotNil(t, doc)
			require.Len(t, doc.Statements, 1)

			stmt, ok := doc.Statements[0].(*ast.ExpressionStatement)
			require.True(t, ok)

			arr, ok := stmt.Expression.(*ast.ArrayLiteral)
			require.True(t, ok)
			require.Len(t, arr.Elements, 2)
			testLiteralExpression(t, arr.Elements[0], "one")
			testLiteralExpression(t, arr.Elements[1], "two")
		})
	}
}

func TestObjectKeyTypes(t *testing.T) {
	t.Run("Unquoted and quoted keys", func(t *testing.T) {
		tests := []struct {
			name        string
			input       string
			expectedKey string
		}{
			{
				name:        "Unquoted key",
				input:       "{ my-key_123: 1 }",
				expectedKey: "my-key_123",
			},
			{
				name:        "Quoted key with spaces and symbols",
				input:       `{ "key with spaces!": 1 }`,
				expectedKey: "key with spaces!",
			},
			{
				name:        "Empty quoted key",
				input:       `{ "": 1 }`,
				expectedKey: "",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				l := lexer.New([]byte(tt.input))
				p := parser.New(l)
				doc := p.Parse()

				require.Empty(t, p.Errors(), "parser has errors")
				require.NotNil(t, doc)
				require.Len(t, doc.Statements, 1)

				stmt, ok := doc.Statements[0].(*ast.ExpressionStatement)
				require.True(t, ok)

				obj, ok := stmt.Expression.(*ast.ObjectLiteral)
				require.True(t, ok)
				require.Len(t, obj.Pairs, 1)

				testLiteralExpression(t, obj.Pairs[0].Key, tt.expectedKey)
				testLiteralExpression(t, obj.Pairs[0].Value, int64(1))
			})
		}
	})

	t.Run("Unquoted key with only digits", func(t *testing.T) {
		input := `{ 123: 1 }`
		l := lexer.New([]byte(input))
		p := parser.New(l)
		doc := p.Parse()

		require.Empty(t, p.Errors(), "parser has errors")
		stmt, ok := doc.Statements[0].(*ast.ExpressionStatement)
		require.True(t, ok)
		obj, ok := stmt.Expression.(*ast.ObjectLiteral)
		require.True(t, ok)
		require.Len(t, obj.Pairs, 1)

		key, ok := obj.Pairs[0].Key.(*ast.Identifier)
		require.True(t, ok, "key should be an Identifier, got=%T", obj.Pairs[0].Key)
		require.Equal(t, "123", key.Value)
	})
}

func TestStringLiterals(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Unicode characters",
			input:    `"hello 👋"`,
			expected: "hello 👋",
		},
		{
			name:     "Escaped quote",
			input:    `"hello \"world\""`,
			expected: `hello "world"`,
		},
		{
			name:     "Escaped backslash",
			input:    `"c:\\path\\"`,
			expected: `c:\path\`,
		},
		{
			name:     "Escaped tab",
			input:    `"hello\tworld"`,
			expected: "hello\tworld",
		},
		{
			name:     "Escaped backspace",
			input:    `"hello\bworld"`,
			expected: "hello\bworld",
		},
		{
			name:     "Escaped formfeed",
			input:    `"hello\fworld"`,
			expected: "hello\fworld",
		},
		{
			name:     "Escaped newline",
			input:    `"hello\nworld"`,
			expected: "hello\nworld",
		},
		{
			name:     "Escaped carriage return",
			input:    `"hello\rworld"`,
			expected: "hello\rworld",
		},
		{
			name:     "Unicode escape sequence",
			input:    `"a \u0022 quote"`,
			expected: `a " quote`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New([]byte(tt.input))
			p := parser.New(l)
			doc := p.Parse()

			require.Empty(t, p.Errors(), "parser has errors for %s", tt.name)
			require.NotNil(t, doc)
			require.Len(t, doc.Statements, 1)

			stmt, ok := doc.Statements[0].(*ast.ExpressionStatement)
			require.True(t, ok)

			testLiteralExpression(t, stmt.Expression, tt.expected)
		})
	}
}

func TestObjectParsingScenarios(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedPairs int
		shouldError   bool
	}{
		{
			name:          "Trailing comma",
			input:         `{ "a": 1, }`,
			expectedPairs: 1,
			shouldError:   false,
		},
		{
			name:          "Trailing comma with newline",
			input:         "{ \"a\": 1,\n }",
			expectedPairs: 1,
			shouldError:   false,
		},
		{
			name:          "No comma, just newline",
			input:         "{\n\"a\": 1\n\"b\": 2\n}",
			expectedPairs: 2,
			shouldError:   false,
		},
		{
			name:          "Mixed commas and newlines",
			input:         "{ \"a\": 1,\n\"b\": 2 }",
			expectedPairs: 2,
			shouldError:   false,
		},
		{
			name:          "Empty object",
			input:         `{}`,
			expectedPairs: 0,
			shouldError:   false,
		},
		{
			name:          "Object with all value types",
			input:         `{ a: 1, b: "s", c: true, d: 1.2, e: null, f: [1], g: {h:1} }`,
			expectedPairs: 7,
			shouldError:   false,
		},
		{
			name:          "Duplicate key",
			input:         `{ a: 1, a: 2 }`,
			expectedPairs: 0,
			shouldError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New([]byte(tt.input))
			p := parser.New(l)
			doc := p.Parse()

			if tt.shouldError {
				require.NotEmpty(t, p.Errors(), "expected parser errors for %s", tt.name)
				return
			}

			require.Empty(t, p.Errors(), "parser has errors for %s", tt.name)
			require.NotNil(t, doc)
			require.Len(t, doc.Statements, 1)

			stmt, ok := doc.Statements[0].(*ast.ExpressionStatement)
			require.True(t, ok)

			obj, ok := stmt.Expression.(*ast.ObjectLiteral)
			require.True(t, ok)

			require.Len(t, obj.Pairs, tt.expectedPairs)
		})
	}
}

func TestIntegerOverflow(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldError bool
	}{
		{
			name:        "int64 max",
			input:       "9223372036854775807",
			shouldError: false,
		},
		{
			name:        "int64 overflow",
			input:       "9223372036854775808",
			shouldError: true,
		},
		{
			name:        "int64 min",
			input:       "-9223372036854775808",
			shouldError: false,
		},
		{
			name:        "int64 underflow",
			input:       "-9223372036854775809",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New([]byte(tt.input))
			p := parser.New(l)
			p.Parse()

			if tt.shouldError {
				require.NotEmpty(t, p.Errors(), "expected parser errors for %s", tt.name)
			} else {
				require.Empty(t, p.Errors(), "parser has errors for %s", tt.name)
			}
		})
	}
}

func TestFloatLiterals(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected float64
	}{
		{"fractional", "1.0", 1.0},
		{"pi", "3.1415", 3.1415},
		{"negative", "-0.01", -0.01},
		{"exponent", "5e+22", 5e+22},
		{"exponent uppercase", "1E06", 1e06},
		{"negative exponent", "-2E-2", -2e-2},
		{"both", "6.626e-34", 6.626e-34},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New([]byte(tt.input))
			p := parser.New(l)
			doc := p.Parse()

			require.Empty(t, p.Errors(), "parser has errors for %s", tt.name)
			stmt, ok := doc.Statements[0].(*ast.ExpressionStatement)
			require.True(t, ok)
			testLiteralExpression(t, stmt.Expression, tt.expected)
		})
	}
}

func TestMultilineStringLiterals(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Basic multiline string",
			input:    `"""hello"""`,
			expected: "hello",
		},
		{
			name:     "Leading newline ignored",
			input:    "\"\"\"\nhello\nworld\"\"\"",
			expected: "hello\nworld",
		},
		{
			name:     "Empty multiline string",
			input:    `""""""`,
			expected: "",
		},
		{
			name:     "Single newline",
			input:    "\"\"\"\n\n\"\"\"",
			expected: "\n",
		},
		{
			name:     "Preserves whitespace",
			input:    "\"\"\"\n  line1\n  line2\n\"\"\"",
			expected: "  line1\n  line2\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New([]byte(tt.input))
			p := parser.New(l)
			doc := p.Parse()

			require.Empty(t, p.Errors(), "parser has errors for %s", tt.name)
			require.NotNil(t, doc)
			require.Len(t, doc.Statements, 1)

			stmt, ok := doc.Statements[0].(*ast.ExpressionStatement)
			require.True(t, ok)

			testLiteralExpression(t, stmt.Expression, tt.expected)
		})
	}
}

func TestInvalidIntegerFormats(t *testing.T) {
	tests := []string{
		"01",
		"+100",
		"+05",
	}
	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			l := lexer.New([]byte(tt))
			p := parser.New(l)
			p.Parse()
			require.NotEmpty(t, p.Errors(), "expected parser errors for %s", tt)
		})
	}
}

func TestInvalidFloatFormats(t *testing.T) {
	tests := []string{
		".1",
		"1.",
		"-.1",
	}
	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			l := lexer.New([]byte(tt))
			p := parser.New(l)
			p.Parse()
			require.NotEmpty(t, p.Errors(), "expected parser errors for %s", tt)
		})
	}
}

func TestStringReservedEscapeSequences(t *testing.T) {
	tests := []string{
		`"\x"`,
		`"\q"`,
	}
	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			l := lexer.New([]byte(tt))
			p := parser.New(l)
			p.Parse()
			require.NotEmpty(t, p.Errors(), "expected parser errors for %s", tt)
		})
	}
}

func TestMultilineStringInvalidQuotes(t *testing.T) {
	input := `"""A string with """ three quotes"""`
	l := lexer.New([]byte(input))
	p := parser.New(l)
	p.Parse()
	require.NotEmpty(t, p.Errors(), "expected parser errors for invalid quotes in multiline string")
}

func TestInvalidUTF8(t *testing.T) {
	input := []byte("{\"\xff\": 1}") // Invalid UTF-8 sequence
	l := lexer.New(input)
	p := parser.New(l)
	p.Parse()
	require.NotEmpty(t, p.Errors(), "expected parser errors for invalid UTF-8")
}

func TestStringWithControlCharacter(t *testing.T) {
	input := `"a string with a \x01 control character"`
	l := lexer.New([]byte(input))
	p := parser.New(l)
	p.Parse()
	require.NotEmpty(t, p.Errors(), "parser should have errors for control characters in strings")
}

func TestEmptyAndWhitespaceInput(t *testing.T) {
	tests := []string{
		"",
		"   ",
		"\n\t \n",
		"# a comment\n#another comment\n",
	}
	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			l := lexer.New([]byte(tt))
			p := parser.New(l)
			doc := p.Parse()
			require.Empty(t, p.Errors(), "parser should have no errors on empty/whitespace input")
			require.NotNil(t, doc)
			require.Len(t, doc.Statements, 0, "document should have zero statements")
		})
	}
}

func TestSyntaxErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "Mismatched array delimiter",
			input: "[1, 2)",
		},
		{
			name:  "Mismatched object delimiter",
			input: `{ "key": "value" ]`,
		},
		{
			name:  "Missing colon in object",
			input: `{ "key" 1 }`,
		},
		{
			name:  "Missing value in object",
			input: `{ "key": , }`,
		},
		// {
		// 	name:  "Extra comma in array",
		// 	input: "[1, , 2]",
		// },
		{
			name:  "Unexpected token after expression",
			input: `[1, 2] "hello"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New([]byte(tt.input))
			p := parser.New(l)
			p.Parse()
			require.NotEmpty(t, p.Errors(), "expected parser errors for input: %s", tt.input)
		})
	}
}
