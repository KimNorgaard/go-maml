package ast

import "github.com/KimNorgaard/go-maml/token"

// Node is the base interface for all AST nodes.
type Node interface {
	// Token returns the first token of the node.
	Token() token.Token
	// String returns a string representation of the node.
	String() string
}
