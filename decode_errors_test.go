package maml_test

import (
	"testing"

	"github.com/KimNorgaard/go-maml"
	"github.com/stretchr/testify/require"
)

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
	require.EqualError(t, err, "maml: parsing error: illegal token encountered: invalid utf-8 sequence in string\nunterminated object literal, expected '}' got EOF")
	require.EqualError(t, err, "maml: parsing error: illegal token encountered: invalid utf-8 sequence in string\nunterminated object literal, expected '}' got EOF")
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
			expectedErr: "maml: parsing error: illegal token encountered: unterminated string\nunterminated object literal, expected '}' got EOF",
		},
		{
			name:        "Missing colon",
			input:       `{ key "value" }`,
			expectedErr: "maml: parsing error: expected ':' after key, got STRING",
		},
		{
			name:        "Unbalanced braces",
			input:       `{ key: "value"`,
			expectedErr: "maml: parsing error: unterminated object literal, expected '}' got EOF",
		},
		{
			name:        "Invalid map key",
			input:       `{ []: "value" }`,
			expectedErr: "maml: parsing error: invalid token for object key: [ ('[')",
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
