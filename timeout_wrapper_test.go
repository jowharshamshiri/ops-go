package ops_test

import (
	"strings"
	"testing"
	"time"

	ops "github.com/machinefabric/ops-go"
)

// slowOpTimeout sleeps 50ms then succeeds.
type slowOpTimeout struct{ ops.DefaultRollback }

func (o *slowOpTimeout) Perform(_ *ops.DryContext, _ *ops.WetContext) (int, error) {
	time.Sleep(50 * time.Millisecond)
	return 42, nil
}
func (o *slowOpTimeout) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("SlowOpTimeout").Build()
}

// verySlowOpTimeout sleeps 200ms (will exceed short timeouts).
type verySlowOpTimeout struct{ ops.DefaultRollback }

func (o *verySlowOpTimeout) Perform(_ *ops.DryContext, _ *ops.WetContext) (int, error) {
	time.Sleep(200 * time.Millisecond)
	return 42, nil
}
func (o *verySlowOpTimeout) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("VerySlowOpTimeout").Build()
}

// timeoutStringOp returns a fixed string quickly.
type timeoutStringOp struct{ ops.DefaultRollback }

func (o *timeoutStringOp) Perform(_ *ops.DryContext, _ *ops.WetContext) (string, error) {
	return "success", nil
}
func (o *timeoutStringOp) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("TimeoutStringOp").Build()
}

// timeoutIntOp returns a fixed int quickly.
type timeoutIntOp struct{ ops.DefaultRollback }

func (o *timeoutIntOp) Perform(_ *ops.DryContext, _ *ops.WetContext) (int, error) {
	return 100, nil
}
func (o *timeoutIntOp) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("TimeoutIntOp").Build()
}

// compositeTimeoutOp returns a string quickly.
type compositeTimeoutOp struct{ ops.DefaultRollback }

func (o *compositeTimeoutOp) Perform(_ *ops.DryContext, _ *ops.WetContext) (string, error) {
	return "logged and timed", nil
}
func (o *compositeTimeoutOp) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("CompositeTimeoutOp").Build()
}

// TEST033: Wrap a fast op in TimeBoundWrapper and confirm it completes before the timeout
func Test033_timeout_wrapper_success(t *testing.T) {
	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	wrapper := ops.NewTimeBoundWrapper[int](&slowOpTimeout{}, 200*time.Millisecond)
	result, err := wrapper.Perform(dry, wet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 42 {
		t.Errorf("expected 42, got %d", result)
	}
}

// TEST034: Wrap a slow op in TimeBoundWrapper with a short timeout and verify a Timeout error is returned
func Test034_timeout_wrapper_timeout(t *testing.T) {
	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	wrapper := ops.NewTimeBoundWrapper[int](&verySlowOpTimeout{}, 50*time.Millisecond)
	_, err := wrapper.Perform(dry, wet)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	oe := ops.AsOpError(err)
	if oe == nil {
		t.Fatalf("expected *OpError, got %T", err)
	}
	if oe.Kind != ops.ErrTimeout {
		t.Errorf("expected ErrTimeout, got %s", oe.Kind)
	}
	if oe.TimeoutMs != 50 {
		t.Errorf("expected TimeoutMs=50, got %d", oe.TimeoutMs)
	}
}

// TEST035: Create a named TimeBoundWrapper and verify the op succeeds and returns the expected value
func Test035_timeout_wrapper_with_name(t *testing.T) {
	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	wrapper := ops.NewTimeBoundWrapperWithName[string](&timeoutStringOp{}, 100*time.Millisecond, "TestOp")
	result, err := wrapper.Perform(dry, wet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "success" {
		t.Errorf("expected 'success', got %q", result)
	}
}

// TEST036: Use CreateTimeoutWrapperWithCallerName helper and verify the op result is returned
func Test036_caller_name_wrapper(t *testing.T) {
	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	wrapper := ops.CreateTimeoutWrapperWithCallerName[int](&timeoutIntOp{}, 100*time.Millisecond)
	result, err := wrapper.Perform(dry, wet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 100 {
		t.Errorf("expected 100, got %d", result)
	}
}

// TEST037: Use CreateLoggedTimeoutWrapper to compose logging and timeout wrappers and verify success
func Test037_logged_timeout_wrapper(t *testing.T) {
	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	wrapped := ops.CreateLoggedTimeoutWrapper[string](&compositeTimeoutOp{}, 100*time.Millisecond, "CompositeOp")
	result, err := wrapped.Perform(dry, wet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "logged and timed") {
		t.Errorf("expected result to contain 'logged and timed', got %q", result)
	}
}
