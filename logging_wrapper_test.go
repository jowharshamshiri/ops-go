package ops_test

import (
	"strings"
	"testing"

	ops "github.com/filegrind/ops-go"
)

// loggingTestOp returns a fixed int for logging wrapper tests.
type loggingTestOp struct{ ops.DefaultRollback }

func (o *loggingTestOp) Perform(_ *ops.DryContext, _ *ops.WetContext) (int, error) {
	return 42, nil
}
func (o *loggingTestOp) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("LoggingTestOp").Build()
}

// loggingFailingOp always fails.
type loggingFailingOp struct{ ops.DefaultRollback }

func (o *loggingFailingOp) Perform(_ *ops.DryContext, _ *ops.WetContext) (int, error) {
	return 0, ops.NewExecutionFailedError("test error")
}
func (o *loggingFailingOp) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("LoggingFailingOp").Build()
}

// loggingStringOp returns a fixed string.
type loggingStringOp struct{ ops.DefaultRollback }

func (o *loggingStringOp) Perform(_ *ops.DryContext, _ *ops.WetContext) (string, error) {
	return "test", nil
}
func (o *loggingStringOp) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("LoggingStringOp").Build()
}

// TEST029: Wrap a successful op in LoggingWrapper and verify it passes through the result unchanged
func Test029_logging_wrapper_success(t *testing.T) {
	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	wrapper := ops.NewLoggingWrapper[int](&loggingTestOp{}, "LoggingTestOp")
	result, err := wrapper.Perform(dry, wet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 42 {
		t.Errorf("expected 42, got %d", result)
	}
}

// TEST030: Wrap a failing op in LoggingWrapper and verify the error includes the op name context
func Test030_logging_wrapper_failure(t *testing.T) {
	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	wrapper := ops.NewLoggingWrapper[int](&loggingFailingOp{}, "FailingOp")
	_, err := wrapper.Perform(dry, wet)
	if err == nil {
		t.Fatal("expected error from failing op")
	}
	oe := ops.AsOpError(err)
	if oe == nil {
		t.Fatalf("expected *OpError, got %T", err)
	}
	if oe.Kind != ops.ErrExecutionFailed {
		t.Errorf("expected ErrExecutionFailed, got %s", oe.Kind)
	}
	if !strings.Contains(oe.Message, "FailingOp") {
		t.Errorf("expected message to contain 'FailingOp', got %q", oe.Message)
	}
}

// TEST031: Use CreateContextAwareLogger helper and verify the wrapped op returns its result
func Test031_context_aware_logger(t *testing.T) {
	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	wrapper := ops.CreateContextAwareLogger[string](&loggingStringOp{})
	result, err := wrapper.Perform(dry, wet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "test" {
		t.Errorf("expected 'test', got %q", result)
	}
}

// TEST032: Verify ANSI color escape code constants have the expected ANSI sequence values
func Test032_ansi_color_constants(t *testing.T) {
	if ops.ANSIYellow != "\x1b[33m" {
		t.Errorf("expected ANSIYellow=\\x1b[33m, got %q", ops.ANSIYellow)
	}
	if ops.ANSIGreen != "\x1b[32m" {
		t.Errorf("expected ANSIGreen=\\x1b[32m, got %q", ops.ANSIGreen)
	}
	if ops.ANSIRed != "\x1b[31m" {
		t.Errorf("expected ANSIRed=\\x1b[31m, got %q", ops.ANSIRed)
	}
	if ops.ANSIReset != "\x1b[0m" {
		t.Errorf("expected ANSIReset=\\x1b[0m, got %q", ops.ANSIReset)
	}
}
