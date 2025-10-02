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
	o := options{}
	for _, opt := range e.opts {
		if err := opt(&o); err != nil {
			return err
		}
	}

	node, err := marshaler.Marshal(v)
	if err != nil {
		return fmt.Errorf("maml: %w", err)
	}

	f := formatter.New(e.w, o.indent)
	return f.Format(node)
}
