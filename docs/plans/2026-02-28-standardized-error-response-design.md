# Standardized Error Response Design

**Date:** 2026-02-28
**Author:** Claude Code
**Status:** Approved

## Overview

Create a standardized error response helper for raw Encore endpoints to match the response structure used by `encore.dev/beta/errs` across the API.

## Problem

Raw endpoints (`auth/login`, `auth/refresh`, `auth/logout`) currently use `http.Error()` which returns plain text responses. This creates inconsistency with non-raw endpoints that return structured JSON errors via `encore.dev/beta/errs`.

**Example of current issue:**
```go
http.Error(w, "Invalid email or password", http.StatusUnauthorized)
// Returns: "Invalid email or password" (plain text)
```

**Desired output (matching Encore errs):**
```json
{
  "code": "unauthenticated",
  "message": "Invalid email or password",
  "details": null
}
```

## Solution

Create a minimal `pkg/response/error.go` package with a `writeError()` helper function.

## Design

### Package Structure

```
pkg/response/
└── error.go       # writeError() helper + HTTP status mapping
```

### API

```go
// writeError writes a standardized error response matching Encore's errs.Error format
func writeError(w http.ResponseWriter, code errs.ErrCode, message string)
```

### Response Structure

```json
{
  "code": "invalid_argument",
  "message": "email is required",
  "details": null
}
```

### HTTP Status Code Mapping

| ErrCode | HTTP Status |
|---------|-------------|
| `ok` | 200 |
| `cancelled` | 499 |
| `unknown` | 500 |
| `invalid_argument` | 400 |
| `deadline_exceeded` | 504 |
| `not_found` | 404 |
| `already_exists` | 409 |
| `permission_denied` | 403 |
| `resource_exhausted` | 429 |
| `failed_precondition` | 400 |
| `aborted` | 409 |
| `out_of_range` | 400 |
| `unimplemented` | 501 |
| `internal` | 500 |
| `unavailable` | 503 |
| `data_loss` | 500 |
| `unauthenticated` | 401 |

## Implementation Files

| File | Change |
|------|--------|
| `pkg/response/error.go` | **New** - Create helper package |
| `auth/login.go` | Replace `http.Error()` with `writeError()` |
| `auth/refresh.go` | Replace `http.Error()` with `writeError()` |
| `auth/logout.go` | Replace `http.Error()` with `writeError()` (if any) |

## Testing

1. Unit test for `writeError()` - verify correct JSON and HTTP status
2. Integration tests for auth endpoints - verify error response format
3. Run existing tests to ensure no regressions

## Success Criteria

- All raw endpoints return JSON error responses matching `encore.dev/beta/errs` structure
- Consistent HTTP status codes mapped from error codes
- All existing tests pass
- No breaking changes to successful response formats
