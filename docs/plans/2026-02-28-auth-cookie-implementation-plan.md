# Authentication Cookie - Backend Part 1 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add login and logout endpoints to backend with httpOnly cookie support

**Architecture:** Backend proxies to Supabase Auth REST API, sets httpOnly cookies for access tokens

**Tech Stack:** Encore.go, Supabase Auth REST API, httpOnly cookies

---

## Prerequisites

**Add Supabase Anon Key to secrets:**

Edit `.secrets.local.cue`:
```cue
SupabaseURL:       "https://<project>.supabase.co"
SupabaseDBPassword: "<db-password>"
SupabaseJWKSURL:   "https://<project>.supabase.co/auth/v1/jwks"
NetlifyBuildHook:   "https://api.netlify.com/build_hooks/<hook-id>"
WebhookSecret:      "<webhook-token>"
SupabaseAnonKey:    "<anon-key>"  // ADD THIS
```

Update `auth/auth.go` secrets struct:
```go
var secrets struct {
	SupabaseJWKSURL string
	SupabaseAnonKey string  // ADD THIS
}
```

---

### Task 1: Add SupabaseAnonKey secret to auth package

**Files:**
- Modify: `auth/auth.go:23-26`

**Step 1: Update secrets struct**

Edit the secrets struct to include SupabaseAnonKey:

```go
// secrets holds the auth-related secrets.
var secrets struct {
	SupabaseJWKSURL string
	SupabaseAnonKey string
}
```

**Step 2: Add secret declaration at top of file**

After the import section, add:
```go
//secrets:auth
type secrets struct {
	SupabaseJWKSURL string
	SupabaseAnonKey string
}

//go:generate encore.exe parse
var _ struct{}
```

**Step 3: Run encore to verify secret is registered**

Run: `encore check`
Expected: Success, no errors

**Step 4: Commit**

```bash
git add auth/auth.go
git commit -m "feat: add SupabaseAnonKey secret to auth package"
```

---

### Task 2: Create login endpoint structure

**Files:**
- Create: `auth/login.go`

**Step 1: Write the file structure with request/response types**

Create `auth/login.go`:

```go
package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"encore.app/pkg/database"
	"encore.dev/beta/errs"
)

// LoginRequest is the request body for login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse is the response for successful login.
type LoginResponse struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
}

// Validate validates the login request.
func (r *LoginRequest) Validate() error {
	if r.Email == "" {
		return &errs.Error{Code: errs.InvalidArgument, Message: "email is required"}
	}
	if r.Password == "" {
		return &errs.Error{Code: errs.InvalidArgument, Message: "password is required"}
	}
	return nil
}

//encore:api public path=/auth/login method=POST
func Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	return nil, errs.NotImplemented
}
```

**Step 2: Run encore to verify endpoint is registered**

Run: `encore check`
Expected: Success, no errors

**Step 3: Commit**

```bash
git add auth/login.go
git commit -m "feat: add login endpoint structure"
```

---

### Task 3: Write login test - happy path

**Files:**
- Create: `auth/login_test.go`

**Step 1: Write the happy path test**

Create `auth/login_test.go`:

```go
package auth

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLogin_HappyPath(t *testing.T) {
	ctx := context.Background()

	// Mock Supabase API - will need to implement
	// For now, skip this test
	t.Skip("waiting for Supabase mock implementation")
}
```

**Step 2: Run test to verify it's skipped**

Run: `encore test ./auth -run TestLogin_HappyPath -v`
Expected: SKIP with "waiting for Supabase mock implementation"

**Step 3: Commit**

```bash
git add auth/login_test.go
git commit -m "test: add login test structure"
```

---

### Task 4: Implement login endpoint - call Supabase

**Files:**
- Modify: `auth/login.go:24-30`

**Step 1: Implement Supabase API call**

Replace the Login function:

```go
// supabaseAuthResponse represents Supabase's auth API response.
type supabaseAuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	User         struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Role  string `json:"role"`
	} `json:"user"`
}

//encore:api public path=/auth/login method=POST
func Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Call Supabase Auth API
	httpClient := &http.Client{Timeout: 10 * time.Second}
	url := secrets.SupabaseURL + "/auth/v1/token?grant_type=password"

	body, err := json.Marshal(map[string]string{
		"email":    req.Email,
		"password": req.Password,
	})
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to encode request")
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to create request")
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("apikey", secrets.SupabaseAnonKey)

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to call Supabase auth")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusUnauthorized {
			return nil, &errs.Error{
				Code:    errs.Unauthenticated,
				Message: "Invalid email or password",
			}
		}
		return nil, &errs.Error{
			Code:    errs.Internal,
			Message: "Authentication service error",
		}
	}

	var authResp supabaseAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to decode response")
	}

	return &LoginResponse{
		UserID: authResp.User.ID,
		Email:  authResp.User.Email,
		Role:   authResp.User.Role,
	}, nil
}
```

**Step 2: Run encore to check for errors**

Run: `encore check`
Expected: Success (may have import errors for `bytes`, fix by adding to imports)

**Step 3: Fix imports**

Add to imports:
```go
"bytes"
```

**Step 4: Run encore check again**

Run: `encore check`
Expected: Success

**Step 5: Commit**

```bash
git add auth/login.go
git commit -m "feat: implement login endpoint with Supabase API call"
```

---

### Task 5: Add httpOnly cookie support to login

**Files:**
- Modify: `auth/login.go:66-100`

**Step 1: Check Encore's cookie support documentation**

Run: `encore docs search cookie` or check Encore docs
(Note: Encore's raw endpoint pattern needed for custom headers)

**Step 2: Rewrite as raw endpoint for cookie support**

Replace the Login function:

```go
//encore:api public raw path=/auth/login method=POST
func Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := req.Validate(); err != nil {
		http.Error(w, err.Message, http.StatusBadRequest)
		return
	}

	// Call Supabase Auth API
	httpClient := &http.Client{Timeout: 10 * time.Second}
	url := secrets.SupabaseURL + "/auth/v1/token?grant_type=password"

	body, err := json.Marshal(map[string]string{
		"email":    req.Email,
		"password": req.Password,
	})
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("apikey", secrets.SupabaseAnonKey)

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusUnauthorized {
			http.Error(w, "Invalid email or password", http.StatusUnauthorized)
			return
		}
		http.Error(w, "Authentication service error", http.StatusInternalServerError)
		return
	}

	var authResp supabaseAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// Set httpOnly cookie for access token
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    authResp.AccessToken,
		Expires:  time.Now().Add(time.Duration(authResp.ExpiresIn) * time.Second),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})

	// Set refresh token cookie (non-httpOnly for frontend to read)
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    authResp.RefreshToken,
		Expires:  time.Now().Add(30 * 24 * time.Hour), // 30 days
		HttpOnly: false,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})

	// Return user data
	respData := LoginResponse{
		UserID: authResp.User.ID,
		Email:  authResp.User.Email,
		Role:   authResp.User.Role,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(respData)
}
```

**Step 3: Run encore check**

Run: `encore check`
Expected: Success

**Step 4: Commit**

```bash
git add auth/login.go
git commit -m "feat: add httpOnly cookie support to login endpoint"
```

---

### Task 6: Create logout endpoint

**Files:**
- Create: `auth/logout.go`

**Step 1: Write logout endpoint**

Create `auth/logout.go`:

```go
package auth

import (
	"net/http"
	"time"
)

//encore:api public raw path=/auth/logout method=POST
func Logout(w http.ResponseWriter, r *http.Request) {
	// Clear auth_token cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})

	// Clear refresh_token cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: false,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})

	w.WriteHeader(http.StatusOK)
}
```

**Step 2: Run encore check**

Run: `encore check`
Expected: Success

**Step 3: Commit**

```bash
git add auth/logout.go
git commit -m "feat: add logout endpoint"
```

---

### Task 7: Create refresh token endpoint

**Files:**
- Create: `auth/refresh.go`

**Step 1: Write refresh endpoint structure**

Create `auth/refresh.go`:

```go
package auth

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"encore.dev/beta/errs"
)

// RefreshResponse is the response for successful token refresh.
type RefreshResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

//encore:api public raw path=/auth/refresh method=POST
func Refresh(w http.ResponseWriter, r *http.Request) {
	// Get refresh token from cookie
	refreshCookie, err := r.Cookie("refresh_token")
	if err != nil {
		http.Error(w, "Refresh token not found", http.StatusUnauthorized)
		return
	}

	if refreshCookie.Value == "" {
		http.Error(w, "Refresh token is empty", http.StatusUnauthorized)
		return
	}

	// Call Supabase Auth API to refresh
	httpClient := &http.Client{Timeout: 10 * time.Second}
	apiURL := secrets.SupabaseURL + "/auth/v1/token?grant_type=refresh_token"

	formData := url.Values{}
	formData.Set("refresh_token", refreshCookie.Value)

	httpReq, err := http.NewRequest("POST", apiURL, strings.NewReader(formData.Encode()))
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpReq.Header.Set("apikey", secrets.SupabaseAnonKey)

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, "Invalid or expired refresh token", http.StatusUnauthorized)
		return
	}

	var authResp supabaseAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// Update access token cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    authResp.AccessToken,
		Expires:  time.Now().Add(time.Duration(authResp.ExpiresIn) * time.Second),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})

	// Update refresh token cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    authResp.RefreshToken,
		Expires:  time.Now().Add(30 * 24 * time.Hour), // 30 days
		HttpOnly: false,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})

	// Return new access token
	respData := RefreshResponse{
		AccessToken: authResp.AccessToken,
		ExpiresIn:   authResp.ExpiresIn,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(respData)
}
```

**Step 2: Run encore check**

Run: `encore check`
Expected: Success

**Step 3: Commit**

```bash
git add auth/refresh.go
git commit -m "feat: add refresh token endpoint"
```

---

### Task 8: Update AuthHandler to read from Cookie header

**Files:**
- Modify: `auth/auth.go:126-136`

**Step 1: Update AuthHandler to read cookie first**

Replace the AuthHandler function:

```go
//encore:authhandler
func AuthHandler(ctx context.Context, token string) (auth.UID, *UserData, error) {
	// First try to get token from Authorization header
	// This is handled by Encore's framework - token parameter is the Bearer token

	// If no token in header, we need to read from cookie
	// But Encore's authhandler doesn't provide access to the request
	// We'll use a different approach: create a custom auth middleware

	// For now, keep existing behavior - token is passed as Bearer header
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

**Step 2: Run encore check**

Run: `encore check`
Expected: Success

**Step 3: Create custom auth middleware for cookie support**

Create `auth/middleware.go`:

```go
package auth

import (
	"context"
	"net/http"
	"strings"

	"encore.dev/beta/auth"
)

// CookieAuth extracts JWT from cookie and sets it in the auth context.
// This is used as middleware for endpoints that support cookie auth.
func CookieAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to get token from Authorization header first
		authHeader := r.Header.Get("Authorization")
		var token string

		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
				token = parts[1]
			}
		}

		// If no token in header, try cookie
		if token == "" {
			if cookie, err := r.Cookie("auth_token"); err == nil && cookie.Value != "" {
				token = cookie.Value
			}
		}

		// If we have a token, validate and set auth context
		if token != "" {
			userData, err := ValidateToken(r.Context(), token)
			if err == nil {
				ctx := auth.WithContext(r.Context(), auth.UID(userData.UserID), &auth.UserData{
					UserID: userData.UserID,
					Email:  userData.Email,
					Role:   userData.Role,
				})
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		// No valid token, continue without auth context
		// Protected endpoints will fail with 401
		next.ServeHTTP(w, r)
	})
}
```

**Step 4: Run encore check**

Run: `encore check`
Expected: Success

**Step 5: Commit**

```bash
git add auth/middleware.go
git commit -m "feat: add cookie auth middleware"
```

---

### Task 9: Write refresh endpoint test

**Files:**
- Create: `auth/refresh_test.go`

**Step 1: Write refresh test structure**

Create `auth/refresh_test.go`:

```go
package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRefresh_HappyPath(t *testing.T) {
	// This test will require mocking Supabase API
	// For now, skip
	t.Skip("waiting for Supabase mock implementation")
}

func TestRefresh_NoCookie(t *testing.T) {
	req := httptest.NewRequest("POST", "/auth/refresh", nil)
	w := httptest.NewRecorder()

	Refresh(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestRefresh_EmptyCookie(t *testing.T) {
	req := httptest.NewRequest("POST", "/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: ""})
	w := httptest.NewRecorder()

	Refresh(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}
```

**Step 2: Run tests**

Run: `encore test ./auth -run TestRefresh -v`
Expected: One PASS, two SKIP or FAIL depending on mock implementation

**Step 3: Commit**

```bash
git add auth/refresh_test.go
git commit -m "test: add refresh endpoint tests"
```

---

### Task 10: Write logout endpoint test

**Files:**
- Modify: `auth/logout.go` (add test file)

**Step 1: Write logout tests**

Create `auth/logout_test.go`:

```go
package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLogout_ClearsCookies(t *testing.T) {
	req := httptest.NewRequest("POST", "/auth/logout", nil)
	// Add cookies to simulate logged in state
	req.AddCookie(&http.Cookie{
		Name:     "auth_token",
		Value:    "some-token",
		Expires:  time.Now().Add(time.Hour),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})
	req.AddCookie(&http.Cookie{
		Name:     "refresh_token",
		Value:    "some-refresh-token",
		Expires:  time.Now().Add(30 * 24 * time.Hour),
		HttpOnly: false,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})

	w := httptest.NewRecorder()

	Logout(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Check that cookies were cleared
	authToken := resp.Cookies()[0]
	require.Equal(t, "auth_token", authToken.Name)
	require.Equal(t, "", authToken.Value)
	require.True(t, authToken.Expires.Before(time.Now()))

	refreshToken := resp.Cookies()[1]
	require.Equal(t, "refresh_token", refreshToken.Name)
	require.Equal(t, "", refreshToken.Value)
	require.True(t, refreshToken.Expires.Before(time.Now()))
}
```

**Step 2: Run tests**

Run: `encore test ./auth -run TestLogout -v`
Expected: PASS

**Step 3: Commit**

```bash
git add auth/logout_test.go
git commit -m "test: add logout endpoint tests"
```

---

**Next Steps (Frontend):**
1. Create auth client with 401 retry logic
2. Implement login page
3. Update API client to use credentials
4. Remove localStorage JWT handling
