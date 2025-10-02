package formatter

import (
	"fmt"
	"io"
	"strings"

	"github.com/KimNorgaard/go-maml/internal/ast"
)

const (
	defaultIndent = 2
)

// Formatter writes a MAML AST to an output stream.
type Formatter struct {
	w      io.Writer
	indent string
	depth  int
}

// New returns a new formatter that writes to w.
func New(w io.Writer, indentSpaces *int) *Formatter {
	spaces := defaultIndent
	if indentSpaces != nil {
		spaces = *indentSpaces
	}
	var indentStr string
	if spaces > 0 {
		indentStr = strings.Repeat(" ", spaces)
	}
	return &Formatter{w: w, indent: indentStr}
}

// Format writes the MAML string representation of the AST node to the writer.
func (f *Formatter) Format(node ast.Node) error {
	return f.writeNode(node)
}

func (f *Formatter) writeIndent() error {
	if f.indent == "" {
		return nil
	}
	for i := 0; i < f.depth; i++ {
		if _, err := f.w.Write([]byte(f.indent)); err != nil {
			return err
		}
	}
	return nil
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
		if len(n.Pairs) > 0 {
			if f.indent != "" {
				f.depth++
				for i, pair := range n.Pairs {
					if _, err := f.w.Write([]byte("\n")); err != nil {
						return err
					}
					if err := f.writeIndent(); err != nil {
						return err
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
					if i < len(n.Pairs)-1 {
						if _, err := f.w.Write([]byte(",")); err != nil {
							return err
						}
					}
				}
				f.depth--
				if _, err := f.w.Write([]byte("\n")); err != nil {
					return err
				}
				if err := f.writeIndent(); err != nil {
					return err
				}
			} else {
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
			}
		}
		_, err := f.w.Write([]byte("}"))
		return err

	case *ast.ArrayLiteral:
		if _, err := f.w.Write([]byte("[")); err != nil {
			return err
		}
		if len(n.Elements) > 0 {
			if f.indent != "" {
				f.depth++
				for i, elem := range n.Elements {
					if _, err := f.w.Write([]byte("\n")); err != nil {
						return err
					}
					if err := f.writeIndent(); err != nil {
						return err
					}
					if err := f.writeNode(elem); err != nil {
						return err
					}
					if i < len(n.Elements)-1 {
						if _, err := f.w.Write([]byte(",")); err != nil {
							return err
						}
					}
				}
				f.depth--
				if _, err := f.w.Write([]byte("\n")); err != nil {
					return err
				}
				if err := f.writeIndent(); err != nil {
					return err
				}
			} else {
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
