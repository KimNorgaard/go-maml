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

	// TODO: This is the beginning of the core unmarshaling logic.
	// It will be expanded with a large switch statement to handle all AST node types
	// and map them to the corresponding reflect.Value kinds (string, int, struct, map, slice, etc.).
	switch node := expr.(type) {
	case *ast.NullLiteral:
		// For null, we set the Go value to its zero value.
		rv.Set(reflect.Zero(rv.Type()))
		return nil
	default:
		return fmt.Errorf("maml: mapping for AST node type %T not yet implemented", node)
	}
}
