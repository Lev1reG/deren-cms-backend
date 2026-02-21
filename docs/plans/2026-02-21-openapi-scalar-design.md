# OpenAPI + Scalar API Documentation Design

**Date:** 2026-02-21
**Status:** Approved

## Overview

Add OpenAPI-standard API documentation served at `/docs` using Scalar for the interactive documentation UI.

## Goals

- Provide self-hosted API reference documentation
- Leverage Encore's built-in OpenAPI generation
- Keep implementation simple and maintainable

## Architecture

### Endpoints

| Path        | Method | Description                           |
|-------------|--------|---------------------------------------|
| `/docs`     | GET    | Scalar HTML UI (loads JS from CDN)    |
| `/openapi.json` | GET | Generated OpenAPI 3.0 spec            |

### Component Structure

```
docs/
├── openapi.json    # Generated OpenAPI spec (committed to repo)
└── docs.go         # Encore raw endpoints for /docs and /openapi.json
```

## Implementation Details

### 1. OpenAPI Spec Generation

Use Encore's built-in generator:

```bash
encore gen client --lang=openapi > docs/openapi.json
```

- Spec is regenerated manually when API changes
- File is committed to the repository
- Can be automated in CI later if needed

### 2. `/openapi.json` Endpoint

```go
//encore:api public raw path=/openapi.json
func OpenAPISpec(w http.ResponseWriter, req *http.Request) {
    // Serve docs/openapi.json with Content-Type: application/json
}
```

### 3. `/docs` Endpoint

```go
//encore:api public raw path=/docs
func Docs(w http.ResponseWriter, req *http.Request) {
    // Serve HTML that loads Scalar from CDN
    // Configure Scalar to fetch /openapi.json
}
```

Scalar HTML loads from CDN:
```html
<script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
```

## Data Flow

```
User visits /docs
       ↓
Encore serves Scalar HTML
       ↓
Scalar JS loads from CDN
       ↓
Scalar fetches /openapi.json
       ↓
Interactive API docs rendered
```

## Authentication

- OpenAPI spec documents all endpoints including auth requirements
- Scalar "Try It Out" requires manual Bearer token entry
- No special auth handling needed in docs endpoints (both are public)

## Maintenance

- Regenerate spec after API changes: `encore gen client --lang=openapi > docs/openapi.json`
- Commit updated spec to repository
- Scalar UI updates automatically via CDN

## Trade-offs

| Choice | Trade-off |
|--------|-----------|
| Encore built-in gen | Less control than hand-written, but stays in sync with code |
| Raw endpoints | Slightly more code than static files, but keeps everything in Encore |
| CDN Scalar | Requires internet connection, but always gets latest features |
| Manual regeneration | Could forget to update spec, but simpler than CI automation |
