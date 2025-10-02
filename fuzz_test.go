//go:build go1.18

package maml_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/KimNorgaard/go-maml"
	"github.com/stretchr/testify/require"
)

func FuzzRoundTrip(f *testing.F) {
	// Seed the corpus with valid MAML files from the testdata directory.
	// This gives the fuzzer good starting points for valid syntax.
	seedFiles, err := filepath.Glob("testdata/*.maml")
	if err != nil {
		f.Fatalf("failed to find seed files: %v", err)
	}

	for _, file := range seedFiles {
		data, err := os.ReadFile(file)
		if err != nil {
			f.Fatalf("failed to read seed file %s: %v", file, err)
		}
		f.Add(data)
	}

	// Add some simple but important edge cases manually.
	f.Add([]byte("{}"))
	f.Add([]byte("[]"))
	f.Add([]byte("null"))
	f.Add([]byte(`"a simple string"`))
	f.Add([]byte("12345"))
	f.Add([]byte("true"))

	f.Fuzz(func(t *testing.T, originalData []byte) {
		// 1. Try to unmarshal the fuzzed data into a generic interface.
		var v1 any
		err := maml.Unmarshal(originalData, &v1)
		if err != nil {
			// If there's an error, the input was invalid MAML, which is expected.
			// The fuzzer's main job is to find inputs that cause a panic.
			// The fuzz engine detects panics automatically, so we can just return.
			return
		}

		// 2. If unmarshaling succeeded, marshal it back to bytes.
		// This step should *never* fail or panic for a value our own unmarshaler
		// just successfully created.
		marshaledData, err := maml.Marshal(v1)
		require.NoError(t, err, "Marshal failed for a successfully unmarshaled value")

		// 3. Unmarshal the marshaled data again into a new variable.
		// This must also succeed without error or panic.
		var v2 any
		err = maml.Unmarshal(marshaledData, &v2)
		require.NoError(t, err, "Unmarshal failed on our own marshaled output")

		// 4. Compare the results. They must be identical.
		// This ensures the library is symmetric (what goes in, comes out).
		// require.Equal uses reflect.DeepEqual under the hood, which is correct
		// for comparing the complex nested structures we might get.
		require.Equal(t, v1, v2, "Value is not the same after a marshal/unmarshal round trip")
	})
}
