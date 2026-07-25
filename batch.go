package ops

import (
	"encoding/json"
	"fmt"
	"log"
)

// BatchOp executes a sequence of ops. On failure, succeeded ops are rolled back in LIFO order.
// BatchOp[T] implements Op[[]T].
type BatchOp[T any] struct {
	ops             []Op[T]
	continueOnError bool
}

// NewBatchOp creates a BatchOp from a slice of ops.
func NewBatchOp[T any](ops []Op[T]) *BatchOp[T] {
	return &BatchOp[T]{ops: ops}
}

// WithContinueOnError sets whether the batch continues past op failures.
func (b *BatchOp[T]) WithContinueOnError(continueOnError bool) *BatchOp[T] {
	b.continueOnError = continueOnError
	return b
}

// AddOp appends an op to the batch.
func (b *BatchOp[T]) AddOp(op Op[T]) {
	b.ops = append(b.ops, op)
}

// Len returns the number of ops in the batch.
func (b *BatchOp[T]) Len() int {
	return len(b.ops)
}

// IsEmpty returns true when the batch contains no ops.
func (b *BatchOp[T]) IsEmpty() bool {
	return len(b.ops) == 0
}

// Perform executes all ops in sequence, rolling back on failure.
func (b *BatchOp[T]) Perform(dry *DryContext, wet *WetContext) ([]T, error) {
	results := make([]T, 0, len(b.ops))
	var errors []struct {
		index int
		err   error
	}
	succeededOps := make([]Op[T], 0, len(b.ops))

	for index, op := range b.ops {
		// Check abort before each op
		if dry.IsAborted() {
			b.rollbackSucceededOps(succeededOps, dry, wet)
			reason := "Batch operation aborted"
			if r := dry.AbortReason(); r != nil {
				reason = *r
			}
			return nil, NewAbortedError(reason)
		}

		result, err := op.Perform(dry, wet)
		if err != nil {
			opErr := AsOpError(err)
			if opErr != nil && opErr.Kind == ErrAborted {
				b.rollbackSucceededOps(succeededOps, dry, wet)
				return nil, err
			}
			if b.continueOnError {
				errors = append(errors, struct {
					index int
					err   error
				}{index, err})
			} else {
				b.rollbackSucceededOps(succeededOps, dry, wet)
				// A classified child failure keeps its declared identity
				// through the batch wrap (docs/failure-taxonomy.md) — the
				// wrap enriches the human chain only.
				chain := fmt.Sprintf("Op %d-%s failed: %v", index, op.Metadata().Name, err)
				if opErr != nil && opErr.FailureCode() != "" {
					wrapped := NewWrappedClassifiedError(
						chain, opErr.Code, opErr.AttributionClassOf(), opErr.FailureReason(),
					)
					wrapped.ArgURN = opErr.FailureArgURN()
					return nil, wrapped
				}
				return nil, NewBatchFailedError(chain)
			}
		} else {
			results = append(results, result)
			succeededOps = append(succeededOps, op)
		}
	}

	if len(errors) > 0 && !b.continueOnError {
		b.rollbackSucceededOps(succeededOps, dry, wet)
		return nil, NewBatchFailedError(fmt.Sprintf("Batch op had %d errors", len(errors)))
	}

	return results, nil
}

// Metadata returns a dynamically-computed metadata for this batch.
func (b *BatchOp[T]) Metadata() OpMetadata {
	return NewBatchMetadataBuilder(b.ops).Build()
}

// Rollback is a no-op for BatchOp (rollback is internal).
func (b *BatchOp[T]) Rollback(_ *DryContext, _ *WetContext) error {
	return nil
}

// rollbackSucceededOps calls Rollback on each succeeded op in LIFO order.
func (b *BatchOp[T]) rollbackSucceededOps(succeeded []Op[T], dry *DryContext, wet *WetContext) {
	for i := len(succeeded) - 1; i >= 0; i-- {
		op := succeeded[i]
		if err := op.Rollback(dry, wet); err != nil {
			log.Printf("Failed to rollback op %s: %v", op.Metadata().Name, err)
		}
	}
}

// BatchMetadataBuilder builds metadata for a BatchOp by analyzing data flow between ops.
type BatchMetadataBuilder struct {
	opsMetadata []OpMetadata
}

// NewBatchMetadataBuilder creates a builder from a slice of ops.
func NewBatchMetadataBuilder[T any](ops []Op[T]) *BatchMetadataBuilder {
	metadata := make([]OpMetadata, len(ops))
	for i, op := range ops {
		metadata[i] = op.Metadata()
	}
	return &BatchMetadataBuilder{opsMetadata: metadata}
}

// Build constructs the merged OpMetadata for the batch.
func (b *BatchMetadataBuilder) Build() OpMetadata {
	inputSchema := b.analyzeInputRequirements()
	refSchema := b.mergeReferenceSchemas()
	outputSchema := b.constructOutputSchema()

	desc := fmt.Sprintf("Batch of %d operations with data flow analysis", len(b.opsMetadata))
	return NewOpMetadataBuilder("BatchOp").
		Description(desc).
		InputSchema(inputSchema).
		ReferenceSchema(refSchema).
		OutputSchema(outputSchema).
		Build()
}

func (b *BatchMetadataBuilder) analyzeInputRequirements() json.RawMessage {
	requiredInputs := make(map[string]bool)
	availableOutputs := make(map[string]bool)
	propertiesMap := make(map[string]json.RawMessage)

	for _, meta := range b.opsMetadata {
		// Collect required inputs not already satisfied
		if meta.InputSchema != nil {
			for _, field := range extractRequiredFields(meta.InputSchema) {
				if !availableOutputs[field] {
					requiredInputs[field] = true
				}
			}
			// Collect properties for schema building
			var schemaObj map[string]json.RawMessage
			if err := json.Unmarshal(meta.InputSchema, &schemaObj); err == nil {
				if propsRaw, ok := schemaObj["properties"]; ok {
					var props map[string]json.RawMessage
					if err := json.Unmarshal(propsRaw, &props); err == nil {
						for k, v := range props {
							if _, exists := propertiesMap[k]; !exists {
								propertiesMap[k] = v
							}
						}
					}
				}
			}
		}
		// Track outputs
		for _, field := range extractOutputFields(meta.OutputSchema) {
			availableOutputs[field] = true
		}
	}

	// Build required list from only the unsatisfied inputs
	required := make([]string, 0)
	filteredProps := make(map[string]json.RawMessage)
	for field := range requiredInputs {
		required = append(required, field)
		if v, ok := propertiesMap[field]; ok {
			filteredProps[field] = v
		}
	}

	propsJSON, _ := json.Marshal(filteredProps)
	reqJSON, _ := json.Marshal(required)
	schemaJSON := fmt.Sprintf(`{"type":"object","properties":%s,"required":%s,"additionalProperties":false}`,
		string(propsJSON), string(reqJSON))
	return json.RawMessage(schemaJSON)
}

func (b *BatchMetadataBuilder) mergeReferenceSchemas() json.RawMessage {
	allProps := make(map[string]json.RawMessage)
	allRequired := make(map[string]bool)

	for _, meta := range b.opsMetadata {
		if meta.ReferenceSchema == nil {
			continue
		}
		var schemaObj map[string]json.RawMessage
		if err := json.Unmarshal(meta.ReferenceSchema, &schemaObj); err != nil {
			continue
		}
		if propsRaw, ok := schemaObj["properties"]; ok {
			var props map[string]json.RawMessage
			if err := json.Unmarshal(propsRaw, &props); err == nil {
				for k, v := range props {
					if _, exists := allProps[k]; !exists {
						allProps[k] = v
					}
				}
			}
		}
		if reqRaw, ok := schemaObj["required"]; ok {
			var required []string
			if err := json.Unmarshal(reqRaw, &required); err == nil {
				for _, field := range required {
					allRequired[field] = true
				}
			}
		}
	}

	required := make([]string, 0, len(allRequired))
	for field := range allRequired {
		required = append(required, field)
	}

	propsJSON, _ := json.Marshal(allProps)
	reqJSON, _ := json.Marshal(required)
	schemaJSON := fmt.Sprintf(`{"type":"object","properties":%s,"required":%s,"additionalProperties":false}`,
		string(propsJSON), string(reqJSON))
	return json.RawMessage(schemaJSON)
}

func (b *BatchMetadataBuilder) constructOutputSchema() json.RawMessage {
	n := len(b.opsMetadata)
	schemaJSON := fmt.Sprintf(
		`{"type":"array","items":{"type":"object","description":"Output from individual ops in the batch"},"minItems":%d,"maxItems":%d}`,
		n, n,
	)
	return json.RawMessage(schemaJSON)
}

// extractRequiredFields returns the fields listed in a schema's "required" array.
func extractRequiredFields(schema json.RawMessage) []string {
	if schema == nil {
		return nil
	}
	var schemaObj map[string]json.RawMessage
	if err := json.Unmarshal(schema, &schemaObj); err != nil {
		return nil
	}
	reqRaw, ok := schemaObj["required"]
	if !ok {
		return nil
	}
	var required []string
	_ = json.Unmarshal(reqRaw, &required)
	return required
}

// extractOutputFields returns the property names from a schema's "properties" object.
func extractOutputFields(schema json.RawMessage) []string {
	if schema == nil {
		return nil
	}
	var schemaObj map[string]json.RawMessage
	if err := json.Unmarshal(schema, &schemaObj); err != nil {
		return nil
	}
	propsRaw, ok := schemaObj["properties"]
	if !ok {
		// Check for scalar string output
		if typeRaw, ok := schemaObj["type"]; ok {
			var t string
			if err := json.Unmarshal(typeRaw, &t); err == nil && t == "string" {
				return []string{"result"}
			}
		}
		return nil
	}
	var props map[string]json.RawMessage
	if err := json.Unmarshal(propsRaw, &props); err != nil {
		return nil
	}
	fields := make([]string, 0, len(props))
	for k := range props {
		fields = append(fields, k)
	}
	return fields
}
