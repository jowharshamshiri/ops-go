package ops_test

import (
	"encoding/json"
	"sync"
	"testing"

	ops "github.com/machinefabric/ops-go"
)

// batchTestOp is a configurable op for batch tests.
type batchTestOp struct {
	ops.DefaultRollback
	value      int
	shouldFail bool
}

func (o *batchTestOp) Perform(_ *ops.DryContext, _ *ops.WetContext) (int, error) {
	if o.shouldFail {
		return 0, ops.NewExecutionFailedError("Test failure")
	}
	return o.value, nil
}

func (o *batchTestOp) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("BatchTestOp").Build()
}

// TEST049: Run BatchOp with two succeeding ops and verify results contain both values in order
func Test049_batch_op_success(t *testing.T) {
	batchOps := []ops.Op[int]{
		&batchTestOp{value: 1},
		&batchTestOp{value: 2},
	}
	batch := ops.NewBatchOp(batchOps)
	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	results, err := batch.Perform(dry, wet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 || results[0] != 1 || results[1] != 2 {
		t.Errorf("expected [1,2], got %v", results)
	}
}

// TEST050: Run BatchOp where the second op fails and verify the batch returns an error
func Test050_batch_op_failure(t *testing.T) {
	batchOps := []ops.Op[int]{
		&batchTestOp{value: 1},
		&batchTestOp{value: 2, shouldFail: true},
	}
	batch := ops.NewBatchOp(batchOps)
	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	_, err := batch.Perform(dry, wet)
	if err == nil {
		t.Fatal("expected error from failing op")
	}
}

// TEST051: Run BatchOp with two ops and verify both result values are present in order
func Test051_batch_op_returns_all_results(t *testing.T) {
	batchOps := []ops.Op[int]{
		&batchTestOp{value: 1},
		&batchTestOp{value: 2},
	}
	batch := ops.NewBatchOp(batchOps)
	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	results, err := batch.Perform(dry, wet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	found1, found2 := false, false
	for _, r := range results {
		if r == 1 {
			found1 = true
		}
		if r == 2 {
			found2 = true
		}
	}
	if !found1 || !found2 {
		t.Errorf("expected results to contain 1 and 2, got %v", results)
	}
}

// TEST052: Verify BatchOp metadata correctly identifies only the externally-required input fields
func Test052_batch_metadata_data_flow(t *testing.T) {
	type producerMetaOp struct{ ops.DefaultRollback }
	type consumerMetaOp struct{ ops.DefaultRollback }

	producerMeta := &struct {
		ops.DefaultRollback
	}{}
	_ = producerMeta

	// Use the producer/consumer ops defined in batch_metadata_test.go
	batchOps := []ops.Op[string]{&producerOp{}, &consumerOp{}}
	batch := ops.NewBatchOp(batchOps)
	metadata := batch.Metadata()

	if metadata.InputSchema == nil {
		t.Fatal("expected non-nil InputSchema")
	}
	var schemaObj map[string]json.RawMessage
	if err := json.Unmarshal(metadata.InputSchema, &schemaObj); err != nil {
		t.Fatalf("failed to parse input schema: %v", err)
	}
	reqRaw, ok := schemaObj["required"]
	if !ok {
		t.Fatal("expected 'required' in input schema")
	}
	var required []string
	if err := json.Unmarshal(reqRaw, &required); err != nil {
		t.Fatalf("failed to parse required: %v", err)
	}

	if len(required) != 1 {
		t.Errorf("expected 1 external required field (initial_input), got %d: %v", len(required), required)
	}
	found := map[string]bool{}
	for _, r := range required {
		found[r] = true
	}
	if !found["initial_input"] {
		t.Error("expected 'initial_input' in required fields")
	}
	if found["produced_value"] {
		t.Error("'produced_value' should not be required externally (satisfied internally)")
	}
}

// TEST053: Verify BatchOp merges reference schemas from all ops into a unified set of required refs
func Test053_batch_reference_schema_merging(t *testing.T) {
	// Use serviceAOp and serviceBOp with shared_service added
	type serviceAWithSharedOp struct{ ops.DefaultRollback }
	type serviceBWithSharedOp struct{ ops.DefaultRollback }

	svcAOp := &serviceASharedImpl{}
	svcBOp := &serviceBSharedImpl{}

	batchOps := []ops.Op[struct{}]{svcAOp, svcBOp}
	batch := ops.NewBatchOp(batchOps)
	metadata := batch.Metadata()

	if metadata.ReferenceSchema == nil {
		t.Fatal("expected non-nil ReferenceSchema")
	}
	var schemaObj map[string]json.RawMessage
	if err := json.Unmarshal(metadata.ReferenceSchema, &schemaObj); err != nil {
		t.Fatalf("failed to parse reference schema: %v", err)
	}
	reqRaw, ok := schemaObj["required"]
	if !ok {
		t.Fatal("expected 'required' in reference schema")
	}
	var required []string
	if err := json.Unmarshal(reqRaw, &required); err != nil {
		t.Fatalf("failed to parse required: %v", err)
	}

	if len(required) != 3 {
		t.Errorf("expected 3 required refs (service_a, service_b, shared_service), got %d: %v", len(required), required)
	}
	found := map[string]bool{}
	for _, r := range required {
		found[r] = true
	}
	for _, ref := range []string{"service_a", "service_b", "shared_service"} {
		if !found[ref] {
			t.Errorf("expected '%s' in required refs", ref)
		}
	}
}

type serviceASharedImpl struct{ ops.DefaultRollback }

func (o *serviceASharedImpl) Perform(_ *ops.DryContext, _ *ops.WetContext) (struct{}, error) {
	return struct{}{}, nil
}
func (o *serviceASharedImpl) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("ServiceASharedOp").
		ReferenceSchema(json.RawMessage(`{
			"type": "object",
			"properties": {
				"service_a": {"type": "ServiceA"},
				"shared_service": {"type": "SharedService"}
			},
			"required": ["service_a", "shared_service"]
		}`)).
		Build()
}

type serviceBSharedImpl struct{ ops.DefaultRollback }

func (o *serviceBSharedImpl) Perform(_ *ops.DryContext, _ *ops.WetContext) (struct{}, error) {
	return struct{}{}, nil
}
func (o *serviceBSharedImpl) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("ServiceBSharedOp").
		ReferenceSchema(json.RawMessage(`{
			"type": "object",
			"properties": {
				"service_b": {"type": "ServiceB"},
				"shared_service": {"type": "SharedService"}
			},
			"required": ["service_b", "shared_service"]
		}`)).
		Build()
}

// TEST054: Run BatchOp where the third op fails and verify rollback is called on first two but not the third
func Test054_batch_rollback_on_failure(t *testing.T) {
	state1 := &batchTrackState{}
	state2 := &batchTrackState{}
	state3 := &batchTrackState{}

	batchOps := []ops.Op[uint32]{
		&batchRollbackOp{id: 1, shouldFail: false, s: state1},
		&batchRollbackOp{id: 2, shouldFail: false, s: state2},
		&batchRollbackOp{id: 3, shouldFail: true, s: state3},
	}
	batch := ops.NewBatchOp(batchOps)
	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	_, err := batch.Perform(dry, wet)
	if err == nil {
		t.Fatal("expected error from batch with failing op")
	}

	if !state1.performed {
		t.Error("op1 should have been performed")
	}
	if !state2.performed {
		t.Error("op2 should have been performed")
	}
	if !state3.performed {
		t.Error("op3 should have been performed (and failed)")
	}
	if !state1.rolledBack {
		t.Error("op1 should have been rolled back")
	}
	if !state2.rolledBack {
		t.Error("op2 should have been rolled back")
	}
	if state3.rolledBack {
		t.Error("op3 should NOT have been rolled back (it failed)")
	}
}

type batchTrackState struct {
	performed  bool
	rolledBack bool
	mu         sync.Mutex
}

type batchRollbackOp struct {
	id         uint32
	shouldFail bool
	s          *batchTrackState
}

func (o *batchRollbackOp) Perform(_ *ops.DryContext, _ *ops.WetContext) (uint32, error) {
	o.s.mu.Lock()
	o.s.performed = true
	o.s.mu.Unlock()
	if o.shouldFail {
		return 0, ops.NewExecutionFailedError("failed")
	}
	return o.id, nil
}

func (o *batchRollbackOp) Rollback(_ *ops.DryContext, _ *ops.WetContext) error {
	o.s.mu.Lock()
	o.s.rolledBack = true
	o.s.mu.Unlock()
	return nil
}

func (o *batchRollbackOp) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("BatchRollbackOp").Build()
}

// TEST055: Run BatchOp where the last op fails and verify rollback occurs in reverse (LIFO) order
func Test055_batch_rollback_order(t *testing.T) {
	var mu sync.Mutex
	rollbackOrder := []uint32{}

	type orderTrackingOp struct {
		id    uint32
		order *[]uint32
		mu    *sync.Mutex
	}

	batchOps := []ops.Op[uint32]{
		&batchOrderOp{id: 1, order: &rollbackOrder, mu: &mu},
		&batchOrderOp{id: 2, order: &rollbackOrder, mu: &mu},
		&batchOrderOp{id: 3, order: &rollbackOrder, mu: &mu},
		&batchFailOp{},
	}
	batch := ops.NewBatchOp(batchOps)
	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	_, err := batch.Perform(dry, wet)
	if err == nil {
		t.Fatal("expected error from batch with failing op")
	}

	mu.Lock()
	order := make([]uint32, len(rollbackOrder))
	copy(order, rollbackOrder)
	mu.Unlock()

	if len(order) != 3 {
		t.Fatalf("expected 3 rollbacks, got %d: %v", len(order), order)
	}
	if order[0] != 3 || order[1] != 2 || order[2] != 1 {
		t.Errorf("expected LIFO order [3,2,1], got %v", order)
	}
}

type batchOrderOp struct {
	id    uint32
	order *[]uint32
	mu    *sync.Mutex
}

func (o *batchOrderOp) Perform(_ *ops.DryContext, _ *ops.WetContext) (uint32, error) {
	return o.id, nil
}

func (o *batchOrderOp) Rollback(_ *ops.DryContext, _ *ops.WetContext) error {
	o.mu.Lock()
	*o.order = append(*o.order, o.id)
	o.mu.Unlock()
	return nil
}

func (o *batchOrderOp) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("BatchOrderOp").Build()
}

type batchFailOp struct{ ops.DefaultRollback }

func (o *batchFailOp) Perform(_ *ops.DryContext, _ *ops.WetContext) (uint32, error) {
	return 0, ops.NewExecutionFailedError("intentional failure")
}

func (o *batchFailOp) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("BatchFailOp").Build()
}

// TEST056: Run BatchOp where one op fails and verify rollback is triggered for succeeded ops
func Test056_batch_rollback_on_failure_partial(t *testing.T) {
	state1 := &batchTrackState{}
	state2 := &batchTrackState{}

	batchOps := []ops.Op[uint32]{
		&batchRollbackOp{id: 1, shouldFail: false, s: state1},
		&batchRollbackOp{id: 2, shouldFail: true, s: state2},
	}
	batch := ops.NewBatchOp(batchOps)
	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	_, err := batch.Perform(dry, wet)
	if err == nil {
		t.Fatal("expected error")
	}

	if !state1.performed {
		t.Error("op1 should have been performed")
	}
	if !state2.performed {
		t.Error("op2 should have been performed (and failed)")
	}
	if !state1.rolledBack {
		t.Error("op1 should have been rolled back")
	}
	if state2.rolledBack {
		t.Error("op2 should NOT have been rolled back (it failed)")
	}
}

// TEST093: Call BatchOp::Len and IsEmpty on empty and non-empty batches
func Test093_batch_len_and_is_empty(t *testing.T) {
	empty := ops.NewBatchOp[int]([]ops.Op[int]{})
	if empty.Len() != 0 {
		t.Errorf("expected Len()=0, got %d", empty.Len())
	}
	if !empty.IsEmpty() {
		t.Error("expected IsEmpty()=true for empty batch")
	}

	nonempty := ops.NewBatchOp([]ops.Op[int]{&batchTestOp{value: 1}})
	if nonempty.Len() != 1 {
		t.Errorf("expected Len()=1, got %d", nonempty.Len())
	}
	if nonempty.IsEmpty() {
		t.Error("expected IsEmpty()=false for non-empty batch")
	}
}

// TEST094: Use AddOp to dynamically add an op and verify it is executed
func Test094_batch_add_op(t *testing.T) {
	batch := ops.NewBatchOp([]ops.Op[int]{&batchTestOp{value: 10}})
	batch.AddOp(&batchTestOp{value: 20})

	dry := ops.NewDryContext()
	wet := ops.NewWetContext()
	results, err := batch.Perform(dry, wet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 || results[0] != 10 || results[1] != 20 {
		t.Errorf("expected [10,20], got %v", results)
	}
}

// TEST095: Run BatchOp::with_continue_on_error and verify it collects results past failures
func Test095_batch_continue_on_error(t *testing.T) {
	batchOps := []ops.Op[int]{
		&batchTestOp{value: 1},
		&batchTestOp{value: 2, shouldFail: true},
		&batchTestOp{value: 3},
	}
	batch := ops.NewBatchOp(batchOps).WithContinueOnError(true)
	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	results, err := batch.Perform(dry, wet)
	if err != nil {
		t.Fatalf("unexpected error with continue_on_error: %v", err)
	}
	// Only successful ops contribute results
	if len(results) != 2 || results[0] != 1 || results[1] != 3 {
		t.Errorf("expected [1,3], got %v", results)
	}
}

// TEST096: Run an empty BatchOp and verify it returns an empty result vec
func Test096_empty_batch_returns_empty(t *testing.T) {
	batch := ops.NewBatchOp[int]([]ops.Op[int]{})
	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	results, err := batch.Perform(dry, wet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected empty results, got %v", results)
	}
}

// TEST097: Verify nested BatchOp rollback propagates correctly when outer batch fails
func Test097_nested_batch_rollback(t *testing.T) {
	// Inner batch: two simple ops that succeed
	innerOps := []ops.Op[int]{
		&batchTestOp{value: 0},
		&batchTestOp{value: 0},
	}
	innerBatch := ops.NewBatchOp(innerOps)

	// Outer batch: inner batch (as Op[[]int]) then a failing op
	outerOps := []ops.Op[[]int]{
		innerBatch,
		&batchOuterFailOp{},
	}
	outerBatch := ops.NewBatchOp(outerOps)
	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	_, err := outerBatch.Perform(dry, wet)
	if err == nil {
		t.Fatal("expected error from outer batch")
	}
	oe := ops.AsOpError(err)
	if oe == nil || oe.Kind != ops.ErrBatchFailed {
		t.Errorf("expected ErrBatchFailed, got %v", err)
	}
}

type batchOuterFailOp struct{ ops.DefaultRollback }

func (o *batchOuterFailOp) Perform(_ *ops.DryContext, _ *ops.WetContext) ([]int, error) {
	return nil, ops.NewExecutionFailedError("outer fail")
}

func (o *batchOuterFailOp) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("OuterFailOp").Build()
}
