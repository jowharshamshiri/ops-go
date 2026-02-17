package ops

import (
	"log"
	"time"
)

// TimeBoundWrapper wraps an Op[T] and enforces a timeout.
type TimeBoundWrapper[T any] struct {
	wrappedOp   Op[T]
	timeout     time.Duration
	triggerName string
	warnOnTimeout bool
}

// NewTimeBoundWrapper creates a TimeBoundWrapper with the given timeout.
func NewTimeBoundWrapper[T any](op Op[T], timeout time.Duration) *TimeBoundWrapper[T] {
	return &TimeBoundWrapper[T]{wrappedOp: op, timeout: timeout, warnOnTimeout: true}
}

// NewTimeBoundWrapperWithName creates a TimeBoundWrapper with an explicit name.
func NewTimeBoundWrapperWithName[T any](op Op[T], timeout time.Duration, name string) *TimeBoundWrapper[T] {
	return &TimeBoundWrapper[T]{wrappedOp: op, timeout: timeout, triggerName: name, warnOnTimeout: true}
}

// NewTimeBoundWrapperWithWarningControl creates a TimeBoundWrapper with configurable timeout warnings.
func NewTimeBoundWrapperWithWarningControl[T any](op Op[T], timeout time.Duration, warn bool) *TimeBoundWrapper[T] {
	return &TimeBoundWrapper[T]{wrappedOp: op, timeout: timeout, warnOnTimeout: warn}
}

func (t *TimeBoundWrapper[T]) getTriggerName() string {
	if t.triggerName != "" {
		return t.triggerName
	}
	return "TimeBoundOp"
}

func (t *TimeBoundWrapper[T]) logTimeoutWarning() {
	if t.warnOnTimeout {
		log.Printf("Op '%s' was terminated due to timeout after %v", t.getTriggerName(), t.timeout)
	}
}

func (t *TimeBoundWrapper[T]) logNearTimeoutCompletion(duration time.Duration) {
	ratio := float64(duration) / float64(t.timeout)
	if ratio > 0.8 {
		log.Printf("Op '%s' completed in %.3fs (%.0f%% of %v timeout)",
			t.getTriggerName(), duration.Seconds(), ratio*100, t.timeout)
	}
}

// Perform executes the wrapped op with a timeout.
func (t *TimeBoundWrapper[T]) Perform(dry *DryContext, wet *WetContext) (T, error) {
	type result struct {
		val T
		err error
	}
	ch := make(chan result, 1)

	start := time.Now()
	go func() {
		val, err := t.wrappedOp.Perform(dry, wet)
		ch <- result{val: val, err: err}
	}()

	select {
	case r := <-ch:
		duration := time.Since(start)
		t.logNearTimeoutCompletion(duration)
		return r.val, r.err
	case <-time.After(t.timeout):
		t.logTimeoutWarning()
		var zero T
		return zero, NewTimeoutError(uint64(t.timeout.Milliseconds()))
	}
}

// Metadata delegates to the wrapped op's metadata with timeout info appended.
func (t *TimeBoundWrapper[T]) Metadata() OpMetadata {
	meta := t.wrappedOp.Metadata()
	if t.triggerName != "" {
		desc := t.triggerName + " (timeout: " + t.timeout.String() + ")"
		meta.Description = &desc
	}
	return meta
}

// Rollback delegates to the wrapped op's rollback.
func (t *TimeBoundWrapper[T]) Rollback(dry *DryContext, wet *WetContext) error {
	return t.wrappedOp.Rollback(dry, wet)
}

// CreateTimeoutWrapperWithCallerName creates a TimeBoundWrapper using the caller's location as the name.
func CreateTimeoutWrapperWithCallerName[T any](op Op[T], timeout time.Duration) *TimeBoundWrapper[T] {
	callerName := GetCallerTriggerName()
	return NewTimeBoundWrapperWithName(op, timeout, callerName)
}

// CreateLoggedTimeoutWrapper creates a TimeBoundWrapper wrapped in a LoggingWrapper.
func CreateLoggedTimeoutWrapper[T any](op Op[T], timeout time.Duration, triggerName string) *LoggingWrapper[T] {
	timeoutWrapper := NewTimeBoundWrapperWithName(op, timeout, triggerName)
	return NewLoggingWrapper[T](timeoutWrapper, "TimeBound["+triggerName+"]")
}
