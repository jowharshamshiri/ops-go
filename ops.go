package ops

import (
	"fmt"
	"runtime"
	"strings"
)

// Perform executes an op with automatic logging wrapping.
// This is the central execution function equivalent to Java OPS.perform().
func Perform[T any](op Op[T], dry *DryContext, wet *WetContext) (T, error) {
	triggerName := GetCallerTriggerName()
	logged := NewLoggingWrapper(op, triggerName)
	return logged.Perform(dry, wet)
}

// GetCallerTriggerName returns a string identifying the immediate caller's location.
// Format: "filename.lineNumber" (e.g., "ops_test.42").
func GetCallerTriggerName() string {
	pcs := make([]uintptr, 3)
	n := runtime.Callers(2, pcs)
	if n == 0 {
		return "unknown.0"
	}
	frames := runtime.CallersFrames(pcs[:n])
	frame, _ := frames.Next()
	parts := strings.Split(frame.File, "/")
	filename := parts[len(parts)-1]
	filename = strings.TrimSuffix(filename, ".go")
	return fmt.Sprintf("%s.%d", filename, frame.Line)
}

// WrapNestedOpException wraps an OpError with the op name in the error message.
// Equivalent to Java wrapNestedOpException(String, Exception).
func WrapNestedOpException(triggerName string, err error) error {
	opErr := AsOpError(err)
	if opErr == nil {
		return NewExecutionFailedError(fmt.Sprintf("Op '%s' failed: %v", triggerName, err))
	}
	switch opErr.Kind {
	case ErrExecutionFailed:
		return NewExecutionFailedError(fmt.Sprintf("Op '%s' failed: %s", triggerName, opErr.Message))
	case ErrTimeout:
		return NewExecutionFailedError(fmt.Sprintf("Op '%s' timed out after %dms", triggerName, opErr.TimeoutMs))
	case ErrContext:
		return NewContextError(fmt.Sprintf("Op '%s' context error: %s", triggerName, opErr.Message))
	case ErrBatchFailed:
		return NewBatchFailedError(fmt.Sprintf("Batch op '%s' failed: %s", triggerName, opErr.Message))
	// Classified failures keep their identity through wrapping — the wrap
	// adds human context to the CHAIN, never touches class/code/reason
	// (docs/failure-taxonomy.md).
	case ErrWrappedClassified:
		return NewWrappedClassifiedError(
			fmt.Sprintf("Batch op '%s' failed: %s", triggerName, opErr.Message),
			opErr.Code, opErr.Class, opErr.Reason,
		)
	case ErrClassified:
		return NewWrappedClassifiedError(
			fmt.Sprintf("Op '%s' failed: %s: %s", triggerName, opErr.Code, opErr.Message),
			opErr.Code, opErr.Class, opErr.Message,
		)
	case ErrAborted:
		return NewAbortedError(fmt.Sprintf("Op '%s' aborted: %s", triggerName, opErr.Message))
	case ErrTrigger:
		return NewTriggerError(fmt.Sprintf("Op '%s' internal error: %s", triggerName, opErr.Message))
	default:
		return NewExecutionFailedError(fmt.Sprintf("Op '%s' failed: %v", triggerName, opErr.Wrapped))
	}
}

// WrapNestedException wraps any error as an OpError (ErrOther).
// Equivalent to Java wrapNestedOpException(Exception).
func WrapNestedException(err error) error {
	return NewOtherError(err)
}

// WrapRuntimeException converts any error to an ExecutionFailed OpError.
// Equivalent to Java wrapNestedRuntimeException(Exception).
func WrapRuntimeException(err error) error {
	return NewExecutionFailedError(fmt.Sprintf("Runtime error: %v", err))
}
