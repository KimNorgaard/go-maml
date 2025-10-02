package maml

import (
	"fmt"
	"io"
	"math"
	"reflect"
	"strings"

	"github.com/KimNorgaard/go-maml/internal/ast"
	"github.com/KimNorgaard/go-maml/internal/lexer"
	"github.com/KimNorgaard/go-maml/internal/parser"
	"github.com/KimNorgaard/go-maml/internal/token"
)

// Encoder writes MAML values to an output stream.
type Encoder struct {
	w    io.Writer
	opts []Option
}

// NewEncoder returns a new encoder that writes to w.
func NewEncoder(w io.Writer, opts ...Option) *Encoder {
	return &Encoder{w: w, opts: opts}
}

// Encode writes the MAML encoding of v to the stream.
func (e *Encoder) Encode(v any) error {
	o := options{}
	for _, opt := range e.opts {
		if err := opt(&o); err != nil {
			return err
		}
	}

	es := &encodeState{}
	node, err := es.marshalValue(reflect.ValueOf(v))
	if err != nil {
		return fmt.Errorf("maml: %w", err)
	}

	f := newFormatter(e.w, &o)
	return f.format(node)
}

type encodeState struct {
	// Future state like depth counters can be added here.
}

func (e *encodeState) marshalCustom(v reflect.Value, u Marshaler) (ast.Node, error) {
	b, err := u.MarshalMAML()
	if err != nil {
		return nil, &MarshalerError{Type: v.Type(), Err: err}
	}

	// The user's marshaled output must be parsed back into an AST node
	// to be integrated into the main AST being built.
	l := lexer.New(b)
	p := parser.New(l)
	doc := p.Parse()

	if len(p.Errors()) > 0 {
		return nil, &MarshalerError{
			Type: v.Type(),
			Err:  fmt.Errorf("invalid MAML output: %s", strings.Join(p.Errors(), "; ")),
		}
	}

	if len(doc.Statements) == 0 {
		// An empty document from a custom marshaler is treated as a null value.
		return &ast.NullLiteral{Token: token.Token{Type: token.NULL, Literal: "null"}}, nil
	}

	if len(doc.Statements) != 1 {
		return nil, &MarshalerError{
			Type: v.Type(),
			Err:  fmt.Errorf("expected single MAML expression, got %d statements", len(doc.Statements)),
		}
	}

	stmt, ok := doc.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		return nil, &MarshalerError{
			Type: v.Type(),
			Err:  fmt.Errorf("expected MAML expression, but got a different statement type"),
		}
	}

	return stmt.Expression, nil
}

// parseTag splits a maml struct tag into its name and options.
func parseTag(tag string) (string, map[string]bool) {
	parts := strings.Split(tag, ",")
	name := parts[0]
	options := make(map[string]bool)
	for _, part := range parts[1:] {
		options[strings.TrimSpace(part)] = true
	}
	return name, options
}

// isEmptyValue reports whether the value v is empty.
func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	return false
}

func (e *encodeState) marshalValue(v reflect.Value) (ast.Node, error) {
	// Handle nil interfaces explicitly to avoid panics.
	if !v.IsValid() || (v.Kind() == reflect.Interface && v.IsNil()) {
		return &ast.NullLiteral{Token: token.Token{Type: token.NULL, Literal: "null"}}, nil
	}

	// Check for custom Marshaler implementation.
	// We must check the value itself and a pointer to the value,
	// to handle both value and pointer receivers.
	if v.Type().NumMethod() > 0 && v.CanInterface() {
		if u, ok := v.Interface().(Marshaler); ok {
			return e.marshalCustom(v, u)
		}
	}
	if v.Kind() != reflect.Pointer {
		var pv reflect.Value
		if v.CanAddr() {
			pv = v.Addr()
		} else {
			// For non-addressable values (like struct literals),
			// create a pointer to a copy to check for the interface.
			pv = reflect.New(v.Type())
			pv.Elem().Set(v)
		}
		if pv.Type().NumMethod() > 0 && pv.CanInterface() {
			if u, ok := pv.Interface().(Marshaler); ok {
				return e.marshalCustom(pv, u)
			}
		}
	}

	// Follow pointers and interfaces to find the concrete value.
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return &ast.NullLiteral{Token: token.Token{Type: token.NULL, Literal: "null"}}, nil
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.String:
		lit := v.String()
		return &ast.StringLiteral{Token: token.Token{Type: token.STRING, Literal: lit}, Value: lit}, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val := v.Int()
		lit := fmt.Sprintf("%d", val)
		return &ast.IntegerLiteral{Token: token.Token{Type: token.INT, Literal: lit}, Value: val}, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		val := v.Uint()
		if val > math.MaxInt64 {
			return nil, fmt.Errorf("maml: cannot marshal uint64 %d into MAML (overflows int64)", val)
		}
		lit := fmt.Sprintf("%d", val)
		return &ast.IntegerLiteral{Token: token.Token{Type: token.INT, Literal: lit}, Value: int64(val)}, nil
	case reflect.Float32, reflect.Float64:
		val := v.Float()
		lit := fmt.Sprintf("%g", val)
		return &ast.FloatLiteral{Token: token.Token{Type: token.FLOAT, Literal: lit}, Value: val}, nil
	case reflect.Bool:
		val := v.Bool()
		lit := fmt.Sprintf("%t", val)
		tokType := token.FALSE
		if val {
			tokType = token.TRUE
		}
		return &ast.BooleanLiteral{Token: token.Token{Type: tokType, Literal: lit}, Value: val}, nil
	case reflect.Slice, reflect.Array:
		if v.Kind() == reflect.Slice && v.IsNil() {
			return &ast.NullLiteral{Token: token.Token{Type: token.NULL, Literal: "null"}}, nil
		}

		elements := make([]ast.Expression, v.Len())
		for i := 0; i < v.Len(); i++ {
			elemNode, err := e.marshalValue(v.Index(i))
			if err != nil {
				return nil, err
			}
			elemExpr, ok := elemNode.(ast.Expression)
			if !ok {
				return nil, fmt.Errorf("maml: marshaled element is not an expression")
			}
			elements[i] = elemExpr
		}
		return &ast.ArrayLiteral{
			Token:    token.Token{Type: token.LBRACK, Literal: "["},
			Elements: elements,
		}, nil
	case reflect.Map:
		if v.IsNil() {
			return &ast.NullLiteral{Token: token.Token{Type: token.NULL, Literal: "null"}}, nil
		}

		if v.Type().Key().Kind() != reflect.String {
			return nil, fmt.Errorf("maml: map key type must be a string, got %s", v.Type().Key())
		}

		pairs := make([]*ast.KeyValueExpression, 0, v.Len())
		for _, key := range v.MapKeys() {
			value := v.MapIndex(key)

			valueNode, err := e.marshalValue(value)
			if err != nil {
				return nil, err
			}
			valueExpr, ok := valueNode.(ast.Expression)
			if !ok {
				return nil, fmt.Errorf("maml: marshaled map value is not an expression")
			}

			keyStr := key.String()
			keyIdent := &ast.Identifier{
				Token: token.Token{Type: token.IDENT, Literal: keyStr},
				Value: keyStr,
			}

			pairs = append(pairs, &ast.KeyValueExpression{
				Token: token.Token{Type: token.COLON, Literal: ":"},
				Key:   keyIdent,
				Value: valueExpr,
			})
		}

		return &ast.ObjectLiteral{
			Token: token.Token{Type: token.LBRACE, Literal: "{"},
			Pairs: pairs,
		}, nil
	case reflect.Struct:
		pairs := make([]*ast.KeyValueExpression, 0, v.NumField())
		t := v.Type()

		for i := 0; i < v.NumField(); i++ {
			field := t.Field(i)
			fieldValue := v.Field(i)

			if !field.IsExported() {
				continue
			}

			tagStr := field.Tag.Get("maml")
			tagName, opts := parseTag(tagStr)

			if tagName == "-" {
				continue
			}

			if opts["omitempty"] && isEmptyValue(fieldValue) {
				continue
			}

			keyStr := field.Name
			if tagName != "" {
				keyStr = tagName
			}

			valueNode, err := e.marshalValue(fieldValue)
			if err != nil {
				return nil, err
			}
			valueExpr, ok := valueNode.(ast.Expression)
			if !ok {
				return nil, fmt.Errorf("maml: marshaled struct field value is not an expression")
			}

			keyIdent := &ast.Identifier{
				Token: token.Token{Type: token.IDENT, Literal: keyStr},
				Value: keyStr,
			}

			pairs = append(pairs, &ast.KeyValueExpression{
				Token: token.Token{Type: token.COLON, Literal: ":"},
				Key:   keyIdent,
				Value: valueExpr,
			})
		}

		return &ast.ObjectLiteral{
			Token: token.Token{Type: token.LBRACE, Literal: "{"},
			Pairs: pairs,
		}, nil
	default:
		// nil can be a valid value for some kinds (e.g. chan, func, map, ptr, slice)
		if !v.IsValid() || v.IsZero() {
			return &ast.NullLiteral{Token: token.Token{Type: token.NULL, Literal: "null"}}, nil
		}
		return nil, fmt.Errorf("maml: unsupported type for marshaling: %s", v.Type())
	}
}
