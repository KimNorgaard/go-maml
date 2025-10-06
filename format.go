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
	opts   *options
}

const (
	defaultIndent = 2
	tripleQuote   = `"""`
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
	return &formatter{w: w, indent: indentStr, opts: opts}
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

func (f *formatter) writeNode(node ast.Node) error { //nolint:gocognit
	switch n := node.(type) {
	case *ast.Document:
		for _, comment := range n.HeadComments {
			if err := f.write("# " + comment.Value + "\n"); err != nil {
				return err
			}
		}

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
		if f.opts.inlineStrings || f.indent == "" || !strings.ContainsRune(n.Value, '\n') || strings.Contains(n.Value, `"""`) {
			return f.write(n.String())
		}
		return f.writeMultilineString(n.Value)

	case *ast.IntegerLiteral, *ast.FloatLiteral, *ast.BooleanLiteral:
		return f.write(n.TokenLiteral())

	case *ast.NullLiteral:
		return f.write("null")

	default:
		return fmt.Errorf("maml: unsupported node type for formatting: %T", n)
	}
}

// writeMultilineString formats and writes a string value, deciding between standard
// and multiline string literals based on options and content.
func (f *formatter) writeMultilineString(s string) error {
	return f.write(tripleQuote + "\n" + s + tripleQuote)
}

func (f *formatter) writePrettyObject(obj *ast.ObjectLiteral) error {
	f.depth++
	for i, pair := range obj.Pairs {
		if err := f.writePairPrefix(i, pair); err != nil {
			return err
		}
		if err := f.writePairKeyValue(pair); err != nil {
			return err
		}
		if err := f.writePairSuffix(i, len(obj.Pairs), pair); err != nil {
			return err
		}
		if err := f.writePairFootComments(pair); err != nil {
			return err
		}
	}
	f.depth--

	if len(obj.Pairs) > 0 {
		if err := f.write("\n"); err != nil {
			return err
		}
	}
	return f.writeIndent()
}

// writePairPrefix handles writing newlines, head comments, and indentation before the key.
func (f *formatter) writePairPrefix(i int, pair *ast.KeyValueExpression) error {
	// Use the recorded number of newlines from the source to preserve vertical spacing.
	numNewlines := pair.NewlinesBefore
	if i == 0 {
		// First pair is always one newline after '{'.
		numNewlines = 1
	} else if numNewlines == 0 {
		// Subsequent pairs need at least one newline for pretty printing.
		numNewlines = 1
	}

	for j := 0; j < numNewlines; j++ {
		if err := f.write("\n"); err != nil {
			return err
		}
	}

	for _, comment := range pair.HeadComments {
		if err := f.writeIndent(); err != nil {
			return err
		}
		if err := f.write("# " + comment.Value + "\n"); err != nil {
			return err
		}
	}

	return f.writeIndent()
}

// writePairKeyValue handles writing the "key: value" part of a pair.
func (f *formatter) writePairKeyValue(pair *ast.KeyValueExpression) error {
	if err := f.write(pair.Key.String() + ": "); err != nil {
		return err
	}
	return f.writeNode(pair.Value)
}

// writePairSuffix handles writing commas and line comments after the value.
func (f *formatter) writePairSuffix(i, pairCount int, pair *ast.KeyValueExpression) error {
	if f.opts.useFieldCommas {
		isLast := i == pairCount-1
		if !isLast {
			if err := f.write(","); err != nil {
				return err
			}
		} else if f.opts.useTrailingCommas {
			if err := f.write(","); err != nil {
				return err
			}
		}
	}

	if pair.LineComment != nil {
		if err := f.write(" # " + pair.LineComment.Value); err != nil {
			return err
		}
	}
	return nil
}

// writePairFootComments handles writing foot comments after a key-value pair.
func (f *formatter) writePairFootComments(pair *ast.KeyValueExpression) error {
	for _, comment := range pair.FootComments {
		if err := f.write("\n"); err != nil {
			return err
		}
		if err := f.writeIndent(); err != nil {
			return err
		}
		if err := f.write("# " + comment.Value); err != nil {
			return err
		}
	}
	return nil
}

func (f *formatter) writeCompactObject(obj *ast.ObjectLiteral) error {
	for i, pair := range obj.Pairs {
		if i > 0 {
			if err := f.write(","); err != nil {
				return err
			}
		}
		if err := f.write(pair.Key.String()); err != nil {
			return err
		}
		if err := f.write(":"); err != nil {
			return err
		}
		if err := f.writeNode(pair.Value); err != nil {
			return err
		}
	}
	return nil
}

func (f *formatter) writeObject(obj *ast.ObjectLiteral) error {
	if err := f.write("{"); err != nil {
		return err
	}

	if len(obj.Pairs) > 0 {
		if f.indent != "" {
			if err := f.writePrettyObject(obj); err != nil {
				return err
			}
		} else {
			if err := f.writeCompactObject(obj); err != nil {
				return err
			}
		}
	}

	return f.write("}")
}

func (f *formatter) writePrettyArray(arr *ast.ArrayLiteral) error { //nolint:gocognit
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
		if f.opts.useFieldCommas {
			if i < len(arr.Elements)-1 {
				if err := f.write(","); err != nil {
					return err
				}
			} else if f.opts.useTrailingCommas {
				if err := f.write(","); err != nil {
					return err
				}
			}
		}
	}
	f.depth--
	if err := f.write("\n"); err != nil {
		return err
	}
	return f.writeIndent()
}

func (f *formatter) writeCompactArray(arr *ast.ArrayLiteral) error {
	for i, elem := range arr.Elements {
		if i > 0 {
			if err := f.write(","); err != nil {
				return err
			}
		}
		if err := f.writeNode(elem); err != nil {
			return err
		}
	}
	return nil
}

func (f *formatter) writeArray(arr *ast.ArrayLiteral) error {
	if err := f.write("["); err != nil {
		return err
	}

	if len(arr.Elements) > 0 {
		switch {
		case f.opts.inlineArrays:
			if err := f.writeCompactArray(arr); err != nil {
				return err
			}
		case f.indent != "":
			if err := f.writePrettyArray(arr); err != nil {
				return err
			}
		default:
			if err := f.writeCompactArray(arr); err != nil {
				return err
			}
		}
	}

	return f.write("]")
}
