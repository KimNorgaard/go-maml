package ast

import (
	"bytes"
	"strconv"
	"strings"

	"github.com/KimNorgaard/go-maml/token"
)

// Node is the base interface for all AST nodes.
type Node interface {
	// TokenLiteral returns the literal value of the token associated with the node.
	TokenLiteral() string
	// String returns a string representation of the node.
	String() string
}

// Statement is a node that represents a statement.
type Statement interface {
	Node
	statementNode()
}

// Expression is a node that represents an expression.
type Expression interface {
	Node
	expressionNode()
}

// Document is the root node of a MAML document.
type Document struct {
	Statements []Statement
}

// TokenLiteral returns the literal value of the token associated with the node.
func (p *Document) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	}
	return ""
}

// String returns a string representation of the node.
func (p *Document) String() string {
	var out bytes.Buffer
	for _, s := range p.Statements {
		out.WriteString(s.String())
	}
	return out.String()
}

// ExpressionStatement represents a statement that consists of a single expression.
type ExpressionStatement struct {
	Token      token.Token // the first token of the expression
	Expression Expression
}

func (es *ExpressionStatement) statementNode()       {}
func (es *ExpressionStatement) TokenLiteral() string { return es.Token.Literal }
func (es *ExpressionStatement) String() string {
	if es.Expression != nil {
		return es.Expression.String()
	}
	return ""
}

// Identifier represents an identifier.
type Identifier struct {
	Token token.Token // the token.IDENT token
	Value string
}

func (i *Identifier) expressionNode()      {}
func (i *Identifier) TokenLiteral() string { return i.Token.Literal }
func (i *Identifier) String() string       { return i.Value }

// BooleanLiteral represents a boolean literal.
type BooleanLiteral struct {
	Token token.Token
	Value bool
}

func (b *BooleanLiteral) expressionNode()      {}
func (b *BooleanLiteral) TokenLiteral() string { return b.Token.Literal }
func (b *BooleanLiteral) String() string       { return b.Token.Literal }

// IntegerLiteral represents an integer literal.
type IntegerLiteral struct {
	Token token.Token
	Value int64
}

func (il *IntegerLiteral) expressionNode()      {}
func (il *IntegerLiteral) TokenLiteral() string { return il.Token.Literal }
func (il *IntegerLiteral) String() string       { return il.Token.Literal }

// FloatLiteral represents a float literal.
type FloatLiteral struct {
	Token token.Token
	Value float64
}

func (fl *FloatLiteral) expressionNode()      {}
func (fl *FloatLiteral) TokenLiteral() string { return fl.Token.Literal }
func (fl *FloatLiteral) String() string       { return fl.Token.Literal }

// StringLiteral represents a string literal.
type StringLiteral struct {
	Token token.Token
	Value string
}

func (sl *StringLiteral) expressionNode()      {}
func (sl *StringLiteral) TokenLiteral() string { return sl.Token.Literal }
func (sl *StringLiteral) String() string       { return strconv.Quote(sl.Token.Literal) }

// ArrayLiteral represents an array literal.
type ArrayLiteral struct {
	Token    token.Token // the '[' token
	Elements []Expression
}

func (al *ArrayLiteral) expressionNode()      {}
func (al *ArrayLiteral) TokenLiteral() string { return al.Token.Literal }
func (al *ArrayLiteral) String() string {
	var out bytes.Buffer
	elements := []string{}
	for _, el := range al.Elements {
		elements = append(elements, el.String())
	}
	out.WriteString("[")
	out.WriteString(strings.Join(elements, ", "))
	out.WriteString("]")
	return out.String()
}

// ObjectLiteral represents an object literal.
type ObjectLiteral struct {
	Token token.Token // the '{' token
	Pairs []*PairExpression
}

func (ol *ObjectLiteral) expressionNode()      {}
func (ol *ObjectLiteral) TokenLiteral() string { return ol.Token.Literal }
func (ol *ObjectLiteral) String() string {
	var out bytes.Buffer
	pairs := []string{}
	for _, p := range ol.Pairs {
		pairs = append(pairs, p.String())
	}
	out.WriteString("{")
	out.WriteString(strings.Join(pairs, ", "))
	out.WriteString("}")
	return out.String()
}

// PairExpression represents a key-value pair in an object literal.
type PairExpression struct {
	Token token.Token // The ':' token
	Key   Expression
	Value Expression
}

func (pe *PairExpression) expressionNode()      {}
func (pe *PairExpression) TokenLiteral() string { return pe.Token.Literal }
func (pe *PairExpression) String() string {
	return pe.Key.String() + ":" + pe.Value.String()
}

// NullLiteral represents a null literal.
type NullLiteral struct {
	Token token.Token
}

func (nl *NullLiteral) expressionNode()      {}
func (nl *NullLiteral) TokenLiteral() string { return nl.Token.Literal }
func (nl *NullLiteral) String() string       { return "null" }
