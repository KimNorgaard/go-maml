package mapper

import (
	"reflect"
	"strings"
	"sync"
)

// field represents a cached struct field.
type field struct {
	name      string
	idx       []int
	tagged    bool
	omitEmpty bool
}

// fieldCache caches a map of field names to their properties for a given struct type.
var fieldCache sync.Map

// cachedFields uses reflection to parse a struct's tags and build a cache
// of its fields. This is a performance optimization to avoid re-parsing tags
// for the same struct type on every unmarshal operation.
// It skips unexported fields and fields tagged with "maml:-"
func cachedFields(t reflect.Type) map[string]field {
	if f, ok := fieldCache.Load(t); ok {
		return f.(map[string]field)
	}

	fields := make(map[string]field)
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if sf.Anonymous {
			// TODO: Handle embedded structs if desired in the future.
			continue
		}
		if !sf.IsExported() {
			continue
		}

		tag := sf.Tag.Get("maml")
		if tag == "-" {
			continue
		}

		f := field{idx: sf.Index}
		name, opts, _ := strings.Cut(tag, ",")
		if name != "" {
			f.name = name
			f.tagged = true
		} else {
			f.name = sf.Name
		}

		for opts != "" {
			var opt string
			opt, opts, _ = strings.Cut(opts, ",")
			if opt == "omitempty" {
				f.omitEmpty = true
			}
		}
		fields[f.name] = f
	}

	fieldCache.Store(t, fields)
	return fields
}
