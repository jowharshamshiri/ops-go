package ops_test

import (
	"strings"
	"testing"

	ops "github.com/filegrind/ops-go"
)

// TEST005: Confirm the Perform() utility wraps an op with automatic logging and returns its result
func Test005_perform_with_auto_logging(t *testing.T) {
	op := &testOpInt{value: 42}
	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	result, err := ops.Perform[int](op, dry, wet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 42 {
		t.Errorf("expected 42, got %d", result)
	}
}

// TEST006: Verify GetCallerTriggerName returns a string containing the file name with a "." separator
func Test006_caller_trigger_name(t *testing.T) {
	name := ops.GetCallerTriggerName()
	if !strings.Contains(name, "ops_test") {
		t.Errorf("expected name to contain 'ops_test', got %q", name)
	}
	if !strings.Contains(name, ".") {
		t.Errorf("expected name to contain '.', got %q", name)
	}
}

// TEST007: Confirm WrapNestedOpException wraps an error with the op name in the message
func Test007_wrap_nested_op_exception(t *testing.T) {
	originalErr := ops.NewExecutionFailedError("original error")
	wrapped := ops.WrapNestedOpException("TestOp", originalErr)

	oe := ops.AsOpError(wrapped)
	if oe == nil {
		t.Fatal("expected *OpError from WrapNestedOpException")
	}
	if oe.Kind != ops.ErrExecutionFailed {
		t.Errorf("expected ErrExecutionFailed, got %s", oe.Kind)
	}
	if !strings.Contains(oe.Message, "TestOp") {
		t.Errorf("expected message to contain 'TestOp', got %q", oe.Message)
	}
	if !strings.Contains(oe.Message, "original error") {
		t.Errorf("expected message to contain 'original error', got %q", oe.Message)
	}
}

// TEST008: Verify WrapRuntimeException converts any error into an OpError::ExecutionFailed
func Test008_wrap_runtime_exception(t *testing.T) {
	stdErr := &simpleStdError{msg: "test error"}
	wrapped := ops.WrapRuntimeException(stdErr)

	oe := ops.AsOpError(wrapped)
	if oe == nil {
		t.Fatal("expected *OpError from WrapRuntimeException")
	}
	if oe.Kind != ops.ErrExecutionFailed {
		t.Errorf("expected ErrExecutionFailed, got %s", oe.Kind)
	}
	if !strings.Contains(oe.Message, "Runtime error") {
		t.Errorf("expected message to contain 'Runtime error', got %q", oe.Message)
	}
	if !strings.Contains(oe.Message, "test error") {
		t.Errorf("expected message to contain 'test error', got %q", oe.Message)
	}
}

// simpleStdError is a helper error type for tests.
type simpleStdError struct{ msg string }

func (e *simpleStdError) Error() string { return e.msg }
