package maml

import (
	"fmt"
	"io"
)

// Encoder writes MAML values to an output stream.
type Encoder struct {
	w    io.Writer
	opts []Option
}

// NewEncoder returns a new encoder that writes to w.
func NewEncoder(w io.Writer, opts ...Option) *Encoder {
	return &Encoder{w: w, opts: opts}
}

// Encode writes the MAML encoding of v to the stream.
func (e *Encoder) Encode(v any) error {
	// TODO: Implement the marshaling logic. This will involve walking the
	// Go value `v` using reflection, building an AST, and then formatting
	// that AST to the writer `e.w`.
	return fmt.Errorf("maml: Encode not yet implemented")
}
