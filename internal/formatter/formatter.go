package formatter

import (
	"fmt"
	"io"

	"github.com/KimNorgaard/go-maml/internal/ast"
)

// Formatter writes a MAML AST to an output stream.
type Formatter struct {
	w io.Writer
}

// New returns a new formatter that writes to w.
func New(w io.Writer) *Formatter {
	return &Formatter{w: w}
}

// Format writes the MAML string representation of the AST node to the writer.
func (f *Formatter) Format(node ast.Node) error {
	return f.writeNode(node)
}

func (f *Formatter) writeNode(node ast.Node) error {
	switch n := node.(type) {
	case *ast.Document:
		// A document can have multiple statements, but for marshaling,
		// it will typically be just one.
		for i, stmt := range n.Statements {
			if err := f.writeNode(stmt); err != nil {
				return err
			}
			// Add a newline if there are multiple top-level statements.
			if i < len(n.Statements)-1 {
				if _, err := f.w.Write([]byte("\n")); err != nil {
					return err
				}
			}
		}
		return nil

	case *ast.ExpressionStatement:
		return f.writeNode(n.Expression)

	case *ast.ObjectLiteral:
		if _, err := f.w.Write([]byte("{")); err != nil {
			return err
		}
		for i, pair := range n.Pairs {
			if i > 0 {
				if _, err := f.w.Write([]byte(", ")); err != nil {
					return err
				}
			} else {
				if _, err := f.w.Write([]byte(" ")); err != nil {
					return err
				}
			}

			if _, err := f.w.Write([]byte(pair.Key.String())); err != nil {
				return err
			}
			if _, err := f.w.Write([]byte(": ")); err != nil {
				return err
			}
			if err := f.writeNode(pair.Value); err != nil {
				return err
			}
		}
		if len(n.Pairs) > 0 {
			if _, err := f.w.Write([]byte(" ")); err != nil {
				return err
			}
		}
		_, err := f.w.Write([]byte("}"))
		return err

	case *ast.ArrayLiteral:
		if _, err := f.w.Write([]byte("[")); err != nil {
			return err
		}
		for i, elem := range n.Elements {
			if i > 0 {
				if _, err := f.w.Write([]byte(", ")); err != nil {
					return err
				}
			}
			if err := f.writeNode(elem); err != nil {
				return err
			}
		}
		_, err := f.w.Write([]byte("]"))
		return err

	case *ast.StringLiteral:
		_, err := f.w.Write([]byte(n.String()))
		return err

	case *ast.IntegerLiteral, *ast.FloatLiteral, *ast.BooleanLiteral:
		_, err := f.w.Write([]byte(n.TokenLiteral()))
		return err

	case *ast.NullLiteral:
		_, err := f.w.Write([]byte("null"))
		return err

	default:
		return fmt.Errorf("maml: unsupported node type for formatting: %T", n)
	}
}
