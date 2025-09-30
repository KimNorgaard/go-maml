package maml

import (
	"testing"

	"github.com/KimNorgaard/go-maml/ast"
	"github.com/stretchr/testify/require"
)

func TestIdentifierExpression(t *testing.T) {
	input := `foobar`

	l := NewLexer([]byte(input))
	p := NewParser(l)
	doc := p.ParseDocument()
	require.Empty(t, p.Errors(), "parser has errors")

	require.Len(t, doc.Statements, 1, "doc.Statements does not contain 1 statement")

	stmt, ok := doc.Statements[0].(*ast.ExpressionStatement)
	require.True(t, ok, "doc.Statements[0] is not ast.ExpressionStatement")

	ident, ok := stmt.Expression.(*ast.Identifier)
	require.True(t, ok, "exp not *ast.Identifier")

	require.Equal(t, "foobar", ident.Value, "ident.Value not %s", "foobar")
	require.Equal(t, "foobar", ident.TokenLiteral(), "ident.TokenLiteral not %s", "foobar")
}

func TestIntegerLiteralExpression(t *testing.T) {
	input := `5`

	l := NewLexer([]byte(input))
	p := NewParser(l)
	doc := p.ParseDocument()
	require.Empty(t, p.Errors(), "parser has errors")

	require.Len(t, doc.Statements, 1, "doc.Statements does not contain 1 statement")

	stmt, ok := doc.Statements[0].(*ast.ExpressionStatement)
	require.True(t, ok, "doc.Statements[0] is not ast.ExpressionStatement")

	literal, ok := stmt.Expression.(*ast.IntegerLiteral)
	require.True(t, ok, "exp not *ast.IntegerLiteral")

	require.Equal(t, int64(5), literal.Value, "literal.Value not %d", 5)
	require.Equal(t, "5", literal.TokenLiteral(), "literal.TokenLiteral not %s", "5")
}

func TestBooleanLiteralExpression(t *testing.T) {
	tests := []struct {
		input         string
		expectedValue bool
	}{
		{"true", true},
		{"false", false},
	}

	for _, tt := range tests {
		l := NewLexer([]byte(tt.input))
		p := NewParser(l)
		doc := p.ParseDocument()
		require.Empty(t, p.Errors(), "parser has errors")

		require.Len(t, doc.Statements, 1, "doc.Statements does not contain 1 statement")

		stmt, ok := doc.Statements[0].(*ast.ExpressionStatement)
		require.True(t, ok, "doc.Statements[0] is not ast.ExpressionStatement")

		literal, ok := stmt.Expression.(*ast.BooleanLiteral)
		require.True(t, ok, "exp not *ast.BooleanLiteral")

		require.Equal(t, tt.expectedValue, literal.Value, "literal.Value not %t", tt.expectedValue)
	}
}

func TestStringLiteralExpression(t *testing.T) {
	input := `"hello world"`

	l := NewLexer([]byte(input))
	p := NewParser(l)
	doc := p.ParseDocument()
	require.Empty(t, p.Errors(), "parser has errors")

	require.Len(t, doc.Statements, 1, "doc.Statements does not contain 1 statement")

	stmt, ok := doc.Statements[0].(*ast.ExpressionStatement)
	require.True(t, ok, "doc.Statements[0] is not ast.ExpressionStatement")

	literal, ok := stmt.Expression.(*ast.StringLiteral)
	require.True(t, ok, "exp not *ast.StringLiteral")

	require.Equal(t, "hello world", literal.Value, "literal.Value not %s", "hello world")
}

func TestFloatLiteralExpression(t *testing.T) {
	input := `1.23`

	l := NewLexer([]byte(input))
	p := NewParser(l)
	doc := p.ParseDocument()
	require.Empty(t, p.Errors(), "parser has errors")

	require.Len(t, doc.Statements, 1, "doc.Statements does not contain 1 statement")

	stmt, ok := doc.Statements[0].(*ast.ExpressionStatement)
	require.True(t, ok, "doc.Statements[0] is not ast.ExpressionStatement")

	literal, ok := stmt.Expression.(*ast.FloatLiteral)
	require.True(t, ok, "exp not *ast.FloatLiteral")

	require.Equal(t, float64(1.23), literal.Value, "literal.Value not %f", 1.23)
	require.Equal(t, "1.23", literal.TokenLiteral(), "literal.TokenLiteral not %s", "1.23")
}

func TestNullLiteralExpression(t *testing.T) {
	input := `null`

	l := NewLexer([]byte(input))
	p := NewParser(l)
	doc := p.ParseDocument()
	require.Empty(t, p.Errors(), "parser has errors")

	require.Len(t, doc.Statements, 1, "doc.Statements does not contain 1 statement")

	stmt, ok := doc.Statements[0].(*ast.ExpressionStatement)
	require.True(t, ok, "doc.Statements[0] is not ast.ExpressionStatement")

	_, ok = stmt.Expression.(*ast.NullLiteral)
	require.True(t, ok, "exp not *ast.NullLiteral")
}
