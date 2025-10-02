package maml_test

import (
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
		expected := `{
  Name: "Test",
  Data: [
    1,
    2
  ]
}`
		b, err := maml.Marshal(v)
		require.NoError(t, err)
		require.Equal(t, expected, string(b))
	})

	t.Run("Compact output with Indent(0)", func(t *testing.T) {
		expected := `{ Name: "Test", Data: [1, 2] }`
		b, err := maml.Marshal(v, maml.Indent(0))
		require.NoError(t, err)
		require.Equal(t, expected, string(b))
	})

	t.Run("Custom indentation with Indent(4)", func(t *testing.T) {
		expected := `{
    Name: "Test",
    Data: [
        1,
        2
    ]
}`
		b, err := maml.Marshal(v, maml.Indent(4))
		require.NoError(t, err)
		require.Equal(t, expected, string(b))
	})

	t.Run("Invalid Indent option", func(t *testing.T) {
		_, err := maml.Marshal(v, maml.Indent(-1))
		require.Error(t, err)
		require.Contains(t, err.Error(), "indent spaces cannot be negative")
	})
}
