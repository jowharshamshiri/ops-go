package ops_test

import (
	"encoding/json"
	"strings"
	"testing"

	ops "github.com/filegrind/ops-go"
)

// validationTestOutput is the structured output for validating wrapper tests.
type validationTestOutput struct {
	Value int `json:"value"`
}

// validatedOp reads "value" from dry context, returns it as validationTestOutput.
type validatedOp struct{ ops.DefaultRollback }

func (o *validatedOp) Perform(dry *ops.DryContext, _ *ops.WetContext) (validationTestOutput, error) {
	value, err := ops.DryGetRequired[int](dry, "value")
	if err != nil {
		return validationTestOutput{}, err
	}
	return validationTestOutput{Value: value}, nil
}

func (o *validatedOp) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("ValidatedOp").
		Description("Op with schema validation").
		InputSchema(json.RawMessage(`{
			"type": "object",
			"properties": {
				"value": {"type": "integer", "minimum": 0, "maximum": 100}
			},
			"required": ["value"]
		}`)).
		OutputSchema(json.RawMessage(`{
			"type": "object",
			"properties": {
				"value": {"type": "integer"}
			},
			"required": ["value"]
		}`)).
		Build()
}

// TEST038: Run ValidatingWrapper with a valid input and verify the op executes and returns the result
func Test038_valid_input_output(t *testing.T) {
	validator := ops.NewValidatingWrapper[validationTestOutput](&validatedOp{})

	dry := ops.NewDryContext().WithValue("value", 42)
	wet := ops.NewWetContext()

	result, err := validator.Perform(dry, wet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Value != 42 {
		t.Errorf("expected value=42, got %d", result.Value)
	}
}

// TEST039: Run ValidatingWrapper without a required input field and verify a Context validation error
func Test039_invalid_input_missing_required(t *testing.T) {
	validator := ops.NewValidatingWrapper[validationTestOutput](&validatedOp{})

	dry := ops.NewDryContext() // missing "value"
	wet := ops.NewWetContext()

	_, err := validator.Perform(dry, wet)
	if err == nil {
		t.Fatal("expected validation error")
	}
	oe := ops.AsOpError(err)
	if oe == nil || oe.Kind != ops.ErrContext {
		t.Errorf("expected ErrContext, got %v", err)
	}
	if !strings.Contains(oe.Message, "Input validation failed") {
		t.Errorf("expected 'Input validation failed' in message, got %q", oe.Message)
	}
}

// TEST040: Run ValidatingWrapper with an input exceeding the schema maximum and verify a validation error
func Test040_invalid_input_out_of_range(t *testing.T) {
	validator := ops.NewValidatingWrapper[validationTestOutput](&validatedOp{})

	dry := ops.NewDryContext().WithValue("value", 150) // exceeds maximum of 100
	wet := ops.NewWetContext()

	_, err := validator.Perform(dry, wet)
	if err == nil {
		t.Fatal("expected validation error for value exceeding maximum")
	}
	oe := ops.AsOpError(err)
	if oe == nil || oe.Kind != ops.ErrContext {
		t.Errorf("expected ErrContext, got %v", err)
	}
	// gojsonschema reports exceeded maximum as "Must be less than or equal to"
	if !strings.Contains(oe.Message, "100") && !strings.Contains(oe.Message, "maximum") {
		t.Errorf("expected exceeded-maximum error in message, got %q", oe.Message)
	}
}

// TEST041: Use NewValidatingWrapperInputOnly and confirm input is validated while output is not
func Test041_input_only_validation(t *testing.T) {
	wrapper := ops.NewValidatingWrapperInputOnly[int](&noOutputSchemaOpImpl{})

	dry := ops.NewDryContext().WithValue("value", 42)
	wet := ops.NewWetContext()

	result, err := wrapper.Perform(dry, wet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 42 {
		t.Errorf("expected 42, got %d", result)
	}
}

type noOutputSchemaOpImpl struct{ ops.DefaultRollback }

func (o *noOutputSchemaOpImpl) Perform(dry *ops.DryContext, _ *ops.WetContext) (int, error) {
	return ops.DryGetRequired[int](dry, "value")
}

func (o *noOutputSchemaOpImpl) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("NoOutputSchemaOp").
		InputSchema(json.RawMessage(`{
			"type": "object",
			"properties": {
				"value": {"type": "integer"}
			},
			"required": ["value"]
		}`)).
		Build()
}

// TEST042: Use NewValidatingWrapperOutputOnly and confirm output is validated while input is not
func Test042_output_only_validation(t *testing.T) {
	op := &noInputSchemaOpImpl{}
	validator := ops.NewValidatingWrapperOutputOnly[validationTestOutput](op)

	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	result, err := validator.Perform(dry, wet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Value != 99 {
		t.Errorf("expected value=99, got %d", result.Value)
	}
}

type noInputSchemaOpImpl struct{ ops.DefaultRollback }

func (o *noInputSchemaOpImpl) Perform(_ *ops.DryContext, _ *ops.WetContext) (validationTestOutput, error) {
	return validationTestOutput{Value: 99}, nil
}

func (o *noInputSchemaOpImpl) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("NoInputSchemaOp").
		OutputSchema(json.RawMessage(`{
			"type": "object",
			"properties": {
				"value": {"type": "integer", "maximum": 100}
			},
			"required": ["value"]
		}`)).
		Build()
}

// TEST043: Wrap an op with no schemas in ValidatingWrapper and confirm it still succeeds
func Test043_no_schema_validation(t *testing.T) {
	op := &noSchemaOpImpl{}
	validator := ops.NewValidatingWrapper[int](op)

	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	result, err := validator.Perform(dry, wet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 123 {
		t.Errorf("expected 123, got %d", result)
	}
}

type noSchemaOpImpl struct{ ops.DefaultRollback }

func (o *noSchemaOpImpl) Perform(_ *ops.DryContext, _ *ops.WetContext) (int, error) {
	return 123, nil
}

func (o *noSchemaOpImpl) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("NoSchemaOp").Build()
}

// TEST044: Verify ValidatingWrapper::Metadata() delegates to the inner op's metadata unchanged
func Test044_metadata_transparency(t *testing.T) {
	validator := ops.NewValidatingWrapper[validationTestOutput](&validatedOp{})
	metadata := validator.Metadata()

	if metadata.Name != "ValidatedOp" {
		t.Errorf("expected name 'ValidatedOp', got %q", metadata.Name)
	}
	if metadata.Description == nil || *metadata.Description != "Op with schema validation" {
		t.Errorf("expected description 'Op with schema validation', got %v", metadata.Description)
	}
	if metadata.InputSchema == nil {
		t.Error("expected InputSchema to be non-nil")
	}
	if metadata.OutputSchema == nil {
		t.Error("expected OutputSchema to be non-nil")
	}
}

// TEST045: Verify ValidatingWrapper checks reference_schema and rejects when required refs are missing
func Test045_reference_validation(t *testing.T) {
	op := &serviceRequiringOpImpl{}
	validator := ops.NewValidatingWrapper[string](op)

	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	// Missing all required references
	_, err := validator.Perform(dry, wet)
	if err == nil {
		t.Fatal("expected error when required references are missing")
	}
	oe := ops.AsOpError(err)
	if oe == nil || oe.Kind != ops.ErrContext {
		t.Errorf("expected ErrContext, got %v", err)
	}
	if !strings.Contains(oe.Message, "database") {
		t.Errorf("expected 'database' in error message, got %q", oe.Message)
	}

	// Add database but not cache
	wet.InsertRef("database", "postgresql")
	_, err = validator.Perform(dry, wet)
	if err == nil {
		t.Fatal("expected error when 'cache' reference is missing")
	}
	oe = ops.AsOpError(err)
	if oe == nil || oe.Kind != ops.ErrContext {
		t.Errorf("expected ErrContext, got %v", err)
	}
	if !strings.Contains(oe.Message, "cache") {
		t.Errorf("expected 'cache' in error message, got %q", oe.Message)
	}

	// Add all required references
	wet.InsertRef("cache", "redis")
	result, err := validator.Perform(dry, wet)
	if err != nil {
		t.Fatalf("unexpected error with all references present: %v", err)
	}
	if result != "Used service: postgresql" {
		t.Errorf("expected 'Used service: postgresql', got %q", result)
	}
}

type serviceRequiringOpImpl struct{ ops.DefaultRollback }

func (o *serviceRequiringOpImpl) Perform(_ *ops.DryContext, wet *ops.WetContext) (string, error) {
	service, err := ops.WetGetRefRequired[string](wet, "database")
	if err != nil {
		return "", err
	}
	return "Used service: " + service, nil
}

func (o *serviceRequiringOpImpl) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("ServiceRequiringOp").
		ReferenceSchema(json.RawMessage(`{
			"type": "object",
			"required": ["database", "cache"],
			"properties": {
				"database": {"type": "string"},
				"cache": {"type": "string"}
			}
		}`)).
		Build()
}

// TEST046: Wrap an op with no reference schema in ValidatingWrapper and confirm it succeeds
func Test046_no_reference_schema(t *testing.T) {
	op := &noRefSchemaOpImpl{}
	validator := ops.NewValidatingWrapper[int](op)

	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	result, err := validator.Perform(dry, wet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 456 {
		t.Errorf("expected 456, got %d", result)
	}
}

type noRefSchemaOpImpl struct{ ops.DefaultRollback }

func (o *noRefSchemaOpImpl) Perform(_ *ops.DryContext, _ *ops.WetContext) (int, error) {
	return 456, nil
}

func (o *noRefSchemaOpImpl) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("NoRefSchemaOp").Build()
}

// TEST112: Verify ValidatingWrapper::output_only validates references even when input validation is disabled
func Test112_output_only_still_validates_references(t *testing.T) {
	op := &refRequiringOpImpl{}
	validator := ops.NewValidatingWrapperOutputOnly[int](op)

	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	// Missing required reference - must be rejected even though input validation is off
	_, err := validator.Perform(dry, wet)
	if err == nil {
		t.Fatal("output_only must still validate references")
	}
	oe := ops.AsOpError(err)
	if oe == nil || oe.Kind != ops.ErrContext {
		t.Errorf("expected ErrContext, got %v", err)
	}
	if !strings.Contains(oe.Message, "database") {
		t.Errorf("expected 'database' in message, got %q", oe.Message)
	}

	// With the reference present - must succeed
	wet.InsertRef("database", "postgres://localhost")
	result, err := validator.Perform(dry, wet)
	if err != nil {
		t.Fatalf("unexpected error when required reference is present: %v", err)
	}
	if result != 42 {
		t.Errorf("expected 42, got %d", result)
	}
}

type refRequiringOpImpl struct{ ops.DefaultRollback }

func (o *refRequiringOpImpl) Perform(_ *ops.DryContext, _ *ops.WetContext) (int, error) {
	return 42, nil
}

func (o *refRequiringOpImpl) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("RefRequiringOp").
		ReferenceSchema(json.RawMessage(`{
			"type": "object",
			"required": ["database"],
			"properties": {
				"database": {"type": "string"}
			}
		}`)).
		Build()
}
