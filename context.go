package ops

import (
	"encoding/json"
	"fmt"
)

// ControlFlags tracks abort signals in DryContext.
type ControlFlags struct {
	Aborted     bool    `json:"aborted"`
	AbortReason *string `json:"abort_reason,omitempty"`
}

// DryContext contains only serializable (JSON-compatible) data values.
// It is passed to ops and can be serialized/deserialized for persistence.
type DryContext struct {
	values       map[string]json.RawMessage
	controlFlags ControlFlags
}

// NewDryContext creates an empty DryContext.
func NewDryContext() *DryContext {
	return &DryContext{values: make(map[string]json.RawMessage)}
}

// WithValue returns a new DryContext with the given key/value added (builder pattern).
func (d *DryContext) WithValue(key string, value any) *DryContext {
	d.Insert(key, value)
	return d
}

// Insert stores a value in the context. Panics if the value cannot be marshaled to JSON.
func (d *DryContext) Insert(key string, value any) {
	data, err := json.Marshal(value)
	if err != nil {
		panic(fmt.Sprintf("Failed to serialize value for key '%s': %v", key, err))
	}
	d.values[key] = data
}

// Contains returns true if the given key exists in the context.
func (d *DryContext) Contains(key string) bool {
	_, ok := d.values[key]
	return ok
}

// Keys returns all keys currently in the context.
func (d *DryContext) Keys() []string {
	keys := make([]string, 0, len(d.values))
	for k := range d.values {
		keys = append(keys, k)
	}
	return keys
}

// Values returns the raw JSON map backing this context.
func (d *DryContext) Values() map[string]json.RawMessage {
	return d.values
}

// SetAbort flags this context as aborted with an optional reason.
// Pass nil for no reason.
func (d *DryContext) SetAbort(reason *string) {
	d.controlFlags.Aborted = true
	d.controlFlags.AbortReason = reason
}

// IsAborted returns true if the abort flag has been set.
func (d *DryContext) IsAborted() bool {
	return d.controlFlags.Aborted
}

// AbortReason returns the abort reason, or nil if none was set.
func (d *DryContext) AbortReason() *string {
	return d.controlFlags.AbortReason
}

// ClearControlFlags resets all abort flags.
func (d *DryContext) ClearControlFlags() {
	d.controlFlags = ControlFlags{}
}

// Merge copies all values from other into d. If other is aborted and d is not,
// d inherits the abort state. If d is already aborted, its abort reason is preserved.
func (d *DryContext) Merge(other *DryContext) {
	for k, v := range other.values {
		d.values[k] = v
	}
	if other.controlFlags.Aborted && !d.controlFlags.Aborted {
		d.controlFlags.Aborted = true
		d.controlFlags.AbortReason = other.controlFlags.AbortReason
	}
}

// Clone returns an independent deep copy of this DryContext.
func (d *DryContext) Clone() *DryContext {
	clone := &DryContext{
		values:       make(map[string]json.RawMessage, len(d.values)),
		controlFlags: d.controlFlags,
	}
	for k, v := range d.values {
		vCopy := make(json.RawMessage, len(v))
		copy(vCopy, v)
		clone.values[k] = vCopy
	}
	return clone
}

// MarshalJSON serializes DryContext to JSON (values + control_flags).
func (d *DryContext) MarshalJSON() ([]byte, error) {
	type alias struct {
		Values       map[string]json.RawMessage `json:"values"`
		ControlFlags ControlFlags               `json:"control_flags"`
	}
	return json.Marshal(&alias{Values: d.values, ControlFlags: d.controlFlags})
}

// UnmarshalJSON deserializes a DryContext from JSON.
func (d *DryContext) UnmarshalJSON(data []byte) error {
	type alias struct {
		Values       map[string]json.RawMessage `json:"values"`
		ControlFlags ControlFlags               `json:"control_flags"`
	}
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	if a.Values == nil {
		a.Values = make(map[string]json.RawMessage)
	}
	d.values = a.Values
	d.controlFlags = a.ControlFlags
	return nil
}

// DryGet retrieves a typed value from the context by key.
// Returns (zero, false) if the key does not exist or the type does not match.
func DryGet[T any](d *DryContext, key string) (T, bool) {
	var zero T
	data, ok := d.values[key]
	if !ok {
		return zero, false
	}
	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		return zero, false
	}
	return result, true
}

// DryGetRequired retrieves a typed value from the context. Returns a Context error
// if the key is missing or the stored value cannot be decoded as type T.
func DryGetRequired[T any](d *DryContext, key string) (T, error) {
	var zero T
	data, ok := d.values[key]
	if !ok {
		return zero, NewContextError(fmt.Sprintf("Required dry context key '%s' not found", key))
	}
	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		actualType := jsonTypeName(data)
		expectedType := fmt.Sprintf("%T", zero)
		return zero, NewContextError(fmt.Sprintf(
			"Type mismatch for dry context key '%s': expected type '%s', but found '%s' value: %s",
			key, expectedType, actualType, string(data),
		))
	}
	return result, nil
}

// DryGetOrInsertWith returns the value for key if present, otherwise calls factory,
// stores the result, and returns it.
func DryGetOrInsertWith[T any](d *DryContext, key string, factory func() T) (T, error) {
	if val, ok := DryGet[T](d, key); ok {
		return val, nil
	}
	newVal := factory()
	d.Insert(key, newVal)
	return newVal, nil
}

// DryGetOrComputeWith returns the value for key if present, otherwise calls computer
// with the context and key, stores the result, and returns it. The computer may also
// insert additional values into the context.
func DryGetOrComputeWith[T any](d *DryContext, key string, computer func(*DryContext, string) T) (T, error) {
	if val, ok := DryGet[T](d, key); ok {
		return val, nil
	}
	newVal := computer(d, key)
	d.Insert(key, newVal)
	return newVal, nil
}

// jsonTypeName returns a human-readable JSON type name for the given raw JSON value.
func jsonTypeName(data json.RawMessage) string {
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return "unknown"
	}
	switch raw.(type) {
	case nil:
		return "null"
	case bool:
		return "boolean"
	case float64:
		return "number"
	case string:
		return "string"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	default:
		return "unknown"
	}
}

// WetContext contains non-serializable runtime references (services, connections, etc.).
type WetContext struct {
	references map[string]any
}

// NewWetContext creates an empty WetContext.
func NewWetContext() *WetContext {
	return &WetContext{references: make(map[string]any)}
}

// WithRef returns a new WetContext with the given key/value added (builder pattern).
func (w *WetContext) WithRef(key string, value any) *WetContext {
	w.InsertRef(key, value)
	return w
}

// InsertRef stores any value under the given key.
func (w *WetContext) InsertRef(key string, value any) {
	w.references[key] = value
}

// InsertArc stores any value under the given key (identical to InsertRef; provided for API parity).
func (w *WetContext) InsertArc(key string, value any) {
	w.references[key] = value
}

// Contains returns true if the given key exists in the context.
func (w *WetContext) Contains(key string) bool {
	_, ok := w.references[key]
	return ok
}

// Keys returns all keys currently in the context.
func (w *WetContext) Keys() []string {
	keys := make([]string, 0, len(w.references))
	for k := range w.references {
		keys = append(keys, k)
	}
	return keys
}

// Merge copies all references from other into w.
func (w *WetContext) Merge(other *WetContext) {
	for k, v := range other.references {
		w.references[k] = v
	}
}

// WetGetRef retrieves a typed value from the WetContext.
// Returns (zero, false) if the key does not exist or the type does not match.
func WetGetRef[T any](w *WetContext, key string) (T, bool) {
	var zero T
	val, ok := w.references[key]
	if !ok {
		return zero, false
	}
	typed, ok := val.(T)
	return typed, ok
}

// WetGetRefRequired retrieves a typed value from the WetContext.
// Returns a Context error if the key is missing or the type does not match.
func WetGetRefRequired[T any](w *WetContext, key string) (T, error) {
	var zero T
	val, ok := w.references[key]
	if !ok {
		return zero, NewContextError(fmt.Sprintf(
			"Required wet context reference '%s' not found", key,
		))
	}
	typed, ok := val.(T)
	if !ok {
		return zero, NewContextError(fmt.Sprintf(
			"Type mismatch for wet context reference '%s': expected type '%T', but found a different type",
			key, zero,
		))
	}
	return typed, nil
}
