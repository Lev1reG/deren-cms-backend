# Standardized Error Response Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Create a standardized error response helper for raw Encore endpoints to match `encore.dev/beta/errs` response structure.

**Architecture:** Create a `pkg/response/error.go` package with a `writeError()` helper function that returns JSON errors with `code`, `message`, and `details` fields, with appropriate HTTP status codes mapped from error codes.

**Tech Stack:** Encore.go, `encore.dev/beta/errs`, standard `net/http`, `encoding/json`

---

## Task 1: Create response helper package

**Files:**
- Create: `pkg/response/error.go`

**Step 1: Create the response package directory and error.go file**

```bash
mkdir -p pkg/response
```

Create `pkg/response/error.go`:

```go
// Package response provides standardized response helpers for raw Encore endpoints.
package response

import (
	"encoding/json"
	"net/http"

	"encore.dev/beta/errs"
)

// ErrorResponse represents a standardized error response matching encore.dev/beta/errs format.
type ErrorResponse struct {
	Code    errs.ErrCode           `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details"`
}

// writeError writes a standardized error response to the HTTP response writer.
// It sets the Content-Type header to application/json, sets the appropriate
// HTTP status code based on the error code, and writes the error response as JSON.
//
// Example:
//
//	writeError(w, errs.Unauthenticated, "Invalid email or password")
func writeError(w http.ResponseWriter, code errs.ErrCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatusForCode(code))

	resp := ErrorResponse{
		Code:    code,
		Message: message,
		Details: nil,
	}

	json.NewEncoder(w).Encode(resp)
}

// httpStatusForCode maps Encore error codes to HTTP status codes.
// This matches the mapping used by Encore for non-raw endpoints.
func httpStatusForCode(code errs.ErrCode) int {
	switch code {
	case errs.OK:
		return http.StatusOK
	case errs.Canceled:
		return 499
	case errs.Unknown:
		return http.StatusInternalServerError
	case errs.InvalidArgument:
		return http.StatusBadRequest
	case errs.DeadlineExceeded:
		return http.StatusGatewayTimeout
	case errs.NotFound:
		return http.StatusNotFound
	case errs.AlreadyExists:
		return http.StatusConflict
	case errs.PermissionDenied:
		return http.StatusForbidden
	case errs.ResourceExhausted:
		return http.StatusTooManyRequests
	case errs.FailedPrecondition:
		return http.StatusBadRequest
	case errs.Aborted:
		return http.StatusConflict
	case errs.OutOfRange:
		return http.StatusBadRequest
	case errs.Unimplemented:
		return http.StatusNotImplemented
	case errs.Internal:
		return http.StatusInternalServerError
	case errs.Unavailable:
		return http.StatusServiceUnavailable
	case errs.DataLoss:
		return http.StatusInternalServerError
	case errs.Unauthenticated:
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}
```

**Step 2: Run Encore check to verify compilation**

Run: `encore check`
Expected: No compilation errors

**Step 3: Commit**

```bash
git add pkg/response/error.go
git commit -m "feat: add standardized error response helper

Add pkg/response package with writeError() helper to match
encore.dev/beta/errs response format for raw endpoints.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 2: Write unit tests for writeError

**Files:**
- Create: `pkg/response/error_test.go`

**Step 1: Write failing tests**

Create `pkg/response/error_test.go`:

```go
package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"encore.dev/beta/errs"
	"testing"
)

func TestWriteError(t *testing.T) {
	tests := []struct {
		name           string
		code           errs.ErrCode
		message        string
		expectedStatus int
	}{
		{
			name:           "invalid argument",
			code:           errs.InvalidArgument,
			message:        "email is required",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "not found",
			code:           errs.NotFound,
			message:        "resource not found",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "unauthenticated",
			code:           errs.Unauthenticated,
			message:        "invalid token",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "internal error",
			code:           errs.Internal,
			message:        "something went wrong",
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			writeError(w, tt.code, tt.message)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if w.Header().Get("Content-Type") != "application/json" {
				t.Errorf("expected Content-Type application/json, got %s", w.Header().Get("Content-Type"))
			}

			var resp ErrorResponse
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if resp.Code != tt.code {
				t.Errorf("expected code %s, got %s", tt.code, resp.Code)
			}

			if resp.Message != tt.message {
				t.Errorf("expected message %q, got %q", tt.message, resp.Message)
			}

			if resp.Details != nil {
				t.Errorf("expected details to be nil, got %v", resp.Details)
			}
		})
	}
}

func TestHTTPStatusForCode(t *testing.T) {
	tests := []struct {
		name           string
		code           errs.ErrCode
		expectedStatus int
	}{
		{"ok", errs.OK, http.StatusOK},
		{"cancelled", errs.Canceled, 499},
		{"unknown", errs.Unknown, http.StatusInternalServerError},
		{"invalid argument", errs.InvalidArgument, http.StatusBadRequest},
		{"deadline exceeded", errs.DeadlineExceeded, http.StatusGatewayTimeout},
		{"not found", errs.NotFound, http.StatusNotFound},
		{"already exists", errs.AlreadyExists, http.StatusConflict},
		{"permission denied", errs.PermissionDenied, http.StatusForbidden},
		{"resource exhausted", errs.ResourceExhausted, http.StatusTooManyRequests},
		{"failed precondition", errs.FailedPrecondition, http.StatusBadRequest},
		{"aborted", errs.Aborted, http.StatusConflict},
		{"out of range", errs.OutOfRange, http.StatusBadRequest},
		{"unimplemented", errs.Unimplemented, http.StatusNotImplemented},
		{"internal", errs.Internal, http.StatusInternalServerError},
		{"unavailable", errs.Unavailable, http.StatusServiceUnavailable},
		{"data loss", errs.DataLoss, http.StatusInternalServerError},
		{"unauthenticated", errs.Unauthenticated, http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := httpStatusForCode(tt.code)
			if got != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, got)
			}
		})
	}
}
```

**Step 2: Run tests to verify they pass**

Run: `encore test ./pkg/response/...`
Expected: All tests PASS

**Step 3: Commit**

```bash
git add pkg/response/error_test.go
git commit -m "test: add unit tests for error response helper

Test writeError() function and httpStatusForCode() mapping
to ensure correct JSON response structure and HTTP status codes.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 3: Update login.go to use writeError

**Files:**
- Modify: `auth/login.go`

**Step 1: Replace http.Error calls with writeError**

In `auth/login.go`, replace the import block:

```go
import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"encore.app/pkg/response"
	"encore.dev/beta/errs"
)
```

Replace the validation error handling (lines 56-64):

```go
	if err := req.Validate(); err != nil {
		var eErr *errs.Error
		if errors.As(err, &eErr) {
			response.WriteError(w, eErr.Code, eErr.Message)
		} else {
			response.WriteError(w, errs.InvalidArgument, err.Error())
		}
		return
	}
```

Replace the marshal error (lines 74-77):

```go
	if err != nil {
		response.WriteError(w, errs.Internal, "Internal error")
		return
	}
```

Replace the new request error (lines 79-83):

```go
	if err != nil {
		response.WriteError(w, errs.Internal, "Internal error")
		return
	}
```

Replace the HTTP client error (lines 87-91):

```go
	if err != nil {
		response.WriteError(w, errs.Internal, "Internal error")
		return
	}
```

Replace the authentication error handling (lines 94-101):

```go
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusUnauthorized {
			response.WriteError(w, errs.Unauthenticated, "Invalid email or password")
			return
		}
		response.WriteError(w, errs.Internal, "Authentication service error")
		return
	}
```

Replace the decode error (lines 104-107):

```go
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		response.WriteError(w, errs.Internal, "Internal error")
		return
	}
```

**Step 2: Run Encore check to verify compilation**

Run: `encore check`
Expected: No compilation errors

**Step 3: Run auth tests**

Run: `encore test ./auth/...`
Expected: All tests PASS

**Step 4: Commit**

```bash
git add auth/login.go
git commit -m "refactor: use standardized error response in login endpoint

Replace http.Error() with response.WriteError() to return
JSON errors matching encore.dev/beta/errs format.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 4: Update refresh.go to use writeError

**Files:**
- Modify: `auth/refresh.go`

**Step 1: Replace http.Error calls with writeError**

In `auth/refresh.go`, replace the import block:

```go
import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"encore.app/pkg/response"
	"encore.dev/beta/errs"
)
```

Replace the cookie error handling (lines 20-29):

```go
	// Get refresh token from cookie
	refreshCookie, err := r.Cookie("refresh_token")
	if err != nil {
		response.WriteError(w, errs.Unauthenticated, "Refresh token not found")
		return
	}

	if refreshCookie.Value == "" {
		response.WriteError(w, errs.Unauthenticated, "Refresh token is empty")
		return
	}
```

Replace the new request error (lines 38-42):

```go
	httpReq, err := http.NewRequest("POST", apiURL, strings.NewReader(formData.Encode()))
	if err != nil {
		response.WriteError(w, errs.Internal, "Internal error")
		return
	}
```

Replace the HTTP client error (lines 46-50):

```go
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		response.WriteError(w, errs.Internal, "Internal error")
		return
	}
```

Replace the authentication error handling (lines 53-56):

```go
	if resp.StatusCode != http.StatusOK {
		response.WriteError(w, errs.Unauthenticated, "Invalid or expired refresh token")
		return
	}
```

Replace the decode error (lines 58-62):

```go
	var authResp supabaseAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		response.WriteError(w, errs.Internal, "Internal error")
		return
	}
```

**Step 2: Run Encore check to verify compilation**

Run: `encore check`
Expected: No compilation errors

**Step 3: Run auth tests**

Run: `encore test ./auth/...`
Expected: All tests PASS

**Step 4: Commit**

```bash
git add auth/refresh.go
git commit -m "refactor: use standardized error response in refresh endpoint

Replace http.Error() with response.WriteError() to return
JSON errors matching encore.dev/beta/errs format.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 5: Update logout.go (if needed)

**Files:**
- Modify: `auth/logout.go`

**Step 1: Check if logout.go has error handling**

Read `auth/logout.go`:
Run: `cat auth/logout.go`

Expected outcome: If no `http.Error()` calls exist, skip this task. If they exist, replace with `response.WriteError()`.

**Step 2 (if changes needed): Run Encore check to verify compilation**

Run: `encore check`
Expected: No compilation errors

**Step 3 (if changes needed): Run auth tests**

Run: `encore test ./auth/...`
Expected: All tests PASS

**Step 4 (if changes needed): Commit**

```bash
git add auth/logout.go
git commit -m "refactor: use standardized error response in logout endpoint

Replace http.Error() with response.WriteError() to return
JSON errors matching encore.dev/beta/errs format.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 6: Run full test suite

**Files:**
- None (verification step)

**Step 1: Run all tests**

Run: `encore test ./...`
Expected: All tests PASS

**Step 2: Test raw endpoints manually (optional but recommended)**

Test login endpoint with invalid credentials:

```bash
curl -X POST http://localhost:4000/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"invalid@test.com","password":"wrong"}'
```

Expected response:
```json
{
  "code": "unauthenticated",
  "message": "Invalid email or password",
  "details": null
}
```

HTTP Status: 401

**Step 3: Regenerate OpenAPI spec**

Run: `encore gen openapi > docs/openapi.json`

**Step 4: Commit**

```bash
git add docs/openapi.json
git commit -m "docs: update OpenAPI spec for error responses

Regenerate OpenAPI spec to reflect standardized error
response format in auth endpoints.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 7: Verify and cleanup

**Files:**
- None (verification step)

**Step 1: Run Encore check**

Run: `encore check`
Expected: No compilation errors

**Step 2: Run full test suite one more time**

Run: `encore test ./...`
Expected: All tests PASS

**Step 3: Review changes**

Run: `git diff HEAD~5`

Verify:
- All raw endpoints use `response.WriteError()`
- No `http.Error()` calls remain in auth package
- Error response format matches encore.dev/beta/errs

**Step 4: Final commit**

```bash
git commit --allow-empty -m "feat: complete standardized error response implementation

All raw endpoints now return JSON errors matching
encore.dev/beta/errs format with code, message, and details fields.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Completion Checklist

- [ ] `pkg/response/error.go` created with writeError() helper
- [ ] Unit tests for writeError() and httpStatusForCode() passing
- [ ] `auth/login.go` updated to use writeError()
- [ ] `auth/refresh.go` updated to use writeError()
- [ ] `auth/logout.go` updated if needed
- [ ] All tests passing
- [ ] OpenAPI spec regenerated
- [ ] Manual endpoint verification confirms JSON error responses
