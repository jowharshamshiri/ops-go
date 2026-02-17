package ops_test

import (
	"sync"
	"testing"

	ops "github.com/filegrind/ops-go"
)

// loopTestOpInt returns a fixed value each iteration.
type loopTestOpInt struct {
	ops.DefaultRollback
	value int
}

func (o *loopTestOpInt) Perform(_ *ops.DryContext, _ *ops.WetContext) (int, error) {
	return o.value, nil
}

func (o *loopTestOpInt) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("LoopTestOp").Build()
}

// counterReadOp reads the loop counter from context.
type counterReadOp struct{ ops.DefaultRollback }

func (o *counterReadOp) Perform(dry *ops.DryContext, _ *ops.WetContext) (int, error) {
	counter, _ := ops.DryGet[int](dry, "loop_counter")
	return counter, nil
}

func (o *counterReadOp) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("CounterReadOp").Build()
}

// TEST067: Run a LoopOp for 3 iterations with 2 ops each and verify all 6 results in order
func Test067_loop_op_basic(t *testing.T) {
	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	loopOps := []ops.Op[int]{
		&loopTestOpInt{value: 10},
		&loopTestOpInt{value: 20},
	}
	loop := ops.NewLoopOp("loop_counter", 3, loopOps)
	results, err := loop.Perform(dry, wet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 6 {
		t.Fatalf("expected 6 results (2 ops × 3 iters), got %d", len(results))
	}
	expected := []int{10, 20, 10, 20, 10, 20}
	for i, r := range results {
		if r != expected[i] {
			t.Errorf("results[%d]: expected %d, got %d", i, expected[i], r)
		}
	}
}

// TEST068: Run a LoopOp where each op reads the loop counter and verify values are 0, 1, 2
func Test068_loop_op_with_counter_access(t *testing.T) {
	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	loopOps := []ops.Op[int]{&counterReadOp{}}
	loop := ops.NewLoopOp("loop_counter", 3, loopOps)
	results, err := loop.Perform(dry, wet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []int{0, 1, 2}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	for i, r := range results {
		if r != expected[i] {
			t.Errorf("results[%d]: expected %d, got %d", i, expected[i], r)
		}
	}
}

// TEST069: Start a LoopOp with a pre-initialized counter and verify it only executes the remaining iterations
func Test069_loop_op_existing_counter(t *testing.T) {
	dry := ops.NewDryContext().WithValue("my_counter", 2)
	wet := ops.NewWetContext()

	loopOps := []ops.Op[int]{&loopTestOpInt{value: 42}}
	loop := ops.NewLoopOp("my_counter", 4, loopOps)
	results, err := loop.Perform(dry, wet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Counter starts at 2, limit is 4 → 2 iterations
	if len(results) != 2 {
		t.Fatalf("expected 2 results (counter 2..4), got %d", len(results))
	}
	if results[0] != 42 || results[1] != 42 {
		t.Errorf("expected [42, 42], got %v", results)
	}
}

// TEST070: Run a LoopOp with a zero iteration limit and verify no ops are executed
func Test070_loop_op_zero_limit(t *testing.T) {
	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	loopOps := []ops.Op[int]{&loopTestOpInt{value: 99}}
	loop := ops.NewLoopOp("counter", 0, loopOps)
	results, err := loop.Perform(dry, wet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for zero limit, got %d: %v", len(results), results)
	}
}

// TEST071: Build a LoopOp with AddOp chaining and verify all added ops run across all iterations
func Test071_loop_op_builder_pattern(t *testing.T) {
	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	loop := ops.NewLoopOp[int]("builder_counter", 2, []ops.Op[int]{}).
		AddOp(&loopTestOpInt{value: 1}).
		AddOp(&loopTestOpInt{value: 2})

	results, err := loop.Perform(dry, wet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 4 {
		t.Fatalf("expected 4 results (2 ops × 2 iters), got %d", len(results))
	}
	expected := []int{1, 2, 1, 2}
	for i, r := range results {
		if r != expected[i] {
			t.Errorf("results[%d]: expected %d, got %d", i, expected[i], r)
		}
	}
}

// TEST072: Run a LoopOp where the third op fails and verify succeeded ops are rolled back in reverse order
func Test072_loop_op_rollback_on_iteration_failure(t *testing.T) {
	var mu sync.Mutex
	performed := []uint32{}
	rolledBack := []uint32{}

	loopOps := []ops.Op[uint32]{
		&loopRollbackOp{id: 1, shouldFail: false, performed: &performed, rolledBack: &rolledBack, mu: &mu},
		&loopRollbackOp{id: 2, shouldFail: false, performed: &performed, rolledBack: &rolledBack, mu: &mu},
		&loopRollbackOp{id: 3, shouldFail: true, performed: &performed, rolledBack: &rolledBack, mu: &mu},
	}
	loop := ops.NewLoopOp("test_counter", 2, loopOps)
	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	_, err := loop.Perform(dry, wet)
	if err == nil {
		t.Fatal("expected error from failing op")
	}

	mu.Lock()
	perf := make([]uint32, len(performed))
	copy(perf, performed)
	rb := make([]uint32, len(rolledBack))
	copy(rb, rolledBack)
	mu.Unlock()

	if len(perf) != 3 || perf[0] != 1 || perf[1] != 2 || perf[2] != 3 {
		t.Errorf("expected performed=[1,2,3], got %v", perf)
	}
	// Only ops 1 and 2 succeeded (op 3 failed); rolled back in LIFO order
	if len(rb) != 2 || rb[0] != 2 || rb[1] != 1 {
		t.Errorf("expected rolledBack=[2,1], got %v", rb)
	}
}

type loopRollbackOp struct {
	id         uint32
	shouldFail bool
	performed  *[]uint32
	rolledBack *[]uint32
	mu         *sync.Mutex
}

func (o *loopRollbackOp) Perform(_ *ops.DryContext, _ *ops.WetContext) (uint32, error) {
	o.mu.Lock()
	*o.performed = append(*o.performed, o.id)
	o.mu.Unlock()
	if o.shouldFail {
		return 0, ops.NewExecutionFailedError("failed")
	}
	return o.id, nil
}

func (o *loopRollbackOp) Rollback(_ *ops.DryContext, _ *ops.WetContext) error {
	o.mu.Lock()
	*o.rolledBack = append(*o.rolledBack, o.id)
	o.mu.Unlock()
	return nil
}

func (o *loopRollbackOp) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("LoopRollbackOp").Build()
}

// TEST073: Run a LoopOp where the last op fails and verify rollback occurs in LIFO order within the iteration
func Test073_loop_op_rollback_order_within_iteration(t *testing.T) {
	var mu sync.Mutex
	rollbackOrder := []uint32{}

	type orderOp struct {
		id    uint32
		order *[]uint32
		mu    *sync.Mutex
	}

	loopOps := []ops.Op[uint32]{
		&loopOrderOp{id: 1, order: &rollbackOrder, mu: &mu},
		&loopOrderOp{id: 2, order: &rollbackOrder, mu: &mu},
		&loopOrderOp{id: 3, order: &rollbackOrder, mu: &mu},
		&loopFailOp{},
	}
	loop := ops.NewLoopOp("test_counter", 1, loopOps)
	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	_, err := loop.Perform(dry, wet)
	if err == nil {
		t.Fatal("expected error from failing op")
	}

	mu.Lock()
	order := make([]uint32, len(rollbackOrder))
	copy(order, rollbackOrder)
	mu.Unlock()

	if len(order) != 3 || order[0] != 3 || order[1] != 2 || order[2] != 1 {
		t.Errorf("expected LIFO rollback [3,2,1], got %v", order)
	}
}

type loopOrderOp struct {
	id    uint32
	order *[]uint32
	mu    *sync.Mutex
}

func (o *loopOrderOp) Perform(_ *ops.DryContext, _ *ops.WetContext) (uint32, error) { return o.id, nil }
func (o *loopOrderOp) Rollback(_ *ops.DryContext, _ *ops.WetContext) error {
	o.mu.Lock()
	*o.order = append(*o.order, o.id)
	o.mu.Unlock()
	return nil
}
func (o *loopOrderOp) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("LoopOrderOp").Build()
}

type loopFailOp struct{ ops.DefaultRollback }

func (o *loopFailOp) Perform(_ *ops.DryContext, _ *ops.WetContext) (uint32, error) {
	return 0, ops.NewExecutionFailedError("intentional failure")
}
func (o *loopFailOp) Metadata() ops.OpMetadata { return ops.NewOpMetadataBuilder("LoopFailOp").Build() }

// TEST074: Run a LoopOp that fails on iteration 2 and verify previously completed iterations are not rolled back
func Test074_loop_op_successful_iterations_not_rolled_back(t *testing.T) {
	var mu sync.Mutex
	performedIters := []int{}
	rolledBackIters := []int{}

	loopOps := []ops.Op[uint32]{
		&iterTrackOp{id: 1, failOnIter: 2, counterVar: "test_counter", performed: &performedIters, rolledBack: &rolledBackIters, mu: &mu},
	}
	loop := ops.NewLoopOp("test_counter", 5, loopOps)
	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	_, err := loop.Perform(dry, wet)
	if err == nil {
		t.Fatal("expected error from failing op")
	}

	mu.Lock()
	perf := make([]int, len(performedIters))
	copy(perf, performedIters)
	rb := make([]int, len(rolledBackIters))
	copy(rb, rolledBackIters)
	mu.Unlock()

	// Should have run iterations 0, 1, 2
	if len(perf) != 3 || perf[0] != 0 || perf[1] != 1 || perf[2] != 2 {
		t.Errorf("expected performed=[0,1,2], got %v", perf)
	}
	// The failing op itself was not successfully performed, so no rollback
	if len(rb) != 0 {
		t.Errorf("expected no rollbacks (failing op was not in succeeded list), got %v", rb)
	}
}

type iterTrackOp struct {
	id         uint32
	failOnIter int
	counterVar string
	performed  *[]int
	rolledBack *[]int
	mu         *sync.Mutex
}

func (o *iterTrackOp) Perform(dry *ops.DryContext, _ *ops.WetContext) (uint32, error) {
	counter, _ := ops.DryGet[int](dry, o.counterVar)
	o.mu.Lock()
	*o.performed = append(*o.performed, counter)
	o.mu.Unlock()
	if counter == o.failOnIter {
		return 0, ops.NewExecutionFailedError("failed on iteration")
	}
	return o.id, nil
}

func (o *iterTrackOp) Rollback(dry *ops.DryContext, _ *ops.WetContext) error {
	counter, _ := ops.DryGet[int](dry, o.counterVar)
	o.mu.Lock()
	*o.rolledBack = append(*o.rolledBack, counter)
	o.mu.Unlock()
	return nil
}

func (o *iterTrackOp) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("IterTrackOp").Build()
}

// TEST075: Run a LoopOp where op2 fails on iteration 1 and verify only op1 from that iteration is rolled back
func Test075_loop_op_mixed_iteration_with_rollback(t *testing.T) {
	var mu sync.Mutex
	performedOps := []perfEntry{}
	rolledBackOps := []perfEntry{}

	loopOps := []ops.Op[uint32]{
		&mixedIterOp{id: 1, failOnIter: -1, counterVar: "test_counter", performed: &performedOps, rolledBack: &rolledBackOps, mu: &mu},
		&mixedIterOp{id: 2, failOnIter: 1, counterVar: "test_counter", performed: &performedOps, rolledBack: &rolledBackOps, mu: &mu},
	}
	loop := ops.NewLoopOp("test_counter", 3, loopOps)
	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	_, err := loop.Perform(dry, wet)
	if err == nil {
		t.Fatal("expected error")
	}

	mu.Lock()
	perf := make([]perfEntry, len(performedOps))
	copy(perf, performedOps)
	rb := make([]perfEntry, len(rolledBackOps))
	copy(rb, rolledBackOps)
	mu.Unlock()

	// iter 0: op1(1,0), op2(2,0) both succeed
	// iter 1: op1(1,1) succeeds, op2(2,1) fails
	expected := []perfEntry{{1, 0}, {2, 0}, {1, 1}, {2, 1}}
	if len(perf) != len(expected) {
		t.Fatalf("expected %d performed, got %d: %v", len(expected), len(perf), perf)
	}
	for i, e := range expected {
		if perf[i].opID != e.opID || perf[i].iter != e.iter {
			t.Errorf("performed[%d]: expected {%d,%d}, got {%d,%d}", i, e.opID, e.iter, perf[i].opID, perf[i].iter)
		}
	}

	// Only op1 from iteration 1 should be rolled back
	if len(rb) != 1 || rb[0].opID != 1 || rb[0].iter != 1 {
		t.Errorf("expected rolledBack=[{1,1}], got %v", rb)
	}
}

type perfEntry struct{ opID uint32; iter int }

type mixedIterOp struct {
	id         uint32
	failOnIter int // -1 means never fail
	counterVar string
	performed  *[]perfEntry
	rolledBack *[]perfEntry
	mu         *sync.Mutex
}

func (o *mixedIterOp) Perform(dry *ops.DryContext, _ *ops.WetContext) (uint32, error) {
	counter, _ := ops.DryGet[int](dry, o.counterVar)
	o.mu.Lock()
	*o.performed = append(*o.performed, perfEntry{o.id, counter})
	o.mu.Unlock()
	if o.failOnIter >= 0 && counter == o.failOnIter {
		return 0, ops.NewExecutionFailedError("failed on iteration")
	}
	return o.id, nil
}

func (o *mixedIterOp) Rollback(dry *ops.DryContext, _ *ops.WetContext) error {
	counter, _ := ops.DryGet[int](dry, o.counterVar)
	o.mu.Lock()
	*o.rolledBack = append(*o.rolledBack, perfEntry{o.id, counter})
	o.mu.Unlock()
	return nil
}

func (o *mixedIterOp) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("MixedIterOp").Build()
}

// TEST076: Run a LoopOp configured to continue on error and verify subsequent iterations still execute
func Test076_loop_op_continue_on_error(t *testing.T) {
	var mu sync.Mutex
	performedOps := []ceEntry{}
	rolledBackOps := []ceEntry{}

	loopOps := []ops.Op[uint32]{
		&continueOnErrOp{id: 1, failOnIter: -1, counterVar: "test_counter", performed: &performedOps, rolledBack: &rolledBackOps, mu: &mu},
		&continueOnErrOp{id: 2, failOnIter: 1, counterVar: "test_counter", performed: &performedOps, rolledBack: &rolledBackOps, mu: &mu},
	}
	loop := ops.NewLoopOp("test_counter", 3, loopOps).WithContinueOnError(true)
	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	results, err := loop.Perform(dry, wet)
	if err != nil {
		t.Fatalf("unexpected error with continue_on_error: %v", err)
	}

	mu.Lock()
	perf := make([]ceEntry, len(performedOps))
	copy(perf, performedOps)
	rb := make([]ceEntry, len(rolledBackOps))
	copy(rb, rolledBackOps)
	mu.Unlock()

	// iter0: op1(1,0)✓, op2(2,0)✓ → [1,2]
	// iter1: op1(1,1)✓, op2(2,1)✗ (continue) → rollback op1 → [1]
	// iter2: op1(1,2)✓, op2(2,2)✓ → [1,2]
	expectedPerf := []ceEntry{{1, 0}, {2, 0}, {1, 1}, {2, 1}, {1, 2}, {2, 2}}
	if len(perf) != len(expectedPerf) {
		t.Fatalf("expected %d performed, got %d: %v", len(expectedPerf), len(perf), perf)
	}
	for i, e := range expectedPerf {
		if perf[i].opID != e.opID || perf[i].iter != e.iter {
			t.Errorf("performed[%d]: expected {%d,%d}, got {%d,%d}", i, e.opID, e.iter, perf[i].opID, perf[i].iter)
		}
	}

	// Results: [1,2,1,1,2] (iter1 op2 failed, so only op1 result from iter1)
	expected := []uint32{1, 2, 1, 1, 2}
	if len(results) != len(expected) {
		t.Fatalf("expected %d results, got %d: %v", len(expected), len(results), results)
	}
	for i, r := range results {
		if r != expected[i] {
			t.Errorf("results[%d]: expected %d, got %d", i, expected[i], r)
		}
	}

	// Only op1 from iteration 1 rolled back
	if len(rb) != 1 || rb[0].opID != 1 || rb[0].iter != 1 {
		t.Errorf("expected rolledBack=[{1,1}], got %v", rb)
	}
}

type ceEntry struct{ opID uint32; iter int }

type continueOnErrOp struct {
	id         uint32
	failOnIter int
	counterVar string
	performed  *[]ceEntry
	rolledBack *[]ceEntry
	mu         *sync.Mutex
}

func (o *continueOnErrOp) Perform(dry *ops.DryContext, _ *ops.WetContext) (uint32, error) {
	counter, _ := ops.DryGet[int](dry, o.counterVar)
	o.mu.Lock()
	*o.performed = append(*o.performed, ceEntry{o.id, counter})
	o.mu.Unlock()
	if o.failOnIter >= 0 && counter == o.failOnIter {
		return 0, ops.NewExecutionFailedError("failed on iteration")
	}
	return o.id, nil
}

func (o *continueOnErrOp) Rollback(dry *ops.DryContext, _ *ops.WetContext) error {
	counter, _ := ops.DryGet[int](dry, o.counterVar)
	o.mu.Lock()
	*o.rolledBack = append(*o.rolledBack, ceEntry{o.id, counter})
	o.mu.Unlock()
	return nil
}

func (o *continueOnErrOp) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("ContinueOnErrOp").Build()
}

// breakOpLT sets the break flag for the current loop.
type breakOpLT struct {
	ops.DefaultRollback
	shouldBreak bool
	value       int
}

func (o *breakOpLT) Perform(dry *ops.DryContext, _ *ops.WetContext) (int, error) {
	if o.shouldBreak {
		ops.BreakLoop(dry)
	}
	return o.value, nil
}

func (o *breakOpLT) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("BreakOp").Build()
}

// TEST113: Run a LoopOp where an op sets the break flag and verify the loop terminates early
func Test113_loop_op_break_terminates_loop(t *testing.T) {
	loopOps := []ops.Op[int]{
		&loopTestOpInt{value: 10},
		&breakOpLT{shouldBreak: true, value: 99},
		&loopTestOpInt{value: 20}, // should NOT execute after break
	}
	loop := ops.NewLoopOp("counter", 5, loopOps)
	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	results, err := loop.Perform(dry, wet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Only TestOp(10) and BreakOp(99) from iteration 0, then loop stops
	if len(results) != 2 || results[0] != 10 || results[1] != 99 {
		t.Errorf("expected [10, 99], got %v", results)
	}
}

// TEST114: Run LoopOp with_continue_on_error where an op fails and verify the loop continues
func Test114_loop_op_continue_on_error_skips_failed_iterations(t *testing.T) {
	var mu sync.Mutex
	itersSeen := []int{}

	type iterLogOp struct {
		failOn int // -1 = never
		log    *[]int
		mu     *sync.Mutex
	}

	loopOps := []ops.Op[int]{
		&iterLogOpImpl{failOn: 1, counterVar: "it_counter", log: &itersSeen, mu: &mu},
	}
	loop := ops.NewLoopOp("it_counter", 4, loopOps).WithContinueOnError(true)
	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	results, err := loop.Perform(dry, wet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mu.Lock()
	seen := make([]int, len(itersSeen))
	copy(seen, itersSeen)
	mu.Unlock()

	// All 4 iterations attempted despite failure on iteration 1
	if len(seen) != 4 || seen[0] != 0 || seen[1] != 1 || seen[2] != 2 || seen[3] != 3 {
		t.Errorf("expected seen=[0,1,2,3], got %v", seen)
	}
	// Iteration 1 produced no result (failed), others did: [0, 2, 3]
	if len(results) != 3 || results[0] != 0 || results[1] != 2 || results[2] != 3 {
		t.Errorf("expected results=[0,2,3], got %v", results)
	}
}

type iterLogOpImpl struct {
	failOn     int
	counterVar string
	log        *[]int
	mu         *sync.Mutex
}

func (o *iterLogOpImpl) Perform(dry *ops.DryContext, _ *ops.WetContext) (int, error) {
	counter, _ := ops.DryGet[int](dry, o.counterVar)
	o.mu.Lock()
	*o.log = append(*o.log, counter)
	o.mu.Unlock()
	if counter == o.failOn {
		return 0, ops.NewExecutionFailedError("failed")
	}
	return counter, nil
}

func (o *iterLogOpImpl) Rollback(_ *ops.DryContext, _ *ops.WetContext) error { return nil }
func (o *iterLogOpImpl) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("IterLogOp").Build()
}

// TEST115: Run an empty LoopOp with a non-zero limit and verify it produces no results
func Test115_loop_op_with_no_ops_produces_no_results(t *testing.T) {
	loop := ops.NewLoopOp[int]("counter", 5, []ops.Op[int]{})
	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	results, err := loop.Perform(dry, wet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty loop body, got %d", len(results))
	}
	// Counter advances to limit
	counter, ok := ops.DryGet[int](dry, "counter")
	if !ok || counter != 5 {
		t.Errorf("expected counter=5, got %d ok=%v", counter, ok)
	}
}
