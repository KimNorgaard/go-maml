package marshaler_test

import (
	"math"
	"testing"

	"github.com/KimNorgaard/go-maml/internal/ast"
	"github.com/KimNorgaard/go-maml/internal/marshaler"
	"github.com/stretchr/testify/require"
)

func TestMarshal_Scalars(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		node, err := marshaler.Marshal(nil)
		require.NoError(t, err)
		require.IsType(t, &ast.NullLiteral{}, node)
	})

	t.Run("String", func(t *testing.T) {
		node, err := marshaler.Marshal("hello")
		require.NoError(t, err)
		lit, ok := node.(*ast.StringLiteral)
		require.True(t, ok)
		require.Equal(t, "hello", lit.Value)
	})

	t.Run("Integer", func(t *testing.T) {
		node, err := marshaler.Marshal(123)
		require.NoError(t, err)
		lit, ok := node.(*ast.IntegerLiteral)
		require.True(t, ok)
		require.Equal(t, int64(123), lit.Value)
	})

	t.Run("Float", func(t *testing.T) {
		node, err := marshaler.Marshal(3.14)
		require.NoError(t, err)
		lit, ok := node.(*ast.FloatLiteral)
		require.True(t, ok)
		require.Equal(t, 3.14, lit.Value)
	})

	t.Run("Boolean", func(t *testing.T) {
		node, err := marshaler.Marshal(true)
		require.NoError(t, err)
		lit, ok := node.(*ast.BooleanLiteral)
		require.True(t, ok)
		require.Equal(t, true, lit.Value)
	})

	t.Run("Uint64 Overflow", func(t *testing.T) {
		input := uint64(math.MaxInt64 + 1)
		_, err := marshaler.Marshal(input)
		require.Error(t, err)
		require.Contains(t, err.Error(), "overflows int64")
	})
}

func TestMarshal_SlicesAndArrays(t *testing.T) {
	t.Run("Slice of integers", func(t *testing.T) {
		input := []int{1, 2, 3}
		node, err := marshaler.Marshal(input)
		require.NoError(t, err)

		arr, ok := node.(*ast.ArrayLiteral)
		require.True(t, ok)
		require.Len(t, arr.Elements, 3)

		for i, v := range []int64{1, 2, 3} {
			elem, ok := arr.Elements[i].(*ast.IntegerLiteral)
			require.True(t, ok)
			require.Equal(t, v, elem.Value)
		}
	})

	t.Run("Array of strings", func(t *testing.T) {
		input := [2]string{"a", "b"}
		node, err := marshaler.Marshal(input)
		require.NoError(t, err)

		arr, ok := node.(*ast.ArrayLiteral)
		require.True(t, ok)
		require.Len(t, arr.Elements, 2)

		for i, v := range []string{"a", "b"} {
			elem, ok := arr.Elements[i].(*ast.StringLiteral)
			require.True(t, ok)
			require.Equal(t, v, elem.Value)
		}
	})

	t.Run("Nil slice", func(t *testing.T) {
		var input []int
		node, err := marshaler.Marshal(input)
		require.NoError(t, err)
		require.IsType(t, &ast.NullLiteral{}, node)
	})

	t.Run("Empty slice", func(t *testing.T) {
		input := []int{}
		node, err := marshaler.Marshal(input)
		require.NoError(t, err)

		arr, ok := node.(*ast.ArrayLiteral)
		require.True(t, ok)
		require.Len(t, arr.Elements, 0)
	})
}

func TestMarshal_Maps(t *testing.T) {
	t.Run("Map of string to int", func(t *testing.T) {
		input := map[string]int{"a": 1, "b": 2}
		node, err := marshaler.Marshal(input)
		require.NoError(t, err)

		obj, ok := node.(*ast.ObjectLiteral)
		require.True(t, ok)
		require.Len(t, obj.Pairs, 2)

		expected := map[string]int64{"a": 1, "b": 2}
		found := make(map[string]int64)

		for _, pair := range obj.Pairs {
			key, ok := pair.Key.(*ast.Identifier)
			require.True(t, ok)
			val, ok := pair.Value.(*ast.IntegerLiteral)
			require.True(t, ok)
			found[key.Value] = val.Value
		}
		require.Equal(t, expected, found)
	})

	t.Run("Nil map", func(t *testing.T) {
		var input map[string]any
		node, err := marshaler.Marshal(input)
		require.NoError(t, err)
		require.IsType(t, &ast.NullLiteral{}, node)
	})

	t.Run("Non-string key error", func(t *testing.T) {
		input := map[int]string{1: "a"}
		_, err := marshaler.Marshal(input)
		require.Error(t, err)
		require.Contains(t, err.Error(), "map key type must be a string")
	})
}

func TestMarshal_Structs(t *testing.T) {
	type testStruct struct {
		FirstName  string
		LastName   string `maml:"surname"`
		Age        int
		unexported bool
		Ignored    string `maml:"-"`
		Notes      *string
	}

	t.Run("Basic struct", func(t *testing.T) {
		notes := "some notes"
		input := testStruct{
			FirstName:  "John",
			LastName:   "Doe",
			Age:        42,
			unexported: true,
			Ignored:    "should be ignored",
			Notes:      &notes,
		}

		node, err := marshaler.Marshal(input)
		require.NoError(t, err)

		obj, ok := node.(*ast.ObjectLiteral)
		require.True(t, ok)
		require.Len(t, obj.Pairs, 4) // FirstName, surname, Age, Notes

		expectedValues := map[string]any{
			"FirstName": "John",
			"surname":   "Doe",
			"Age":       int64(42),
			"Notes":     "some notes",
		}
		foundValues := make(map[string]any)

		for _, pair := range obj.Pairs {
			key := pair.Key.String()
			switch v := pair.Value.(type) {
			case *ast.StringLiteral:
				foundValues[key] = v.Value
			case *ast.IntegerLiteral:
				foundValues[key] = v.Value
			}
		}
		require.Equal(t, expectedValues, foundValues)
	})

	t.Run("Struct with nil pointer field", func(t *testing.T) {
		input := testStruct{FirstName: "Jane"} // Notes is nil
		node, err := marshaler.Marshal(input)
		require.NoError(t, err)

		obj, ok := node.(*ast.ObjectLiteral)
		require.True(t, ok)

		for _, pair := range obj.Pairs {
			if pair.Key.String() == "Notes" {
				require.IsType(t, &ast.NullLiteral{}, pair.Value)
			}
		}
	})
}

func TestMarshal_Pointers(t *testing.T) {
	t.Run("Pointer to string", func(t *testing.T) {
		s := "hello"
		ps := &s
		node, err := marshaler.Marshal(ps)
		require.NoError(t, err)
		lit, ok := node.(*ast.StringLiteral)
		require.True(t, ok)
		require.Equal(t, "hello", lit.Value)
	})

	t.Run("Nil pointer", func(t *testing.T) {
		var ps *string
		node, err := marshaler.Marshal(ps)
		require.NoError(t, err)
		require.IsType(t, &ast.NullLiteral{}, node)
	})

	t.Run("Pointer to struct", func(t *testing.T) {
		type simple struct{ A int }
		v := &simple{A: 10}
		node, err := marshaler.Marshal(v)
		require.NoError(t, err)

		obj, ok := node.(*ast.ObjectLiteral)
		require.True(t, ok)
		require.Len(t, obj.Pairs, 1)

		pair := obj.Pairs[0]
		require.Equal(t, "A", pair.Key.String())
		val, ok := pair.Value.(*ast.IntegerLiteral)
		require.True(t, ok)
		require.Equal(t, int64(10), val.Value)
	})
}
