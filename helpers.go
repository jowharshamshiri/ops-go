package ops

import "fmt"

// --- Dry context helpers (equivalent to Rust macros) ---

// DryPut stores a value in the dry context under the given key.
func DryPut(dry *DryContext, key string, value any) {
	dry.Insert(key, value)
}

// DryGet retrieves an optionally typed value from the dry context.
// This is the generic version — Go callers use DryGet[T](dry, key).
// Non-generic alias for use in tests needing a concrete signature.
func DryGetAny(dry *DryContext, key string) (any, bool) {
	return DryGet[any](dry, key)
}

// DryResult stores a result in the dry context under both the op name and "result".
func DryResult(dry *DryContext, opName string, result any) {
	dry.Insert(opName, result)
	dry.Insert("result", result)
}

// WetPutRef stores a value in the wet context under the given key.
func WetPutRef(wet *WetContext, key string, value any) {
	wet.InsertRef(key, value)
}

// WetPutArc stores a value in the wet context under the given key.
// Equivalent to wet_put_arc! — in Go there is no Arc distinction.
func WetPutArc(wet *WetContext, key string, value any) {
	wet.InsertArc(key, value)
}

// --- Control flow helpers (equivalent to Rust abort!/continue_loop!/break_loop!/check_abort! macros) ---

// Abort sets the abort flag with no reason and returns an Aborted error.
// The caller's op should return this error immediately.
func Abort(dry *DryContext) error {
	dry.SetAbort(nil)
	return NewAbortedError("Operation aborted")
}

// AbortWithReason sets the abort flag with a reason and returns an Aborted error.
func AbortWithReason(dry *DryContext, reason string) error {
	dry.SetAbort(&reason)
	return NewAbortedError(reason)
}

// ContinueLoop sets the continue flag for the innermost loop.
// The op should return (zero, nil) after calling this.
func ContinueLoop(dry *DryContext) {
	if id, ok := DryGet[string](dry, "__current_loop_id"); ok {
		continueVar := fmt.Sprintf("__continue_loop_%s", id)
		dry.Insert(continueVar, true)
	}
}

// BreakLoop sets the break flag for the innermost loop.
// The op should return (zero, nil) after calling this.
func BreakLoop(dry *DryContext) {
	if id, ok := DryGet[string](dry, "__current_loop_id"); ok {
		breakVar := fmt.Sprintf("__break_loop_%s", id)
		dry.Insert(breakVar, true)
	}
}

// BreakLoopScoped sets the break flag for the loop with the given ID.
func BreakLoopScoped(dry *DryContext, loopID string) {
	breakVar := fmt.Sprintf("__break_loop_%s", loopID)
	dry.Insert(breakVar, true)
}

// CheckAbort returns an Aborted error if the abort flag is set, otherwise nil.
// Ops should call this at the start of their perform to short-circuit.
func CheckAbort(dry *DryContext) error {
	if !dry.IsAborted() {
		return nil
	}
	reason := "Operation aborted"
	if r := dry.AbortReason(); r != nil {
		reason = *r
	}
	return NewAbortedError(reason)
}

// --- Aggregation strategy (used by batch/loop composition patterns) ---

// AggregationStrategy determines how results are aggregated in composed ops.
type AggregationStrategy int

const (
	AggregateAll   AggregationStrategy = iota // return all results as []T
	AggregateLast                             // return only the last result
	AggregateFirst                            // return only the first result
	AggregateMerge                            // return a merged result
	AggregateUnit                             // discard all results, return ()
)
