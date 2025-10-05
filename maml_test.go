package maml_test

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/KimNorgaard/go-maml"
	"github.com/stretchr/testify/require"
)

func TestMarshal_IndentOption(t *testing.T) {
	type testStruct struct {
		Name string
		Data []int
	}
	v := testStruct{
		Name: "Test",
		Data: []int{1, 2},
	}

	t.Run("Default indentation (2 spaces)", func(t *testing.T) {
		// Note: The order of fields in a marshaled struct is not guaranteed.
		// We test for containment of parts instead of exact string equality.
		b, err := maml.Marshal(v)
		require.NoError(t, err)
		s := string(b)

		require.Contains(t, s, `Name: "Test"`)
		require.Contains(t, s, "Data: [\n    1\n    2\n  ]")
		require.True(t, strings.HasPrefix(s, "{"), "should start with {")
		require.True(t, strings.HasSuffix(s, "}"), "should end with }")
	})

	t.Run("Compact output with Indent(0)", func(t *testing.T) {
		// With compact output, we can't guarantee field order.
		b, err := maml.Marshal(v, maml.Indent(0))
		require.NoError(t, err)
		s := string(b)

		require.Contains(t, s, `Name:"Test"`)
		require.Contains(t, s, `Data:[1,2]`)
	})

	t.Run("Custom indentation with Indent(4)", func(t *testing.T) {
		b, err := maml.Marshal(v, maml.Indent(4))
		require.NoError(t, err)
		s := string(b)

		require.Contains(t, s, `Name: "Test"`)
		require.Contains(t, s, "Data: [\n        1\n        2\n    ]")
		require.True(t, strings.HasPrefix(s, "{"), "should start with {")
		require.True(t, strings.HasSuffix(s, "}"), "should end with }")
	})

	t.Run("Invalid Indent option", func(t *testing.T) {
		_, err := maml.Marshal(v, maml.Indent(-1))
		require.Error(t, err)
		require.Contains(t, err.Error(), "indent spaces cannot be negative")
	})
}

// Helper types for custom marshaler tests
type CustomValue struct {
	Value int
}

func (c CustomValue) MarshalMAML() ([]byte, error) {
	// Note: Produces a JSON-style string key
	return []byte(`{ "custom_value": ` + strconv.Itoa(c.Value) + ` }`), nil
}

type CustomPointer struct {
	Data string
}

func (c *CustomPointer) MarshalMAML() ([]byte, error) {
	return []byte(`"` + c.Data + ` (custom)"`), nil
}

type CustomError struct{}

func (c CustomError) MarshalMAML() ([]byte, error) {
	return nil, errors.New("custom error")
}

type CustomInvalidMAML struct{}

func (c CustomInvalidMAML) MarshalMAML() ([]byte, error) {
	return []byte(`{ key: "unterminated string }`), nil
}

type CustomEmpty struct{}

func (c CustomEmpty) MarshalMAML() ([]byte, error) {
	return []byte(""), nil
}

func TestMarshal_CustomMarshaler(t *testing.T) {
	t.Run("Marshaler on value", func(t *testing.T) {
		v := CustomValue{Value: 123}
		b, err := maml.Marshal(v, maml.Indent(0))
		require.NoError(t, err)
		require.Equal(t, `{"custom_value":123}`, string(b))
	})

	t.Run("Marshaler on pointer", func(t *testing.T) {
		v := &CustomPointer{Data: "hello"}
		b, err := maml.Marshal(v)
		require.NoError(t, err)
		require.Equal(t, `"hello (custom)"`, string(b))
	})

	t.Run("Marshaler on pointer for a non-pointer value", func(t *testing.T) {
		v := CustomPointer{Data: "world"}
		b, err := maml.Marshal(v)
		require.NoError(t, err)
		require.Equal(t, `"world (custom)"`, string(b))
	})

	t.Run("Marshaler that returns an error", func(t *testing.T) {
		v := CustomError{}
		_, err := maml.Marshal(v)
		require.Error(t, err)
		require.Contains(t, err.Error(), "custom error")
	})

	t.Run("Marshaler that returns invalid MAML", func(t *testing.T) {
		v := CustomInvalidMAML{}
		_, err := maml.Marshal(v)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid MAML output")
	})

	t.Run("Marshaler that returns empty bytes", func(t *testing.T) {
		v := CustomEmpty{}
		b, err := maml.Marshal(v)
		require.NoError(t, err)
		require.Equal(t, "null", string(b))
	})
}

// Helper types for cycle detection tests
type NodeA struct {
	B *NodeB
}
type NodeB struct {
	A *NodeA
}

func TestMarshal_CycleDetection(t *testing.T) {
	t.Run("Pointer to self", func(t *testing.T) {
		type Node struct {
			Next *Node
		}
		var n Node
		n.Next = &n
		_, err := maml.Marshal(n)
		require.Error(t, err)
		require.Contains(t, err.Error(), "encountered a cycle")
	})

	t.Run("Map containing self", func(t *testing.T) {
		m := make(map[string]any)
		m["self"] = m
		_, err := maml.Marshal(m)
		require.Error(t, err)
		require.Contains(t, err.Error(), "encountered a cycle")
	})

	t.Run("Slice containing self", func(t *testing.T) {
		s := make([]any, 1)
		s[0] = s
		_, err := maml.Marshal(s)
		require.Error(t, err)
		require.Contains(t, err.Error(), "encountered a cycle")
	})

	t.Run("Struct with two fields pointing to the same object (valid)", func(t *testing.T) {
		type Child struct{ Name string }
		type Parent struct {
			A *Child
			B *Child
		}
		c := &Child{Name: "Shared"}
		p := Parent{A: c, B: c}
		b, err := maml.Marshal(p, maml.Indent(0))
		require.NoError(t, err)
		// Field order not guaranteed
		require.Contains(t, string(b), `A:{Name:"Shared"}`)
		require.Contains(t, string(b), `B:{Name:"Shared"}`)
	})

	t.Run("Indirect cycle through multiple structs", func(t *testing.T) {
		a := &NodeA{}
		b := &NodeB{A: a}
		a.B = b
		_, err := maml.Marshal(a)
		require.Error(t, err)
		require.Contains(t, err.Error(), "encountered a cycle")
	})
}

func TestMarshalUnmarshal_CollectionEdgeCases(t *testing.T) {
	t.Run("Marshal nil slice", func(t *testing.T) {
		var s []int // nil slice
		b, err := maml.Marshal(s)
		require.NoError(t, err)
		require.Equal(t, "null", string(b))
	})

	t.Run("Marshal empty slice", func(t *testing.T) {
		s := []int{} // empty but not nil
		b, err := maml.Marshal(s, maml.Indent(0))
		require.NoError(t, err)
		require.Equal(t, "[]", string(b))
	})

	t.Run("Marshal nil map", func(t *testing.T) {
		var m map[string]int // nil map
		b, err := maml.Marshal(m)
		require.NoError(t, err)
		require.Equal(t, "null", string(b))
	})

	t.Run("Marshal empty map", func(t *testing.T) {
		m := map[string]int{} // empty but not nil
		b, err := maml.Marshal(m, maml.Indent(0))
		require.NoError(t, err)
		require.Equal(t, "{}", string(b))
	})

	t.Run("Unmarshal null into slice", func(t *testing.T) {
		var s []int = []int{1, 2, 3} // pre-populate to ensure it's cleared
		err := maml.Unmarshal([]byte("null"), &s)
		require.NoError(t, err)
		require.Nil(t, s, "unmarshaling null into a slice should make it nil")
	})

	t.Run("Unmarshal null into map", func(t *testing.T) {
		var m map[string]int = map[string]int{"a": 1} // pre-populate
		err := maml.Unmarshal([]byte("null"), &m)
		require.NoError(t, err)
		require.Nil(t, m, "unmarshaling null into a map should make it nil")
	})

	t.Run("Unmarshal empty array into slice", func(t *testing.T) {
		var s []int = []int{1, 2, 3} // pre-populate to ensure it's overwritten
		err := maml.Unmarshal([]byte("[]"), &s)
		require.NoError(t, err)
		require.NotNil(t, s, "unmarshaling an empty array should produce a non-nil slice")
		require.Len(t, s, 0, "unmarshaling an empty array should produce a zero-length slice")
	})

	t.Run("Unmarshal empty object into map", func(t *testing.T) {
		var m map[string]int = map[string]int{"a": 1} // pre-populate
		err := maml.Unmarshal([]byte("{}"), &m)
		require.NoError(t, err)
		require.NotNil(t, m, "unmarshaling an empty object should produce a non-nil map")
		require.Len(t, m, 0, "unmarshaling an empty object should produce a zero-length map")
	})
}

// TestMarshal_OmitEmpty tests the functionality of the ",omitempty" struct tag.
func TestMarshal_OmitEmpty(t *testing.T) {
	// Struct where all exportable fields are tagged with omitempty.
	type OmitStruct struct {
		String     string         `maml:"string,omitempty"`
		Int        int            `maml:"int,omitempty"`
		Float      float64        `maml:"float,omitempty"`
		Bool       bool           `maml:"bool,omitempty"`
		Slice      []string       `maml:"slice,omitempty"`
		Map        map[string]int `maml:"map,omitempty"`
		Pointer    *int           `maml:"pointer,omitempty"`
		Struct     *OmitStruct    `maml:"struct,omitempty"`
		unexported string         // Unexported fields are always ignored.
	}

	t.Run("All fields are zero-valued and should be omitted", func(t *testing.T) {
		v := OmitStruct{unexported: "should be ignored"}
		b, err := maml.Marshal(v, maml.Indent(0))
		require.NoError(t, err)
		// Expect an empty object because all exported fields are zero and tagged with omitempty.
		require.Equal(t, "{}", string(b))
	})

	t.Run("All fields have non-zero values and should be included", func(t *testing.T) {
		pointerVal := 123
		v := OmitStruct{
			String:  "hello",
			Int:     1,
			Float:   3.14,
			Bool:    true, // Bool is tricky, false is the zero value
			Slice:   []string{"a"},
			Map:     map[string]int{"b": 2},
			Pointer: &pointerVal,
			Struct:  &OmitStruct{String: "nested"},
		}
		b, err := maml.Marshal(v, maml.Indent(0))
		require.NoError(t, err)
		s := string(b)

		// Check that all fields are present. Field order isn't guaranteed.
		// Check that all fields are present. Field order isn't guaranteed.
		require.Contains(t, s, `string:"hello"`)
		require.Contains(t, s, `int:1`)
		require.Contains(t, s, `float:3.14`)
		require.Contains(t, s, `bool:true`)
		require.Contains(t, s, `slice:["a"]`)
		require.Contains(t, s, `map:{b:2}`)
		require.Contains(t, s, `pointer:123`)
		require.Contains(t, s, `struct:{string:"nested"}`)
	})

	t.Run("Bool field with false value (zero) should be omitted", func(t *testing.T) {
		v := OmitStruct{
			Bool: false, // This is the zero value for bool
			Int:  1,     // Add another field to avoid an empty object
		}
		b, err := maml.Marshal(v, maml.Indent(0))
		require.NoError(t, err)
		s := string(b)
		require.NotContains(t, s, "bool:")
		require.Contains(t, s, "int:1")
	})

	// Struct where fields do NOT have omitempty.
	type NoOmitStruct struct {
		String  string `maml:"string"`
		Int     int    `maml:"int"`
		Pointer *int   `maml:"pointer"`
	}

	t.Run("Fields without omitempty should be included even if zero-valued", func(t *testing.T) {
		v := NoOmitStruct{}
		b, err := maml.Marshal(v, maml.Indent(0))
		require.NoError(t, err)
		s := string(b)

		// Check that all fields are present even with zero values.
		require.Contains(t, s, `string:""`)
		require.Contains(t, s, `int:0`)
		require.Contains(t, s, `pointer:null`)
	})
}

// CustomUnmarshalValue implements maml.Unmarshaler
type CustomUnmarshalValue struct {
	Value string
}

func (c *CustomUnmarshalValue) UnmarshalMAML(data []byte) error {
	// A simple custom format: expecting `{ custom: "value" }`
	var inner struct {
		Value string `maml:"custom"`
	}
	if err := maml.Unmarshal(data, &inner); err != nil {
		return err
	}
	c.Value = inner.Value
	return nil
}

// CustomTextValue implements encoding.TextUnmarshaler
type CustomTextValue struct {
	Value string
}

func (c *CustomTextValue) UnmarshalText(text []byte) error {
	c.Value = "text(" + string(text) + ")"
	return nil
}

// CustomUnmarshalError implements maml.Unmarshaler and always returns an error.
type CustomUnmarshalError struct{}

func (c *CustomUnmarshalError) UnmarshalMAML(data []byte) error {
	return errors.New("custom unmarshal error")
}

func TestUnmarshal_CustomUnmarshaler(t *testing.T) {
	t.Run("Unmarshaler with pointer receiver", func(t *testing.T) {
		input := `{ custom: "hello world" }`
		var v CustomUnmarshalValue
		err := maml.Unmarshal([]byte(input), &v)
		require.NoError(t, err)
		require.Equal(t, "hello world", v.Value)
	})

	t.Run("TextUnmarshaler on string value", func(t *testing.T) {
		input := `"a string"`
		var v CustomTextValue
		err := maml.Unmarshal([]byte(input), &v)
		require.NoError(t, err)
		require.Equal(t, "text(a string)", v.Value)
	})

	t.Run("TextUnmarshaler is not called for non-string value", func(t *testing.T) {
		// The TextUnmarshaler should only be called if the MAML value is a string.
		// Here we provide an integer, so the default unmarshaler should fail.
		input := `123`
		var v CustomTextValue
		err := maml.Unmarshal([]byte(input), &v)
		require.Error(t, err)
		// We expect an error because the default unmarshaler will try to put an int
		// into a struct, which is not supported without field mapping.
		require.Contains(t, err.Error(), "cannot unmarshal integer into Go value of type maml_test.CustomTextValue")
	})

	t.Run("Unmarshaler that returns an error", func(t *testing.T) {
		input := `{}`
		var v CustomUnmarshalError
		err := maml.Unmarshal([]byte(input), &v)
		require.Error(t, err)
		require.Contains(t, err.Error(), "custom unmarshal error")
	})
}
