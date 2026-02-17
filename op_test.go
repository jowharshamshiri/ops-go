package ops_test

import (
	"testing"

	ops "github.com/filegrind/ops-go"
)

// Concrete test op used across TEST001-TEST004
type testOpInt struct {
	ops.DefaultRollback
	value int
}

func (o *testOpInt) Perform(_ *ops.DryContext, _ *ops.WetContext) (int, error) {
	return o.value, nil
}

func (o *testOpInt) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("TestOp").
		Description("Simple test op").
		Build()
}

// TEST001: Run Op::perform and verify the returned value matches what the op was configured with
func Test001_op_execution(t *testing.T) {
	op := &testOpInt{value: 42}
	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	result, err := op.Perform(dry, wet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 42 {
		t.Errorf("expected 42, got %d", result)
	}
}

type contextUsingOp struct{ ops.DefaultRollback }

func (o *contextUsingOp) Perform(dry *ops.DryContext, _ *ops.WetContext) (string, error) {
	name, err := ops.DryGetRequired[string](dry, "name")
	if err != nil {
		return "", err
	}
	return "Hello, " + name + "!", nil
}

func (o *contextUsingOp) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("ContextUsingOp").Build()
}

// TEST002: Verify Op reads from DryContext and produces a formatted result using that data
func Test002_op_with_contexts(t *testing.T) {
	op := &contextUsingOp{}
	dry := ops.NewDryContext().WithValue("name", "World")
	wet := ops.NewWetContext()

	result, err := op.Perform(dry, wet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Hello, World!" {
		t.Errorf("expected 'Hello, World!', got %q", result)
	}
}

type simpleVoidOp struct{ ops.DefaultRollback }

func (o *simpleVoidOp) Perform(_ *ops.DryContext, _ *ops.WetContext) (struct{}, error) {
	return struct{}{}, nil
}
func (o *simpleVoidOp) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("SimpleOp").Build()
}

// TEST003: Confirm that the default rollback implementation is a no-op that always succeeds
func Test003_op_default_rollback(t *testing.T) {
	op := &simpleVoidOp{}
	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	if err := op.Rollback(dry, wet); err != nil {
		t.Errorf("default rollback should be no-op, got error: %v", err)
	}
}

type rollbackTrackingOpBool struct {
	performed   *bool
	rolledBack  *bool
}

func (o *rollbackTrackingOpBool) Perform(_ *ops.DryContext, _ *ops.WetContext) (struct{}, error) {
	*o.performed = true
	return struct{}{}, nil
}

func (o *rollbackTrackingOpBool) Rollback(_ *ops.DryContext, _ *ops.WetContext) error {
	*o.rolledBack = true
	return nil
}

func (o *rollbackTrackingOpBool) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("RollbackTrackingOp").Build()
}

// TEST004: Verify a custom rollback implementation is called and sets the rolled_back flag
func Test004_op_custom_rollback(t *testing.T) {
	performed := false
	rolledBack := false

	op := &rollbackTrackingOpBool{performed: &performed, rolledBack: &rolledBack}
	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	if _, err := op.Perform(dry, wet); err != nil {
		t.Fatalf("perform failed: %v", err)
	}
	if !performed {
		t.Error("expected performed=true after Perform")
	}
	if rolledBack {
		t.Error("expected rolledBack=false before Rollback")
	}

	if err := op.Rollback(dry, wet); err != nil {
		t.Fatalf("rollback failed: %v", err)
	}
	if !performed {
		t.Error("expected performed=true after Rollback")
	}
	if !rolledBack {
		t.Error("expected rolledBack=true after Rollback")
	}
}
