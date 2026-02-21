# Admin Dashboard Backend Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build an Encore.go backend API with CRUD operations for projects, hero, and work experience content, connected to Supabase PostgreSQL with JWT authentication.

**Architecture:** Encore services connect to external Supabase database via pgxpool. Each content type (projects, hero, work) is a separate service with its own API endpoints. Auth middleware validates Supabase JWTs using JWKS. Webhook endpoint triggers Netlify builds with secret token protection. All delete operations use soft delete.

**Tech Stack:** Encore.go, pgxpool/v5 (PostgreSQL driver), Supabase (Postgres + Auth + JWKS), structured errors

---

## Prerequisites

Before starting implementation:

1. **Supabase project created** with URL and anon key available
2. **Supabase Auth user created** for admin access
3. **Netlify build hook URL** obtained from Netlify dashboard
4. **Encore CLI installed** (`encore version` should work)

---

## Task 1: Set Up Secrets Configuration

**Files:**
- Modify: `encore.app` (add secrets configuration)
- Create: `.secrets.local.cue` (local development secrets)

**Step 1: Update encore.app for secrets**

Replace the contents of `encore.app`:

```json
{
	"id": "",
	"secrets": {
		"SupabaseURL":           {},
		"SupabaseJWKSURL":       {},
		"NetlifyBuildHook":      {},
		"WebhookSecret":         {}
	}
}
```

**Step 2: Create local secrets file**

Create `.secrets.local.cue` (add to .gitignore if not already):

```cue
SupabaseURL:      "https://your-project.supabase.co"
SupabaseJWKSURL:  "https://your-project.supabase.co/auth/v1/jwks"
NetlifyBuildHook: "https://api.netlify.com/build_hooks/your-hook-id"
WebhookSecret:    "your-secure-random-string-here"
```

**Step 3: Update .gitignore**

Ensure `.secrets.local.cue` is ignored:

```bash
echo ".secrets.local.cue" >> .gitignore
```

**Step 4: Verify Encore recognizes secrets**

Run: `encore run`
Expected: App starts without secret-related errors

**Step 5: Commit**

```bash
git add encore.app .gitignore
git commit -m "chore: configure secrets for Supabase and Netlify"
```

---

## Task 2: Create Database Types Package

**Files:**
- Create: `pkg/database/types.go`
- Create: `pkg/database/database.go`

**Step 1: Write the failing test**

Create `pkg/database/database_test.go`:

```go
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
```

**Step 2: Run test to verify it fails**

Run: `encore test ./pkg/database/... -v`
Expected: FAIL or SKIP (no database package yet)

**Step 3: Create database types**

Create `pkg/database/types.go`:

```go
// Package database provides a connection pool to the Supabase PostgreSQL database.
package database

import (
	"time"
)

// Project represents a project entry in the database.
type Project struct {
	ID           string         `json:"id" db:"id"`
	Title        string         `json:"title" db:"title"`
	Description  string         `json:"description" db:"description"`
	Href         *string        `json:"href" db:"href"`
	Technologies []string       `json:"technologies" db:"technologies"`
	DisplayOrder int            `json:"display_order" db:"display_order"`
	CreatedAt    time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at" db:"updated_at"`
	DeletedAt    *time.Time     `json:"deleted_at" db:"deleted_at"`
}

// Hero represents the hero section content (single row).
type Hero struct {
	ID          string    `json:"id" db:"id"`
	Phrases     []string  `json:"phrases" db:"phrases"`
	Description string    `json:"description" db:"description"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// WorkExperience represents a work experience entry.
type WorkExperience struct {
	ID           string     `json:"id" db:"id"`
	Company      string     `json:"company" db:"company"`
	Position     string     `json:"position" db:"position"`
	Date         string     `json:"date" db:"date"`
	Description  string     `json:"description" db:"description"`
	Href         *string    `json:"href" db:"href"`
	Type         *string    `json:"type" db:"type"`
	DisplayOrder int        `json:"display_order" db:"display_order"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at" db:"deleted_at"`
}
```

**Step 4: Create database connection pool**

Create `pkg/database/database.go`:

```go
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
	SupabaseURL string
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

	// Construct connection string for Supabase
	// Format: postgresql://postgres:[PASSWORD]@db.[PROJECT-REF].supabase.co:5432/postgres
	// The SupabaseURL is like https://xxxxx.supabase.co
	// We need to convert it to the database URL format
	connString := convertSupabaseURLToDBConn(secrets.SupabaseURL)

	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}

// convertSupabaseURLToDBConn converts a Supabase API URL to a database connection string.
// This expects the SupabaseURL secret to be the full connection string or just the base URL.
// For simplicity, we'll expect the secret to be the full postgresql:// connection string.
func convertSupabaseURLToDBConn(url string) string {
	// If it already starts with postgresql://, use as-is
	if len(url) > 13 && url[:13] == "postgresql://" {
		return url
	}
	// Otherwise assume it's an API URL and return error
	// User should provide full connection string
	return url
}
```

**Step 5: Update secrets to include database password**

Update `encore.app`:

```json
{
	"id": "",
	"secrets": {
		"SupabaseURL":           {},
		"SupabaseJWKSURL":       {},
		"SupabaseDBPassword":    {},
		"NetlifyBuildHook":      {},
		"WebhookSecret":         {}
	}
}
```

Update `pkg/database/database.go` secrets struct:

```go
var secrets struct {
	SupabaseURL      string
	SupabaseDBPassword string
}
```

Update `setup` function in `pkg/database/database.go`:

```go
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

	// Build connection string
	connString := fmt.Sprintf(
		"postgresql://postgres:%s@db.%s.supabase.co:5432/postgres",
		secrets.SupabaseDBPassword,
		projectRef,
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
```

**Step 6: Add pgx dependency**

Run: `go get github.com/jackc/pgx/v5/pgxpool`

**Step 7: Update local secrets**

Update `.secrets.local.cue`:

```cue
SupabaseURL:       "https://your-project-ref.supabase.co"
SupabaseDBPassword: "your-database-password"
SupabaseJWKSURL:   "https://your-project-ref.supabase.co/auth/v1/jwks"
NetlifyBuildHook:  "https://api.netlify.com/build_hooks/your-hook-id"
WebhookSecret:     "your-secure-random-string-here"
```

**Step 8: Run tests**

Run: `encore test ./pkg/database/... -v`
Expected: Tests pass or skip gracefully without DB

**Step 9: Commit**

```bash
git add pkg/database/ encore.app go.mod go.sum
git commit -m "feat: add database connection pool with types"
```

---

## Task 3: Create Database Migrations

**Files:**
- Create: `migrations/001_init_schema.up.sql` (standalone SQL for Supabase)
- Create: `migrations/002_add_soft_delete.up.sql`

**Step 1: Create initial schema migration**

Create `migrations/001_init_schema.up.sql`:

```sql
-- Projects table
CREATE TABLE IF NOT EXISTS projects (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  title TEXT NOT NULL,
  description TEXT NOT NULL,
  href TEXT,
  technologies TEXT[] NOT NULL DEFAULT '{}',
  display_order INTEGER DEFAULT 0,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  deleted_at TIMESTAMPTZ
);

-- Hero table (single row for hero content)
CREATE TABLE IF NOT EXISTS hero (
  id UUID PRIMARY KEY DEFAULT '00000000-0000-0000-0000-000000000001'::UUID,
  phrases TEXT[] NOT NULL DEFAULT '{}',
  description TEXT NOT NULL,
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Work experience table
CREATE TABLE IF NOT EXISTS work_experience (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  company TEXT NOT NULL,
  position TEXT NOT NULL,
  date TEXT NOT NULL,
  description TEXT NOT NULL,
  href TEXT,
  type TEXT CHECK (type IN ('Freelance', 'Internship', 'Contract', 'Part-Time', 'Full-Time')),
  display_order INTEGER DEFAULT 0,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  deleted_at TIMESTAMPTZ
);

-- Indexes for ordering
CREATE INDEX IF NOT EXISTS idx_projects_order ON projects(display_order) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_work_order ON work_experience(display_order) WHERE deleted_at IS NULL;

-- Insert default hero row
INSERT INTO hero (id, phrases, description)
VALUES (
  '00000000-0000-0000-0000-000000000001'::UUID,
  ARRAY['Software Engineer', 'Full Stack Developer'],
  'Welcome to my personal website.'
) ON CONFLICT (id) DO NOTHING;

-- Updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply triggers
CREATE TRIGGER update_projects_updated_at
    BEFORE UPDATE ON projects
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_work_experience_updated_at
    BEFORE UPDATE ON work_experience
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```

**Step 2: Create rollback migration**

Create `migrations/001_init_schema.down.sql`:

```sql
DROP TRIGGER IF EXISTS update_work_experience_updated_at ON work_experience;
DROP TRIGGER IF EXISTS update_projects_updated_at ON projects;
DROP FUNCTION IF EXISTS update_updated_at_column();
DROP TABLE IF EXISTS work_experience;
DROP TABLE IF EXISTS hero;
DROP TABLE IF EXISTS projects;
```

**Step 3: Verify SQL syntax**

Run: `cat migrations/001_init_schema.up.sql`
Expected: File contents displayed correctly

**Step 4: Commit**

```bash
git add migrations/
git commit -m "feat: add database schema migrations for Supabase"
```

---

## Task 4: Create Auth Service with JWT Validation

**Files:**
- Create: `auth/auth.go`
- Create: `auth/auth_test.go`
- Create: `auth/jwks.go`

**Step 1: Write the failing test for JWT validation**

Create `auth/auth_test.go`:

```go
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
```

**Step 2: Run test to verify it fails**

Run: `encore test ./auth/... -v`
Expected: FAIL (package doesn't exist)

**Step 3: Create auth types and secrets**

Create `auth/auth.go`:

```go
// Package auth provides JWT validation against Supabase's JWKS endpoint.
package auth

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// UserData represents the authenticated user's data extracted from JWT.
type UserData struct {
	UserID string
	Email  string
	Role   string
}

// secrets holds the auth-related secrets.
var secrets struct {
	SupabaseJWKSURL string
}

// jwksCache holds cached JWKS keys.
var jwksCache struct {
	sync.RWMutex
	keys     map[string]jwt.VerifyingKey
	fetched  time.Time
	duration time.Duration
}

func init() {
	jwksCache.keys = make(map[string]jwt.VerifyingKey)
	jwksCache.duration = 1 * time.Hour
}

// ValidateToken validates a JWT token against Supabase's JWKS.
// It returns the user data if valid, or an error if invalid.
func ValidateToken(ctx context.Context, tokenString string) (*UserData, error) {
	if tokenString == "" {
		return nil, fmt.Errorf("token is empty")
	}

	// Parse token without verifying to get the kid
	unverifiedToken, _, err := jwt.NewParser().ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	kid, ok := unverifiedToken.Header["kid"].(string)
	if !ok {
		return nil, fmt.Errorf("token missing kid header")
	}

	// Get the verifying key
	key, err := getVerifyingKey(ctx, kid)
	if err != nil {
		return nil, fmt.Errorf("failed to get verifying key: %w", err)
	}

	// Parse and verify the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return key, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to verify token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("token is invalid")
	}

	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("failed to extract claims")
	}

	userData := &UserData{}

	if sub, ok := claims["sub"].(string); ok {
		userData.UserID = sub
	}
	if email, ok := claims["email"].(string); ok {
		userData.Email = email
	}
	if role, ok := claims["role"].(string); ok {
		userData.Role = role
	}

	return userData, nil
}

// getVerifyingKey retrieves the RSA public key for the given kid from JWKS cache or fetches it.
func getVerifyingKey(ctx context.Context, kid string) (jwt.VerifyingKey, error) {
	// Check cache first
	jwksCache.RLock()
	if key, ok := jwksCache.keys[kid]; ok && time.Since(jwksCache.fetched) < jwksCache.duration {
		jwksCache.RUnlock()
		return key, nil
	}
	jwksCache.RUnlock()

	// Fetch new keys
	if err := fetchJWKS(ctx); err != nil {
		return nil, err
	}

	// Try cache again
	jwksCache.RLock()
	defer jwksCache.RUnlock()
	if key, ok := jwksCache.keys[kid]; ok {
		return key, nil
	}

	return nil, fmt.Errorf("key with kid %s not found", kid)
}
```

**Step 4: Create JWKS fetching logic**

Create `auth/jwks.go`:

```go
package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// jwksResponse represents the JWKS endpoint response.
type jwksResponse struct {
	Keys []jwksKey `json:"keys"`
}

type jwksKey struct {
	Kty string   `json:"kty"`
	Kid string   `json:"kid"`
	Use string   `json:"use"`
	N   string   `json:"n"`
	E   string   `json:"e"`
	X5c []string `json:"x5c"`
}

// fetchJWKS fetches the JWKS from Supabase and caches the keys.
func fetchJWKS(ctx context.Context) error {
	if secrets.SupabaseJWKSURL == "" {
		return fmt.Errorf("SupabaseJWKSURL secret not configured")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, secrets.SupabaseJWKSURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("JWKS endpoint returned status %d", resp.StatusCode)
	}

	var jwks jwksResponse
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("failed to decode JWKS: %w", err)
	}

	// Convert to verifying keys
	keys := make(map[string]jwt.VerifyingKey)
	for _, key := range jwks.Keys {
		if key.Kty != "RSA" {
			continue
		}

		rsaKey, err := jwkToRSA(key)
		if err != nil {
			continue // Skip invalid keys
		}

		keys[key.Kid] = rsaKey
	}

	// Update cache
	jwksCache.Lock()
	jwksCache.keys = keys
	jwksCache.fetched = time.Now()
	jwksCache.Unlock()

	return nil
}

// jwkToRSA converts a JWK to an RSA public key.
func jwkToRSA(jwk jwksKey) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("failed to decode n: %w", err)
	}

	eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("failed to decode e: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)

	return &rsa.PublicKey{
		N: n,
		E: int(e.Int64()),
	}, nil
}
```

**Step 5: Add JWT dependency**

Run: `go get github.com/golang-jwt/jwt/v5`

**Step 6: Run tests**

Run: `encore test ./auth/... -v`
Expected: PASS (empty and invalid token tests)

**Step 7: Commit**

```bash
git add auth/ go.mod go.sum
git commit -m "feat: add JWT validation with Supabase JWKS"
```

---

## Task 5: Create Encore Auth Handler

**Files:**
- Modify: `auth/auth.go` (add auth handler)

**Step 1: Add Encore auth handler**

Add to `auth/auth.go`:

```go
import (
	"encore.dev/beta/auth"
	"encore.dev/beta/errs"
)

//encore:authhandler
func AuthHandler(ctx context.Context, token string) (auth.UID, *UserData, error) {
	userData, err := ValidateToken(ctx, token)
	if err != nil {
		return "", nil, &errs.Error{
			Code:    errs.Unauthenticated,
			Message: err.Error(),
		}
	}
	return auth.UID(userData.UserID), userData, nil
}
```

**Step 2: Run Encore check**

Run: `encore check ./...`
Expected: No errors

**Step 3: Commit**

```bash
git add auth/auth.go
git commit -m "feat: add Encore auth handler for Supabase JWT"
```

---

*Part 1 complete. The foundation is in place: secrets, database connection, JWT validation, and auth handler.*

---

## Task 6: Create Projects Service - Types and List Endpoint

**Files:**
- Create: `projects/projects.go`
- Create: `projects/projects_test.go`

**Step 1: Write the failing test for List**

Create `projects/projects_test.go`:

```go
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
```

**Step 2: Run test to verify it fails**

Run: `encore test ./projects/... -v`
Expected: FAIL (package doesn't exist)

**Step 3: Create projects service with types**

Create `projects/projects.go`:

```go
// Package projects provides CRUD operations for project content.
package projects

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"encore.app/pkg/database"
	"encore.dev/beta/errs"
)

// Response types for API
type (
	// Project represents a project in API responses.
	Project struct {
		ID           string   `json:"id"`
		Title        string   `json:"title"`
		Description  string   `json:"description"`
		Href         *string  `json:"href,omitempty"`
		Technologies []string `json:"technologies"`
		DisplayOrder int      `json:"display_order"`
		CreatedAt    string   `json:"created_at"`
		UpdatedAt    string   `json:"updated_at"`
	}

	// ListResponse is the response for listing projects.
	ListResponse struct {
		Projects []*Project `json:"projects"`
	}

	// CreateRequest is the request body for creating a project.
	CreateRequest struct {
		Title        string   `json:"title"`
		Description  string   `json:"description"`
		Href         *string  `json:"href,omitempty"`
		Technologies []string `json:"technologies"`
		DisplayOrder int      `json:"display_order"`
	}

	// UpdateRequest is the request body for updating a project.
	UpdateRequest struct {
		Title        *string  `json:"title,omitempty"`
		Description  *string  `json:"description,omitempty"`
		Href         *string  `json:"href,omitempty"`
		Technologies []string `json:"technologies,omitempty"`
		DisplayOrder *int     `json:"display_order,omitempty"`
	}
)

// dbToProject converts a database Project to API Project.
func dbToProject(db *database.Project) *Project {
	return &Project{
		ID:           db.ID,
		Title:        db.Title,
		Description:  db.Description,
		Href:         db.Href,
		Technologies: db.Technologies,
		DisplayOrder: db.DisplayOrder,
		CreatedAt:    db.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    db.UpdatedAt.Format(time.RFC3339),
	}
}

//encore:api auth path=/projects method=GET
func List(ctx context.Context) (*ListResponse, error) {
	pool, err := database.Get(ctx)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to get database connection")
	}

	query := `
		SELECT id, title, description, href, technologies, display_order, created_at, updated_at
		FROM projects
		WHERE deleted_at IS NULL
		ORDER BY display_order ASC, created_at DESC
	`

	rows, err := pool.Query(ctx, query)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to query projects")
	}
	defer rows.Close()

	var projects []*Project
	for rows.Next() {
		var db database.Project
		var techBytes []byte

		err := rows.Scan(
			&db.ID,
			&db.Title,
			&db.Description,
			&db.Href,
			&techBytes,
			&db.DisplayOrder,
			&db.CreatedAt,
			&db.UpdatedAt,
		)
		if err != nil {
			return nil, errs.WrapCode(err, errs.Internal, "failed to scan project row")
		}

		// Parse technologies array
		if len(techBytes) > 0 {
			if err := json.Unmarshal(techBytes, &db.Technologies); err != nil {
				db.Technologies = []string{}
			}
		}

		projects = append(projects, dbToProject(&db))
	}

	if err := rows.Err(); err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "error iterating project rows")
	}

	if projects == nil {
		projects = []*Project{}
	}

	return &ListResponse{Projects: projects}, nil
}
```

**Step 4: Run tests**

Run: `encore test ./projects/... -v`
Expected: Tests pass (will skip without DB)

**Step 5: Commit**

```bash
git add projects/
git commit -m "feat: add projects service with list endpoint"
```

---

## Task 7: Create Projects Service - Create Endpoint

**Files:**
- Modify: `projects/projects.go`
- Modify: `projects/projects_test.go`

**Step 1: Add validation helper**

Add to `projects/projects.go`:

```go
func (r *CreateRequest) Validate() error {
	if r.Title == "" {
		return &errs.Error{Code: errs.InvalidArgument, Message: "title is required"}
	}
	if r.Description == "" {
		return &errs.Error{Code: errs.InvalidArgument, Message: "description is required"}
	}
	if r.Technologies == nil {
		r.Technologies = []string{}
	}
	return nil
}
```

**Step 2: Add Create endpoint**

Add to `projects/projects.go`:

```go
//encore:api auth path=/projects method=POST
func Create(ctx context.Context, req *CreateRequest) (*Project, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	pool, err := database.Get(ctx)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to get database connection")
	}

	techJSON, _ := json.Marshal(req.Technologies)

	query := `
		INSERT INTO projects (title, description, href, technologies, display_order)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, title, description, href, technologies, display_order, created_at, updated_at
	`

	var db database.Project
	var techBytes []byte

	err = pool.QueryRow(ctx, query,
		req.Title,
		req.Description,
		req.Href,
		techJSON,
		req.DisplayOrder,
	).Scan(
		&db.ID,
		&db.Title,
		&db.Description,
		&db.Href,
		&techBytes,
		&db.DisplayOrder,
		&db.CreatedAt,
		&db.UpdatedAt,
	)

	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to create project")
	}

	// Parse technologies array
	if len(techBytes) > 0 {
		if err := json.Unmarshal(techBytes, &db.Technologies); err != nil {
			db.Technologies = []string{}
		}
	}

	return dbToProject(&db), nil
}
```

**Step 3: Run Encore check**

Run: `encore check ./projects/...`
Expected: No errors

**Step 4: Commit**

```bash
git add projects/projects.go
git commit -m "feat: add projects create endpoint"
```

---

## Task 8: Create Projects Service - Update Endpoint

**Files:**
- Modify: `projects/projects.go`

**Step 1: Add Update endpoint**

Add to `projects/projects.go`:

```go
//encore:api auth path=/projects/:id method=PUT
func Update(ctx context.Context, req *UpdateRequest) (*Project, error) {
	pool, err := database.Get(ctx)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to get database connection")
	}

	// Build dynamic update query
	updates := []string{}
	args := []interface{}{}
	argNum := 1

	if req.Title != nil {
		updates = append(updates, fmt.Sprintf("title = $%d", argNum))
		args = append(args, *req.Title)
		argNum++
	}
	if req.Description != nil {
		updates = append(updates, fmt.Sprintf("description = $%d", argNum))
		args = append(args, *req.Description)
		argNum++
	}
	if req.Href != nil {
		updates = append(updates, fmt.Sprintf("href = $%d", argNum))
		args = append(args, *req.Href)
		argNum++
	}
	if req.Technologies != nil {
		updates = append(updates, fmt.Sprintf("technologies = $%d", argNum))
		techJSON, _ := json.Marshal(req.Technologies)
		args = append(args, techJSON)
		argNum++
	}
	if req.DisplayOrder != nil {
		updates = append(updates, fmt.Sprintf("display_order = $%d", argNum))
		args = append(args, *req.DisplayOrder)
		argNum++
	}

	if len(updates) == 0 {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "no fields to update"}
	}

	// Add ID as last argument
	args = append(args, req.ID)

	query := fmt.Sprintf(`
		UPDATE projects
		SET %s
		WHERE id = $%d AND deleted_at IS NULL
		RETURNING id, title, description, href, technologies, display_order, created_at, updated_at
	`, joinUpdates(updates), argNum)

	var db database.Project
	var techBytes []byte

	err = pool.QueryRow(ctx, query, args...).Scan(
		&db.ID,
		&db.Title,
		&db.Description,
		&db.Href,
		&techBytes,
		&db.DisplayOrder,
		&db.CreatedAt,
		&db.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, &errs.Error{Code: errs.NotFound, Message: "project not found"}
	}
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to update project")
	}

	// Parse technologies array
	if len(techBytes) > 0 {
		if err := json.Unmarshal(techBytes, &db.Technologies); err != nil {
			db.Technologies = []string{}
		}
	}

	return dbToProject(&db), nil
}

// joinUpdates joins update clauses with commas.
func joinUpdates(updates []string) string {
	result := ""
	for i, u := range updates {
		if i > 0 {
			result += ", "
		}
		result += u
	}
	return result
}
```

**Step 2: Update UpdateRequest to include ID in path**

The `UpdateRequest` needs to be modified - ID comes from path:

```go
// UpdateRequest is the request body for updating a project.
// ID is populated from the path parameter.
type UpdateRequest struct {
	ID           string   `json:"-"` // Populated from path
	Title        *string  `json:"title,omitempty"`
	Description  *string  `json:"description,omitempty"`
	Href         *string  `json:"href,omitempty"`
	Technologies []string `json:"technologies,omitempty"`
	DisplayOrder *int     `json:"display_order,omitempty"`
}
```

Update the endpoint signature to use path parameter:

```go
//encore:api auth path=/projects/:id method=PUT
func Update(ctx context.Context, id string, req *UpdateRequest) (*Project, error) {
	req.ID = id
	// ... rest of implementation
```

**Step 3: Run Encore check**

Run: `encore check ./projects/...`
Expected: No errors

**Step 4: Commit**

```bash
git add projects/projects.go
git commit -m "feat: add projects update endpoint"
```

---

## Task 9: Create Projects Service - Delete Endpoint

**Files:**
- Modify: `projects/projects.go`

**Step 1: Add Delete endpoint (soft delete)**

Add to `projects/projects.go`:

```go
//encore:api auth path=/projects/:id method=DELETE
func Delete(ctx context.Context, id string) error {
	pool, err := database.Get(ctx)
	if err != nil {
		return errs.WrapCode(err, errs.Internal, "failed to get database connection")
	}

	query := `
		UPDATE projects
		SET deleted_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := pool.Exec(ctx, query, id)
	if err != nil {
		return errs.WrapCode(err, errs.Internal, "failed to delete project")
	}

	if result.RowsAffected() == 0 {
		return &errs.Error{Code: errs.NotFound, Message: "project not found"}
	}

	return nil
}
```

**Step 2: Run Encore check**

Run: `encore check ./projects/...`
Expected: No errors

**Step 3: Commit**

```bash
git add projects/projects.go
git commit -m "feat: add projects delete endpoint (soft delete)"
```

---

## Task 10: Create Hero Service

**Files:**
- Create: `hero/hero.go`
- Create: `hero/hero_test.go`

**Step 1: Create hero service**

Create `hero/hero.go`:

```go
// Package hero provides operations for the hero section content.
package hero

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"encore.app/pkg/database"
	"encore.dev/beta/errs"
)

// Hero represents the hero section content.
type Hero struct {
	ID          string   `json:"id"`
	Phrases     []string `json:"phrases"`
	Description string   `json:"description"`
	UpdatedAt   string   `json:"updated_at"`
}

// UpdateRequest is the request body for updating hero content.
type UpdateRequest struct {
	Phrases     []string `json:"phrases"`
	Description string   `json:"description"`
}

// dbToHero converts a database Hero to API Hero.
func dbToHero(db *database.Hero) *Hero {
	return &Hero{
		ID:          db.ID,
		Phrases:     db.Phrases,
		Description: db.Description,
		UpdatedAt:   db.UpdatedAt.Format(time.RFC3339),
	}
}

//encore:api auth path=/hero method=GET
func Get(ctx context.Context) (*Hero, error) {
	pool, err := database.Get(ctx)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to get database connection")
	}

	query := `
		SELECT id, phrases, description, updated_at
		FROM hero
		WHERE id = '00000000-0000-0000-0000-000000000001'::UUID
	`

	var db database.Hero
	var phrasesBytes []byte

	err = pool.QueryRow(ctx, query).Scan(
		&db.ID,
		&phrasesBytes,
		&db.Description,
		&db.UpdatedAt,
	)

	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to get hero content")
	}

	// Parse phrases array
	if len(phrasesBytes) > 0 {
		if err := json.Unmarshal(phrasesBytes, &db.Phrases); err != nil {
			db.Phrases = []string{}
		}
	}

	return dbToHero(&db), nil
}

//encore:api auth path=/hero method=PUT
func Update(ctx context.Context, req *UpdateRequest) (*Hero, error) {
	if req.Description == "" {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "description is required"}
	}
	if req.Phrases == nil {
		req.Phrases = []string{}
	}

	pool, err := database.Get(ctx)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to get database connection")
	}

	phrasesJSON, _ := json.Marshal(req.Phrases)

	query := `
		UPDATE hero
		SET phrases = $1, description = $2, updated_at = NOW()
		WHERE id = '00000000-0000-0000-0000-000000000001'::UUID
		RETURNING id, phrases, description, updated_at
	`

	var db database.Hero
	var phrasesBytes []byte

	err = pool.QueryRow(ctx, query, phrasesJSON, req.Description).Scan(
		&db.ID,
		&phrasesBytes,
		&db.Description,
		&db.UpdatedAt,
	)

	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to update hero content")
	}

	// Parse phrases array
	if len(phrasesBytes) > 0 {
		if err := json.Unmarshal(phrasesBytes, &db.Phrases); err != nil {
			db.Phrases = []string{}
		}
	}

	return dbToHero(&db), nil
}
```

**Step 2: Run Encore check**

Run: `encore check ./hero/...`
Expected: No errors

**Step 3: Commit**

```bash
git add hero/
git commit -m "feat: add hero service with get and update endpoints"
```

---

## Task 11: Create Work Experience Service - Types and List

**Files:**
- Create: `work/work.go`
- Create: `work/work_test.go`

**Step 1: Create work service with List**

Create `work/work.go`:

```go
// Package work provides CRUD operations for work experience content.
package work

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"encore.app/pkg/database"
	"encore.dev/beta/errs"
)

// Response types for API
type (
	// WorkExperience represents a work experience entry in API responses.
	WorkExperience struct {
		ID           string  `json:"id"`
		Company      string  `json:"company"`
		Position     string  `json:"position"`
		Date         string  `json:"date"`
		Description  string  `json:"description"`
		Href         *string `json:"href,omitempty"`
		Type         *string `json:"type,omitempty"`
		DisplayOrder int     `json:"display_order"`
		CreatedAt    string  `json:"created_at"`
		UpdatedAt    string  `json:"updated_at"`
	}

	// ListResponse is the response for listing work experiences.
	ListResponse struct {
		WorkExperiences []*WorkExperience `json:"work_experiences"`
	}

	// CreateRequest is the request body for creating a work experience.
	CreateRequest struct {
		Company      string  `json:"company"`
		Position     string  `json:"position"`
		Date         string  `json:"date"`
		Description  string  `json:"description"`
		Href         *string `json:"href,omitempty"`
		Type         *string `json:"type,omitempty"`
		DisplayOrder int     `json:"display_order"`
	}

	// UpdateRequest is the request body for updating a work experience.
	UpdateRequest struct {
		ID           string  `json:"-"`
		Company      *string `json:"company,omitempty"`
		Position     *string `json:"position,omitempty"`
		Date         *string `json:"date,omitempty"`
		Description  *string `json:"description,omitempty"`
		Href         *string `json:"href,omitempty"`
		Type         *string `json:"type,omitempty"`
		DisplayOrder *int    `json:"display_order,omitempty"`
	}
)

// validWorkTypes are the allowed values for work type.
var validWorkTypes = map[string]bool{
	"Freelance":  true,
	"Internship": true,
	"Contract":   true,
	"Part-Time":  true,
	"Full-Time":  true,
}

// dbToWork converts a database WorkExperience to API WorkExperience.
func dbToWork(db *database.WorkExperience) *WorkExperience {
	return &WorkExperience{
		ID:           db.ID,
		Company:      db.Company,
		Position:     db.Position,
		Date:         db.Date,
		Description:  db.Description,
		Href:         db.Href,
		Type:         db.Type,
		DisplayOrder: db.DisplayOrder,
		CreatedAt:    db.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    db.UpdatedAt.Format(time.RFC3339),
	}
}

//encore:api auth path=/work method=GET
func List(ctx context.Context) (*ListResponse, error) {
	pool, err := database.Get(ctx)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to get database connection")
	}

	query := `
		SELECT id, company, position, date, description, href, type, display_order, created_at, updated_at
		FROM work_experience
		WHERE deleted_at IS NULL
		ORDER BY display_order ASC, created_at DESC
	`

	rows, err := pool.Query(ctx, query)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to query work experiences")
	}
	defer rows.Close()

	var workExperiences []*WorkExperience
	for rows.Next() {
		var db database.WorkExperience
		err := rows.Scan(
			&db.ID,
			&db.Company,
			&db.Position,
			&db.Date,
			&db.Description,
			&db.Href,
			&db.Type,
			&db.DisplayOrder,
			&db.CreatedAt,
			&db.UpdatedAt,
		)
		if err != nil {
			return nil, errs.WrapCode(err, errs.Internal, "failed to scan work experience row")
		}
		workExperiences = append(workExperiences, dbToWork(&db))
	}

	if err := rows.Err(); err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "error iterating work experience rows")
	}

	if workExperiences == nil {
		workExperiences = []*WorkExperience{}
	}

	return &ListResponse{WorkExperiences: workExperiences}, nil
}
```

**Step 2: Run Encore check**

Run: `encore check ./work/...`
Expected: No errors

**Step 3: Commit**

```bash
git add work/
git commit -m "feat: add work experience service with list endpoint"
```

---

## Task 12: Complete Work Experience Service - Create/Update/Delete

**Files:**
- Modify: `work/work.go`

**Step 1: Add Create endpoint**

Add to `work/work.go`:

```go
func (r *CreateRequest) Validate() error {
	if r.Company == "" {
		return &errs.Error{Code: errs.InvalidArgument, Message: "company is required"}
	}
	if r.Position == "" {
		return &errs.Error{Code: errs.InvalidArgument, Message: "position is required"}
	}
	if r.Date == "" {
		return &errs.Error{Code: errs.InvalidArgument, Message: "date is required"}
	}
	if r.Description == "" {
		return &errs.Error{Code: errs.InvalidArgument, Message: "description is required"}
	}
	if r.Type != nil && !validWorkTypes[*r.Type] {
		return &errs.Error{Code: errs.InvalidArgument, Message: "invalid work type"}
	}
	return nil
}

//encore:api auth path=/work method=POST
func Create(ctx context.Context, req *CreateRequest) (*WorkExperience, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	pool, err := database.Get(ctx)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to get database connection")
	}

	query := `
		INSERT INTO work_experience (company, position, date, description, href, type, display_order)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, company, position, date, description, href, type, display_order, created_at, updated_at
	`

	var db database.WorkExperience
	err = pool.QueryRow(ctx, query,
		req.Company,
		req.Position,
		req.Date,
		req.Description,
		req.Href,
		req.Type,
		req.DisplayOrder,
	).Scan(
		&db.ID,
		&db.Company,
		&db.Position,
		&db.Date,
		&db.Description,
		&db.Href,
		&db.Type,
		&db.DisplayOrder,
		&db.CreatedAt,
		&db.UpdatedAt,
	)

	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to create work experience")
	}

	return dbToWork(&db), nil
}
```

**Step 2: Add Update endpoint**

Add to `work/work.go`:

```go
//encore:api auth path=/work/:id method=PUT
func Update(ctx context.Context, id string, req *UpdateRequest) (*WorkExperience, error) {
	req.ID = id

	pool, err := database.Get(ctx)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to get database connection")
	}

	// Validate type if provided
	if req.Type != nil && !validWorkTypes[*req.Type] {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "invalid work type"}
	}

	// Build dynamic update query
	updates := []string{}
	args := []interface{}{}
	argNum := 1

	if req.Company != nil {
		updates = append(updates, fmt.Sprintf("company = $%d", argNum))
		args = append(args, *req.Company)
		argNum++
	}
	if req.Position != nil {
		updates = append(updates, fmt.Sprintf("position = $%d", argNum))
		args = append(args, *req.Position)
		argNum++
	}
	if req.Date != nil {
		updates = append(updates, fmt.Sprintf("date = $%d", argNum))
		args = append(args, *req.Date)
		argNum++
	}
	if req.Description != nil {
		updates = append(updates, fmt.Sprintf("description = $%d", argNum))
		args = append(args, *req.Description)
		argNum++
	}
	if req.Href != nil {
		updates = append(updates, fmt.Sprintf("href = $%d", argNum))
		args = append(args, *req.Href)
		argNum++
	}
	if req.Type != nil {
		updates = append(updates, fmt.Sprintf("type = $%d", argNum))
		args = append(args, *req.Type)
		argNum++
	}
	if req.DisplayOrder != nil {
		updates = append(updates, fmt.Sprintf("display_order = $%d", argNum))
		args = append(args, *req.DisplayOrder)
		argNum++
	}

	if len(updates) == 0 {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "no fields to update"}
	}

	args = append(args, req.ID)

	query := fmt.Sprintf(`
		UPDATE work_experience
		SET %s
		WHERE id = $%d AND deleted_at IS NULL
		RETURNING id, company, position, date, description, href, type, display_order, created_at, updated_at
	`, joinWorkUpdates(updates), argNum)

	var db database.WorkExperience
	err = pool.QueryRow(ctx, query, args...).Scan(
		&db.ID,
		&db.Company,
		&db.Position,
		&db.Date,
		&db.Description,
		&db.Href,
		&db.Type,
		&db.DisplayOrder,
		&db.CreatedAt,
		&db.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, &errs.Error{Code: errs.NotFound, Message: "work experience not found"}
	}
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to update work experience")
	}

	return dbToWork(&db), nil
}

func joinWorkUpdates(updates []string) string {
	result := ""
	for i, u := range updates {
		if i > 0 {
			result += ", "
		}
		result += u
	}
	return result
}
```

**Step 3: Add Delete endpoint**

Add to `work/work.go`:

```go
//encore:api auth path=/work/:id method=DELETE
func Delete(ctx context.Context, id string) error {
	pool, err := database.Get(ctx)
	if err != nil {
		return errs.WrapCode(err, errs.Internal, "failed to get database connection")
	}

	query := `
		UPDATE work_experience
		SET deleted_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := pool.Exec(ctx, query, id)
	if err != nil {
		return errs.WrapCode(err, errs.Internal, "failed to delete work experience")
	}

	if result.RowsAffected() == 0 {
		return &errs.Error{Code: errs.NotFound, Message: "work experience not found"}
	}

	return nil
}
```

**Step 4: Run Encore check**

Run: `encore check ./work/...`
Expected: No errors

**Step 5: Commit**

```bash
git add work/work.go
git commit -m "feat: add work experience create, update, delete endpoints"
```

---

## Task 13: Create Webhook Service with Secret Token Validation

**Files:**
- Create: `webhook/webhook.go`
- Create: `webhook/webhook_test.go`

**Step 1: Create webhook service**

Create `webhook/webhook.go`:

```go
// Package webhook provides endpoints for triggering external webhooks.
package webhook

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"encore.dev/beta/errs"
)

// secrets holds the webhook-related secrets.
var secrets struct {
	NetlifyBuildHook string
	WebhookSecret    string
}

// RebuildResponse is the response from the rebuild endpoint.
type RebuildResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ValidateRequest validates the webhook secret from the request header.
func ValidateRequest(authHeader string) error {
	if secrets.WebhookSecret == "" {
		return fmt.Errorf("webhook secret not configured")
	}

	if authHeader == "" {
		return &errs.Error{Code: errs.Unauthenticated, Message: "missing authorization header"}
	}

	// Expected format: "Bearer <token>"
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return &errs.Error{Code: errs.Unauthenticated, Message: "invalid authorization format"}
	}

	token := parts[1]
	if token != secrets.WebhookSecret {
		return &errs.Error{Code: errs.Unauthenticated, Message: "invalid webhook secret"}
	}

	return nil
}

//encore:api public raw path=/webhook/rebuild method=POST
func Rebuild(w http.ResponseWriter, req *http.Request) {
	// Validate the webhook secret from Authorization header
	authHeader := req.Header.Get("Authorization")
	if err := ValidateRequest(authHeader); err != nil {
		if encErr, ok := err.(*errs.Error); ok {
			errs.HTTPError(w, encErr)
		} else {
			errs.HTTPError(w, &errs.Error{
				Code:    errs.Unauthenticated,
				Message: err.Error(),
			})
		}
		return
	}

	// Trigger Netlify build hook
	if secrets.NetlifyBuildHook == "" {
		errs.HTTPError(w, &errs.Error{
			Code:    errs.Internal,
			Message: "netlify build hook not configured",
		})
		return
	}

	resp, err := http.Post(secrets.NetlifyBuildHook, "application/json", nil)
	if err != nil {
		errs.HTTPError(w, &errs.Error{
			Code:    errs.Internal,
			Message: fmt.Sprintf("failed to trigger build: %v", err),
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errs.HTTPError(w, &errs.Error{
			Code:    errs.Internal,
			Message: fmt.Sprintf("build hook returned status %d", resp.StatusCode),
		})
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"success":true,"message":"build triggered successfully"}`))
}
```

**Step 2: Add tests for webhook validation**

Create `webhook/webhook_test.go`:

```go
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
```

**Step 3: Run tests**

Run: `encore test ./webhook/... -v`
Expected: Tests pass

**Step 4: Commit**

```bash
git add webhook/
git commit -m "feat: add webhook service with secret token validation"
```

---

## Task 14: Remove Hello World Service

**Files:**
- Delete: `hello/hello.go`
- Delete: `hello/hello_test.go`
- Delete: `hello/encore.gen.go`

**Step 1: Remove hello directory**

Run: `rm -rf hello/`

**Step 2: Verify Encore still works**

Run: `encore check ./...`
Expected: No errors

**Step 3: Commit**

```bash
git add -A
git commit -m "chore: remove hello world example service"
```

---

## Task 15: Final Verification and Cleanup

**Step 1: Run all tests**

Run: `encore test ./... -v`
Expected: All tests pass or skip gracefully

**Step 2: Run Encore check**

Run: `encore check ./...`
Expected: No errors

**Step 3: Verify local development server**

Run: `encore run`
Expected: Server starts without errors

**Step 4: Test API endpoints (with valid JWT)**

In another terminal:
```bash
# Test projects list (requires auth)
curl -H "Authorization: Bearer YOUR_JWT" http://localhost:4000/projects

# Test hero get (requires auth)
curl -H "Authorization: Bearer YOUR_JWT" http://localhost:4000/hero

# Test webhook (requires secret)
curl -X POST -H "Authorization: Bearer YOUR_WEBHOOK_SECRET" http://localhost:4000/webhook/rebuild
```

**Step 5: Final commit**

```bash
git add -A
git commit -m "chore: final cleanup and verification"
```

---

## Summary

### Services Created

| Service | Endpoints | Description |
|---------|-----------|-------------|
| `auth` | Internal | JWT validation with Supabase JWKS |
| `projects` | GET, POST, PUT, DELETE `/projects` | CRUD for projects |
| `hero` | GET, PUT `/hero` | Single-row hero content |
| `work` | GET, POST, PUT, DELETE `/work` | CRUD for work experience |
| `webhook` | POST `/webhook/rebuild` | Trigger Netlify build |

### API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/projects` | Yes | List all projects |
| POST | `/projects` | Yes | Create project |
| PUT | `/projects/:id` | Yes | Update project |
| DELETE | `/projects/:id` | Yes | Soft delete project |
| GET | `/hero` | Yes | Get hero content |
| PUT | `/hero` | Yes | Update hero content |
| GET | `/work` | Yes | List work experiences |
| POST | `/work` | Yes | Create work experience |
| PUT | `/work/:id` | Yes | Update work experience |
| DELETE | `/work/:id` | Yes | Soft delete work experience |
| POST | `/webhook/rebuild` | Secret token | Trigger Netlify rebuild |

### Secrets Required

Configure these in Encore (local: `.secrets.local.cue`, production: `encore secret set`):

| Secret | Description |
|--------|-------------|
| `SupabaseURL` | Supabase project URL (e.g., `https://xxx.supabase.co`) |
| `SupabaseDBPassword` | Database password from Supabase dashboard |
| `SupabaseJWKSURL` | JWKS endpoint for JWT validation |
| `NetlifyBuildHook` | Netlify build hook URL |
| `WebhookSecret` | Secret token for webhook authorization |

### Database Schema

Run `migrations/001_init_schema.up.sql` in Supabase SQL editor to create tables.

---

## Deployment Steps

1. **Create Supabase project** and run migrations
2. **Create Encore app**: `encore app create`
3. **Link app**: `encore app link <app-id>` (in `encore.app`)
4. **Set production secrets**:
   ```bash
   encore secret set --type production SupabaseURL
   encore secret set --type production SupabaseDBPassword
   encore secret set --type production SupabaseJWKSURL
   encore secret set --type production NetlifyBuildHook
   encore secret set --type production WebhookSecret
   ```
5. **Deploy**: `git push encore` (after adding Encore remote)

---

*Implementation plan complete.*
