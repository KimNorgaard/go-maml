package maml

import (
	"fmt"
	"io"
	"math"
	"reflect"
	"sort"
	"strconv"
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

// Encode writes the MAML encoding of in to the stream.
func (e *Encoder) Encode(in any) error {
	o := options{}
	for _, opt := range e.opts {
		if err := opt(&o); err != nil {
			return err
		}
	}

	es := &encodeState{seen: make(map[uintptr]struct{})}
	node, err := es.marshalValue(reflect.ValueOf(in))
	if err != nil {
		return fmt.Errorf("maml: %w", err)
	}

	f := newFormatter(e.w, &o)
	return f.format(node)
}

type encodeState struct {
	// Keep track of pointers seen so far.
	seen map[uintptr]struct{}
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
		var errs []string
		for _, err := range p.Errors() {
			errs = append(errs, err.Message)
		}
		return nil, &MarshalerError{
			Type: v.Type(),
			Err:  fmt.Errorf("invalid MAML output: %s", strings.Join(errs, "; ")),
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
	case reflect.Interface, reflect.Pointer:
		return v.IsNil()
	}
	return false
}

// isBareKey checks if a string can be used as a bare key in an object.
// Bare keys can be identifiers or numbers, but not keywords.
func isBareKey(s string) bool {
	if s == "" {
		return false
	}

	// Keywords must be quoted.
	if token.LookupIdent(s) != token.IDENT {
		return false
	}

	// If it can be parsed as a number, it can be a bare key.
	if _, ok := lexer.ParseAsNumber(s); ok {
		return true
	}

	// Otherwise, it must be a valid identifier.
	// Must not start with a hyphen (unless it's a number, handled above).
	if s[0] == '-' {
		return false
	}

	for _, r := range s {
		if !isIdentifierChar(r) {
			return false
		}
	}

	return true
}

// isIdentifierChar checks if a rune is a valid character for a MAML identifier.
func isIdentifierChar(r rune) bool {
	return ('a' <= r && r <= 'z') || ('A' <= r && r <= 'Z') ||
		('0' <= r && r <= '9') || r == '_' || r == '-'
}

func (e *encodeState) marshalValue(v reflect.Value) (ast.Node, error) { //nolint:gocyclo
	if !v.IsValid() {
		return &ast.NullLiteral{Token: token.Token{Type: token.NULL, Literal: "null"}}, nil
	}

	// Check for custom Marshaler implementation first.
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
			pv = reflect.New(v.Type())
			pv.Elem().Set(v)
		}
		if pv.Type().NumMethod() > 0 && pv.CanInterface() {
			if u, ok := pv.Interface().(Marshaler); ok {
				return e.marshalCustom(pv, u)
			}
		}
	}

	switch v.Kind() {
	case reflect.Pointer:
		return e.marshalPointer(v)
	case reflect.Interface:
		return e.marshalInterface(v)
	case reflect.String:
		return e.marshalString(v)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return e.marshalInt(v)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return e.marshalUint(v)
	case reflect.Float32, reflect.Float64:
		return e.marshalFloat(v)
	case reflect.Bool:
		return e.marshalBool(v)
	case reflect.Slice, reflect.Array:
		return e.marshalSlice(v)
	case reflect.Map:
		return e.marshalMap(v)
	case reflect.Struct:
		return e.marshalStruct(v)
	default:
		// nil can be a valid value for some kinds (e.g. chan, func, map, ptr, slice)
		if !v.IsValid() || v.IsZero() {
			return &ast.NullLiteral{Token: token.Token{Type: token.NULL, Literal: "null"}}, nil
		}
		return nil, fmt.Errorf("maml: unsupported type for marshaling: %s", v.Type())
	}
}

func (e *encodeState) marshalPointer(v reflect.Value) (ast.Node, error) {
	if v.IsNil() {
		return &ast.NullLiteral{Token: token.Token{Type: token.NULL, Literal: "null"}}, nil
	}
	ptr := v.Pointer()
	if _, ok := e.seen[ptr]; ok {
		return nil, fmt.Errorf("maml: encountered a cycle via type %s", v.Type())
	}
	e.seen[ptr] = struct{}{}
	result, err := e.marshalValue(v.Elem())
	delete(e.seen, ptr)
	return result, err
}

func (e *encodeState) marshalInterface(v reflect.Value) (ast.Node, error) {
	if v.IsNil() {
		return &ast.NullLiteral{Token: token.Token{Type: token.NULL, Literal: "null"}}, nil
	}
	return e.marshalValue(v.Elem())
}

func (e *encodeState) marshalString(v reflect.Value) (ast.Node, error) {
	lit := v.String()
	return &ast.StringLiteral{Token: token.Token{Type: token.STRING, Literal: lit}, Value: lit}, nil
}

func (e *encodeState) marshalInt(v reflect.Value) (ast.Node, error) {
	val := v.Int()
	lit := fmt.Sprintf("%d", val)
	return &ast.IntegerLiteral{Token: token.Token{Type: token.INT, Literal: lit}, Value: val}, nil
}

func (e *encodeState) marshalUint(v reflect.Value) (ast.Node, error) {
	val := v.Uint()
	if val > math.MaxInt64 {
		return nil, fmt.Errorf("maml: cannot marshal uint64 %d into MAML (overflows int64)", val)
	}
	lit := fmt.Sprintf("%d", val)
	return &ast.IntegerLiteral{Token: token.Token{Type: token.INT, Literal: lit}, Value: int64(val)}, nil
}

func (e *encodeState) marshalFloat(v reflect.Value) (ast.Node, error) {
	val := v.Float()
	lit := strconv.FormatFloat(val, 'g', -1, 64)
	// If the formatted string doesn't contain a decimal or an exponent, add .0
	// to ensure it's treated as a float.
	if !strings.ContainsAny(lit, ".eE") {
		lit += ".0"
	}
	return &ast.FloatLiteral{Token: token.Token{Type: token.FLOAT, Literal: lit}, Value: val}, nil
}

func (e *encodeState) marshalBool(v reflect.Value) (ast.Node, error) {
	val := v.Bool()
	lit := fmt.Sprintf("%t", val)
	tokType := token.FALSE
	if val {
		tokType = token.TRUE
	}
	return &ast.BooleanLiteral{Token: token.Token{Type: tokType, Literal: lit}, Value: val}, nil
}

func (e *encodeState) marshalSlice(v reflect.Value) (ast.Node, error) {
	if v.Kind() == reflect.Slice {
		if v.IsNil() {
			return &ast.NullLiteral{Token: token.Token{Type: token.NULL, Literal: "null"}}, nil
		}
		ptr := v.Pointer()
		if _, ok := e.seen[ptr]; ok {
			return nil, fmt.Errorf("maml: encountered a cycle via type %s", v.Type())
		}
		e.seen[ptr] = struct{}{}
		defer delete(e.seen, ptr)
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
}

func (e *encodeState) marshalMap(v reflect.Value) (ast.Node, error) {
	if v.IsNil() {
		return &ast.NullLiteral{Token: token.Token{Type: token.NULL, Literal: "null"}}, nil
	}
	ptr := v.Pointer()
	if _, ok := e.seen[ptr]; ok {
		return nil, fmt.Errorf("maml: encountered a cycle via type %s", v.Type())
	}
	e.seen[ptr] = struct{}{}
	defer delete(e.seen, ptr)

	if v.Type().Key().Kind() != reflect.String {
		return nil, fmt.Errorf("maml: map key type must be a string, got %s", v.Type().Key())
	}

	pairs := make([]*ast.KeyValueExpression, 0, v.Len())
	keys := v.MapKeys()
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].String() < keys[j].String()
	})
	for _, key := range keys {
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

		var keyNode ast.Expression
		if isBareKey(keyStr) {
			tok := token.Token{Literal: keyStr}
			if typ, ok := lexer.ParseAsNumber(keyStr); ok {
				tok.Type = typ
			} else {
				tok.Type = token.IDENT
			}
			keyNode = &ast.Identifier{Token: tok, Value: keyStr}
		} else {
			keyNode = &ast.StringLiteral{
				Token: token.Token{Type: token.STRING, Literal: keyStr},
				Value: keyStr,
			}
		}

		pairs = append(pairs, &ast.KeyValueExpression{
			Token: token.Token{Type: token.COLON, Literal: ":"},
			Key:   keyNode,
			Value: valueExpr,
		})
	}

	return &ast.ObjectLiteral{
		Token: token.Token{Type: token.LBRACE, Literal: "{"},
		Pairs: pairs,
	}, nil
}

func (e *encodeState) marshalStruct(v reflect.Value) (ast.Node, error) { //nolint:gocognit
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

		var keyNode ast.Expression
		if isBareKey(keyStr) {
			tok := token.Token{Literal: keyStr}
			if typ, ok := lexer.ParseAsNumber(keyStr); ok {
				tok.Type = typ
			} else {
				tok.Type = token.IDENT
			}
			keyNode = &ast.Identifier{Token: tok, Value: keyStr}
		} else {
			keyNode = &ast.StringLiteral{
				Token: token.Token{Type: token.STRING, Literal: keyStr},
				Value: keyStr,
			}
		}

		pairs = append(pairs, &ast.KeyValueExpression{
			Token: token.Token{Type: token.COLON, Literal: ":"},
			Key:   keyNode,
			Value: valueExpr,
		})
	}

	return &ast.ObjectLiteral{
		Token: token.Token{Type: token.LBRACE, Literal: "{"},
		Pairs: pairs,
	}, nil
}
