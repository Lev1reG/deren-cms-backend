package database

import (
	"context"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	once sync.Once
	pool *pgxpool.Pool
)

// secrets holds the database connection secrets.
var secrets struct {
	SupabaseURL       string
	SupabaseDBPassword string
}

// Get returns the database connection pool, initializing it on first call.
func Get(ctx context.Context) (*pgxpool.Pool, error) {
	var initErr error
	once.Do(func() {
		pool, initErr = setup(ctx)
	})
	if initErr != nil {
		return nil, initErr
	}
	return pool, nil
}

// setup creates and configures the database connection pool.
func setup(ctx context.Context) (*pgxpool.Pool, error) {
	if secrets.SupabaseURL == "" {
		return nil, fmt.Errorf("SupabaseURL secret not configured")
	}
	if secrets.SupabaseDBPassword == "" {
		return nil, fmt.Errorf("SupabaseDBPassword secret not configured")
	}

	// Extract project ref from Supabase URL
	// URL format: https://xxxxx.supabase.co
	projectRef := extractProjectRef(secrets.SupabaseURL)
	if projectRef == "" {
		return nil, fmt.Errorf("failed to extract project ref from SupabaseURL")
	}

	// Build connection string using Session Pooler for better IPv4 support
	// Format: postgresql://postgres.[project-ref]:[password]@aws-1-ap-southeast-2.pooler.supabase.com:5432/postgres
	connString := fmt.Sprintf(
		"postgresql://postgres.%s:%s@aws-1-ap-southeast-2.pooler.supabase.com:5432/postgres",
		projectRef,
		secrets.SupabaseDBPassword,
	)

	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}

// extractProjectRef extracts the project reference from a Supabase URL.
func extractProjectRef(url string) string {
	// Expected format: https://xxxxx.supabase.co
	// Extract xxxxx
	if len(url) < 20 {
		return ""
	}
	// Remove https:// prefix
	if url[:8] == "https://" {
		url = url[8:]
	}
	// Find .supabase.co
	idx := len(url)
	for i, c := range url {
		if c == '.' {
			idx = i
			break
		}
	}
	return url[:idx]
}
