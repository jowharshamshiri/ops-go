package ops_test

import (
	"encoding/json"
	"strings"
	"testing"

	ops "github.com/machinefabric/ops-go"
)

// TEST104: Verify OpError::ExecutionFailed displays with the correct message format
func Test104_op_error_display_execution_failed(t *testing.T) {
	err := ops.NewExecutionFailedError("something broke")
	if err.Error() != "Op execution failed: something broke" {
		t.Errorf("unexpected error string: %q", err.Error())
	}
}

// TEST105: Verify OpError::Timeout displays with the correct timeout_ms value
func Test105_op_error_display_timeout(t *testing.T) {
	err := ops.NewTimeoutError(250)
	if err.Error() != "Op timeout after 250ms" {
		t.Errorf("unexpected error string: %q", err.Error())
	}
}

// TEST106: Verify OpError::Context displays with the correct message format
func Test106_op_error_display_context(t *testing.T) {
	err := ops.NewContextError("missing key")
	if err.Error() != "Context error: missing key" {
		t.Errorf("unexpected error string: %q", err.Error())
	}
}

// TEST107: Verify OpError::Aborted displays with the correct message format
func Test107_op_error_display_aborted(t *testing.T) {
	err := ops.NewAbortedError("user cancelled")
	if err.Error() != "Op aborted: user cancelled" {
		t.Errorf("unexpected error string: %q", err.Error())
	}
}

// TEST108: Copy an OpError::ExecutionFailed and verify the copy is identical
func Test108_op_error_copy_execution_failed(t *testing.T) {
	err := ops.NewExecutionFailedError("fail msg")
	copied := *err // value copy
	if err.Error() != copied.Error() {
		t.Errorf("copy should equal original: %q vs %q", err.Error(), copied.Error())
	}
	if copied.Kind != ops.ErrExecutionFailed {
		t.Errorf("expected ErrExecutionFailed kind, got %s", copied.Kind)
	}
	if copied.Message != "fail msg" {
		t.Errorf("expected message 'fail msg', got %q", copied.Message)
	}
}

// TEST109: Copy OpError::Timeout and verify timeout_ms is preserved
func Test109_op_error_copy_timeout(t *testing.T) {
	err := ops.NewTimeoutError(500)
	copied := *err
	if copied.Kind != ops.ErrTimeout {
		t.Errorf("expected ErrTimeout kind, got %s", copied.Kind)
	}
	if copied.TimeoutMs != 500 {
		t.Errorf("expected TimeoutMs=500, got %d", copied.TimeoutMs)
	}
}

// TEST110: Copy an OpError wrapping a std error and verify the error message is accessible
func Test110_op_error_copy_other_preserves_message(t *testing.T) {
	import_err := &wrappedStdError{msg: "file missing"}
	err := ops.NewOtherError(import_err)
	copied := *err
	if !strings.Contains(copied.Error(), "file missing") {
		t.Errorf("expected copied error to contain 'file missing', got: %s", copied.Error())
	}
}

// TEST111: Convert a json.Unmarshal error into OpError via conversion function
func Test111_op_error_from_json_error(t *testing.T) {
	var x int32
	jsonErr := json.Unmarshal([]byte(`"not_a_number"`), &x)
	if jsonErr == nil {
		t.Fatal("expected json unmarshal error for string->int32")
	}
	opErr := ops.OpErrorFromJSONError(jsonErr)
	if opErr == nil {
		t.Fatal("expected non-nil OpError")
	}
	if opErr.Kind != ops.ErrOther {
		t.Errorf("expected ErrOther kind, got %s", opErr.Kind)
	}
}

// wrappedStdError is a test helper implementing the error interface.
type wrappedStdError struct{ msg string }

func (e *wrappedStdError) Error() string { return e.msg }
