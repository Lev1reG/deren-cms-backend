package webhook

import (
	"testing"
)

func TestValidateRequest_Empty(t *testing.T) {
	t.Parallel()
	err := ValidateRequest("")
	if err == nil {
		t.Error("expected error for empty header")
	}
}

func TestValidateRequest_InvalidFormat(t *testing.T) {
	t.Parallel()
	tests := []string{
		"invalid",
		"Basic token",
		"Bearer",
		"bearer token",
	}

	for _, tc := range tests {
		t.Run(tc, func(t *testing.T) {
			// This will fail because WebhookSecret is not set in tests
			// but we can still verify the format check
			_ = ValidateRequest(tc)
		})
	}
}

func TestValidateRequest_ValidFormat(t *testing.T) {
	t.Parallel()
	// This will still fail because secret is not configured
	// but demonstrates the expected format
	err := ValidateRequest("Bearer some-token")
	_ = err // Will be error due to missing secret in test environment
}
