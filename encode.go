package maml

import (
	"fmt"
	"io"

	"github.com/KimNorgaard/go-maml/internal/formatter"
	"github.com/KimNorgaard/go-maml/internal/marshaler"
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
	// TODO: Process options from e.opts here before marshaling.
	// For example, passing them to the marshaler if needed.

	node, err := marshaler.Marshal(v)
	if err != nil {
		return fmt.Errorf("maml: %w", err)
	}

	// TODO: Process options from e.opts here before formatting.
	// For example, passing them to the formatter.

	f := formatter.New(e.w)
	return f.Format(node)
}
