package ops

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/xeipuuv/gojsonschema"
)

// ValidatingWrapper wraps an Op[T] and validates input/output against JSON schemas.
// T must be JSON-serializable.
type ValidatingWrapper[T any] struct {
	wrappedOp      Op[T]
	validateInput  bool
	validateOutput bool
}

// NewValidatingWrapper creates a wrapper that validates both input and output.
func NewValidatingWrapper[T any](op Op[T]) *ValidatingWrapper[T] {
	return &ValidatingWrapper[T]{wrappedOp: op, validateInput: true, validateOutput: true}
}

// NewValidatingWrapperInputOnly creates a wrapper that validates only input.
func NewValidatingWrapperInputOnly[T any](op Op[T]) *ValidatingWrapper[T] {
	return &ValidatingWrapper[T]{wrappedOp: op, validateInput: true, validateOutput: false}
}

// NewValidatingWrapperOutputOnly creates a wrapper that validates only output.
func NewValidatingWrapperOutputOnly[T any](op Op[T]) *ValidatingWrapper[T] {
	return &ValidatingWrapper[T]{wrappedOp: op, validateInput: false, validateOutput: true}
}

// Perform validates input, executes the op, then validates output.
func (v *ValidatingWrapper[T]) Perform(dry *DryContext, wet *WetContext) (T, error) {
	var zero T
	metadata := v.wrappedOp.Metadata()

	// Validate input schema
	if err := v.validateInputSchema(dry, &metadata); err != nil {
		return zero, err
	}

	// Validate reference schema (always, regardless of input/output flag)
	if err := v.validateReferencesSchema(wet, &metadata); err != nil {
		return zero, err
	}

	// Execute wrapped op
	result, err := v.wrappedOp.Perform(dry, wet)
	if err != nil {
		return zero, err
	}

	// Validate output schema
	if err := v.validateOutputSchema(&result, &metadata); err != nil {
		return zero, err
	}

	return result, nil
}

// Metadata delegates to the wrapped op's metadata.
func (v *ValidatingWrapper[T]) Metadata() OpMetadata {
	return v.wrappedOp.Metadata()
}

// Rollback delegates to the wrapped op's rollback.
func (v *ValidatingWrapper[T]) Rollback(dry *DryContext, wet *WetContext) error {
	return v.wrappedOp.Rollback(dry, wet)
}

func (v *ValidatingWrapper[T]) validateInputSchema(dry *DryContext, meta *OpMetadata) error {
	if !v.validateInput || meta.InputSchema == nil {
		return nil
	}
	contextJSON, err := json.Marshal(dry.Values())
	if err != nil {
		return NewContextError(fmt.Sprintf("Failed to serialize context for validation: %v", err))
	}
	return runJSONSchemaValidation(contextJSON, meta.InputSchema, "Input", meta.Name)
}

func (v *ValidatingWrapper[T]) validateReferencesSchema(wet *WetContext, meta *OpMetadata) error {
	if meta.ReferenceSchema == nil {
		return nil
	}
	var schemaObj map[string]json.RawMessage
	if err := json.Unmarshal(meta.ReferenceSchema, &schemaObj); err != nil {
		return nil
	}
	reqRaw, ok := schemaObj["required"]
	if !ok {
		return nil
	}
	var required []string
	if err := json.Unmarshal(reqRaw, &required); err != nil {
		return nil
	}
	for _, refName := range required {
		if !wet.Contains(refName) {
			return NewContextError(fmt.Sprintf(
				"Required reference '%s' not found in WetContext for op '%s'",
				refName, meta.Name,
			))
		}
	}
	return nil
}

func (v *ValidatingWrapper[T]) validateOutputSchema(output *T, meta *OpMetadata) error {
	if !v.validateOutput || meta.OutputSchema == nil {
		return nil
	}
	outputJSON, err := json.Marshal(output)
	if err != nil {
		return NewContextError(fmt.Sprintf("Failed to serialize output for validation: %v", err))
	}
	return runJSONSchemaValidation(outputJSON, meta.OutputSchema, "Output", meta.Name)
}

// runJSONSchemaValidation validates a JSON value against a JSON schema using gojsonschema.
func runJSONSchemaValidation(valueJSON, schemaJSON json.RawMessage, kind, opName string) error {
	schemaLoader := gojsonschema.NewBytesLoader(schemaJSON)
	docLoader := gojsonschema.NewBytesLoader(valueJSON)

	result, err := gojsonschema.Validate(schemaLoader, docLoader)
	if err != nil {
		return NewContextError(fmt.Sprintf("Invalid %s schema for %s: %v", kind, opName, err))
	}
	if !result.Valid() {
		msgs := make([]string, 0, len(result.Errors()))
		for _, e := range result.Errors() {
			msgs = append(msgs, fmt.Sprintf("%s: %s", e.Field(), e.Description()))
		}
		return NewContextError(fmt.Sprintf(
			"%s validation failed for %s: %s",
			kind, opName, strings.Join(msgs, ", "),
		))
	}
	return nil
}
