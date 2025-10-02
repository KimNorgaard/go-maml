package maml

import (
	"fmt"
	"io"
	"strings"

	"github.com/KimNorgaard/go-maml/internal/ast"
)

// formatter writes a MAML AST to an output stream.
type formatter struct {
	w      io.Writer
	indent string
	depth  int
}

const (
	defaultIndent = 2
)

// newFormatter returns a new formatter that writes to w.
func newFormatter(w io.Writer, opts *options) *formatter {
	spaces := defaultIndent
	if opts.indent != nil {
		spaces = *opts.indent
	}
	var indentStr string
	if spaces > 0 {
		indentStr = strings.Repeat(" ", spaces)
	}
	return &formatter{w: w, indent: indentStr}
}

// format writes the MAML string representation of the AST node to the writer.
func (f *formatter) format(node ast.Node) error {
	return f.writeNode(node)
}

func (f *formatter) write(s string) error {
	_, err := f.w.Write([]byte(s))
	return err
}

func (f *formatter) writeIndent() error {
	if f.indent == "" {
		return nil
	}
	for i := 0; i < f.depth; i++ {
		if err := f.write(f.indent); err != nil {
			return err
		}
	}
	return nil
}

func (f *formatter) writeNode(node ast.Node) error {
	switch n := node.(type) {
	case *ast.Document:
		for i, stmt := range n.Statements {
			if err := f.writeNode(stmt); err != nil {
				return err
			}
			if i < len(n.Statements)-1 {
				if err := f.write("\n"); err != nil {
					return err
				}
			}
		}
		return nil

	case *ast.ExpressionStatement:
		return f.writeNode(n.Expression)

	case *ast.ObjectLiteral:
		return f.writeObject(n)

	case *ast.ArrayLiteral:
		return f.writeArray(n)

	case *ast.StringLiteral:
		return f.write(n.String())

	case *ast.IntegerLiteral, *ast.FloatLiteral, *ast.BooleanLiteral:
		return f.write(n.TokenLiteral())

	case *ast.NullLiteral:
		return f.write("null")

	default:
		return fmt.Errorf("maml: unsupported node type for formatting: %T", n)
	}
}

func (f *formatter) writeObject(obj *ast.ObjectLiteral) error {
	if err := f.write("{"); err != nil {
		return err
	}

	if len(obj.Pairs) == 0 {
		return f.write("}")
	}

	if f.indent != "" { // Pretty-print mode
		f.depth++
		for i, pair := range obj.Pairs {
			if err := f.write("\n"); err != nil {
				return err
			}
			if err := f.writeIndent(); err != nil {
				return err
			}
			if err := f.write(pair.Key.String() + ": "); err != nil {
				return err
			}
			if err := f.writeNode(pair.Value); err != nil {
				return err
			}
			if i < len(obj.Pairs)-1 {
				if err := f.write(","); err != nil {
					return err
				}
			}
		}
		f.depth--
		if err := f.write("\n"); err != nil {
			return err
		}
		if err := f.writeIndent(); err != nil {
			return err
		}
	} else { // Compact mode
		if err := f.write(" "); err != nil {
			return err
		}
		for i, pair := range obj.Pairs {
			if i > 0 {
				if err := f.write(", "); err != nil {
					return err
				}
			}
			if err := f.write(pair.Key.String() + ": "); err != nil {
				return err
			}
			if err := f.writeNode(pair.Value); err != nil {
				return err
			}
		}
		if err := f.write(" "); err != nil {
			return err
		}
	}

	return f.write("}")
}

func (f *formatter) writeArray(arr *ast.ArrayLiteral) error {
	if err := f.write("["); err != nil {
		return err
	}

	if len(arr.Elements) == 0 {
		return f.write("]")
	}

	if f.indent != "" { // Pretty-print mode
		f.depth++
		for i, elem := range arr.Elements {
			if err := f.write("\n"); err != nil {
				return err
			}
			if err := f.writeIndent(); err != nil {
				return err
			}
			if err := f.writeNode(elem); err != nil {
				return err
			}
			if i < len(arr.Elements)-1 {
				if err := f.write(","); err != nil {
					return err
				}
			}
		}
		f.depth--
		if err := f.write("\n"); err != nil {
			return err
		}
		if err := f.writeIndent(); err != nil {
			return err
		}
	} else { // Compact mode
		for i, elem := range arr.Elements {
			if i > 0 {
				if err := f.write(", "); err != nil {
					return err
				}
			}
			if err := f.writeNode(elem); err != nil {
				return err
			}
		}
	}

	return f.write("]")
}
