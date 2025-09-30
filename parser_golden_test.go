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

func TestParserGolden(t *testing.T) {
	files, err := filepath.Glob("testdata/*.maml")
	require.NoError(t, err)

	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			src, err := os.ReadFile(file)
			require.NoError(t, err)

			l := NewLexer(src)
			p := NewParser(l)
			doc := p.ParseDocument()

			var actual string
			errs := p.Errors()
			if len(errs) > 0 {
				actual = strings.Join(errs, "\n")
			} else {
				actual = doc.String()
			}

			goldenFile := strings.Replace(file, ".maml", ".golden", 1)
			if *update {
				err := os.WriteFile(goldenFile, []byte(actual), 0o644)
				require.NoError(t, err)
			}

			expected, err := os.ReadFile(goldenFile)
			// If the golden file doesn't exist, fail with a helpful message
			require.NoError(t, err, "Golden file not found. Run with -update to create it.")

			require.Equal(t, string(expected), actual, "Parser output does not match golden file.")
		})
	}
}
