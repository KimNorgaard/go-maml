package maml

import (
	"io"
)

// DecodeOption is a functional option for configuring a Decoder.
type DecodeOption func(d *Decoder) error

// Decoder reads and decodes MAML values from an input stream.
type Decoder struct {
	r    io.Reader
	opts []DecodeOption

	// Internal configuration fields, set by options.
	maxDepth int
}

// NewDecoder returns a new decoder that reads from r. It stores options
// to be applied later by the Decode method.
func NewDecoder(r io.Reader, opts ...DecodeOption) *Decoder {
	return &Decoder{r: r, opts: opts}
}

// Decode reads the next MAML-encoded value from its input
// and stores it in the value pointed to by v.
// Note: This is a non-streaming implementation. It reads the entire
// reader into memory first before parsing.
func (d *Decoder) Decode(v any) error {
	if d.r == nil {
		return nil
	}
	data, err := io.ReadAll(d.r)
	if err != nil {
		return err
	}

	// Call the core Unmarshal function, passing along the options that
	// were configured when the decoder was created.
	return Unmarshal(data, v, d.opts...)
}
