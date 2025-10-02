package maml_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/KimNorgaard/go-maml"
	"github.com/KimNorgaard/go-maml/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestUnmarshal(t *testing.T) {
	t.Run("Scalar Types", func(t *testing.T) {
		var s string
		err := maml.Unmarshal([]byte(`"hello world"`), &s)
		require.NoError(t, err)
		require.Equal(t, "hello world", s)

		var i int
		err = maml.Unmarshal([]byte(`123`), &i)
		require.NoError(t, err)
		require.Equal(t, 123, i)

		var f float64
		err = maml.Unmarshal([]byte(`3.14`), &f)
		require.NoError(t, err)
		require.Equal(t, 3.14, f)

		var b bool
		err = maml.Unmarshal([]byte(`true`), &b)
		require.NoError(t, err)
		require.Equal(t, true, b)
	})

	t.Run("Null Handling", func(t *testing.T) {
		var s string = "preset"
		err := maml.Unmarshal([]byte(`null`), &s)
		require.NoError(t, err)
		require.Equal(t, "", s, "null should set string to its zero value")

		var i int = 123
		err = maml.Unmarshal([]byte(`null`), &i)
		require.NoError(t, err)
		require.Equal(t, 0, i, "null should set int to its zero value")

		var p *int
		err = maml.Unmarshal([]byte(`null`), &p)
		require.NoError(t, err)
		require.Nil(t, p, "null should set pointer to nil")
	})

	t.Run("Slices", func(t *testing.T) {
		var ints []int
		err := maml.Unmarshal([]byte(`[1, 2, 3]`), &ints)
		require.NoError(t, err)
		require.Equal(t, []int{1, 2, 3}, ints)

		var strings []string
		err = maml.Unmarshal([]byte(`["a", "b"]`), &strings)
		require.NoError(t, err)
		require.Equal(t, []string{"a", "b"}, strings)
	})

	t.Run("Arrays", func(t *testing.T) {
		var arr [3]int
		err := maml.Unmarshal([]byte(`[1, 2, 3]`), &arr)
		require.NoError(t, err)
		require.Equal(t, [3]int{1, 2, 3}, arr)

		var arr2 [2]int
		err = maml.Unmarshal([]byte(`[1, 2, 3]`), &arr2)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot unmarshal array of length 3 into Go array of length 2")

		var arr3 [4]int
		err = maml.Unmarshal([]byte(`[1, 2, 3]`), &arr3)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot unmarshal array of length 3 into Go array of length 4")
	})

	t.Run("Maps", func(t *testing.T) {
		var m map[string]int
		err := maml.Unmarshal([]byte(`{ a: 1, b: 2 }`), &m)
		require.NoError(t, err)
		require.Equal(t, map[string]int{"a": 1, "b": 2}, m)

		var m2 map[string]any
		err = maml.Unmarshal([]byte(`{ str: "s", int: 1, bool: true, float: 1.2 }`), &m2)
		require.NoError(t, err)
		expected := map[string]any{
			"str":   "s",
			"int":   int64(1),
			"bool":  true,
			"float": float64(1.2),
		}
		require.Equal(t, expected, m2)

		var m3 map[string]any
		err = maml.Unmarshal([]byte(`{ bareword: value }`), &m3)
		require.NoError(t, err)
		expected3 := map[string]any{
			"bareword": "value",
		}
		require.Equal(t, expected3, m3)
	})

	t.Run("Type Mismatch Errors", func(t *testing.T) {
		var i int
		err := maml.Unmarshal([]byte(`"not a number"`), &i)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot unmarshal string into Go value of type int")

		var s string
		err = maml.Unmarshal([]byte(`123`), &s)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot unmarshal integer into Go value of type string")
	})

	t.Run("Integer Overflow Error", func(t *testing.T) {
		var i8 int8
		err := maml.Unmarshal([]byte(`128`), &i8)
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
		err := maml.Unmarshal([]byte(input), &s)
		require.NoError(t, err)
		require.Equal(t, "John", s.FirstName)
		require.Equal(t, "Doe", s.LastName)
		require.Equal(t, 42, s.Age)
	})

	t.Run("Case-insensitive mapping", func(t *testing.T) {
		input := `{ firstname: "Jane", SURNAME: "Smith", aGe: 30 }`
		var s testStruct
		err := maml.Unmarshal([]byte(input), &s)
		require.NoError(t, err)
		require.Equal(t, "Jane", s.FirstName)
		require.Equal(t, "Smith", s.LastName)
		require.Equal(t, 30, s.Age)
	})

	t.Run("Pointer fields", func(t *testing.T) {
		notes := "This is a note"
		input := `{ notes: "This is a note" }`
		var s testStruct
		err := maml.Unmarshal([]byte(input), &s)
		require.NoError(t, err)
		require.NotNil(t, s.Notes)
		require.Equal(t, notes, *s.Notes)

		input2 := `{}`
		var s2 testStruct
		err = maml.Unmarshal([]byte(input2), &s2)
		require.NoError(t, err)
		require.Nil(t, s2.Notes)
	})

	t.Run("Ignored and unexported fields", func(t *testing.T) {
		input := `{ Ignored: "should not be set", unexported: "should not be set" }`
		var s testStruct
		s.unexported = "preset"
		err := maml.Unmarshal([]byte(input), &s)
		require.NoError(t, err)
		require.Equal(t, "", s.Ignored)
		require.Equal(t, "preset", s.unexported)
	})
}

func TestUnmarshalMaxDepth(t *testing.T) {
	t.Run("Object nesting", func(t *testing.T) {
		depth := 10
		input := strings.Repeat("{ key: ", depth) + "null" + strings.Repeat(" }", depth)

		var v any
		err := maml.Unmarshal([]byte(input), &v, maml.MaxDepth(depth-1))

		require.Error(t, err)
		require.Contains(t, err.Error(), "reached max recursion depth")
	})

	t.Run("Array nesting", func(t *testing.T) {
		depth := 10
		input := strings.Repeat("[", depth) + "null" + strings.Repeat("]", depth)

		var v any
		err := maml.Unmarshal([]byte(input), &v, maml.MaxDepth(depth-1))

		require.Error(t, err)
		require.Contains(t, err.Error(), "reached max recursion depth")
	})
}

func TestUnmarshalErrorCases(t *testing.T) {
	t.Run("Error on non-pointer destination", func(t *testing.T) {
		var v string
		err := maml.Unmarshal([]byte("true"), v)
		require.Error(t, err)
		require.EqualError(t, err, "maml: Unmarshal(non-pointer string or nil)")
	})

	t.Run("Error on nil pointer destination", func(t *testing.T) {
		var v *string
		err := maml.Unmarshal([]byte("true"), v)
		require.Error(t, err)
		require.EqualError(t, err, "maml: Unmarshal(non-pointer *string or nil)")
	})

	t.Run("Error on nil reader", func(t *testing.T) {
		dec := maml.NewDecoder(nil)
		var v any
		err := dec.Decode(&v)
		require.Error(t, err)
		require.EqualError(t, err, "maml: Decode(nil reader)")
	})
}

func TestUnmarshal_TypeMismatchErrors(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		target      func() any // Use a function to get a fresh pointer for each test
		expectedErr string
	}{
		{
			name:        "Object into String",
			input:       `{ key: "value" }`,
			target:      func() any { return new(string) },
			expectedErr: "maml: cannot unmarshal object into Go value of type string",
		},
		{
			name:        "Object into Int",
			input:       `{ key: "value" }`,
			target:      func() any { return new(int) },
			expectedErr: "maml: cannot unmarshal object into Go value of type int",
		},
		{
			name:        "Object into Slice",
			input:       `{ key: "value" }`,
			target:      func() any { return new([]string) },
			expectedErr: "maml: cannot unmarshal object into Go value of type []string",
		},
		{
			name:        "Array into String",
			input:       `[1, 2, 3]`,
			target:      func() any { return new(string) },
			expectedErr: "maml: cannot unmarshal array into Go value of type string",
		},
		{
			name:        "Array into Int",
			input:       `[1, 2, 3]`,
			target:      func() any { return new(int) },
			expectedErr: "maml: cannot unmarshal array into Go value of type int",
		},
		{
			name:        "Array into Map",
			input:       `[1, 2, 3]`,
			target:      func() any { return new(map[string]int) },
			expectedErr: "maml: cannot unmarshal array into Go value of type map[string]int",
		},
		{
			name:        "String into Int",
			input:       `"hello"`,
			target:      func() any { return new(int) },
			expectedErr: "maml: cannot unmarshal string into Go value of type int",
		},
		{
			name:        "Integer into String",
			input:       `123`,
			target:      func() any { return new(string) },
			expectedErr: "maml: cannot unmarshal integer into Go value of type string",
		},
		{
			name:        "Float into Int",
			input:       `123.45`,
			target:      func() any { return new(int) },
			expectedErr: "maml: cannot unmarshal float into Go value of type int",
		},
		{
			name:        "Boolean into Int",
			input:       `true`,
			target:      func() any { return new(int) },
			expectedErr: "maml: cannot unmarshal boolean into Go value of type int",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			target := tc.target()
			err := maml.Unmarshal([]byte(tc.input), target)
			require.Error(t, err)
			require.EqualError(t, err, tc.expectedErr)
		})
	}
}

func TestUnmarshal_OverflowErrors(t *testing.T) {
	t.Run("Integer Overflow", func(t *testing.T) {
		var i8 int8
		err := maml.Unmarshal([]byte("128"), &i8)
		require.Error(t, err)
		require.EqualError(t, err, "maml: integer value 128 overflows Go value of type int8")

		var i16 int16
		err = maml.Unmarshal([]byte("32768"), &i16)
		require.Error(t, err)
		require.EqualError(t, err, "maml: integer value 32768 overflows Go value of type int16")
	})

	t.Run("Float Overflow", func(t *testing.T) {
		var f32 float32
		// math.MaxFloat32 is approx 3.4e38.
		err := maml.Unmarshal([]byte("3.5e38"), &f32)
		require.Error(t, err)
		require.EqualError(t, err, "maml: float value 350000000000000001565567347835409530880.000000 overflows Go value of type float32")
	})
}

func TestUnmarshal_InvalidUTF8(t *testing.T) {
	// A MAML file must be valid UTF-8. The lexer should produce an error.
	invalidUTF8 := []byte("{ key: \"\xff\" }") // \xff is an invalid start of a UTF-8 sequence
	var v any
	err := maml.Unmarshal(invalidUTF8, &v)
	require.Error(t, err)
	require.EqualError(t, err, "maml: parsing error at line 1, column 8: illegal token encountered: invalid utf-8 sequence in string")
}

func TestUnmarshal_PropagatesSyntaxErrors(t *testing.T) {
	// These are parser tests, but we must ensure Unmarshal bubbles up the errors correctly.
	testCases := []struct {
		name        string
		input       string
		expectedErr string
	}{
		{
			name:        "Unterminated string",
			input:       `{ key: "value }`,
			expectedErr: "maml: parsing error at line 1, column 8: illegal token encountered: unterminated string",
		},
		{
			name:        "Missing colon",
			input:       `{ key "value" }`,
			expectedErr: "maml: parsing error at line 1, column 7: expected ':' after key, got STRING",
		},
		{
			name:        "Unbalanced braces",
			input:       `{ key: "value"`,
			expectedErr: "maml: parsing error at line 1, column 15: unterminated object literal, expected '}' got EOF",
		},
		{
			name:        "Invalid map key",
			input:       `{ []: "value" }`,
			expectedErr: "maml: parsing error at line 1, column 3: invalid token for object key: [ ('[')",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var v any
			err := maml.Unmarshal([]byte(tc.input), &v)
			require.Error(t, err)
			require.EqualError(t, err, tc.expectedErr)
		})
	}
}

func BenchmarkDecode(b *testing.B) {
	benchmarkMAMLInput, err := testutil.ReadTestData("large.maml")
	require.NoError(b, err)

	b.ReportAllocs()
	b.SetBytes(int64(len(benchmarkMAMLInput)))

	var v any
	r := bytes.NewReader(benchmarkMAMLInput)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		r.Seek(0, 0)
		dec := maml.NewDecoder(r)
		if err := dec.Decode(&v); err != nil {
			b.Fatalf("Decode failed during benchmark: %v", err)
		}
	}
}
