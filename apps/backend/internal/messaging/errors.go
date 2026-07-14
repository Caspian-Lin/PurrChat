package messaging

import "fmt"

type panicError struct {
	value any
}

func (e *panicError) Error() string {
	return fmt.Sprintf("sink panic: %v", e.value)
}

func errPanic(value any) error {
	return &panicError{value: value}
}
