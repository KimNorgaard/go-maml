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

func TestParser(t *testing.T) {
	files, err := filepath.Glob("testdata/*.maml")
	require.NoError(t, err)

	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			src, err := os.ReadFile(file)
			require.NoError(t, err)

			node, err := Parse(src)

			var actual string
			if err != nil {
				actual = err.Error()
			} else {
				actual = node.String()
			}

			goldenFile := strings.Replace(file, ".maml", ".golden", 1)
			if *update {
				err := os.WriteFile(goldenFile, []byte(actual), 0o644)
				require.NoError(t, err)
			}

			expected, err := os.ReadFile(goldenFile)
			require.NoError(t, err)

			require.Equal(t, string(expected), actual, "parser output does not match golden file")
		})
	}
}
