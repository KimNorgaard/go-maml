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

// NewDecoder returns a new decoder that reads from r.
//
// The decoder may buffer data from r as necessary. It is the caller's
// responsibility to call Close on r if required.
//
// Functional options can be provided to configure the decoding process,
// such as setting a maximum decoding depth with the MaxDepth option.
func NewDecoder(r io.Reader, opts ...Option) *Decoder {
	return &Decoder{r: r, opts: opts}
}

// Decode reads the next MAML-encoded value from its input and stores it in
// the value pointed to by out. If out is nil or not a pointer, Decode returns
// an error.
//
// See the documentation for Unmarshal for details about the conversion of MAML
// into a Go value.
//
// If the input contains syntax errors, Decode will return a ParseErrors value.
func (d *Decoder) Decode(out any) error {
	if d.r == nil {
		return fmt.Errorf("maml: Decode(nil reader)")
	}

	o := options{}
	for _, opt := range d.opts {
		if err := opt(&o); err != nil {
			return err
		}
	}

	l := lexer.New(d.r)
	parseOpts := []parser.Option{}
	if o.parseComments {
		parseOpts = append(parseOpts, parser.WithParseComments())
	}
	p := parser.New(l, parseOpts...)

	doc := p.Parse()

	if len(p.Errors()) > 0 {
		return p.Errors()
	}

	return d.decodeDocument(doc, out, &o)
}

// decodeDocument processes the options and maps the AST to a Go value.
func (d *Decoder) decodeDocument(doc *ast.Document, v any, o *options) error {
	// If the target is an *ast.Document, just assign it.
	if docPtr, ok := v.(**ast.Document); ok {
		*docPtr = doc
		return nil
	}

	if o.maxDepth == 0 {
		o.maxDepth = defaultMaxDepth
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
	ds := &decodeState{depth: o.maxDepth, opts: o}
	return ds.mapValue(stmt.Expression, rv.Elem())
}

type decodeState struct {
	depth int
	opts  *options
}

func (ds *decodeState) mapValue(expr ast.Expression, rv reflect.Value) error { //nolint:gocyclo,funlen
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

	// Attempt to use a custom unmarshaler if available.
	handled, err := ds.tryCustomUnmarshal(expr, rv)
	if err != nil {
		return err
	}
	if handled {
		return nil
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
	case *ast.Identifier:
		return ds.mapIdentifier(node, rv)
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

// tryCustomUnmarshal attempts to use a custom unmarshaler (maml.Unmarshaler or
// encoding.TextUnmarshaler) on the given reflect.Value. It returns true if a
// custom unmarshaler was found and used, in which case the caller should not
// proceed with default unmarshaling.
func (ds *decodeState) tryCustomUnmarshal(expr ast.Expression, rv reflect.Value) (bool, error) {
	if !rv.CanAddr() {
		return false, nil
	}
	pv := rv.Addr()
	if !pv.CanInterface() {
		return false, nil
	}

	// Check for maml.Unmarshaler
	if u, ok := pv.Interface().(Unmarshaler); ok {
		var buf bytes.Buffer
		compactIndent := 0
		f := newFormatter(&buf, &options{indent: &compactIndent})
		if err := f.format(expr); err != nil {
			return true, fmt.Errorf("maml: failed to re-marshal node for custom unmarshaler: %w", err)
		}
		if err := u.UnmarshalMAML(buf.Bytes()); err != nil {
			return true, &UnmarshalerError{Type: pv.Type(), Err: err}
		}
		return true, nil
	}

	// Check for encoding.TextUnmarshaler
	if u, ok := pv.Interface().(encoding.TextUnmarshaler); ok {
		s, isString := expr.(*ast.StringLiteral)
		if !isString {
			// TextUnmarshaler can only be used on string values.
			return false, nil
		}
		if err := u.UnmarshalText([]byte(s.Value)); err != nil {
			return true, &UnmarshalerError{Type: pv.Type(), Err: err}
		}
		return true, nil
	}

	return false, nil
}

func (ds *decodeState) mapString(s *ast.StringLiteral, rv reflect.Value) error {
	if rv.Kind() != reflect.String {
		return fmt.Errorf("maml: cannot unmarshal string into Go value of type %s", rv.Type())
	}
	rv.SetString(s.Value)
	return nil
}

func (ds *decodeState) mapIdentifier(i *ast.Identifier, rv reflect.Value) error {
	if rv.Kind() != reflect.String {
		return fmt.Errorf("maml: cannot unmarshal identifier into Go value of type %s", rv.Type())
	}
	rv.SetString(i.Value)
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

	// Fallback to a case-insensitive match pre-calculated in the cache.
	if f, ok := fields[strings.ToLower(keyStr)]; ok {
		return &f
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

// resolveFieldPath traverses the given field index path `idx` starting from `rv`.
// It initializes any nil embedded pointers encountered along the path.
// Returns the reflect.Value of the final field at the end of the path.
func (ds *decodeState) resolveFieldPath(rv reflect.Value, idx []int) (reflect.Value, error) {
	currentVal := rv
	for i, fieldIndex := range idx {
		// Handle pointer indirection and nil pointer initialization
		for currentVal.Kind() == reflect.Pointer {
			if currentVal.IsNil() {
				if !currentVal.CanSet() {
					return reflect.Value{}, fmt.Errorf("maml: cannot set nil embedded pointer in path %v", idx[:i+1])
				}
				currentVal.Set(reflect.New(currentVal.Type().Elem()))
			}
			currentVal = currentVal.Elem()
		}

		if currentVal.Kind() != reflect.Struct {
			// This should ideally not happen if cachedFields correctly processes struct fields
			// The idx path should only lead through structs or pointers to structs until the final field.
			return reflect.Value{}, fmt.Errorf("maml: expected struct at path segment %v, got %s", idx[:i], currentVal.Kind())
		}
		currentVal = currentVal.Field(fieldIndex)
	}
	return currentVal, nil
}

func (ds *decodeState) mapStruct(obj *ast.ObjectLiteral, rv reflect.Value) error {
	fields := cachedFields(rv.Type())
	seenFields := make(map[string]struct{})

	for _, pair := range obj.Pairs {
		keyStr, err := resolveMapKey(pair.Key)
		if err != nil {
			return err
		}

		if targetField := findField(fields, keyStr); targetField != nil {
			finalFieldVal, err := ds.resolveFieldPath(rv, targetField.idx)
			if err != nil {
				return err
			}

			if finalFieldVal.IsValid() && finalFieldVal.CanSet() {
				if err := ds.mapValue(pair.Value, finalFieldVal); err != nil {
					return err
				}
				seenFields[keyStr] = struct{}{}
			}
		}
	}

	// Check for unknown fields if disallowUnknownFields is enabled
	if ds.opts.disallowUnknownFields {
		if err := ds.checkUnknownFields(obj, rv.Type(), seenFields); err != nil {
			return err
		}
	}

	return nil
}

// checkUnknownFields iterates through the object literal's pairs and
// returns an error if any field was not found in the seenFields map,
// indicating an unknown field.
func (ds *decodeState) checkUnknownFields(obj *ast.ObjectLiteral, structType reflect.Type, seenFields map[string]struct{}) error {
	for _, pair := range obj.Pairs {
		keyStr, err := resolveMapKey(pair.Key)
		if err != nil {
			// This should ideally not happen as resolveMapKey is called earlier
			return err
		}
		if _, ok := seenFields[keyStr]; !ok {
			return fmt.Errorf("maml: unknown field %q in type %s", keyStr, structType)
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
	case *ast.Identifier:
		var s string
		concreteVal = reflect.ValueOf(&s).Elem()
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
func cachedFields(t reflect.Type) map[string]field { //nolint:gocognit
	if f, ok := fieldCache.Load(t); ok {
		if fields, ok := f.(map[string]field); ok {
			return fields
		}
	}

	// fieldEntry stores information about a field found during traversal,
	// including its depth for precedence resolution.
	type fieldEntry struct {
		f             field
		name          string // The actual name (tag or field name)
		depth         int    // Depth of embedding (0 for top-level)
		originalField reflect.StructField
	}

	var collectedEntries []fieldEntry

	var walkAndCollect func(currentType reflect.Type, currentIdx []int, currentDepth int)
	walkAndCollect = func(currentType reflect.Type, currentIdx []int, currentDepth int) {
		for i := 0; i < currentType.NumField(); i++ {
			sf := currentType.Field(i)
			// Create a new slice for fieldIdx to avoid appendAssign issues and ensure
			// `currentIdx` is not modified by recursive calls using the same underlying array.
			fieldIdx := make([]int, len(currentIdx)+1)
			copy(fieldIdx, currentIdx)
			fieldIdx[len(currentIdx)] = i

			// Dereference embedded pointer types for recursion,
			// but `fieldIdx` still points to the pointer if it was a pointer embed.
			fieldType := sf.Type
			if fieldType.Kind() == reflect.Pointer {
				fieldType = fieldType.Elem()
			}

			if sf.Anonymous && fieldType.Kind() == reflect.Struct {
				// Recurse into embedded structs, increment depth
				walkAndCollect(fieldType, fieldIdx, currentDepth+1)
				continue
			}

			// Skip unexported fields
			if !sf.IsExported() {
				continue
			}

			tag := sf.Tag.Get("maml")
			// Skip fields with `maml:"-"` tag
			if tag == "-" {
				continue
			}

			actualField := field{idx: fieldIdx}
			tagName := strings.Split(tag, ",")[0]

			// Add entries for the tag name (if present) and the field name.
			if tagName != "" {
				collectedEntries = append(collectedEntries, fieldEntry{f: actualField, name: tagName, depth: currentDepth, originalField: sf})
			}
			collectedEntries = append(collectedEntries, fieldEntry{f: actualField, name: sf.Name, depth: currentDepth, originalField: sf})
		}
	}

	walkAndCollect(t, nil, 0) // Start walking from the top-level type at depth 0

	// Now, filter `collectedEntries` to apply precedence rules.
	// Fields at a shallower depth take precedence. If depths are equal,
	// the field declared earlier (in the Go struct definition) takes precedence.
	// `collectedEntries` implicitly preserves declaration order for fields at the same depth,
	// as `append` maintains order and `walkAndCollect` processes fields in declaration order.
	precedenceMap := make(map[string]fieldEntry)

	for _, entry := range collectedEntries {
		if existing, ok := precedenceMap[entry.name]; !ok {
			// First time seeing this name, add it.
			precedenceMap[entry.name] = entry
		} else if entry.depth < existing.depth {
			// Found a field with shallower depth for the same name, replace.
			precedenceMap[entry.name] = entry
		}
		// If entry.depth >= existing.depth, the existing field takes precedence
		// (either shallower, or same depth but declared earlier due to traversal order).
	}

	finalFields := make(map[string]field)

	// Populate finalFields, handling case-insensitive fallback as per original logic.
	// For case-insensitive, if a case-sensitive match already exists (from precedenceMap),
	// we do not overwrite it with a new lowercase entry.
	for name, entry := range precedenceMap {
		// Add the case-sensitive name first (or the chosen name from precedenceMap).
		finalFields[name] = entry.f

		// Now, consider the lowercase version for case-insensitive fallback.
		lowerName := strings.ToLower(name)
		if _, ok := finalFields[lowerName]; !ok {
			// Only add the lowercase version if it doesn't already exist.
			// This means if "Name" was chosen (e.g. from a tag or field name),
			// and "name" (lowercase of "Name") is used for lookup, it should map to the same field.
			// This also respects if another field "name" (case-sensitive) was chosen.
			finalFields[lowerName] = entry.f
		}
	}

	fieldCache.Store(t, finalFields)
	return finalFields
}
