package maml

import (
	"reflect"
)

// A MarshalerError represents an error from calling a MarshalMAML method.
type MarshalerError struct {
	Type reflect.Type
	Err  error
}

func (e *MarshalerError) Error() string {
	return "maml: error calling MarshalMAML for type " + e.Type.String() + ": " + e.Err.Error()
}

func (e *MarshalerError) Unwrap() error { return e.Err }

// UnmarshalerError represents an error that occurred while calling
// the UnmarshalMAML method.
type UnmarshalerError struct {
	Type reflect.Type
	Err  error
}

func (e *UnmarshalerError) Error() string {
	return "maml: error calling UnmarshalMAML for type " + e.Type.String() + ": " + e.Err.Error()
}

func (e *UnmarshalerError) Unwrap() error {
	return e.Err
}
