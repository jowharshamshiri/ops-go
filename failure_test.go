package ops

import "testing"

// TEST1730: the wire vocabulary round-trips exactly and rejects unknowns.
// (mirrors Rust ops/src/failure.rs TEST1730)
func Test1730_attribution_class_wire_tokens_round_trip(t *testing.T) {
	for _, class := range []AttributionClass{FailureInput, FailureResource, FailureEnvironment, FailureInternal, FailureUser} {
		parsed, ok := AttributionClassFromWire(string(class))
		if !ok || parsed != class {
			t.Fatalf("wire token %q must round-trip, got (%q, %v)", class, parsed, ok)
		}
	}
	if _, ok := AttributionClassFromWire("user-error"); ok {
		t.Fatal("unknown token must be rejected")
	}
	if _, ok := AttributionClassFromWire(""); ok {
		t.Fatal("empty token must be rejected")
	}
}

// TEST1731: Input and User are permanent — the retry machinery keys on this
// (a deterministic input failure, or a human's decision, is never retried
// automatically). (mirrors Rust ops/src/failure.rs TEST1731)
func Test1731_only_input_and_user_are_permanent(t *testing.T) {
	if !FailureInput.IsPermanent() {
		t.Fatal("input must be permanent")
	}
	if !FailureUser.IsPermanent() {
		t.Fatal("user must be permanent")
	}
	for _, class := range []AttributionClass{FailureResource, FailureEnvironment, FailureInternal} {
		if class.IsPermanent() {
			t.Fatalf("%q must not be permanent", class)
		}
	}
}

// TEST1901: classified variants carry the emit source's identity through
// the accessors; unclassified variants are Internal with no code — the
// taxonomy's own rule (docs/failure-taxonomy.md).
// (mirrors Rust ops/src/error.rs TEST1901)
func Test1901_classified_accessors(t *testing.T) {
	classified := NewClassifiedError("CONTEXT_OVERFLOW", FailureInput, "prompt too large").
		WithFailureArgURN("media:enc=utf-8;prompt")
	if classified.AttributionClassOf() != FailureInput {
		t.Fatal("classified error must carry its declared class")
	}
	if classified.FailureCode() != "CONTEXT_OVERFLOW" {
		t.Fatal("classified error must carry its declared code")
	}
	if classified.FailureReason() != "prompt too large" {
		t.Fatal("classified error must carry its leaf reason")
	}
	if classified.FailureArgURN() != "media:enc=utf-8;prompt" {
		t.Fatal("classified error must carry its declared argument attribution")
	}
	if classified.Error() != "CONTEXT_OVERFLOW: prompt too large" {
		t.Fatalf("classified Display mismatch: %q", classified.Error())
	}

	wrapped := NewWrappedClassifiedError(
		"Op 3-generate failed: CONTEXT_OVERFLOW: prompt too large",
		"CONTEXT_OVERFLOW", FailureInput, "prompt too large",
	)
	if wrapped.FailureReason() != "prompt too large" {
		t.Fatal("the reason is the LEAF message, not the wrap chain")
	}
	if wrapped.FailureArgURN() != "" {
		t.Fatal("an unattributed classified error must remain unattributed")
	}
	if wrapped.Error() != "Op 3-generate failed: CONTEXT_OVERFLOW: prompt too large" {
		t.Fatalf("wrapped Display keeps the human chain, got %q", wrapped.Error())
	}

	plain := NewExecutionFailedError("boom")
	if plain.AttributionClassOf() != FailureInternal {
		t.Fatal("unclassified errors are Internal — never a guess")
	}
	if plain.FailureCode() != "" {
		t.Fatal("unclassified errors carry no code")
	}
}

// TEST1903: wrapping preserves a classified failure's identity — the wrap
// enriches the human CHAIN only, never the class/code/reason
// (docs/failure-taxonomy.md). (mirrors Rust ops/src/ops.rs TEST1903)
func Test1903_wrap_preserves_classification(t *testing.T) {
	wrapped := WrapNestedOpException("GenerateOp",
		NewClassifiedError("CONTEXT_OVERFLOW", FailureInput, "prompt too large").
			WithFailureArgURN("media:enc=utf-8;prompt"))
	opErr := AsOpError(wrapped)
	if opErr == nil || opErr.Kind != ErrWrappedClassified {
		t.Fatalf("expected WrappedClassified, got %+v", wrapped)
	}
	if opErr.FailureCode() != "CONTEXT_OVERFLOW" || opErr.AttributionClassOf() != FailureInput ||
		opErr.FailureReason() != "prompt too large" {
		t.Fatalf("identity fields must survive the wrap: %+v", opErr)
	}
	if opErr.FailureArgURN() != "media:enc=utf-8;prompt" {
		t.Fatal("argument attribution must survive the wrap")
	}
	if opErr.Error() == "prompt too large" {
		t.Fatal("the chain must name the wrapping op")
	}

	rewrapped := AsOpError(WrapNestedOpException("OuterBatch", opErr))
	if rewrapped == nil || rewrapped.Kind != ErrWrappedClassified {
		t.Fatalf("expected WrappedClassified after re-wrap, got %+v", rewrapped)
	}
	if rewrapped.FailureCode() != "CONTEXT_OVERFLOW" || rewrapped.AttributionClassOf() != FailureInput ||
		rewrapped.FailureReason() != "prompt too large" {
		t.Fatalf("identity fields must survive re-wrapping: %+v", rewrapped)
	}
	if rewrapped.FailureArgURN() != "media:enc=utf-8;prompt" {
		t.Fatal("argument attribution must survive re-wrapping")
	}
}
