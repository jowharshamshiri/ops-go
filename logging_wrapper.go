package ops

import (
	"fmt"
	"log"
	"time"
)

// ANSI color codes for console output.
const (
	ANSIYellow = "\x1b[33m"
	ANSIGreen  = "\x1b[32m"
	ANSIRed    = "\x1b[31m"
	ANSIReset  = "\x1b[0m"
)

// LoggingWrapper wraps an Op[T] and logs start, success, and failure events with timing.
type LoggingWrapper[T any] struct {
	wrappedOp   Op[T]
	triggerName string
	loggerName  string
}

// NewLoggingWrapper creates a LoggingWrapper with the given trigger name.
func NewLoggingWrapper[T any](op Op[T], name string) *LoggingWrapper[T] {
	return &LoggingWrapper[T]{wrappedOp: op, triggerName: name}
}

// NewLoggingWrapperWithLogger creates a LoggingWrapper with a custom logger name.
func NewLoggingWrapperWithLogger[T any](op Op[T], name, loggerName string) *LoggingWrapper[T] {
	return &LoggingWrapper[T]{wrappedOp: op, triggerName: name, loggerName: loggerName}
}

func (lw *LoggingWrapper[T]) getLoggerName() string {
	if lw.loggerName != "" {
		return lw.loggerName
	}
	return "LoggingWrapper"
}

func (lw *LoggingWrapper[T]) logOpStart() {
	log.Printf("[%s] %sStarting op: %s%s", lw.getLoggerName(), ANSIYellow, lw.triggerName, ANSIReset)
}

func (lw *LoggingWrapper[T]) logOpSuccess(duration time.Duration) {
	log.Printf("[%s] %sOp '%s' completed in %.3f seconds%s",
		lw.getLoggerName(), ANSIGreen, lw.triggerName, duration.Seconds(), ANSIReset)
}

func (lw *LoggingWrapper[T]) logOpFailure(err error, duration time.Duration) {
	log.Printf("[%s] %sOp '%s' failed after %.3f seconds: %v%s",
		lw.getLoggerName(), ANSIRed, lw.triggerName, duration.Seconds(), err, ANSIReset)
}

// Perform executes the wrapped op with logging.
func (lw *LoggingWrapper[T]) Perform(dry *DryContext, wet *WetContext) (T, error) {
	var zero T
	start := time.Now()
	lw.logOpStart()

	result, err := lw.wrappedOp.Perform(dry, wet)
	duration := time.Since(start)

	if err != nil {
		lw.logOpFailure(err, duration)
		// Re-wrap error with op context (matches Rust behavior)
		wrapped := WrapNestedOpException(lw.triggerName,
			NewExecutionFailedError(fmt.Sprintf("%v", err)))
		return zero, wrapped
	}
	lw.logOpSuccess(duration)
	return result, nil
}

// Metadata delegates to the wrapped op's metadata.
func (lw *LoggingWrapper[T]) Metadata() OpMetadata {
	return lw.wrappedOp.Metadata()
}

// Rollback delegates to the wrapped op's rollback.
func (lw *LoggingWrapper[T]) Rollback(dry *DryContext, wet *WetContext) error {
	return lw.wrappedOp.Rollback(dry, wet)
}

// CreateContextAwareLogger creates a LoggingWrapper using the caller's location as the name.
func CreateContextAwareLogger[T any](op Op[T]) *LoggingWrapper[T] {
	callerName := GetCallerTriggerName()
	return NewLoggingWrapperWithLogger(op, callerName, callerName)
}
