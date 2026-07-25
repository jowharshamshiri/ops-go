package ops

import "fmt"

// ErrorKind identifies the category of an OpError.
type ErrorKind string

const (
	ErrExecutionFailed ErrorKind = "ExecutionFailed"
	ErrTimeout         ErrorKind = "Timeout"
	ErrContext         ErrorKind = "Context"
	ErrBatchFailed     ErrorKind = "BatchFailed"
	// ErrWrappedClassified is a CLASSIFIED failure inside wrapping context
	// (a batch child, a trigger-wrapped op, …) — the wrapper preserves the
	// origin's failure identity instead of flattening it into prose.
	// Message is the wrapping CHAIN for humans; Code/Class/Reason are the
	// origin's, verbatim. (matches Rust OpError::WrappedClassified)
	ErrWrappedClassified ErrorKind = "WrappedClassified"
	ErrAborted           ErrorKind = "Aborted"
	ErrTrigger           ErrorKind = "Trigger"
	// ErrClassified is a failure carrying its FULL identity from the emit
	// source: the machine-readable Code the origin error declares, the
	// failure Class it declares (whose problem it is), and the leaf human
	// Message. Wrapping layers construct ErrWrappedClassified from
	// classified origins instead of folding everything into prose; the
	// engine's run record and retry policy read it structurally.
	// (matches Rust OpError::Classified)
	ErrClassified ErrorKind = "Classified"
	ErrOther      ErrorKind = "Other"
)

// OpError is the error type for all operations in the framework.
type OpError struct {
	Kind      ErrorKind
	Message   string
	TimeoutMs uint64 // only populated for ErrTimeout
	Wrapped   error  // only populated for ErrOther
	// Code, Class, and Reason carry the failure identity DECLARED at the
	// emit source (docs/failure-taxonomy.md). Populated only for
	// ErrClassified (Message is the leaf message; Reason equals Message)
	// and ErrWrappedClassified (Message is the wrapping chain; Reason is
	// the origin's leaf message).
	Code   string
	Class  AttributionClass
	Reason string
	// ArgURN is the media URN of the argument named by the emit source.
	// Empty means the failure was not attributed to one argument.
	ArgURN string
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
	case ErrWrappedClassified:
		return e.Message
	case ErrClassified:
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
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

// NewClassifiedError creates an ErrClassified OpError carrying the failure
// identity declared at the emit source (docs/failure-taxonomy.md).
func NewClassifiedError(code string, class AttributionClass, message string) *OpError {
	return &OpError{Kind: ErrClassified, Code: code, Class: class, Message: message, Reason: message}
}

// NewWrappedClassifiedError creates an ErrWrappedClassified OpError: the
// chain is the wrapping text for humans; code/class/reason are the origin's,
// verbatim.
func NewWrappedClassifiedError(chain string, code string, class AttributionClass, reason string) *OpError {
	return &OpError{Kind: ErrWrappedClassified, Code: code, Class: class, Message: chain, Reason: reason}
}

// WithFailureArgURN attributes a classified failure to one argument at its
// emit source. Calling it for an unclassified error is a programmer error.
func (e *OpError) WithFailureArgURN(argURN string) *OpError {
	if e.Kind != ErrClassified && e.Kind != ErrWrappedClassified {
		panic("WithFailureArgURN requires a classified OpError")
	}
	if argURN == "" {
		panic("WithFailureArgURN requires a non-empty media URN")
	}
	e.ArgURN = argURN
	return e
}

// AttributionClassOf returns the failure class the error DECLARES. Classified
// variants carry their origin's declaration; everything else is
// FailureInternal — unclassified means "ours", never a guess
// (docs/failure-taxonomy.md). (matches Rust OpError::attribution_class)
func (e *OpError) AttributionClassOf() AttributionClass {
	switch e.Kind {
	case ErrClassified, ErrWrappedClassified:
		return e.Class
	default:
		return FailureInternal
	}
}

// FailureCode returns the machine-readable code declared at the emit
// source, or "" when the failure carried none.
// (matches Rust OpError::failure_code)
func (e *OpError) FailureCode() string {
	switch e.Kind {
	case ErrClassified, ErrWrappedClassified:
		return e.Code
	default:
		return ""
	}
}

// FailureArgURN returns the argument attribution declared at the emit source,
// or "" when no single argument was named. Wrappers preserve it verbatim.
func (e *OpError) FailureArgURN() string {
	switch e.Kind {
	case ErrClassified, ErrWrappedClassified:
		return e.ArgURN
	default:
		return ""
	}
}

// FailureReason returns the LEAF human reason — the origin's own message
// for classified failures, the Error() text otherwise.
// (matches Rust OpError::failure_reason)
func (e *OpError) FailureReason() string {
	switch e.Kind {
	case ErrClassified, ErrWrappedClassified:
		return e.Reason
	default:
		return e.Error()
	}
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
