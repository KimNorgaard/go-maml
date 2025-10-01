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

// mapValue is the main recursive mapping function.
func (m *mapper) mapValue(expr ast.Expression, rv reflect.Value) error {
	m.depth--
	if m.depth <= 0 {
		return fmt.Errorf("maml: reached max recursion depth")
	}
	defer func() { m.depth++ }()

	// Special handling for null before any other logic.
	if _, isNull := expr.(*ast.NullLiteral); isNull {
		switch rv.Kind() {
		case reflect.Interface, reflect.Pointer, reflect.Map, reflect.Slice:
			rv.Set(reflect.Zero(rv.Type())) // Sets pointer/interface/slice/map to nil
			return nil
		default:
			rv.Set(reflect.Zero(rv.Type())) // Sets scalar types to their zero value
			return nil
		}
	}

	// Indirect through pointers to find the concrete value.
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			rv.Set(reflect.New(rv.Type().Elem()))
		}
		rv = rv.Elem()
	}

	// Handle generic interfaces by creating a default concrete type.
	if rv.Kind() == reflect.Interface {
		return m.mapInterface(expr, rv)
	}

	if !rv.CanSet() {
		return fmt.Errorf("maml: cannot set value of type %s", rv.Type())
	}

	switch node := expr.(type) {
	case *ast.StringLiteral:
		return m.mapString(node, rv)
	case *ast.IntegerLiteral:
		return m.mapInt(node, rv)
	case *ast.FloatLiteral:
		return m.mapFloat(node, rv)
	case *ast.BooleanLiteral:
		return m.mapBool(node, rv)
	case *ast.ArrayLiteral:
		switch rv.Kind() {
		case reflect.Slice:
			return m.mapSlice(node, rv)
		case reflect.Array:
			return m.mapArray(node, rv)
		default:
			return fmt.Errorf("maml: cannot unmarshal array into Go value of type %s", rv.Type())
		}
	case *ast.ObjectLiteral:
		switch rv.Kind() {
		case reflect.Struct:
			return m.mapStruct(node, rv)
		case reflect.Map:
			return m.mapMap(node, rv)
		default:
			return fmt.Errorf("maml: cannot unmarshal object into Go value of type %s", rv.Type())
		}
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

func (m *mapper) mapSlice(a *ast.ArrayLiteral, rv reflect.Value) error {
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

func (m *mapper) mapArray(a *ast.ArrayLiteral, rv reflect.Value) error {
	if rv.Len() != len(a.Elements) {
		return fmt.Errorf("maml: cannot unmarshal array of length %d into Go array of length %d", len(a.Elements), rv.Len())
	}
	for i, elemAST := range a.Elements {
		if err := m.mapValue(elemAST, rv.Index(i)); err != nil {
			return err
		}
	}
	return nil
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

	// Create a map for simple, case-insensitive lookup.
	lowerCaseMap := make(map[string]field)
	for name, f := range fields {
		lowerCaseMap[strings.ToLower(name)] = f
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

		keyLower := strings.ToLower(keyStr)
		if f, ok := lowerCaseMap[keyLower]; ok {
			fieldVal := rv.FieldByIndex(f.idx)
			if fieldVal.IsValid() && fieldVal.CanSet() {
				if err := m.mapValue(pair.Value, fieldVal); err != nil {
					return err
				}
			}
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
	default:
		return fmt.Errorf("maml: cannot determine concrete type for interface{} for AST node %T", expr)
	}
	if err := m.mapValue(expr, concreteVal); err != nil {
		return err
	}
	rv.Set(concreteVal)
	return nil
}
