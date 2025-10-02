package token

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLookupIdent(t *testing.T) {
	tests := []struct {
		input    string
		expected Type
	}{
		{"true", TRUE},
		{"false", FALSE},
		{"null", NULL},
		{"foobar", IDENT},
		{"my_var", IDENT},
		{"r2d2", IDENT},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			actual := LookupIdent(tt.input)
			require.Equal(t, tt.expected, actual)
		})
	}
}
