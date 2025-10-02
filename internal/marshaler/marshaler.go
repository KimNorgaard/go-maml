package marshaler

import (
	"fmt"
	"math"
	"reflect"
	"strings"

	"github.com/KimNorgaard/go-maml/internal/ast"
	"github.com/KimNorgaard/go-maml/internal/token"
)

// Marshal converts a Go value into a MAML AST node.
func Marshal(v any) (ast.Node, error) {
	m := &marshaler{}
	return m.marshal(reflect.ValueOf(v))
}

type marshaler struct{}

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
// It is equivalent to the `encoding/json` definition of empty:
// false, 0, a nil pointer, a nil interface value, and any empty array,
// slice, map, or string.
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

func (m *marshaler) marshal(v reflect.Value) (ast.Node, error) {
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
			elemNode, err := m.marshal(v.Index(i))
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

			valueNode, err := m.marshal(value)
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
		// Note: Map iteration is not ordered, so this might produce
		// different output for the same map. This is acceptable for MAML.

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

			// Skip unexported fields
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

			valueNode, err := m.marshal(fieldValue)
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
