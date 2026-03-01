package ops_test

import (
	"strings"
	"testing"
	"time"

	ops "github.com/machinefabric/ops-go"
)

// integrationFailingOp always fails.
type integrationFailingOp struct{ ops.DefaultRollback }

func (o *integrationFailingOp) Perform(_ *ops.DryContext, _ *ops.WetContext) (string, error) {
	return "", ops.NewExecutionFailedError("Simulated failure")
}
func (o *integrationFailingOp) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("FailingOp").
		Description("An op that always fails").
		Build()
}

// integrationSlowOp sleeps for 200ms.
type integrationSlowOp struct{ ops.DefaultRollback }

func (o *integrationSlowOp) Perform(_ *ops.DryContext, _ *ops.WetContext) (string, error) {
	time.Sleep(200 * time.Millisecond)
	return "should_timeout", nil
}
func (o *integrationSlowOp) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("SlowOp").
		Description("An op that takes a long time").
		Build()
}

// TEST083: Compose timeout and logging wrappers around a failing op and verify the error message includes the op name
func Test083_error_handling_and_wrapper_chains(t *testing.T) {
	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	// Wrap with timeout then logging
	timeoutOp := ops.NewTimeBoundWrapperWithName[string](&integrationFailingOp{}, 50*time.Millisecond, "FailingOp")
	loggedOp := ops.NewLoggingWrapper[string](timeoutOp, "TestFailure")

	_, err := loggedOp.Perform(dry, wet)
	if err == nil {
		t.Fatal("expected error from failing op")
	}
	oe := ops.AsOpError(err)
	if oe == nil || oe.Kind != ops.ErrExecutionFailed {
		t.Errorf("expected wrapped ExecutionFailed error, got %v", err)
	}
	if !strings.Contains(oe.Message, "TestFailure") {
		t.Errorf("expected error message to contain 'TestFailure', got %q", oe.Message)
	}
}

// TEST084: Call GetCallerTriggerName from within a test and verify it reflects the integration test module path
func Test084_stack_trace_analysis(t *testing.T) {
	triggerName := ops.GetCallerTriggerName()
	if !strings.Contains(triggerName, "integration_test") {
		t.Errorf("expected trigger name to contain 'integration_test', got %q", triggerName)
	}
	if !strings.Contains(triggerName, ".") {
		t.Errorf("expected trigger name to contain '.', got %q", triggerName)
	}
}

// TEST085: Use WrapNestedOpException and verify the wrapped error contains both op name and original message
func Test085_exception_wrapping_utilities(t *testing.T) {
	originalErr := ops.NewExecutionFailedError("original")
	wrapped := ops.WrapNestedOpException("TestOp", originalErr)

	oe := ops.AsOpError(wrapped)
	if oe == nil || oe.Kind != ops.ErrExecutionFailed {
		t.Errorf("expected wrapped ExecutionFailed, got %v", wrapped)
	}
	if !strings.Contains(oe.Message, "TestOp") {
		t.Errorf("expected message to contain 'TestOp', got %q", oe.Message)
	}
	if !strings.Contains(oe.Message, "original") {
		t.Errorf("expected message to contain 'original', got %q", oe.Message)
	}
}

// TEST086: Wrap a slow op in a short-timeout TimeBoundWrapper and verify the error is wrapped with logging context
func Test086_timeout_wrapper_functionality(t *testing.T) {
	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	timeoutOp := ops.NewTimeBoundWrapperWithName[string](&integrationSlowOp{}, 50*time.Millisecond, "SlowOp")
	loggedTimeoutOp := ops.NewLoggingWrapper[string](timeoutOp, "TimeoutTest")

	_, err := loggedTimeoutOp.Perform(dry, wet)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	oe := ops.AsOpError(err)
	if oe == nil || oe.Kind != ops.ErrExecutionFailed {
		t.Errorf("expected wrapped ExecutionFailed error, got %v", err)
	}
	if !strings.Contains(oe.Message, "TimeoutTest") {
		t.Errorf("expected message to contain 'TimeoutTest', got %q", oe.Message)
	}
}

// integrationConfig is a simple config struct.
type integrationConfig struct {
	DatabaseURL string
	Timeout     int
}

// integrationConfigService provides configuration.
type integrationConfigService struct{}

func (s *integrationConfigService) GetConfig() integrationConfig {
	return integrationConfig{
		DatabaseURL: "postgres://localhost:5432/test",
		Timeout:     30,
	}
}

// integrationConfigOp reads a config service from WetContext.
type integrationConfigOp struct{ ops.DefaultRollback }

func (o *integrationConfigOp) Perform(_ *ops.DryContext, wet *ops.WetContext) (integrationConfig, error) {
	svc, err := ops.WetGetRefRequired[*integrationConfigService](wet, "config_service")
	if err != nil {
		return integrationConfig{}, err
	}
	return svc.GetConfig(), nil
}

func (o *integrationConfigOp) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("ConfigOp").
		Description("Loads configuration from service").
		Build()
}

// TEST087: Run an op that retrieves a service from WetContext and reads config values from it
func Test087_dry_and_wet_context_usage(t *testing.T) {
	dry := ops.NewDryContext().
		WithValue("service", "user_service").
		WithValue("version", "1.0").
		WithValue("debug", true)

	wet := ops.NewWetContext().WithRef("config_service", &integrationConfigService{})

	op := &integrationConfigOp{}
	config, err := op.Perform(dry, wet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if config.DatabaseURL != "postgres://localhost:5432/test" {
		t.Errorf("expected database URL, got %q", config.DatabaseURL)
	}
	if config.Timeout != 30 {
		t.Errorf("expected timeout=30, got %d", config.Timeout)
	}
}

// integrationUser is a sample user struct.
type integrationUser struct {
	ID     uint32
	Name   string
	Email  string
	Active bool
}

// integrationUserOp builds a user from DryContext.
type integrationUserOp struct{ ops.DefaultRollback }

func (o *integrationUserOp) Perform(dry *ops.DryContext, _ *ops.WetContext) (integrationUser, error) {
	userID, err := ops.DryGetRequired[uint32](dry, "user_id")
	if err != nil {
		return integrationUser{}, err
	}
	name, err := ops.DryGetRequired[string](dry, "name")
	if err != nil {
		return integrationUser{}, err
	}
	email, err := ops.DryGetRequired[string](dry, "email")
	if err != nil {
		return integrationUser{}, err
	}
	return integrationUser{ID: userID, Name: name, Email: email, Active: true}, nil
}

func (o *integrationUserOp) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("UserOp").
		Description("Creates a user from context data").
		Build()
}

// TEST088: Run a BatchOp with two identical user-building ops and verify both produce the expected User struct
func Test088_batch_ops(t *testing.T) {
	dry := ops.NewDryContext().
		WithValue("user_id", uint32(1)).
		WithValue("name", "John Doe").
		WithValue("email", "john@example.com")
	wet := ops.NewWetContext()

	batchOps := []ops.Op[integrationUser]{
		&integrationUserOp{},
		&integrationUserOp{},
	}
	batch := ops.NewBatchOp(batchOps)
	results, err := batch.Perform(dry, wet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for i, user := range results {
		if user.ID != 1 {
			t.Errorf("results[%d]: expected ID=1, got %d", i, user.ID)
		}
		if user.Name != "John Doe" {
			t.Errorf("results[%d]: expected name='John Doe', got %q", i, user.Name)
		}
		if user.Email != "john@example.com" {
			t.Errorf("results[%d]: expected email='john@example.com', got %q", i, user.Email)
		}
	}
}

// integrationSimpleOp returns a fixed string.
type integrationSimpleOp struct{ ops.DefaultRollback }

func (o *integrationSimpleOp) Perform(_ *ops.DryContext, _ *ops.WetContext) (string, error) {
	return "success", nil
}
func (o *integrationSimpleOp) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("SimpleOp").Build()
}

// TEST089: Compose TimeBoundWrapper and LoggingWrapper around a simple op and verify the result passes through
func Test089_wrapper_composition(t *testing.T) {
	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	timeoutOp := ops.NewTimeBoundWrapper[string](&integrationSimpleOp{}, 1*time.Second)
	loggedOp := ops.NewLoggingWrapper[string](timeoutOp, "ComposedOp")

	result, err := loggedOp.Perform(dry, wet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "success" {
		t.Errorf("expected 'success', got %q", result)
	}
}

// integrationAutoLoggedOp returns a fixed int.
type integrationAutoLoggedOp struct{ ops.DefaultRollback }

func (o *integrationAutoLoggedOp) Perform(_ *ops.DryContext, _ *ops.WetContext) (int, error) {
	return 42, nil
}
func (o *integrationAutoLoggedOp) Metadata() ops.OpMetadata {
	return ops.NewOpMetadataBuilder("AutoLoggedOp").Build()
}

// TEST090: Use the Perform() utility function directly and verify it returns the op result with auto-logging
func Test090_perform_utility(t *testing.T) {
	dry := ops.NewDryContext()
	wet := ops.NewWetContext()

	result, err := ops.Perform(&integrationAutoLoggedOp{}, dry, wet)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 42 {
		t.Errorf("expected 42, got %d", result)
	}
}
