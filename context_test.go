package ops_test

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"testing"

	ops "github.com/jowharshamshiri/ops-go"
)

// TEST009: Insert typed values into DryContext and verify get/contains work correctly
func Test009_dry_context_basic_operations(t *testing.T) {
	dry := ops.NewDryContext()
	dry.Insert("name", "test")
	dry.Insert("count", 42)

	if name, ok := ops.DryGet[string](dry, "name"); !ok || name != "test" {
		t.Errorf("expected name='test', got %q ok=%v", name, ok)
	}
	if count, ok := ops.DryGet[int](dry, "count"); !ok || count != 42 {
		t.Errorf("expected count=42, got %d ok=%v", count, ok)
	}
	if !dry.Contains("name") {
		t.Error("expected Contains('name')=true")
	}
	if dry.Contains("missing") {
		t.Error("expected Contains('missing')=false")
	}
}

// TEST010: Build a DryContext with chained WithValue calls and verify all values are stored
func Test010_dry_context_builder(t *testing.T) {
	dry := ops.NewDryContext().WithValue("key1", "value1").WithValue("key2", 123)

	if v, ok := ops.DryGet[string](dry, "key1"); !ok || v != "value1" {
		t.Errorf("expected 'value1', got %q ok=%v", v, ok)
	}
	if v, ok := ops.DryGet[int](dry, "key2"); !ok || v != 123 {
		t.Errorf("expected 123, got %d ok=%v", v, ok)
	}
}

type testWetService struct{ name string }

// TEST011: Insert a reference into WetContext and retrieve it by type via WetGetRef
func Test011_wet_context_basic_operations(t *testing.T) {
	wet := ops.NewWetContext()
	wet.InsertRef("service", &testWetService{name: "test"})

	retrieved, ok := ops.WetGetRef[*testWetService](wet, "service")
	if !ok {
		t.Fatal("expected to find service in WetContext")
	}
	if retrieved.name != "test" {
		t.Errorf("expected name='test', got %q", retrieved.name)
	}
}

type testWetService1 struct{}
type testWetService2 struct{}

// TEST012: Build a WetContext with chained WithRef calls and verify contains for each key
func Test012_wet_context_builder(t *testing.T) {
	wet := ops.NewWetContext().WithRef("service1", &testWetService1{}).WithRef("service2", &testWetService2{})

	if !wet.Contains("service1") {
		t.Error("expected Contains('service1')=true")
	}
	if !wet.Contains("service2") {
		t.Error("expected Contains('service2')=true")
	}
}

// TEST013: Confirm get_required succeeds for present keys and returns an error for missing keys
func Test013_required_values(t *testing.T) {
	dry := ops.NewDryContext().WithValue("exists", 42)

	val, err := ops.DryGetRequired[int](dry, "exists")
	if err != nil || val != 42 {
		t.Errorf("expected 42, got %d err=%v", val, err)
	}
	_, err = ops.DryGetRequired[int](dry, "missing")
	if err == nil {
		t.Error("expected error for missing key")
	}
}

// TEST014: Merge two DryContexts and verify values from both are accessible in the target
func Test014_context_merge(t *testing.T) {
	dry1 := ops.NewDryContext().WithValue("a", 1)
	dry2 := ops.NewDryContext().WithValue("b", 2)
	dry1.Merge(dry2)

	if v, ok := ops.DryGet[int](dry1, "a"); !ok || v != 1 {
		t.Errorf("expected a=1, got %d ok=%v", v, ok)
	}
	if v, ok := ops.DryGet[int](dry1, "b"); !ok || v != 2 {
		t.Errorf("expected b=2, got %d ok=%v", v, ok)
	}
}

// TEST015: Verify get_required returns a Type mismatch error when the stored type doesn't match
func Test015_dry_context_type_mismatch_error(t *testing.T) {
	dry := ops.NewDryContext().
		WithValue("count", "not_a_number"). // string stored, int requested
		WithValue("flag", 123)              // number stored, bool requested

	_, err := ops.DryGetRequired[int](dry, "count")
	if err == nil {
		t.Fatal("expected error for string->int mismatch")
	}
	errMsg := err.Error()
	if !strings.Contains(errMsg, "Type mismatch") {
		t.Errorf("expected 'Type mismatch' in error, got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "string") {
		t.Errorf("expected 'string' in error, got: %s", errMsg)
	}

	_, err = ops.DryGetRequired[bool](dry, "flag")
	if err == nil {
		t.Fatal("expected error for number->bool mismatch")
	}
	errMsg = err.Error()
	if !strings.Contains(errMsg, "Type mismatch") {
		t.Errorf("expected 'Type mismatch' in error, got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "number") {
		t.Errorf("expected 'number' in error, got: %s", errMsg)
	}

	// Missing key still gives "not found"
	_, err = ops.DryGetRequired[int](dry, "missing")
	if err == nil {
		t.Fatal("expected error for missing key")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %s", err.Error())
	}
	if strings.Contains(err.Error(), "Type mismatch") {
		t.Error("missing key error should not contain 'Type mismatch'")
	}
}

type wetServiceA struct{ _name string }
type wetServiceB struct{ _id int }

// TEST016: Verify WetContext get_required returns a Type mismatch error when the stored ref type differs
func Test016_wet_context_type_mismatch_error(t *testing.T) {
	wet := ops.NewWetContext()
	wet.InsertRef("service", &wetServiceA{_name: "test"})

	_, err := ops.WetGetRefRequired[*wetServiceB](wet, "service")
	if err == nil {
		t.Fatal("expected type mismatch error")
	}
	errMsg := err.Error()
	if !strings.Contains(errMsg, "Type mismatch") {
		t.Errorf("expected 'Type mismatch' in error, got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "ServiceB") {
		t.Errorf("expected 'ServiceB' in error, got: %s", errMsg)
	}

	_, err = ops.WetGetRefRequired[*wetServiceA](wet, "missing")
	if err == nil {
		t.Fatal("expected error for missing ref")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %s", err.Error())
	}
	if strings.Contains(err.Error(), "Type mismatch") {
		t.Error("missing key error should not contain 'Type mismatch'")
	}
}

// TEST017: Set and clear abort flags on DryContext and verify IsAborted and AbortReason reflect state
func Test017_control_flags(t *testing.T) {
	dry := ops.NewDryContext()

	if dry.IsAborted() {
		t.Error("expected IsAborted=false initially")
	}
	if dry.AbortReason() != nil {
		t.Error("expected AbortReason=nil initially")
	}

	reason := "Test abort reason"
	dry.SetAbort(&reason)
	if !dry.IsAborted() {
		t.Error("expected IsAborted=true after SetAbort")
	}
	if dry.AbortReason() == nil || *dry.AbortReason() != "Test abort reason" {
		t.Errorf("expected AbortReason='Test abort reason', got %v", dry.AbortReason())
	}

	dry.ClearControlFlags()
	if dry.IsAborted() {
		t.Error("expected IsAborted=false after ClearControlFlags")
	}
	if dry.AbortReason() != nil {
		t.Error("expected AbortReason=nil after ClearControlFlags")
	}
}

// TEST018: Merge contexts with abort flags and confirm the target inherits the abort state correctly
func Test018_control_flags_merge(t *testing.T) {
	dry1 := ops.NewDryContext()
	dry2 := ops.NewDryContext()

	reason := "Merged abort"
	dry2.SetAbort(&reason)
	dry1.Merge(dry2)

	if !dry1.IsAborted() {
		t.Error("expected merged abort=true")
	}
	if dry1.AbortReason() == nil || *dry1.AbortReason() != "Merged abort" {
		t.Errorf("expected 'Merged abort', got %v", dry1.AbortReason())
	}

	// Original abort reason is preserved when merging into an already-aborted context.
	dry3 := ops.NewDryContext()
	reason3 := "Original abort"
	dry3.SetAbort(&reason3)

	dry4 := ops.NewDryContext()
	reason4 := "New abort"
	dry4.SetAbort(&reason4)

	dry3.Merge(dry4)
	if dry3.AbortReason() == nil || *dry3.AbortReason() != "Original abort" {
		t.Errorf("expected original abort reason preserved, got %v", dry3.AbortReason())
	}
}

// TEST019: Verify DryGetOrInsertWith inserts when missing and returns existing without calling factory
func Test019_get_or_insert_with(t *testing.T) {
	dry := ops.NewDryContext()

	val, err := ops.DryGetOrInsertWith(dry, "count", func() int { return 42 })
	if err != nil || val != 42 {
		t.Errorf("expected 42, got %d err=%v", val, err)
	}
	if v, _ := ops.DryGet[int](dry, "count"); v != 42 {
		t.Errorf("expected stored count=42, got %d", v)
	}

	factoryCalled := false
	val, err = ops.DryGetOrInsertWith(dry, "count", func() int {
		factoryCalled = true
		return 100
	})
	if err != nil || val != 42 {
		t.Errorf("expected 42 (existing), got %d err=%v", val, err)
	}
	if factoryCalled {
		t.Error("factory should not be called when value already exists")
	}

	nameVal, err := ops.DryGetOrInsertWith(dry, "name", func() string { return "default_name" })
	if err != nil || nameVal != "default_name" {
		t.Errorf("expected 'default_name', got %q err=%v", nameVal, err)
	}
}

// TEST020: Verify DryGetOrComputeWith computes and stores a value using context data
func Test020_get_or_compute_with(t *testing.T) {
	dry := ops.NewDryContext()
	dry.Insert("base_port", 8000)
	dry.Insert("app_name", "test_app")

	computedURL, err := ops.DryGetOrComputeWith(dry, "service_url", func(ctx *ops.DryContext, key string) string {
		basePort, _ := ops.DryGet[int](ctx, "base_port")
		appName, _ := ops.DryGet[string](ctx, "app_name")
		url := appName + ":" + itoa(basePort+80)
		ctx.Insert("computed_port", basePort+80)
		ctx.Insert(key+"_timestamp", "2023-01-01T00:00:00Z")
		return "http://" + url
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if computedURL != "http://test_app:8080" {
		t.Errorf("expected 'http://test_app:8080', got %q", computedURL)
	}
	if v, ok := ops.DryGet[int](dry, "computed_port"); !ok || v != 8080 {
		t.Errorf("expected computed_port=8080, got %d ok=%v", v, ok)
	}

	// Second call should return cached value without calling computer
	computerCalled := false
	existing, err := ops.DryGetOrComputeWith(dry, "service_url", func(_ *ops.DryContext, _ string) string {
		computerCalled = true
		return "should_not_be_called"
	})
	if err != nil || existing != "http://test_app:8080" {
		t.Errorf("expected cached value, got %q err=%v", existing, err)
	}
	if computerCalled {
		t.Error("computer should not be called for existing key")
	}
}

// TEST098: Merge two DryContexts where keys overlap and verify the merging context's values win
func Test098_dry_context_merge_overwrites_keys(t *testing.T) {
	dry1 := ops.NewDryContext().WithValue("shared", 1).WithValue("only_in_1", 10)
	dry2 := ops.NewDryContext().WithValue("shared", 2).WithValue("only_in_2", 20)
	dry1.Merge(dry2)

	if v, _ := ops.DryGet[int](dry1, "shared"); v != 2 {
		t.Errorf("expected shared=2 (from dry2), got %d", v)
	}
	if v, _ := ops.DryGet[int](dry1, "only_in_1"); v != 10 {
		t.Errorf("expected only_in_1=10, got %d", v)
	}
	if v, _ := ops.DryGet[int](dry1, "only_in_2"); v != 20 {
		t.Errorf("expected only_in_2=20, got %d", v)
	}
}

type wetSvcA struct{}
type wetSvcB struct{}

// TEST099: Merge two WetContexts and verify both sets of references are accessible in the target
func Test099_wet_context_merge(t *testing.T) {
	wet1 := ops.NewWetContext()
	wet1.InsertRef("a", &wetSvcA{})

	wet2 := ops.NewWetContext()
	wet2.InsertRef("b", &wetSvcB{})

	wet1.Merge(wet2)
	if !wet1.Contains("a") {
		t.Error("expected 'a' after merge")
	}
	if !wet1.Contains("b") {
		t.Error("expected 'b' after merge")
	}
}

// TEST100: Serialize and deserialize a DryContext and verify all values survive the round-trip
func Test100_dry_context_serde_roundtrip(t *testing.T) {
	original := ops.NewDryContext().
		WithValue("name", "alice").
		WithValue("count", 42).
		WithValue("flag", true)

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var restored ops.DryContext
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if v, _ := ops.DryGet[string](&restored, "name"); v != "alice" {
		t.Errorf("expected name='alice', got %q", v)
	}
	if v, _ := ops.DryGet[int](&restored, "count"); v != 42 {
		t.Errorf("expected count=42, got %d", v)
	}
	if v, _ := ops.DryGet[bool](&restored, "flag"); !v {
		t.Error("expected flag=true")
	}
}

// TEST101: Clone a DryContext and verify the clone is independent (mutations don't propagate)
func Test101_dry_context_clone_is_independent(t *testing.T) {
	original := ops.NewDryContext().WithValue("x", 1)
	cloned := original.Clone()
	cloned.Insert("x", 99)

	if v, _ := ops.DryGet[int](original, "x"); v != 1 {
		t.Errorf("expected original x=1, got %d (mutation propagated)", v)
	}
	if v, _ := ops.DryGet[int](cloned, "x"); v != 99 {
		t.Errorf("expected cloned x=99, got %d", v)
	}
}

// TEST102: Verify DryContext::keys() returns all inserted keys
func Test102_dry_context_keys(t *testing.T) {
	dry := ops.NewDryContext().
		WithValue("alpha", 1).
		WithValue("beta", 2).
		WithValue("gamma", 3)

	keys := dry.Keys()
	sort.Strings(keys)
	if len(keys) != 3 || keys[0] != "alpha" || keys[1] != "beta" || keys[2] != "gamma" {
		t.Errorf("expected [alpha, beta, gamma], got %v", keys)
	}
}

// TEST103: Verify WetContext::keys() returns all inserted reference keys
func Test103_wet_context_keys(t *testing.T) {
	wet := ops.NewWetContext()
	wet.InsertRef("svc1", &wetSvcA{})
	wet.InsertRef("svc2", &wetSvcB{})

	keys := wet.Keys()
	sort.Strings(keys)
	if len(keys) != 2 || keys[0] != "svc1" || keys[1] != "svc2" {
		t.Errorf("expected [svc1, svc2], got %v", keys)
	}
}

// itoa converts int to string (helper for test)
func itoa(n int) string {
	return strconv.Itoa(n)
}
