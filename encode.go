package maml

import (
	"fmt"
	"io"
)

// EncodeOption is a functional option for configuring an Encoder.
type EncodeOption func(e *Encoder) error

// Encoder writes MAML values to an output stream.
type Encoder struct {
	w io.Writer
}

// NewEncoder returns a new encoder that writes to w.
func NewEncoder(w io.Writer, opts ...EncodeOption) *Encoder {
	e := &Encoder{w: w}
	return e
}

// Encode writes the MAML encoding of v to the stream.
func (e *Encoder) Encode(v any) error {
	// TODO: Implement the marshaling logic. This will involve walking the
	// Go value `v` using reflection, building an AST, and then formatting
	// that AST to the writer `e.w`.
	return fmt.Errorf("maml: Encode not yet implemented")
}
