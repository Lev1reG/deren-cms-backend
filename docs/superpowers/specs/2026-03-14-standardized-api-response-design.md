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
| 200 | Success (GET, PUT, POST) |
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

Create helper functions in `pkg/response`:

```go
// WriteSuccess writes a standardized success response
func WriteSuccess(w http.ResponseWriter, data interface{})

// WriteError writes a standardized error response
func WriteError(w http.ResponseWriter, code errs.ErrCode, message string)

// WriteErrorWithStatus writes an error with explicit HTTP status
func WriteErrorWithStatus(w http.ResponseWriter, statusCode int, code string, message string)
```

Usage:

```go
// Success
response.WriteSuccess(w, project)
response.WriteSuccess(w, projects)
response.WriteSuccess(w, nil)

// Error
response.WriteError(w, errs.NotFound, "Project not found")
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
    response.WriteError(w, errs.InvalidArgument, err.Error())
    return
}
```

### Authentication

Keep Encore's `//encore:api auth` annotation. Encore validates JWT before handler runs. Extract user info from context when needed:

```go
uid, userInfo := auth.UserIDFromContext(r.Context())
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
| `/auth/login` | POST | No | Already raw |
| `/auth/refresh` | POST | No | Already raw |
| `/auth/logout` | POST | No | Already raw |

### webhook
| Endpoint | Method | Auth | Notes |
|----------|--------|------|-------|
| `/webhook/rebuild` | POST | Bearer | Already raw, uses custom auth |

## File Changes

1. **pkg/response/error.go** — Update `WriteError` to use new format
2. **pkg/response/response.go** (new) — Add `WriteSuccess` helper
3. **projects/projects.go** — Convert to raw endpoints
4. **work/work.go** — Convert to raw endpoints
5. **hero/hero.go** — Convert to raw endpoints
6. **auth/login.go** — Update response format (already raw)
7. **auth/refresh.go** — Update response format (already raw)
8. **auth/logout.go** — Update response format (already raw)
9. **webhook/webhook.go** — Update response format (already raw)

## OpenAPI Considerations

After conversion, regenerate OpenAPI spec:
```bash
encore gen openapi > docs/openapi.json
```

Note: Raw endpoints may require manual OpenAPI documentation updates since Encore can't infer schemas from raw handlers.

## Breaking Changes

This is a **breaking change** for all clients. Frontend (deren-cms) and any other consumers must update to handle the new response format.

Migration checklist:
1. Deploy backend changes
2. Update frontend to parse new format
3. Test all endpoints
