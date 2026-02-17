package ops

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// OpMetadata describes an op's name, documentation, and JSON schemas.
type OpMetadata struct {
	Name            string          `json:"name"`
	Description     *string         `json:"description,omitempty"`
	InputSchema     json.RawMessage `json:"input_schema,omitempty"`
	ReferenceSchema json.RawMessage `json:"reference_schema,omitempty"`
	OutputSchema    json.RawMessage `json:"output_schema,omitempty"`
}

// OpMetadataBuilder is a fluent builder for OpMetadata.
type OpMetadataBuilder struct {
	name            string
	description     *string
	inputSchema     json.RawMessage
	referenceSchema json.RawMessage
	outputSchema    json.RawMessage
}

// NewOpMetadataBuilder creates a builder with the given op name.
func NewOpMetadataBuilder(name string) *OpMetadataBuilder {
	return &OpMetadataBuilder{name: name}
}

// Description sets the human-readable description.
func (b *OpMetadataBuilder) Description(desc string) *OpMetadataBuilder {
	b.description = &desc
	return b
}

// InputSchema sets the JSON schema for the dry context input.
func (b *OpMetadataBuilder) InputSchema(schema json.RawMessage) *OpMetadataBuilder {
	b.inputSchema = schema
	return b
}

// ReferenceSchema sets the JSON schema for wet context references.
func (b *OpMetadataBuilder) ReferenceSchema(schema json.RawMessage) *OpMetadataBuilder {
	b.referenceSchema = schema
	return b
}

// OutputSchema sets the JSON schema for the op's output.
func (b *OpMetadataBuilder) OutputSchema(schema json.RawMessage) *OpMetadataBuilder {
	b.outputSchema = schema
	return b
}

// Build constructs the OpMetadata.
func (b *OpMetadataBuilder) Build() OpMetadata {
	return OpMetadata{
		Name:            b.name,
		Description:     b.description,
		InputSchema:     b.inputSchema,
		ReferenceSchema: b.referenceSchema,
		OutputSchema:    b.outputSchema,
	}
}

// ValidationError describes a single validation failure.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationWarning describes a non-fatal validation concern.
type ValidationWarning struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationReport is the result of schema validation.
type ValidationReport struct {
	IsValid  bool                `json:"is_valid"`
	Errors   []ValidationError   `json:"errors"`
	Warnings []ValidationWarning `json:"warnings"`
}

// SuccessReport returns a ValidationReport indicating success.
func SuccessReport() ValidationReport {
	return ValidationReport{IsValid: true, Errors: nil, Warnings: nil}
}

// IsFullyValid returns true when there are no errors and no warnings.
func (r *ValidationReport) IsFullyValid() bool {
	return r.IsValid && len(r.Warnings) == 0
}

// ValidateDryContext validates the dry context against the input schema.
// If no input schema is set, returns a success report.
func (m *OpMetadata) ValidateDryContext(ctx *DryContext) (ValidationReport, error) {
	if m.InputSchema == nil {
		return SuccessReport(), nil
	}
	return validateInputAgainstSchema(ctx.Values(), m.InputSchema, m.Name)
}

// ValidateWetContext validates the wet context against the reference schema.
// If no reference schema is set, returns a success report.
func (m *OpMetadata) ValidateWetContext(ctx *WetContext) (ValidationReport, error) {
	if m.ReferenceSchema == nil {
		return SuccessReport(), nil
	}
	return validateReferenceSchema(ctx, m.ReferenceSchema)
}

// ValidateContexts validates both dry and wet contexts and merges the reports.
func (m *OpMetadata) ValidateContexts(dry *DryContext, wet *WetContext) (ValidationReport, error) {
	dryReport, err := m.ValidateDryContext(dry)
	if err != nil {
		return ValidationReport{}, err
	}
	wetReport, err := m.ValidateWetContext(wet)
	if err != nil {
		return ValidationReport{}, err
	}
	return ValidationReport{
		IsValid:  dryReport.IsValid && wetReport.IsValid,
		Errors:   append(dryReport.Errors, wetReport.Errors...),
		Warnings: append(dryReport.Warnings, wetReport.Warnings...),
	}, nil
}

// ValidateOutput validates a serializable output against the output schema.
func (m *OpMetadata) ValidateOutput(output any) (ValidationReport, error) {
	if m.OutputSchema == nil {
		return SuccessReport(), nil
	}
	data, err := json.Marshal(output)
	if err != nil {
		return ValidationReport{}, NewContextError(fmt.Sprintf("Failed to serialize output: %v", err))
	}
	return validateJSONAgainstSchema(data, m.OutputSchema)
}

// validateInputAgainstSchema checks required fields only (simplified schema validation).
func validateInputAgainstSchema(values map[string]json.RawMessage, schema json.RawMessage, _ string) (ValidationReport, error) {
	var schemaObj map[string]json.RawMessage
	if err := json.Unmarshal(schema, &schemaObj); err != nil {
		return ValidationReport{}, nil
	}
	var errors []ValidationError
	if reqRaw, ok := schemaObj["required"]; ok {
		var required []string
		if err := json.Unmarshal(reqRaw, &required); err == nil {
			for _, field := range required {
				if _, present := values[field]; !present {
					errors = append(errors, ValidationError{
						Field:   field,
						Message: fmt.Sprintf("Required field '%s' is missing", field),
					})
				}
			}
		}
	}
	return ValidationReport{IsValid: len(errors) == 0, Errors: errors}, nil
}

// validateReferenceSchema checks required reference keys in WetContext.
func validateReferenceSchema(ctx *WetContext, schema json.RawMessage) (ValidationReport, error) {
	var schemaObj map[string]json.RawMessage
	if err := json.Unmarshal(schema, &schemaObj); err != nil {
		return ValidationReport{}, nil
	}
	var errors []ValidationError
	if reqRaw, ok := schemaObj["required"]; ok {
		var required []string
		if err := json.Unmarshal(reqRaw, &required); err == nil {
			for _, field := range required {
				if !ctx.Contains(field) {
					errors = append(errors, ValidationError{
						Field:   field,
						Message: fmt.Sprintf("Required reference '%s' is missing", field),
					})
				}
			}
		}
	}
	return ValidationReport{IsValid: len(errors) == 0, Errors: errors}, nil
}

// validateJSONAgainstSchema validates JSON data against a JSON schema (required fields only).
func validateJSONAgainstSchema(data json.RawMessage, schema json.RawMessage) (ValidationReport, error) {
	var valueObj map[string]json.RawMessage
	if err := json.Unmarshal(data, &valueObj); err != nil {
		return ValidationReport{IsValid: true}, nil // non-object output, skip
	}
	return validateInputAgainstSchema(valueObj, schema, "")
}

// TriggerFuse represents a saved request to execute an op at a later time.
type TriggerFuse struct {
	ID          string      `json:"id"`
	TriggerName string      `json:"trigger_name"`
	DryCtx      DryContext  `json:"dry_context"`
	CreatedAt   time.Time   `json:"created_at"`
	Metadata    *OpMetadata `json:"metadata,omitempty"`
}

// NewTriggerFuse creates a TriggerFuse with a generated ID and current timestamp.
func NewTriggerFuse(triggerName string) *TriggerFuse {
	return &TriggerFuse{
		ID:          uuid.New().String(),
		TriggerName: triggerName,
		DryCtx:      *NewDryContext(),
		CreatedAt:   time.Now().UTC(),
	}
}

// WithData adds a key/value pair to the trigger fuse's dry context.
func (tf *TriggerFuse) WithData(key string, value any) *TriggerFuse {
	tf.DryCtx.Insert(key, value)
	return tf
}

// WithMetadata attaches metadata to the trigger fuse for validation.
func (tf *TriggerFuse) WithMetadata(metadata OpMetadata) *TriggerFuse {
	tf.Metadata = &metadata
	return tf
}

// ValidateAndGetDryContext validates (if metadata is set) and returns the dry context.
func (tf *TriggerFuse) ValidateAndGetDryContext() (*DryContext, error) {
	if tf.Metadata != nil {
		report, err := tf.Metadata.ValidateDryContext(&tf.DryCtx)
		if err != nil {
			return nil, err
		}
		if !report.IsValid {
			return nil, NewContextError(fmt.Sprintf("Invalid dry context: %v", report.Errors))
		}
	}
	clone := tf.DryCtx.Clone()
	return clone, nil
}
