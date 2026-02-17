package ops

import "fmt"

// ErrorKind identifies the category of an OpError.
type ErrorKind string

const (
	ErrExecutionFailed ErrorKind = "ExecutionFailed"
	ErrTimeout         ErrorKind = "Timeout"
	ErrContext         ErrorKind = "Context"
	ErrBatchFailed     ErrorKind = "BatchFailed"
	ErrAborted         ErrorKind = "Aborted"
	ErrTrigger         ErrorKind = "Trigger"
	ErrOther           ErrorKind = "Other"
)

// OpError is the error type for all operations in the framework.
type OpError struct {
	Kind      ErrorKind
	Message   string
	TimeoutMs uint64 // only populated for ErrTimeout
	Wrapped   error  // only populated for ErrOther
}

// Error implements the error interface.
// The format matches the Rust reference implementation's Display output exactly.
func (e *OpError) Error() string {
	switch e.Kind {
	case ErrExecutionFailed:
		return fmt.Sprintf("Op execution failed: %s", e.Message)
	case ErrTimeout:
		return fmt.Sprintf("Op timeout after %dms", e.TimeoutMs)
	case ErrContext:
		return fmt.Sprintf("Context error: %s", e.Message)
	case ErrBatchFailed:
		return fmt.Sprintf("Batch op failed: %s", e.Message)
	case ErrAborted:
		return fmt.Sprintf("Op aborted: %s", e.Message)
	case ErrTrigger:
		return fmt.Sprintf("Trigger error: %s", e.Message)
	default:
		if e.Wrapped != nil {
			return e.Wrapped.Error()
		}
		return e.Message
	}
}

// Unwrap returns the wrapped error for use with errors.Is/As.
func (e *OpError) Unwrap() error {
	return e.Wrapped
}

// NewExecutionFailedError creates an ErrExecutionFailed OpError.
func NewExecutionFailedError(msg string) *OpError {
	return &OpError{Kind: ErrExecutionFailed, Message: msg}
}

// NewTimeoutError creates an ErrTimeout OpError.
func NewTimeoutError(timeoutMs uint64) *OpError {
	return &OpError{Kind: ErrTimeout, TimeoutMs: timeoutMs}
}

// NewContextError creates an ErrContext OpError.
func NewContextError(msg string) *OpError {
	return &OpError{Kind: ErrContext, Message: msg}
}

// NewBatchFailedError creates an ErrBatchFailed OpError.
func NewBatchFailedError(msg string) *OpError {
	return &OpError{Kind: ErrBatchFailed, Message: msg}
}

// NewAbortedError creates an ErrAborted OpError.
func NewAbortedError(msg string) *OpError {
	return &OpError{Kind: ErrAborted, Message: msg}
}

// NewTriggerError creates an ErrTrigger OpError.
func NewTriggerError(msg string) *OpError {
	return &OpError{Kind: ErrTrigger, Message: msg}
}

// NewOtherError creates an ErrOther OpError wrapping any error.
func NewOtherError(err error) *OpError {
	return &OpError{Kind: ErrOther, Message: err.Error(), Wrapped: err}
}

// OpErrorFromJSONError converts a JSON error into an OpError (ErrOther kind).
func OpErrorFromJSONError(err error) *OpError {
	return NewOtherError(err)
}

// AsOpError extracts *OpError from any error, or returns nil.
func AsOpError(err error) *OpError {
	if err == nil {
		return nil
	}
	if oe, ok := err.(*OpError); ok {
		return oe
	}
	return nil
}
