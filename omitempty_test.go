package maml_test

import (
	"testing"

	"github.com/KimNorgaard/go-maml"
	"github.com/stretchr/testify/require"
)

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
