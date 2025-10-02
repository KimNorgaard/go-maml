package maml

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var update = flag.Bool("update", false, "update golden files")

func TestGolden(t *testing.T) {
	files, err := filepath.Glob("testdata/*.maml")
	require.NoError(t, err)

	for _, file := range files {
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
