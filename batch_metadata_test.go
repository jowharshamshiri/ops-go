package ops_test

import (
	"encoding/json"
	"testing"

	ops "github.com/filegrind/ops-go"
)

// producerOp has an input schema requiring "initial_input" and an output schema with "produced_value".
type producerOp struct{ ops.DefaultRollback }

func (o *producerOp) Perform(dry *ops.DryContext, _ *ops.WetContext) (string, error) {
	input, err := ops.DryGetRequired[string](dry, "initial_input")
	if err != nil {
		return "", err
	}
	return "produced_" + input, nil
}

func (o *producerOp) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("ProducerOp").
		InputSchema(json.RawMessage(`{
			"type": "object",
			"properties": {
				"initial_input": {"type": "string"}
			},
			"required": ["initial_input"]
		}`)).
		OutputSchema(json.RawMessage(`{
			"type": "object",
			"properties": {
				"produced_value": {"type": "string"}
			}
		}`)).
		Build()
}

// consumerOp requires "produced_value" from a previous op.
type consumerOp struct{ ops.DefaultRollback }

func (o *consumerOp) Perform(dry *ops.DryContext, _ *ops.WetContext) (string, error) {
	val, err := ops.DryGetRequired[string](dry, "produced_value")
	if err != nil {
		return "", err
	}
	return "consumed_" + val, nil
}

func (o *consumerOp) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("ConsumerOp").
		InputSchema(json.RawMessage(`{
			"type": "object",
			"properties": {
				"produced_value": {"type": "string"}
			},
			"required": ["produced_value"]
		}`)).
		OutputSchema(json.RawMessage(`{
			"type": "object",
			"properties": {
				"final_result": {"type": "string"}
			}
		}`)).
		Build()
}

// TEST047: Build BatchMetadata from producer/consumer ops and verify only external inputs are required
func Test047_batch_metadata_with_data_flow(t *testing.T) {
	batchOps := []ops.Op[string]{&producerOp{}, &consumerOp{}}
	batch := ops.NewBatchOp(batchOps)
	metadata := batch.Metadata()

	if metadata.InputSchema == nil {
		t.Fatal("expected non-nil InputSchema")
	}

	var schemaObj map[string]json.RawMessage
	if err := json.Unmarshal(metadata.InputSchema, &schemaObj); err != nil {
		t.Fatalf("failed to parse input schema: %v", err)
	}

	reqRaw, ok := schemaObj["required"]
	if !ok {
		t.Fatal("expected 'required' in input schema")
	}
	var required []string
	if err := json.Unmarshal(reqRaw, &required); err != nil {
		t.Fatalf("failed to parse required: %v", err)
	}

	// Only "initial_input" should be required externally; "produced_value" is produced internally.
	if len(required) != 1 {
		t.Errorf("expected 1 required field, got %d: %v", len(required), required)
	}
	if len(required) > 0 && required[0] != "initial_input" {
		t.Errorf("expected required[0]='initial_input', got %q", required[0])
	}
}

// serviceAOp requires "service_a" reference.
type serviceAOp struct{ ops.DefaultRollback }

func (o *serviceAOp) Perform(_ *ops.DryContext, _ *ops.WetContext) (struct{}, error) {
	return struct{}{}, nil
}

func (o *serviceAOp) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("ServiceAOp").
		ReferenceSchema(json.RawMessage(`{
			"type": "object",
			"properties": {
				"service_a": {"type": "ServiceA"}
			},
			"required": ["service_a"]
		}`)).
		Build()
}

// serviceBOp requires "service_b" reference.
type serviceBOp struct{ ops.DefaultRollback }

func (o *serviceBOp) Perform(_ *ops.DryContext, _ *ops.WetContext) (struct{}, error) {
	return struct{}{}, nil
}

func (o *serviceBOp) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("ServiceBOp").
		ReferenceSchema(json.RawMessage(`{
			"type": "object",
			"properties": {
				"service_b": {"type": "ServiceB"}
			},
			"required": ["service_b"]
		}`)).
		Build()
}

// TEST048: Build BatchMetadata from two ops with different reference schemas and verify union of required refs
func Test048_reference_schema_merging(t *testing.T) {
	batchOps := []ops.Op[struct{}]{&serviceAOp{}, &serviceBOp{}}
	batch := ops.NewBatchOp(batchOps)
	metadata := batch.Metadata()

	if metadata.ReferenceSchema == nil {
		t.Fatal("expected non-nil ReferenceSchema")
	}

	var schemaObj map[string]json.RawMessage
	if err := json.Unmarshal(metadata.ReferenceSchema, &schemaObj); err != nil {
		t.Fatalf("failed to parse reference schema: %v", err)
	}

	reqRaw, ok := schemaObj["required"]
	if !ok {
		t.Fatal("expected 'required' in reference schema")
	}
	var required []string
	if err := json.Unmarshal(reqRaw, &required); err != nil {
		t.Fatalf("failed to parse required: %v", err)
	}

	// Both services should be required.
	if len(required) != 2 {
		t.Errorf("expected 2 required refs, got %d: %v", len(required), required)
	}

	found := map[string]bool{}
	for _, r := range required {
		found[r] = true
	}
	if !found["service_a"] {
		t.Error("expected 'service_a' in required refs")
	}
	if !found["service_b"] {
		t.Error("expected 'service_b' in required refs")
	}
}
