package ops

// Op is the core interface for all operations in the framework.
// T is the return type of the operation.
type Op[T any] interface {
	// Perform executes the operation with the given contexts.
	Perform(dry *DryContext, wet *WetContext) (T, error)

	// Metadata returns descriptive metadata about this operation.
	Metadata() OpMetadata

	// Rollback is called when a batch operation needs to undo this op's effects.
	// The default no-op is provided via DefaultRollback embedding.
	Rollback(dry *DryContext, wet *WetContext) error
}

// DefaultRollback provides a no-op Rollback implementation for embedding in Op structs.
// Embed this in your Op implementation to get a free no-op rollback.
type DefaultRollback struct{}

// Rollback is a no-op that always succeeds.
func (*DefaultRollback) Rollback(_ *DryContext, _ *WetContext) error {
	return nil
}
