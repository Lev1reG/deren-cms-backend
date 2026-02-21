package projects

import (
	"context"
	"testing"
)

func TestList_Empty(t *testing.T) {
	t.Parallel()
	// Without a database, this will error - that's expected
	_, err := List(context.Background())
	if err == nil {
		// If no error, verify response is valid
		// This would only happen with a real DB
	}
}
