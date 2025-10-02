package maml_test

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"testing"

	"github.com/KimNorgaard/go-maml"
)

//go:embed testdata/large.json
var benchmarkJSONInput []byte

var benchmarkData any

func init() {
	if err := json.Unmarshal(benchmarkJSONInput, &benchmarkData); err != nil {
		panic("failed to unmarshal benchmark data for encoding benchmark: " + err.Error())
	}
}

func BenchmarkEncode(b *testing.B) {
	b.ReportAllocs()
	// SetBytes informs the benchmark runner how many bytes are processed in a single operation.
	// This is used to calculate ns/op and MB/s.
	// We use the size of the JSON input as a proxy for the data complexity.
	b.SetBytes(int64(len(benchmarkJSONInput)))

	// Encoder writes to an io.Writer. We'll use a buffer that we reset on each iteration.
	var buf bytes.Buffer
	enc := maml.NewEncoder(&buf)

	b.ResetTimer()

	for b.Loop() {
		// The Encode method is what we're benchmarking.
		if err := enc.Encode(benchmarkData); err != nil {
			b.Fatalf("Encode failed during benchmark: %v", err)
		}

		// Reset the buffer for the next run to avoid reallocating it.
		buf.Reset()
	}
}
