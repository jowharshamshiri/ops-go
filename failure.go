package ops

// The failure taxonomy — WHOSE problem a failure is
// (docs/failure-taxonomy.md, mirrors Rust ops::AttributionClass).
//
// Declared at the error's DEFINITION site and carried structurally through
// every hop; no layer ever infers another layer's class from message text.
// An error that reaches a boundary without a declared class is
// FailureInternal — unclassified means "ours", never a guess.

// AttributionClass is whose problem a failure is. The value is the stable
// lowercase wire token — used in the ERR frame meta, the machine_runs
// columns, the gRPC proto, and the loom. One vocabulary everywhere.
type AttributionClass string

const (
	// FailureInput: deterministic on the INPUT (context overflow, invalid
	// request, unsupported format). The user's to fix; retrying can never
	// succeed — tasks failing with this class are marked permanently failed.
	FailureInput AttributionClass = "input"
	// FailureResource: a compute resource was exhausted (GPU VRAM, host
	// memory). Often transient (another process holding memory) — retryable.
	FailureResource AttributionClass = "resource"
	// FailureEnvironment: the environment failed (network, registry, model
	// download/integrity, cartridge process death). Transient by nature —
	// retryable.
	FailureEnvironment AttributionClass = "environment"
	// FailureInternal: everything else — a defect in the engine or a
	// cartridge. Ours, said plainly. Retryable (races un-race), but never
	// blamed on the user.
	FailureInternal AttributionClass = "internal"
)

// AttributionClassFromWire parses a wire token. Returns false for unknown
// tokens — a PROTOCOL error, not a fallback case: the caller decides
// whether to fail hard or treat the value as unclassified (Internal).
// (matches Rust AttributionClass::from_wire)
func AttributionClassFromWire(token string) (AttributionClass, bool) {
	switch token {
	case "input":
		return FailureInput, true
	case "resource":
		return FailureResource, true
	case "environment":
		return FailureEnvironment, true
	case "internal":
		return FailureInternal, true
	default:
		return FailureInternal, false
	}
}

// IsPermanent reports whether retrying can NEVER succeed: the failure is a
// deterministic function of the input. Resource/environment/internal stay
// retryable (memory frees up, networks recover, races un-race).
// (matches Rust AttributionClass::is_permanent)
func (c AttributionClass) IsPermanent() bool {
	return c == FailureInput
}
