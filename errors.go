package maml

import "reflect"

// A MarshalerError represents an error from calling a MarshalMAML method.
type MarshalerError struct {
	Type reflect.Type
	Err  error
}

func (e *MarshalerError) Error() string {
	return "maml: error calling MarshalMAML for type " + e.Type.String() + ": " + e.Err.Error()
}

func (e *MarshalerError) Unwrap() error { return e.Err }
