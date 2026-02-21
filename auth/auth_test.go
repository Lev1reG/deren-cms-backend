package auth

import (
	"context"
	"testing"
)

func TestValidateToken_Empty(t *testing.T) {
	t.Parallel()
	_, err := ValidateToken(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty token")
	}
}

func TestValidateToken_Invalid(t *testing.T) {
	t.Parallel()
	_, err := ValidateToken(context.Background(), "invalid-token")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}
