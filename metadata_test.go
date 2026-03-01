package ops_test

import (
	"encoding/json"
	"testing"

	ops "github.com/machinefabric/ops-go"
)

// TEST021: Build OpMetadata with name, description, and schemas and verify all fields are populated
func Test021_metadata_builder(t *testing.T) {
	inputSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"name": {"type": "string"}
		},
		"required": ["name"]
	}`)
	outputSchema := json.RawMessage(`{"type": "string"}`)

	metadata := ops.NewOpMetadataBuilder("TestOp").
		Description("A test operation").
		InputSchema(inputSchema).
		OutputSchema(outputSchema).
		Build()

	if metadata.Name != "TestOp" {
		t.Errorf("expected name 'TestOp', got %q", metadata.Name)
	}
	if metadata.Description == nil || *metadata.Description != "A test operation" {
		t.Errorf("expected description 'A test operation', got %v", metadata.Description)
	}
	if metadata.InputSchema == nil {
		t.Error("expected InputSchema to be non-nil")
	}
	if metadata.OutputSchema == nil {
		t.Error("expected OutputSchema to be non-nil")
	}
}

// TEST022: Construct a TriggerFuse with data and verify the trigger name and dry context values
func Test022_trigger_fuse(t *testing.T) {
	tf := ops.NewTriggerFuse("ProcessImage").
		WithData("image_path", "/tmp/test.jpg").
		WithData("width", 800)

	if tf.TriggerName != "ProcessImage" {
		t.Errorf("expected trigger_name 'ProcessImage', got %q", tf.TriggerName)
	}

	imagePath, ok := ops.DryGet[string](&tf.DryCtx, "image_path")
	if !ok || imagePath != "/tmp/test.jpg" {
		t.Errorf("expected image_path '/tmp/test.jpg', got %q ok=%v", imagePath, ok)
	}

	width, ok := ops.DryGet[int](&tf.DryCtx, "width")
	if !ok || width != 800 {
		t.Errorf("expected width=800, got %d ok=%v", width, ok)
	}
}

// TEST023: Validate a DryContext against an input schema and confirm valid/invalid reports
func Test023_basic_validation(t *testing.T) {
	schema := json.RawMessage(`{
		"type": "object",
		"required": ["name"]
	}`)

	metadata := ops.NewOpMetadataBuilder("TestOp").
		InputSchema(schema).
		Build()

	validCtx := ops.NewDryContext().WithValue("name", "test")
	report, err := metadata.ValidateDryContext(validCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !report.IsValid {
		t.Error("expected report to be valid with required field present")
	}

	emptyCtx := ops.NewDryContext()
	report, err = metadata.ValidateDryContext(emptyCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.IsValid {
		t.Error("expected report to be invalid with required field missing")
	}
	if len(report.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(report.Errors))
	}
}
