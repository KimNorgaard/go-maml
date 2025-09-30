package maml

import (
	"fmt"

	"github.com/KimNorgaard/go-maml/ast"
	"github.com/KimNorgaard/go-maml/token"
)

// Parse is a placeholder function that will eventually parse MAML source.
// For now, it returns a placeholder node.
func Parse(src []byte) (ast.Node, error) {
	return &placeholderNode{string(src)}, nil
}

// placeholderNode is a temporary struct to allow the test to be written.
// It will be replaced by the actual AST nodes.
type placeholderNode struct {
	content string
}

func (p *placeholderNode) Token() token.Token {
	return token.Token{}
}

func (p *placeholderNode) String() string {
	// This will eventually be a pretty-printed representation of the AST.
	return fmt.Sprintf("AST for %s", p.content)
}

// This ensures placeholderNode satisfies the ast.Node interface.
var _ ast.Node = (*placeholderNode)(nil)
