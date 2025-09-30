package maml

import (
	"testing"

	"github.com/KimNorgaard/go-maml/ast"
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
		t.Run(tt.input, func(t *testing.T) {
			l := NewLexer([]byte(tt.input))
			p := NewParser(l)
			doc := p.ParseDocument()
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

	l := NewLexer([]byte(input))
	p := NewParser(l)
	doc := p.ParseDocument()
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

	l := NewLexer([]byte(input))
	p := NewParser(l)
	doc := p.ParseDocument()
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
