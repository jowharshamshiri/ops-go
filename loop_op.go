package ops

import (
	"fmt"
	"log"

	"github.com/google/uuid"
)

// LoopOp executes a sequence of ops repeatedly up to a limit.
// LoopOp[T] implements Op[[]T].
type LoopOp[T any] struct {
	counterVar      string
	limit           int
	ops             []Op[T]
	loopID          string
	continueVar     string
	breakVar        string
	continueOnError bool
}

// NewLoopOp creates a LoopOp.
// counterVar is the key used to track iteration count in DryContext.
// limit is the maximum number of iterations.
func NewLoopOp[T any](counterVar string, limit int, ops []Op[T]) *LoopOp[T] {
	loopID := uuid.New().String()
	return &LoopOp[T]{
		counterVar:  counterVar,
		limit:       limit,
		ops:         ops,
		loopID:      loopID,
		continueVar: fmt.Sprintf("__continue_loop_%s", loopID),
		breakVar:    fmt.Sprintf("__break_loop_%s", loopID),
	}
}

// AddOp appends an op to the loop body.
func (l *LoopOp[T]) AddOp(op Op[T]) *LoopOp[T] {
	l.ops = append(l.ops, op)
	return l
}

// WithContinueOnError sets whether the loop continues past op failures.
func (l *LoopOp[T]) WithContinueOnError(continueOnError bool) *LoopOp[T] {
	l.continueOnError = continueOnError
	return l
}

// Perform executes the loop.
func (l *LoopOp[T]) Perform(dry *DryContext, wet *WetContext) ([]T, error) {
	results := make([]T, 0)

	// Read current counter (may be pre-set).
	counter := 0
	if val, ok := DryGet[int](dry, l.counterVar); ok {
		counter = val
	}
	if !dry.Contains(l.counterVar) {
		dry.Insert(l.counterVar, counter)
	}

	// Store loop ID for scoped control flow ops.
	dry.Insert("__current_loop_id", l.loopID)

	for counter < l.limit {
		// Check abort before each iteration.
		if dry.IsAborted() {
			reason := "Loop operation aborted"
			if r := dry.AbortReason(); r != nil {
				reason = *r
			}
			return nil, NewAbortedError(reason)
		}

		// Clear scoped control flags for this iteration.
		dry.Insert(l.continueVar, false)
		dry.Insert(l.breakVar, false)

		iterSucceeded := make([]Op[T], 0, len(l.ops))
		iterFailed := false

		for _, op := range l.ops {
			// Check abort before each op.
			if dry.IsAborted() {
				l.rollbackIterationOps(iterSucceeded, dry, wet)
				reason := "Loop operation aborted"
				if r := dry.AbortReason(); r != nil {
					reason = *r
				}
				return nil, NewAbortedError(reason)
			}

			result, err := op.Perform(dry, wet)
			if err != nil {
				opErr := AsOpError(err)
				if opErr != nil && opErr.Kind == ErrAborted {
					l.rollbackIterationOps(iterSucceeded, dry, wet)
					return nil, err
				}
				if l.continueOnError {
					log.Printf("Operation %s failed in loop iteration %d: %v. Continuing.",
						op.Metadata().Name, counter, err)
					l.rollbackIterationOps(iterSucceeded, dry, wet)
					iterFailed = true
					break // move to next iteration
				}
				l.rollbackIterationOps(iterSucceeded, dry, wet)
				return nil, err
			}

			results = append(results, result)
			iterSucceeded = append(iterSucceeded, op)

			// Check scoped continue flag.
			if cont, ok := DryGet[bool](dry, l.continueVar); ok && cont {
				dry.Insert(l.continueVar, false)
				break // skip remaining ops in this iteration
			}

			// Check scoped break flag.
			if brk, ok := DryGet[bool](dry, l.breakVar); ok && brk {
				dry.Insert(l.breakVar, false)
				return results, nil // exit loop entirely
			}
		}

		_ = iterFailed // iterFailed handled by break above

		// Increment counter.
		counter++
		dry.Insert(l.counterVar, counter)
	}

	return results, nil
}

// Metadata returns descriptive metadata for this loop.
func (l *LoopOp[T]) Metadata() OpMetadata {
	var desc string
	if l.continueOnError {
		desc = fmt.Sprintf("Loop %d times over %d ops (continue on error)", l.limit, len(l.ops))
	} else {
		desc = fmt.Sprintf("Loop %d times over %d ops", l.limit, len(l.ops))
	}
	return NewOpMetadataBuilder("LoopOp").Description(desc).Build()
}

// Rollback is a no-op for LoopOp.
func (l *LoopOp[T]) Rollback(_ *DryContext, _ *WetContext) error {
	return nil
}

// rollbackIterationOps rolls back succeeded ops within an iteration in LIFO order.
func (l *LoopOp[T]) rollbackIterationOps(succeeded []Op[T], dry *DryContext, wet *WetContext) {
	for i := len(succeeded) - 1; i >= 0; i-- {
		op := succeeded[i]
		if err := op.Rollback(dry, wet); err != nil {
			log.Printf("Failed to rollback op %s in loop iteration: %v", op.Metadata().Name, err)
		}
	}
}
