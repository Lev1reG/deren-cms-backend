package database

import (
	"context"
	"testing"
)

func TestGetPool(t *testing.T) {
	// This test verifies the pool can be obtained
	// In CI without secrets, this will be skipped
	t.Parallel()

	ctx := context.Background()
	pool, err := Get(ctx)
	if err != nil {
		t.Skipf("skipping test: database not configured: %v", err)
	}
	defer pool.Close()

	if pool == nil {
		t.Error("expected non-nil pool")
	}
}
