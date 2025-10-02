package errors

import "fmt"

// ParseError represents a single error that occurred during parsing.
// It includes the position of the error.
type ParseError struct {
	Message string
	Line    int
	Column  int
}

// ParseErrors is a slice of ParseError that implements the error interface.
// This allows returning all syntax errors found during parsing at once.
type ParseErrors []ParseError

func (p ParseErrors) Error() string {
	if len(p) == 0 {
		return ""
	}
	// For simplicity, the default error message for the collection
	// just reports the first error.
	return fmt.Sprintf("maml: parsing error at line %d, column %d: %s", p[0].Line, p[0].Column, p[0].Message)
}
