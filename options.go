package maml

import "fmt"

// options provides configuration for marshaling and unmarshaling.
type options struct {
	// maxDepth specifies the maximum depth to descend when decoding into
	// a Go value. If not set, a default depth limit is used.
	maxDepth int
}

// Option is a functional option for configuring an Encoder or Decoder.
type Option func(*options) error

// MaxDepth returns an Option that sets the maximum recursion depth
// for the decoder. This helps prevent stack overflows when unmarshaling
// highly nested MAML documents.
//
// The depth n must be a positive integer.
func MaxDepth(n int) Option {
	return func(o *options) error {
		if n <= 0 {
			return fmt.Errorf("maml: max depth must be a positive integer")
		}
		o.maxDepth = n
		return nil
	}
}
