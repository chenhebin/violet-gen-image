package provider

import (
	"context"
	"errors"
	"fmt"
)

type ErrorKind string

const (
	ErrorInvalidConfig    ErrorKind = "invalid_config"
	ErrorInvalidRequest   ErrorKind = "invalid_request"
	ErrorUnsafeURL        ErrorKind = "unsafe_url"
	ErrorTimeout          ErrorKind = "timeout"
	ErrorUnavailable      ErrorKind = "unavailable"
	ErrorRejected         ErrorKind = "rejected"
	ErrorInvalidResponse  ErrorKind = "invalid_response"
	ErrorResponseTooLarge ErrorKind = "response_too_large"
)

// Error intentionally exposes only bounded, non-secret provider metadata.
// The upstream response body and API key are never retained.
type Error struct {
	Kind         ErrorKind
	Operation    string
	StatusCode   int
	ProviderCode string
	Retryable    bool
	Metadata     CallMetadata
	cause        error
}

func (e *Error) Error() string {
	if e == nil {
		return "provider request failed"
	}
	if e.StatusCode > 0 {
		return fmt.Sprintf("provider %s failed with status %d", e.Operation, e.StatusCode)
	}
	return fmt.Sprintf("provider %s failed: %s", e.Operation, e.Kind)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

func newError(kind ErrorKind, operation string, cause error) *Error {
	return &Error{
		Kind:      kind,
		Operation: operation,
		cause:     safeCause(cause),
	}
}

func attachMetadata(err error, metadata CallMetadata) {
	if err == nil {
		return
	}
	var providerErr *Error
	if errors.As(err, &providerErr) {
		providerErr.Metadata = metadata
	}
}

func withMetadata(err error, metadata CallMetadata) error {
	attachMetadata(err, metadata)
	return err
}

func safeCause(err error) error {
	if err == nil {
		return nil
	}
	// Preserve useful classifications without retaining a provider response,
	// URL query string, authorization header, or other potentially secret text.
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return context.DeadlineExceeded
	case errors.Is(err, context.Canceled):
		return context.Canceled
	default:
		return errors.New("provider transport error")
	}
}
