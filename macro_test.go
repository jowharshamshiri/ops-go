package ops_test

import (
	"fmt"
	"testing"

	ops "github.com/filegrind/ops-go"
)

// macroTestService is used in macro tests.
type macroTestService struct {
	value int
}

// TEST077: Use DryPut and DryGet helper functions to store and retrieve a typed value by variable name
func Test077_dry_put_and_get(t *testing.T) {
	dry := ops.NewDryContext()

	value := 42
	ops.DryPut(dry, "value", value)

	retrieved, ok := ops.DryGet[int](dry, "value")
	if !ok {
		t.Fatal("expected to find 'value' in context")
	}
	if retrieved != 42 {
		t.Errorf("expected 42, got %d", retrieved)
	}
}

// TEST078: Use DryGetRequired to retrieve a required value and verify error when key is missing
func Test078_dry_require(t *testing.T) {
	dry := ops.NewDryContext()

	name := "test"
	ops.DryPut(dry, "name", name)

	result, err := ops.DryGetRequired[string](dry, "name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "test" {
		t.Errorf("expected 'test', got %q", result)
	}

	// Missing value should return an error
	_, err = ops.DryGetRequired[int](dry, "missing_value")
	if err == nil {
		t.Error("expected error for missing key")
	}
}

// TEST079: Use DryResult to store a final result and verify it is stored under both "result" and op name
func Test079_dry_result(t *testing.T) {
	dry := ops.NewDryContext()

	finalValue := "completed"
	ops.DryResult(dry, "TestOp", finalValue)

	resultKey, ok := ops.DryGet[string](dry, "result")
	if !ok || resultKey != "completed" {
		t.Errorf("expected result='completed', got %q ok=%v", resultKey, ok)
	}

	opKey, ok := ops.DryGet[string](dry, "TestOp")
	if !ok || opKey != "completed" {
		t.Errorf("expected TestOp='completed', got %q ok=%v", opKey, ok)
	}
}

// TEST080: Use WetPutRef and WetGetRefRequired to store and retrieve a service reference
func Test080_wet_put_ref_and_require_ref(t *testing.T) {
	wet := ops.NewWetContext()

	service := &macroTestService{value: 100}
	ops.WetPutRef(wet, "service", service)

	retrieved, err := ops.WetGetRefRequired[*macroTestService](wet, "service")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if retrieved.value != 100 {
		t.Errorf("expected value=100, got %d", retrieved.value)
	}
}

// TEST081: Use WetPutArc to store a shared service reference and retrieve it via WetGetRefRequired
func Test081_wet_put_arc(t *testing.T) {
	wet := ops.NewWetContext()

	sharedService := &macroTestService{value: 200}
	ops.WetPutArc(wet, "shared_service", sharedService)

	retrieved, err := ops.WetGetRefRequired[*macroTestService](wet, "shared_service")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if retrieved.value != 200 {
		t.Errorf("expected value=200, got %d", retrieved.value)
	}
}

// macroTestOp uses DryGetRequired and WetGetRefRequired internally.
type macroTestOp struct{ ops.DefaultRollback }

func (o *macroTestOp) Perform(dry *ops.DryContext, wet *ops.WetContext) (string, error) {
	input, err := ops.DryGetRequired[string](dry, "input")
	if err != nil {
		return "", err
	}
	count, err := ops.DryGetRequired[int](dry, "count")
	if err != nil {
		return "", err
	}
	service, err := ops.WetGetRefRequired[*macroTestService](wet, "service")
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s x %d = %d", input, count, service.value), nil
}

func (o *macroTestOp) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("MacroTestOp").
		Description("Tests helper function usage in ops").
		Build()
}

// TEST082: Run a full op that uses DryGetRequired and WetGetRefRequired internally and verify the output
func Test082_helpers_in_op(t *testing.T) {
	dry := ops.NewDryContext()
	ops.DryPut(dry, "input", "test")
	ops.DryPut(dry, "count", 3)

	wet := ops.NewWetContext()
	service := &macroTestService{value: 42}
	ops.WetPutRef(wet, "service", service)

	op := &macroTestOp{}
	result, err := op.Perform(dry, wet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "test x 3 = 42" {
		t.Errorf("expected 'test x 3 = 42', got %q", result)
	}
}
