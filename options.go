package maml

import "fmt"

// MaxDepth returns a DecodeOption that sets the maximum recursion depth
// for the decoder. This helps prevent stack overflows when unmarshaling
// highly nested MAML documents.
//
// The depth n must be a positive integer.
func MaxDepth(n int) DecodeOption {
	return func(d *Decoder) error {
		if n <= 0 {
			return fmt.Errorf("maml: max depth must be a positive integer")
		}
		d.maxDepth = n
		return nil
	}
}
