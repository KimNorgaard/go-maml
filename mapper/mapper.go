package mapper

import (
	"fmt"
	"reflect"

	"github.com/KimNorgaard/go-maml/ast"
)

// Map walks the AST from the document root and populates the Go value pointed to by v.
func Map(doc *ast.Document, v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return fmt.Errorf("maml: Unmarshal(non-pointer %T or nil)", v)
	}

	// A MAML document can be empty.
	if len(doc.Statements) == 0 {
		return nil
	}

	// The spec defines a MAML document as a single value.
	stmt, ok := doc.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		return fmt.Errorf("maml: document root is not a valid expression statement")
	}

	m := &mapper{}
	return m.mapValue(stmt.Expression, rv.Elem())
}

// mapper holds the state for the mapping process.
type mapper struct {
	// This can hold state for the mapping, like tracking visited pointers
	// to handle cycles if that feature is added in the future.
}

// mapValue is the core recursive function that maps an AST expression to a reflect.Value.
func (m *mapper) mapValue(expr ast.Expression, rv reflect.Value) error {
	if !rv.CanSet() {
		return fmt.Errorf("maml: cannot set value of type %s", rv.Type())
	}

	switch node := expr.(type) {
	case *ast.NullLiteral:
		rv.Set(reflect.Zero(rv.Type()))
		return nil
	case *ast.IntegerLiteral:
		switch rv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if rv.OverflowInt(node.Value) {
				return fmt.Errorf("maml: integer value %d overflows Go value of type %s", node.Value, rv.Type())
			}
			rv.SetInt(node.Value)
			return nil
		default:
			return fmt.Errorf("maml: cannot unmarshal integer into Go value of type %s", rv.Type())
		}
	case *ast.FloatLiteral:
		switch rv.Kind() {
		case reflect.Float32, reflect.Float64:
			if rv.OverflowFloat(node.Value) {
				return fmt.Errorf("maml: float value %f overflows Go value of type %s", node.Value, rv.Type())
			}
			rv.SetFloat(node.Value)
			return nil
		default:
			return fmt.Errorf("maml: cannot unmarshal float into Go value of type %s", rv.Type())
		}
	case *ast.BooleanLiteral:
		if rv.Kind() != reflect.Bool {
			return fmt.Errorf("maml: cannot unmarshal boolean into Go value of type %s", rv.Type())
		}
		rv.SetBool(node.Value)
		return nil
	case *ast.ArrayLiteral:
		if rv.Kind() != reflect.Slice {
			return fmt.Errorf("maml: cannot unmarshal array into Go value of type %s", rv.Type())
		}

		sliceType := rv.Type()
		newSlice := reflect.MakeSlice(sliceType, len(node.Elements), len(node.Elements))

		for i, elemAST := range node.Elements {
			if err := m.mapValue(elemAST, newSlice.Index(i)); err != nil {
				return err
			}
		}

		rv.Set(newSlice)
		return nil
	case *ast.ObjectLiteral:
		switch rv.Kind() {
		case reflect.Map:
			return m.mapMap(node, rv)
		case reflect.Struct:
			return m.mapStruct(node, rv)
		default:
			return fmt.Errorf("maml: cannot unmarshal object into Go value of type %s", rv.Type())
		}
	default:
		return fmt.Errorf("maml: mapping for AST node type %T not yet implemented", node)
	}
}

func (m *mapper) mapMap(obj *ast.ObjectLiteral, rv reflect.Value) error {
	mapType := rv.Type()
	if mapType.Key().Kind() != reflect.String {
		return fmt.Errorf("maml: cannot unmarshal object into map with non-string key type %s", mapType.Key())
	}

	if rv.IsNil() {
		rv.Set(reflect.MakeMap(mapType))
	}
	elemType := mapType.Elem()

	for _, pair := range obj.Pairs {
		var key string
		switch k := pair.Key.(type) {
		case *ast.Identifier:
			key = k.Value
		case *ast.StringLiteral:
			key = k.Value
		default:
			return fmt.Errorf("maml: invalid key type in object literal: %T", pair.Key)
		}

		newVal := reflect.New(elemType).Elem()
		if err := m.mapValue(pair.Value, newVal); err != nil {
			return err
		}

		rv.SetMapIndex(reflect.ValueOf(key), newVal)
	}

	return nil
}

func (m *mapper) mapStruct(obj *ast.ObjectLiteral, rv reflect.Value) error {
	for _, pair := range obj.Pairs {
		var key string
		switch k := pair.Key.(type) {
		case *ast.Identifier:
			key = k.Value
		case *ast.StringLiteral:
			key = k.Value
		default:
			return fmt.Errorf("maml: invalid key type in object literal: %T", pair.Key)
		}

		// A full implementation would cache struct fields and respect `maml` tags.
		field := rv.FieldByName(key)
		if !field.IsValid() || !field.CanSet() {
			continue // Ignore unknown or unexported fields
		}

		if err := m.mapValue(pair.Value, field); err != nil {
			return err
		}
	}

	return nil
}
