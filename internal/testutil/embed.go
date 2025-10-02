package testutil

import (
	"embed"
	"fmt"
	"io/fs"
)

// TestdataFS holds the embedded test data files.
//
//go:embed testdata
var TestdataFS embed.FS

// ReadTestData reads and returns the content of an embedded test file.
func ReadTestData(name string) ([]byte, error) {
	path := fmt.Sprintf("testdata/%s", name)
	data, err := fs.ReadFile(TestdataFS, path)
	if err != nil {
		return nil, fmt.Errorf("failed to read test data file '%s': %w", name, err)
	}
	return data, nil
}
