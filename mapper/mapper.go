package mapper

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/KimNorgaard/go-maml/ast"
)

const maxDepth = 1000

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

	stmt, ok := doc.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		return fmt.Errorf("maml: document root is not a valid expression statement")
	}

	m := &mapper{depth: maxDepth}
	return m.mapValue(stmt.Expression, rv.Elem())
}

// mapper holds the state for the mapping process.
type mapper struct {
	depth int
}

// mapValue is the core recursive function that maps an AST expression to a reflect.Value.
func (m *mapper) mapValue(expr ast.Expression, rv reflect.Value) error {
	m.depth--
	if m.depth <= 0 {
		return fmt.Errorf("maml: reached max recursion depth")
	}
	defer func() { m.depth++ }()

	// Handle pointers and null values together. Indirect pointers until it
	// reaches a concrete value, while correctly handling nulls.
	for rv.Kind() == reflect.Pointer {
		if _, isNull := expr.(*ast.NullLiteral); isNull {
			// The source is null and the destination is a pointer.
			// Set the pointer to nil and terminate for this value.
			rv.Set(reflect.Zero(rv.Type()))
			return nil
		}
		if rv.IsNil() {
			rv.Set(reflect.New(rv.Type().Elem()))
		}
		rv = rv.Elem()
	}

	// If the destination is a generic interface, we need to instantiate a concrete type.
	if rv.Kind() == reflect.Interface {
		return m.mapInterface(expr, rv)
	}

	if !rv.CanSet() {
		return fmt.Errorf("maml: cannot set value of type %s", rv.Type())
	}

	switch node := expr.(type) {
	case *ast.NullLiteral:
		rv.Set(reflect.Zero(rv.Type()))
		return nil
	case *ast.StringLiteral:
		return m.mapString(node, rv)
	case *ast.IntegerLiteral:
		return m.mapInt(node, rv)
	case *ast.FloatLiteral:
		return m.mapFloat(node, rv)
	case *ast.BooleanLiteral:
		return m.mapBool(node, rv)
	case *ast.ArrayLiteral:
		return m.mapArray(node, rv)
	case *ast.ObjectLiteral:
		return m.mapObject(node, rv)
	default:
		return fmt.Errorf("maml: mapping for AST node type %T not yet implemented", node)
	}
}

func (m *mapper) mapString(s *ast.StringLiteral, rv reflect.Value) error {
	if rv.Kind() != reflect.String {
		return fmt.Errorf("maml: cannot unmarshal string into Go value of type %s", rv.Type())
	}
	rv.SetString(s.Value)
	return nil
}

func (m *mapper) mapInt(i *ast.IntegerLiteral, rv reflect.Value) error {
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if rv.OverflowInt(i.Value) {
			return fmt.Errorf("maml: integer value %d overflows Go value of type %s", i.Value, rv.Type())
		}
		rv.SetInt(i.Value)
		return nil
	default:
		return fmt.Errorf("maml: cannot unmarshal integer into Go value of type %s", rv.Type())
	}
}

func (m *mapper) mapFloat(f *ast.FloatLiteral, rv reflect.Value) error {
	switch rv.Kind() {
	case reflect.Float32, reflect.Float64:
		if rv.OverflowFloat(f.Value) {
			return fmt.Errorf("maml: float value %f overflows Go value of type %s", f.Value, rv.Type())
		}
		rv.SetFloat(f.Value)
		return nil
	default:
		return fmt.Errorf("maml: cannot unmarshal float into Go value of type %s", rv.Type())
	}
}

func (m *mapper) mapBool(b *ast.BooleanLiteral, rv reflect.Value) error {
	if rv.Kind() != reflect.Bool {
		return fmt.Errorf("maml: cannot unmarshal boolean into Go value of type %s", rv.Type())
	}
	rv.SetBool(b.Value)
	return nil
}

func (m *mapper) mapArray(a *ast.ArrayLiteral, rv reflect.Value) error {
	if rv.Kind() != reflect.Slice {
		return fmt.Errorf("maml: cannot unmarshal array into Go value of type %s", rv.Type())
	}

	sliceType := rv.Type()
	newSlice := reflect.MakeSlice(sliceType, len(a.Elements), len(a.Elements))

	for i, elemAST := range a.Elements {
		if err := m.mapValue(elemAST, newSlice.Index(i)); err != nil {
			return err
		}
	}

	rv.Set(newSlice)
	return nil
}

func (m *mapper) mapObject(o *ast.ObjectLiteral, rv reflect.Value) error {
	switch rv.Kind() {
	case reflect.Map:
		return m.mapMap(o, rv)
	case reflect.Struct:
		return m.mapStruct(o, rv)
	}
	return fmt.Errorf("maml: cannot unmarshal object into Go value of type %s", rv.Type())
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
		var keyStr string
		switch k := pair.Key.(type) {
		case *ast.Identifier:
			keyStr = k.Value
		case *ast.StringLiteral:
			keyStr = k.Value
		default:
			return fmt.Errorf("maml: invalid key type in object literal: %T", pair.Key)
		}

		newVal := reflect.New(elemType).Elem()
		if err := m.mapValue(pair.Value, newVal); err != nil {
			return err
		}

		rv.SetMapIndex(reflect.ValueOf(keyStr), newVal)
	}

	return nil
}

func (m *mapper) mapStruct(obj *ast.ObjectLiteral, rv reflect.Value) error {
	fields := cachedFields(rv.Type())

	// For case-insensitive matching, create a lowercase-to-original-case map.
	lowerCaseFields := make(map[string]string)
	for name := range fields {
		lowerCaseFields[strings.ToLower(name)] = name
	}

	for _, pair := range obj.Pairs {
		var keyStr string
		switch k := pair.Key.(type) {
		case *ast.Identifier:
			keyStr = k.Value
		case *ast.StringLiteral:
			keyStr = k.Value
		default:
			return fmt.Errorf("maml: invalid key type in object literal: %T", pair.Key)
		}

		// Find the matching field.
		f, ok := fields[keyStr]
		if !ok {
			// Fallback to case-insensitive matching.
			if caseCorrectedName, ok2 := lowerCaseFields[strings.ToLower(keyStr)]; ok2 {
				f, ok = fields[caseCorrectedName]
			}
		}

		if !ok {
			continue // MAML key has no corresponding field in the struct.
		}

		fieldVal := rv.FieldByIndex(f.idx)
		if !fieldVal.IsValid() || !fieldVal.CanSet() {
			continue // Should not happen for exported fields
		}

		if err := m.mapValue(pair.Value, fieldVal); err != nil {
			return err
		}
	}

	return nil
}

func (m *mapper) mapInterface(expr ast.Expression, rv reflect.Value) error {
	if rv.NumMethod() != 0 {
		return fmt.Errorf("maml: cannot unmarshal into non-empty interface %s", rv.Type())
	}

	var concreteVal reflect.Value
	switch expr.(type) {
	case *ast.StringLiteral:
		var s string
		concreteVal = reflect.ValueOf(&s).Elem()
	case *ast.IntegerLiteral:
		var i int64
		concreteVal = reflect.ValueOf(&i).Elem()
	case *ast.FloatLiteral:
		var f float64
		concreteVal = reflect.ValueOf(&f).Elem()
	case *ast.BooleanLiteral:
		var b bool
		concreteVal = reflect.ValueOf(&b).Elem()
	case *ast.ArrayLiteral:
		var a []any
		concreteVal = reflect.ValueOf(&a).Elem()
	case *ast.ObjectLiteral:
		var o map[string]any
		concreteVal = reflect.ValueOf(&o).Elem()
	case *ast.NullLiteral:
		return nil // Leave interface as nil
	default:
		return fmt.Errorf("maml: cannot determine concrete type for interface{} for AST node %T", expr)
	}

	if err := m.mapValue(expr, concreteVal); err != nil {
		return err
	}
	rv.Set(concreteVal)
	return nil
}

// newValueForInterface returns a new, settable reflect.Value appropriate for holding
// the data from the given AST expression, for when the destination is an interface{}.
func (m *mapper) newValueForInterface(expr ast.Expression) reflect.Value {
	switch expr.(type) {
	case *ast.ObjectLiteral:
		return reflect.ValueOf(make(map[string]any))
	case *ast.ArrayLiteral:
		return reflect.ValueOf(make([]any, 0))
	case *ast.StringLiteral:
		return reflect.New(reflect.TypeOf("")).Elem()
	case *ast.IntegerLiteral:
		return reflect.New(reflect.TypeOf(int64(0))).Elem()
	case *ast.FloatLiteral:
		return reflect.New(reflect.TypeOf(float64(0.0))).Elem()
	case *ast.BooleanLiteral:
		return reflect.New(reflect.TypeOf(false)).Elem()
	}
	// For null, the calling function will set the interface to nil.
	return reflect.Value{}
}
