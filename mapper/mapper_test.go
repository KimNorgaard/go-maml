package mapper_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/KimNorgaard/go-maml/lexer"
	"github.com/KimNorgaard/go-maml/mapper"
	"github.com/KimNorgaard/go-maml/parser"
	"github.com/stretchr/testify/require"
)

func unmarshal(input string, v any) error {
	l := lexer.New([]byte(input))
	p := parser.New(l)
	doc := p.Parse()
	if len(p.Errors()) > 0 {
		return fmt.Errorf("parsing error: %s", p.Errors()[0])
	}
	return mapper.Map(doc, v, mapper.DefaultMaxDepth)
}

func TestUnmarshal(t *testing.T) {
	t.Run("Scalar Types", func(t *testing.T) {
		var s string
		err := unmarshal(`"hello world"`, &s)
		require.NoError(t, err)
		require.Equal(t, "hello world", s)

		var i int
		err = unmarshal(`123`, &i)
		require.NoError(t, err)
		require.Equal(t, 123, i)

		var f float64
		err = unmarshal(`3.14`, &f)
		require.NoError(t, err)
		require.Equal(t, 3.14, f)

		var b bool
		err = unmarshal(`true`, &b)
		require.NoError(t, err)
		require.Equal(t, true, b)
	})

	t.Run("Null Handling", func(t *testing.T) {
		var s string = "preset"
		err := unmarshal(`null`, &s)
		require.NoError(t, err)
		require.Equal(t, "", s, "null should set string to its zero value")

		var i int = 123
		err = unmarshal(`null`, &i)
		require.NoError(t, err)
		require.Equal(t, 0, i, "null should set int to its zero value")

		var p *int
		err = unmarshal(`null`, &p)
		require.NoError(t, err)
		require.Nil(t, p, "null should set pointer to nil")
	})

	t.Run("Slices", func(t *testing.T) {
		var ints []int
		err := unmarshal(`[1, 2, 3]`, &ints)
		require.NoError(t, err)
		require.Equal(t, []int{1, 2, 3}, ints)

		var strings []string
		err = unmarshal(`["a", "b"]`, &strings)
		require.NoError(t, err)
		require.Equal(t, []string{"a", "b"}, strings)
	})

	t.Run("Arrays", func(t *testing.T) {
		var arr [3]int
		err := unmarshal(`[1, 2, 3]`, &arr)
		require.NoError(t, err)
		require.Equal(t, [3]int{1, 2, 3}, arr)

		var arr2 [2]int
		err = unmarshal(`[1, 2, 3]`, &arr2)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot unmarshal array of length 3 into Go array of length 2")

		var arr3 [4]int
		err = unmarshal(`[1, 2, 3]`, &arr3)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot unmarshal array of length 3 into Go array of length 4")
	})

	t.Run("Maps", func(t *testing.T) {
		var m map[string]int
		err := unmarshal(`{ a: 1, b: 2 }`, &m)
		require.NoError(t, err)
		require.Equal(t, map[string]int{"a": 1, "b": 2}, m)

		var m2 map[string]any
		err = unmarshal(`{ str: "s", int: 1, bool: true, float: 1.2 }`, &m2)
		require.NoError(t, err)
		expected := map[string]any{
			"str":   "s",
			"int":   int64(1),
			"bool":  true,
			"float": float64(1.2),
		}
		require.Equal(t, expected, m2)
	})

	t.Run("Type Mismatch Errors", func(t *testing.T) {
		var i int
		err := unmarshal(`"not a number"`, &i)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot unmarshal string into Go value of type int")

		var s string
		err = unmarshal(`123`, &s)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot unmarshal integer into Go value of type string")
	})

	t.Run("Integer Overflow Error", func(t *testing.T) {
		var i8 int8
		err := unmarshal(`128`, &i8)
		require.Error(t, err)
		require.Contains(t, err.Error(), "overflows Go value of type int8")
	})
}

func TestUnmarshalStructs(t *testing.T) {
	type testStruct struct {
		FirstName  string
		LastName   string `maml:"surname"`
		Age        int
		Notes      *string `maml:"notes"`
		unexported string
		Ignored    string `maml:"-"`
	}

	t.Run("Basic struct mapping with tags", func(t *testing.T) {
		input := `{ FirstName: "John", surname: "Doe", Age: 42 }`
		var s testStruct
		err := unmarshal(input, &s)
		require.NoError(t, err)
		require.Equal(t, "John", s.FirstName)
		require.Equal(t, "Doe", s.LastName)
		require.Equal(t, 42, s.Age)
	})

	t.Run("Case-insensitive mapping", func(t *testing.T) {
		input := `{ firstname: "Jane", SURNAME: "Smith", aGe: 30 }`
		var s testStruct
		err := unmarshal(input, &s)
		require.NoError(t, err)
		require.Equal(t, "Jane", s.FirstName)
		require.Equal(t, "Smith", s.LastName)
		require.Equal(t, 30, s.Age)
	})

	t.Run("Pointer fields", func(t *testing.T) {
		notes := "This is a note"
		input := `{ notes: "This is a note" }`
		var s testStruct
		err := unmarshal(input, &s)
		require.NoError(t, err)
		require.NotNil(t, s.Notes)
		require.Equal(t, notes, *s.Notes)

		input2 := `{}`
		var s2 testStruct
		err = unmarshal(input2, &s2)
		require.NoError(t, err)
		require.Nil(t, s2.Notes)
	})

	t.Run("Ignored and unexported fields", func(t *testing.T) {
		input := `{ Ignored: "should not be set", unexported: "should not be set" }`
		var s testStruct
		s.unexported = "preset"
		err := unmarshal(input, &s)
		require.NoError(t, err)
		require.Equal(t, "", s.Ignored)
		require.Equal(t, "preset", s.unexported)
	})
}

func TestUnmarshalMaxDepth(t *testing.T) {
	t.Run("Object nesting", func(t *testing.T) {
		depth := mapper.DefaultMaxDepth + 1
		input := strings.Repeat("{ key: ", depth) + "null" + strings.Repeat(" }", depth)

		var v any
		err := unmarshal(input, &v)

		require.Error(t, err)
		require.Contains(t, err.Error(), "reached max recursion depth")
	})

	t.Run("Array nesting", func(t *testing.T) {
		depth := mapper.DefaultMaxDepth + 1
		input := strings.Repeat("[", depth) + "null" + strings.Repeat("]", depth)

		var v any
		err := unmarshal(input, &v)

		require.Error(t, err)
		require.Contains(t, err.Error(), "reached max recursion depth")
	})
}

func TestUnmarshalErrorCases(t *testing.T) {
	doc := parser.New(lexer.New([]byte("true"))).Parse()

	t.Run("Error on non-pointer destination", func(t *testing.T) {
		var v string
		err := mapper.Map(doc, v, mapper.DefaultMaxDepth)
		require.Error(t, err)
		require.EqualError(t, err, "maml: Unmarshal(non-pointer string or nil)")
	})

	t.Run("Error on nil pointer destination", func(t *testing.T) {
		var v *string
		err := mapper.Map(doc, v, mapper.DefaultMaxDepth)
		require.Error(t, err)
		require.EqualError(t, err, "maml: Unmarshal(non-pointer *string or nil)")
	})
}
