# Standardized API Response Format Design

**Date:** 2026-03-14
**Status:** Draft

## Overview

Standardize all API responses to use a consistent envelope format with `success`, `data`, and `error` fields. All endpoints will be converted to raw endpoints for full control over serialization.

## Response Format

### Success Response

```json
{
  "success": true,
  "data": <T>
}
```

- `success`: Always `true` for successful responses
- `data`: The response payload (object, array, or `null`)

Examples:

```json
// Single object
{ "success": true, "data": { "id": "abc", "title": "My Project" } }

// Array
{ "success": true, "data": [{ "id": "abc" }, { "id": "def" }] }

// Empty (DELETE operations)
{ "success": true, "data": null }
```

### Error Response

```json
{
  "success": false,
  "error": {
    "code": "NOT_FOUND",
    "message": "Project not found"
  }
}
```

- `success`: Always `false` for error responses
- `error.code`: Error code in SCREAMING_SNAKE_CASE
- `error.message`: Human-readable error message

### HTTP Status Codes

Maintain RESTful HTTP status codes:

| Code | Usage |
|------|-------|
| 200 | Success (GET, PUT, POST, DELETE) |
| 201 | Resource created (optional, can use 200) |
| 400 | Invalid argument / bad request |
| 401 | Unauthenticated |
| 403 | Permission denied |
| 404 | Resource not found |
| 500 | Internal error |

## Error Codes

Map Encore error codes to SCREAMING_SNAKE_CASE:

| Encore Code | API Code |
|-------------|----------|
| `ok` | N/A (success) |
| `invalid_argument` | `INVALID_ARGUMENT` |
| `unauthenticated` | `UNAUTHENTICATED` |
| `permission_denied` | `PERMISSION_DENIED` |
| `not_found` | `NOT_FOUND` |
| `already_exists` | `ALREADY_EXISTS` |
| `internal` | `INTERNAL` |
| `unavailable` | `UNAVAILABLE` |

## Implementation

### pkg/response Package

Update `pkg/response/error.go` and create new helpers:

```go
package response

import (
    "encoding/json"
    "net/http"
    "strings"

    "encore.dev/beta/errs"
)

// successResponse wraps successful responses
type successResponse struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data"`
}

// errorDetail contains error information
type errorDetail struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}

// errorResponse wraps error responses
type errorResponse struct {
    Success bool        `json:"success"`
    Error   errorDetail `json:"error"`
}

// WriteSuccess writes a standardized success response with HTTP 200
func WriteSuccess(w http.ResponseWriter, data interface{}) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(successResponse{
        Success: true,
        Data:    data,
    })
}

// WriteCreated writes a standardized success response with HTTP 201
func WriteCreated(w http.ResponseWriter, data interface{}) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(successResponse{
        Success: true,
        Data:    data,
    })
}

// WriteError writes a standardized error response
func WriteError(w http.ResponseWriter, code errs.ErrCode, message string) {
    w.Header().Set("Content-Type", "application/json")
    httpStatus := httpStatusForCode(code)
    w.WriteHeader(httpStatus)
    json.NewEncoder(w).Encode(errorResponse{
        Success: false,
        Error: errorDetail{
            Code:    toScreamingSnake(code),
            Message: message,
        },
    })
}

// toScreamingSnake converts Encore error code to SCREAMING_SNAKE_CASE
func toScreamingSnake(code errs.ErrCode) string {
    return strings.ToUpper(string(code))
}

// httpStatusForCode maps Encore error codes to HTTP status codes
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

### Endpoint Conversion

Convert all endpoints from typed to raw:

**Before (non-raw):**
```go
//encore:api auth path=/projects method=POST
func Create(ctx context.Context, req *CreateRequest) (*Project, error)
```

**After (raw):**
```go
//encore:api auth raw path=/projects method=POST
func Create(w http.ResponseWriter, r *http.Request)
```

### Request Parsing

Parse JSON body manually in raw handlers:

```go
var req CreateRequest
if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
    response.WriteError(w, errs.InvalidArgument, "Invalid request body")
    return
}
if err := req.Validate(); err != nil {
    // Validate() returns *errs.Error
    if eErr, ok := err.(*errs.Error); ok {
        response.WriteError(w, eErr.Code, eErr.Message)
    } else {
        response.WriteError(w, errs.InvalidArgument, err.Error())
    }
    return
}
```

### Path Parameter Extraction

Extract path parameters in raw handlers using Encore's pattern:

```go
// For /projects/:id pattern
id := r.PathValue("id") // Go 1.22+

if id == "" {
    response.WriteError(w, errs.InvalidArgument, "Missing id parameter")
    return
}
```

### Authentication

Keep Encore's `//encore:api auth` annotation. Encore validates JWT before handler runs. Extract user info using Encore's auth package:

```go
import "encore.dev/beta/auth"

// Get authenticated user ID (returns (uid, ok))
uid, ok := auth.UserID()
if !ok {
    // This shouldn't happen if //encore:api auth is set, but handle defensively
    response.WriteError(w, errs.Unauthenticated, "Not authenticated")
    return
}

// Get custom user data returned by auth handler
userData := auth.Data().(*auth.UserData)
```

### Cookie Handling

Auth endpoints continue to set cookies alongside the new response format:

```go
// Set auth cookies
http.SetCookie(w, &http.Cookie{
    Name:     "auth_token",
    Value:    token,
    HttpOnly: true,
    Secure:   true,
    SameSite: http.SameSiteStrictMode,
    Path:     "/",
})

// Return standardized response
response.WriteSuccess(w, LoginResponse{...})
```

## Services to Update

### projects
| Endpoint | Method | Auth | Notes |
|----------|--------|------|-------|
| `/projects` | GET | No | Public list |
| `/projects` | POST | Yes | Create |
| `/projects/:id` | PUT | Yes | Update |
| `/projects/:id` | DELETE | Yes | Soft delete |

### work
| Endpoint | Method | Auth | Notes |
|----------|--------|------|-------|
| `/work` | GET | No | Public list |
| `/work` | POST | Yes | Create |
| `/work/:id` | PUT | Yes | Update |
| `/work/:id` | DELETE | Yes | Soft delete |

### hero
| Endpoint | Method | Auth | Notes |
|----------|--------|------|-------|
| `/hero` | GET | No | Public get |
| `/hero` | PUT | Yes | Update |

### auth
| Endpoint | Method | Auth | Notes |
|----------|--------|------|-------|
| `/auth/login` | POST | No | Already raw, wraps response in format |
| `/auth/refresh` | POST | No | Already raw, wraps response in format |
| `/auth/logout` | POST | No | Already raw, wraps response in format |

### webhook
| Endpoint | Method | Auth | Notes |
|----------|--------|------|-------|
| `/webhook/rebuild` | POST | Bearer | Already raw, wraps response in format |

## File Changes

1. **pkg/response/error.go** — Replace with new implementation (breaking change from old format)
2. **projects/projects.go** — Convert to raw endpoints
3. **work/work.go** — Convert to raw endpoints
4. **hero/hero.go** — Convert to raw endpoints
5. **auth/login.go** — Wrap response in new format
6. **auth/refresh.go** — Wrap response in new format
7. **auth/logout.go** — Wrap response in new format
8. **webhook/webhook.go** — Wrap response in new format

## OpenAPI Considerations

After conversion, regenerate OpenAPI spec:
```bash
encore gen openapi > docs/openapi.json
```

Note: Raw endpoints may require manual OpenAPI documentation updates since Encore can't infer schemas from raw handlers.

## Breaking Changes

This is a **breaking change** for all clients. Frontend (deren-cms) and any other consumers must update to handle the new response format.

### Old Format (Before)
```json
// Success - direct data
{ "projects": [...] }

// Error - Encore format
{ "code": "not_found", "message": "...", "details": null }
```

### New Format (After)
```json
// Success - wrapped
{ "success": true, "data": { "projects": [...] } }

// Error - wrapped with screaming snake code
{ "success": false, "error": { "code": "NOT_FOUND", "message": "..." } }
```

Migration checklist:
1. Deploy backend changes
2. Update frontend to parse new format
3. Test all endpoints
