package ops_test

import (
	"testing"

	ops "github.com/filegrind/ops-go"
)

// abortTestOp can be configured to abort with or without a reason.
type abortTestOp struct {
	ops.DefaultRollback
	shouldAbort bool
	abortReason *string
}

func (o *abortTestOp) Perform(dry *ops.DryContext, _ *ops.WetContext) (int, error) {
	if o.shouldAbort {
		if o.abortReason != nil {
			return 0, ops.AbortWithReason(dry, *o.abortReason)
		}
		return 0, ops.Abort(dry)
	}
	return 42, nil
}

func (o *abortTestOp) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("AbortTestOp").Build()
}

// continueTestOpCF signals continue_loop and returns zero.
type continueTestOpCF struct {
	ops.DefaultRollback
	shouldContinue bool
	value          int
}

func (o *continueTestOpCF) Perform(dry *ops.DryContext, _ *ops.WetContext) (int, error) {
	if o.shouldContinue {
		ops.ContinueLoop(dry)
		return 0, nil
	}
	return o.value, nil
}

func (o *continueTestOpCF) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("ContinueTestOp").Build()
}

// checkAbortOpCF uses CheckAbort to short-circuit.
type checkAbortOpCF struct{ ops.DefaultRollback }

func (o *checkAbortOpCF) Perform(dry *ops.DryContext, _ *ops.WetContext) (int, error) {
	if err := ops.CheckAbort(dry); err != nil {
		return 0, err
	}
	return 100, nil
}

func (o *checkAbortOpCF) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("CheckAbortOp").Build()
}

// TEST057: Invoke Abort without a reason and verify the context is aborted with no reason string
func Test057_abort_without_reason(t *testing.T) {
	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	op := &abortTestOp{shouldAbort: true}
	_, err := op.Perform(dry, wet)

	if err == nil {
		t.Fatal("expected error from Abort")
	}
	if !dry.IsAborted() {
		t.Error("expected context to be aborted")
	}
	if dry.AbortReason() != nil {
		t.Errorf("expected nil abort reason, got %v", dry.AbortReason())
	}
	oe := ops.AsOpError(err)
	if oe == nil || oe.Kind != ops.ErrAborted {
		t.Errorf("expected ErrAborted, got %v", err)
	}
	if oe.Message != "Operation aborted" {
		t.Errorf("expected message 'Operation aborted', got %q", oe.Message)
	}
}

// TEST058: Invoke AbortWithReason and verify abort_reason matches
func Test058_abort_with_reason(t *testing.T) {
	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	reason := "Test reason"
	op := &abortTestOp{shouldAbort: true, abortReason: &reason}
	_, err := op.Perform(dry, wet)

	if err == nil {
		t.Fatal("expected error from AbortWithReason")
	}
	if !dry.IsAborted() {
		t.Error("expected context to be aborted")
	}
	if dry.AbortReason() == nil || *dry.AbortReason() != "Test reason" {
		t.Errorf("expected abort reason 'Test reason', got %v", dry.AbortReason())
	}
	oe := ops.AsOpError(err)
	if oe == nil || oe.Kind != ops.ErrAborted {
		t.Errorf("expected ErrAborted, got %v", err)
	}
	if oe.Message != "Test reason" {
		t.Errorf("expected message 'Test reason', got %q", oe.Message)
	}
}

// TEST059: Use ContinueLoop inside an op and verify the scoped continue flag is set in context
func Test059_continue_loop(t *testing.T) {
	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	// Set up a loop context for ContinueLoop to work
	loopID := "test_loop_123"
	dry.Insert("__current_loop_id", loopID)

	op := &continueTestOpCF{shouldContinue: true, value: 99}
	result, err := op.Perform(dry, wet)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 0 {
		t.Errorf("expected result=0 after continue, got %d", result)
	}

	// Check the scoped continue flag was set
	continueVar := "__continue_loop_" + loopID
	flagVal, ok := ops.DryGet[bool](dry, continueVar)
	if !ok || !flagVal {
		t.Errorf("expected continue flag to be set in context under %q", continueVar)
	}
}

// TEST060: Use CheckAbort to short-circuit when the abort flag is already set in context
func Test060_check_abort(t *testing.T) {
	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	op := &checkAbortOpCF{}

	// Without abort flag - should succeed
	result, err := op.Perform(dry, wet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 100 {
		t.Errorf("expected 100, got %d", result)
	}

	// Set abort flag and try again
	reason := "Pre-existing abort"
	dry.SetAbort(&reason)
	_, err = op.Perform(dry, wet)
	if err == nil {
		t.Fatal("expected error after abort flag set")
	}
	oe := ops.AsOpError(err)
	if oe == nil || oe.Kind != ops.ErrAborted {
		t.Errorf("expected ErrAborted, got %v", err)
	}
	if oe.Message != "Pre-existing abort" {
		t.Errorf("expected message 'Pre-existing abort', got %q", oe.Message)
	}
}

// TEST061: Run a BatchOp where the second op aborts and verify the batch stops and propagates the abort
func Test061_batch_op_with_abort(t *testing.T) {
	batchReason := "Batch abort"
	batchOps := []ops.Op[int]{
		&abortTestOp{shouldAbort: false},
		&abortTestOp{shouldAbort: true, abortReason: &batchReason},
		&abortTestOp{shouldAbort: false}, // should not execute
	}
	batch := ops.NewBatchOp(batchOps)
	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	_, err := batch.Perform(dry, wet)
	if err == nil {
		t.Fatal("expected error from batch with abort")
	}
	oe := ops.AsOpError(err)
	if oe == nil || oe.Kind != ops.ErrAborted {
		t.Errorf("expected ErrAborted, got %v", err)
	}
	if oe.Message != "Batch abort" {
		t.Errorf("expected message 'Batch abort', got %q", oe.Message)
	}
}

// TEST062: Start a BatchOp with an abort flag already set and verify it immediately returns Aborted
func Test062_batch_op_with_pre_existing_abort(t *testing.T) {
	batchOps := []ops.Op[int]{
		&abortTestOp{shouldAbort: false},
		&abortTestOp{shouldAbort: false},
	}
	batch := ops.NewBatchOp(batchOps)
	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	reason := "Pre-existing abort"
	dry.SetAbort(&reason)

	_, err := batch.Perform(dry, wet)
	if err == nil {
		t.Fatal("expected error for pre-existing abort")
	}
	oe := ops.AsOpError(err)
	if oe == nil || oe.Kind != ops.ErrAborted {
		t.Errorf("expected ErrAborted, got %v", err)
	}
	if oe.Message != "Pre-existing abort" {
		t.Errorf("expected message 'Pre-existing abort', got %q", oe.Message)
	}
}

// TEST063: Run a LoopOp where an op signals continue and verify subsequent ops in the iteration are skipped
func Test063_loop_op_with_continue(t *testing.T) {
	loopOps := []ops.Op[int]{
		&continueTestOpCF{shouldContinue: false, value: 10},
		&continueTestOpCF{shouldContinue: true, value: 20}, // will continue, returns 0
		&abortTestOp{shouldAbort: false},                   // should not execute due to continue
	}
	loop := ops.NewLoopOp("test_counter", 2, loopOps)
	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	results, err := loop.Perform(dry, wet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Each iteration: [10, 0] (continue after second op, third skipped)
	// 2 iterations = 4 results
	if len(results) != 4 {
		t.Fatalf("expected 4 results, got %d: %v", len(results), results)
	}
	expected := []int{10, 0, 10, 0}
	for i, r := range results {
		if r != expected[i] {
			t.Errorf("results[%d]: expected %d, got %d", i, expected[i], r)
		}
	}
}

// TEST064: Run a LoopOp where an op aborts mid-loop and verify the loop terminates with the abort error
func Test064_loop_op_with_abort(t *testing.T) {
	loopReason := "Loop abort"
	loopOps := []ops.Op[int]{
		&abortTestOp{shouldAbort: false},
		&abortTestOp{shouldAbort: true, abortReason: &loopReason},
		&abortTestOp{shouldAbort: false}, // should not execute
	}
	loop := ops.NewLoopOp("test_counter", 3, loopOps)
	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	_, err := loop.Perform(dry, wet)
	if err == nil {
		t.Fatal("expected error from abort in loop")
	}
	oe := ops.AsOpError(err)
	if oe == nil || oe.Kind != ops.ErrAborted {
		t.Errorf("expected ErrAborted, got %v", err)
	}
	if oe.Message != "Loop abort" {
		t.Errorf("expected message 'Loop abort', got %q", oe.Message)
	}
}

// TEST065: Start a LoopOp with an abort flag already set and verify it immediately returns Aborted
func Test065_loop_op_with_pre_existing_abort(t *testing.T) {
	loopOps := []ops.Op[int]{&abortTestOp{shouldAbort: false}}
	loop := ops.NewLoopOp("test_counter", 2, loopOps)
	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	reason := "Pre-existing loop abort"
	dry.SetAbort(&reason)

	_, err := loop.Perform(dry, wet)
	if err == nil {
		t.Fatal("expected error for pre-existing abort")
	}
	oe := ops.AsOpError(err)
	if oe == nil || oe.Kind != ops.ErrAborted {
		t.Errorf("expected ErrAborted, got %v", err)
	}
	if oe.Message != "Pre-existing loop abort" {
		t.Errorf("expected message 'Pre-existing loop abort', got %q", oe.Message)
	}
}

// TEST066: Nest a batch with a continue op inside a loop and verify results across all iterations
func Test066_complex_control_flow_scenario(t *testing.T) {
	// Batch of [continueTestOpCF(100, false), continueTestOpCF(200, true)]
	batchOps := []ops.Op[int]{
		&continueTestOpCF{shouldContinue: false, value: 100},
		&continueTestOpCF{shouldContinue: true, value: 200}, // sets continue flag, returns 0
	}
	innerBatch := ops.NewBatchOp(batchOps)

	// Loop over [innerBatch] 2 times; innerBatch is Op[[]int]
	loopOps := []ops.Op[[]int]{innerBatch}
	loop := ops.NewLoopOp("complex_counter", 2, loopOps)
	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	results, err := loop.Perform(dry, wet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Each iteration: innerBatch produces [100, 0], loop stores it as one result
	// 2 iterations = 2 results (each is []int)
	if len(results) != 2 {
		t.Fatalf("expected 2 results from loop, got %d", len(results))
	}

	for i, batchResult := range results {
		if len(batchResult) != 2 {
			t.Errorf("results[%d]: expected len=2, got %d: %v", i, len(batchResult), batchResult)
			continue
		}
		if batchResult[0] != 100 {
			t.Errorf("results[%d][0]: expected 100, got %d", i, batchResult[0])
		}
		if batchResult[1] != 0 {
			t.Errorf("results[%d][1]: expected 0 (from continue), got %d", i, batchResult[1])
		}
	}
}
