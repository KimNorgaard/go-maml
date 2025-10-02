package maml

import (
	"bytes"
	"encoding"
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync"

	"github.com/KimNorgaard/go-maml/internal/ast"
	"github.com/KimNorgaard/go-maml/internal/lexer"
	"github.com/KimNorgaard/go-maml/internal/parser"
)

// Decoder reads and decodes MAML values from an input stream.
type Decoder struct {
	r    io.Reader
	opts []Option
}

const defaultMaxDepth = 1000

// NewDecoder returns a new decoder that reads from r. It stores options
// to be applied later by the Decode method.
func NewDecoder(r io.Reader, opts ...Option) *Decoder {
	return &Decoder{r: r, opts: opts}
}

// Decode reads the next MAML-encoded value from its input
// and stores it in the value pointed to by out.
// Note: This is a non-streaming implementation. It reads the entire
// reader into memory first before parsing.
func (d *Decoder) Decode(out any) error {
	if d.r == nil {
		return fmt.Errorf("maml: Decode(nil reader)")
	}
	data, err := io.ReadAll(d.r)
	if err != nil {
		return err
	}

	l := lexer.New(data)
	p := parser.New(l)
	doc := p.Parse()

	if len(p.Errors()) > 0 {
		var errStr string
		for i, msg := range p.Errors() {
			if i > 0 {
				errStr += "\n"
			}
			errStr += msg
		}
		return fmt.Errorf("maml: parsing error: %s", errStr)
	}

	return d.decodeDocument(doc, out)
}

// decodeDocument processes the options and maps the AST to a Go value.
func (d *Decoder) decodeDocument(doc *ast.Document, v any) error {
	o := options{
		maxDepth: defaultMaxDepth,
	}
	for _, opt := range d.opts {
		if err := opt(&o); err != nil {
			return err
		}
	}

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
	ds := &decodeState{depth: o.maxDepth}
	return ds.mapValue(stmt.Expression, rv.Elem())
}

type decodeState struct {
	depth int
}

func (ds *decodeState) mapValue(expr ast.Expression, rv reflect.Value) error {
	ds.depth--
	if ds.depth <= 0 {
		return fmt.Errorf("maml: reached max recursion depth")
	}
	defer func() { ds.depth++ }()

	if _, isNull := expr.(*ast.NullLiteral); isNull {
		switch rv.Kind() {
		case reflect.Interface, reflect.Pointer, reflect.Map, reflect.Slice:
			rv.Set(reflect.Zero(rv.Type()))
			return nil
		}
	}

	// Check for custom unmarshaler implementations.
	// We check the pointer to the value, as UnmarshalMAML is often implemented on a pointer receiver.
	if rv.CanAddr() {
		pv := rv.Addr()
		if pv.CanInterface() {
			if u, ok := pv.Interface().(Unmarshaler); ok {
				// To use UnmarshalMAML, we need to re-encode the current AST node back to bytes.
				var buf bytes.Buffer
				// Use compact formatting for the re-encoding.
				compactIndent := 0
				f := newFormatter(&buf, &options{indent: &compactIndent})
				if err := f.format(expr); err != nil {
					return fmt.Errorf("maml: failed to re-marshal node for custom unmarshaler: %w", err)
				}
				if err := u.UnmarshalMAML(buf.Bytes()); err != nil {
					return &UnmarshalerError{Type: pv.Type(), Err: err}
				}
				return nil
			}

			if u, ok := pv.Interface().(encoding.TextUnmarshaler); ok {
				if s, isString := expr.(*ast.StringLiteral); isString {
					if err := u.UnmarshalText([]byte(s.Value)); err != nil {
						return &UnmarshalerError{Type: pv.Type(), Err: err}
					}
					return nil
				}
			}
		}
	}

	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			rv.Set(reflect.New(rv.Type().Elem()))
		}
		rv = rv.Elem()
	}

	if rv.Kind() == reflect.Interface {
		return ds.mapInterface(expr, rv)
	}
	if !rv.CanSet() {
		return fmt.Errorf("maml: cannot set value of type %s", rv.Type())
	}

	switch node := expr.(type) {
	case *ast.NullLiteral:
		rv.Set(reflect.Zero(rv.Type()))
		return nil
	case *ast.StringLiteral:
		return ds.mapString(node, rv)
	case *ast.IntegerLiteral:
		return ds.mapInt(node, rv)
	case *ast.FloatLiteral:
		return ds.mapFloat(node, rv)
	case *ast.BooleanLiteral:
		return ds.mapBool(node, rv)
	case *ast.ArrayLiteral:
		switch rv.Kind() {
		case reflect.Slice:
			return ds.mapSlice(node, rv)
		case reflect.Array:
			return ds.mapArray(node, rv)
		default:
			return fmt.Errorf("maml: cannot unmarshal array into Go value of type %s", rv.Type())
		}
	case *ast.ObjectLiteral:
		switch rv.Kind() {
		case reflect.Struct:
			return ds.mapStruct(node, rv)
		case reflect.Map:
			return ds.mapMap(node, rv)
		default:
			return fmt.Errorf("maml: cannot unmarshal object into Go value of type %s", rv.Type())
		}
	default:
		return fmt.Errorf("maml: mapping for AST node type %T not yet implemented", node)
	}
}

func (ds *decodeState) mapString(s *ast.StringLiteral, rv reflect.Value) error {
	if rv.Kind() != reflect.String {
		return fmt.Errorf("maml: cannot unmarshal string into Go value of type %s", rv.Type())
	}
	rv.SetString(s.Value)
	return nil
}

func (ds *decodeState) mapInt(i *ast.IntegerLiteral, rv reflect.Value) error {
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

func (ds *decodeState) mapFloat(f *ast.FloatLiteral, rv reflect.Value) error {
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

func (ds *decodeState) mapBool(b *ast.BooleanLiteral, rv reflect.Value) error {
	if rv.Kind() != reflect.Bool {
		return fmt.Errorf("maml: cannot unmarshal boolean into Go value of type %s", rv.Type())
	}
	rv.SetBool(b.Value)
	return nil
}

func (ds *decodeState) mapSlice(a *ast.ArrayLiteral, rv reflect.Value) error {
	sliceType := rv.Type()
	newSlice := reflect.MakeSlice(sliceType, len(a.Elements), len(a.Elements))
	for i, elemAST := range a.Elements {
		if err := ds.mapValue(elemAST, newSlice.Index(i)); err != nil {
			return err
		}
	}
	rv.Set(newSlice)
	return nil
}

func (ds *decodeState) mapArray(a *ast.ArrayLiteral, rv reflect.Value) error {
	if rv.Len() != len(a.Elements) {
		return fmt.Errorf("maml: cannot unmarshal array of length %d into Go array of length %d", len(a.Elements), rv.Len())
	}
	for i, elemAST := range a.Elements {
		if err := ds.mapValue(elemAST, rv.Index(i)); err != nil {
			return err
		}
	}
	return nil
}

// resolveMapKey extracts the string key from an AST node.
func resolveMapKey(keyExpr ast.Expression) (string, error) {
	switch k := keyExpr.(type) {
	case *ast.Identifier:
		return k.Value, nil
	case *ast.StringLiteral:
		return k.Value, nil
	default:
		return "", fmt.Errorf("maml: invalid key type in object literal: %T", keyExpr)
	}
}

// findField finds the target field in a struct's cached fields.
// It first attempts a case-sensitive match, then falls back to a
// case-insensitive match.
func findField(fields map[string]field, keyStr string) *field {
	// Try a direct, case-sensitive match on the tag/field name.
	if f, ok := fields[keyStr]; ok {
		return &f
	}
	// Fallback to a case-insensitive match on all fields.
	for name, f := range fields {
		if strings.EqualFold(name, keyStr) {
			return &f
		}
	}
	return nil
}

func (ds *decodeState) mapMap(obj *ast.ObjectLiteral, rv reflect.Value) error {
	mapType := rv.Type()
	if mapType.Key().Kind() != reflect.String {
		return fmt.Errorf("maml: cannot unmarshal object into map with non-string key type %s", mapType.Key())
	}
	if rv.IsNil() {
		rv.Set(reflect.MakeMap(mapType))
	} else {
		for _, k := range rv.MapKeys() {
			rv.SetMapIndex(k, reflect.Value{}) // The zero Value deletes the key
		}
	}
	elemType := mapType.Elem()
	for _, pair := range obj.Pairs {
		keyStr, err := resolveMapKey(pair.Key)
		if err != nil {
			return err
		}
		newVal := reflect.New(elemType).Elem()
		if err := ds.mapValue(pair.Value, newVal); err != nil {
			return err
		}
		rv.SetMapIndex(reflect.ValueOf(keyStr), newVal)
	}
	return nil
}

func (ds *decodeState) mapStruct(obj *ast.ObjectLiteral, rv reflect.Value) error {
	fields := cachedFields(rv.Type())
	for _, pair := range obj.Pairs {
		keyStr, err := resolveMapKey(pair.Key)
		if err != nil {
			return err
		}

		if targetField := findField(fields, keyStr); targetField != nil {
			fieldVal := rv.FieldByIndex(targetField.idx)
			if fieldVal.IsValid() && fieldVal.CanSet() {
				if err := ds.mapValue(pair.Value, fieldVal); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (ds *decodeState) mapInterface(expr ast.Expression, rv reflect.Value) error {
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
		return nil
	default:
		return fmt.Errorf("maml: cannot determine concrete type for interface{} for AST node %T", expr)
	}
	if err := ds.mapValue(expr, concreteVal); err != nil {
		return err
	}
	rv.Set(concreteVal)
	return nil
}

// A field represents a single field in a struct.
type field struct {
	idx []int
}

// fieldCache caches a map of struct field names to their properties.
var fieldCache sync.Map // map[reflect.Type]map[string]field

// cachedFields returns a map of field names to field properties for the given type.
// The result is cached to avoid repeated reflection work.
func cachedFields(t reflect.Type) map[string]field {
	if f, ok := fieldCache.Load(t); ok {
		if fields, ok := f.(map[string]field); ok {
			return fields
		}
	}

	fields := make(map[string]field)
	var walk func(t reflect.Type, idx []int)
	walk = func(t reflect.Type, idx []int) {
		for i := 0; i < t.NumField(); i++ {
			sf := t.Field(i)
			if sf.Anonymous {
				// Recurse into embedded structs.
				walk(sf.Type, append(idx, i))
				continue
			}
			if !sf.IsExported() {
				continue
			}

			tag := sf.Tag.Get("maml")
			if tag == "-" {
				continue
			}
			name := sf.Name
			if tag != "" {
				name = strings.Split(tag, ",")[0]
			}

			fields[name] = field{idx: append(idx, i)}
			// Also add the original field name for case-insensitive fallback.
			if _, ok := fields[sf.Name]; !ok {
				fields[sf.Name] = field{idx: append(idx, i)}
			}
		}
	}
	walk(t, nil)

	fieldCache.Store(t, fields)
	return fields
}
