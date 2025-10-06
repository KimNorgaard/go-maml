package maml

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/KimNorgaard/go-maml/internal/ast"
	"github.com/stretchr/testify/require"
)

var update = flag.Bool("update", false, "update golden files")

func TestGolden(t *testing.T) {
	files, err := filepath.Glob("testdata/*.maml")
	require.NoError(t, err)

	for _, file := range files {
		if strings.Contains(file, "format-comments.maml") {
			continue
		}
		t.Run(file, func(t *testing.T) {
			src, err := os.ReadFile(file)
			require.NoError(t, err)

			var v any
			err = Unmarshal(src, &v)

			var actual []byte
			if err != nil {
				// For MAML files that are expected to fail parsing,
				// the golden file will contain the error message.
				actual = []byte(err.Error())
			} else {
				// For valid MAML, we marshal it back out with indentation
				// to create a canonical, readable golden file.
				actual, err = Marshal(v, Indent(2))
				require.NoError(t, err)
			}

			goldenFile := strings.Replace(file, ".maml", ".golden", 1)
			if *update {
				err := os.WriteFile(goldenFile, actual, 0o644)
				require.NoError(t, err)
			}

			expected, err := os.ReadFile(goldenFile)
			require.NoError(t, err, "Golden file not found. Run with -update to create it.")

			require.Equal(t, string(expected), string(actual), "Round-trip output does not match golden file.")
		})
	}
}

// This is a separate golden test to verify the formatter's output
// when working with a comment-rich AST, as opposed to the data-only
// round trip in golden_test.go.
func TestGoldenWithComments(t *testing.T) {
	const inputFile = "testdata/format-comments.maml"

	t.Run(inputFile, func(t *testing.T) {
		src, err := os.ReadFile(inputFile)
		require.NoError(t, err)

		// Parse the source into a full, comment-rich AST by unmarshaling
		// into an *ast.Document with the WithComments option.
		var doc *ast.Document
		err = Unmarshal(src, &doc, ParseComments())
		require.NoError(t, err)

		// Marshal the AST back out with standard formatting.
		// We expect this to match the golden file.
		actual, err := Marshal(doc, Indent(2), UseFieldCommas())
		require.NoError(t, err)

		goldenFile := strings.Replace(inputFile, ".maml", ".golden", 1)

		// The update flag can be used to automatically update the golden file.
		// To use it, run: go test -v ./... -update
		if *update {
			err := os.WriteFile(goldenFile, actual, 0o644)
			require.NoError(t, err)
		}

		expected, err := os.ReadFile(goldenFile)
		require.NoError(t, err, "Golden file not found. Run with -update to create it.")

		// The file system may add a trailing newline to the golden file, but the
		// formatter does not. Trim it for a consistent comparison.
		expected = bytes.TrimSuffix(expected, []byte("\n"))

		require.Equal(t, string(expected), string(actual), "Formatted output does not match golden file.")
	})
}
