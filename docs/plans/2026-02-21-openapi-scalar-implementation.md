# OpenAPI + Scalar Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add interactive API documentation at `/docs` using Scalar, backed by Encore's generated OpenAPI spec.

**Architecture:** Two Encore raw endpoints (`/docs` and `/openapi.json`) served from a new `docs` service. OpenAPI spec is generated via Encore CLI and committed to the repo.

**Tech Stack:** Encore.go raw endpoints, Scalar (CDN), OpenAPI 3.0

---

## Task 1: Generate Initial OpenAPI Spec

**Files:**
- Create: `docs/openapi.json`

**Step 1: Generate the OpenAPI spec using Encore CLI**

Run:
```bash
cd /home/levireg/Documents/projects/Personal_Website/deren-cms-backend
encore gen client --lang=openapi > docs/openapi.json
```

Expected: File `docs/openapi.json` created with valid OpenAPI 3.0 spec

**Step 2: Verify the spec was generated correctly**

Run:
```bash
head -20 docs/openapi.json
```

Expected: JSON starting with `{"openapi": "3.0` containing your API endpoints

**Step 3: Commit the OpenAPI spec**

```bash
git add docs/openapi.json
git commit -m "$(cat <<'EOF'
docs: add generated OpenAPI spec

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Create OpenAPI Spec Endpoint

**Files:**
- Create: `docs/docs.go`

**Step 1: Write the `/openapi.json` endpoint**

Create file `docs/docs.go`:

```go
// Package docs provides API documentation endpoints.
package docs

import (
	"net/http"
	"os"
	"path/filepath"

	"encore.dev/beta/errs"
)

// openAPIPath is the path to the OpenAPI spec file.
// When running locally, Encore uses the project root as working directory.
// In production, we embed the file or read from a known location.
var openAPIPath = filepath.Join("docs", "openapi.json")

//encore:api public raw path=/openapi.json
func OpenAPISpec(w http.ResponseWriter, req *http.Request) {
	data, err := os.ReadFile(openAPIPath)
	if err != nil {
		errs.HTTPError(w, &errs.Error{
			Code:    errs.Internal,
			Message: "failed to read OpenAPI spec",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
```

**Step 2: Verify it compiles**

Run:
```bash
cd /home/levireg/Documents/projects/Personal_Website/deren-cms-backend
encore check
```

Expected: No errors

**Step 3: Commit**

```bash
git add docs/docs.go
git commit -m "$(cat <<'EOF'
feat(docs): add /openapi.json endpoint

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Create Scalar Documentation Endpoint

**Files:**
- Modify: `docs/docs.go`

**Step 1: Add the `/docs` endpoint with Scalar HTML**

Add to `docs/docs.go` after the `OpenAPISpec` function:

```go
//encore:api public raw path=/docs
func Docs(w http.ResponseWriter, req *http.Request) {
	html := `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8" />
  <title>API Documentation</title>
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <style>
    body { margin: 0; }
  </style>
</head>
<body>
  <script id="api-reference" data-url="/openapi.json"></script>
  <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(html))
}
```

**Step 2: Verify it compiles**

Run:
```bash
cd /home/levireg/Documents/projects/Personal_Website/deren-cms-backend
encore check
```

Expected: No errors

**Step 3: Commit**

```bash
git add docs/docs.go
git commit -m "$(cat <<'EOF'
feat(docs): add /docs endpoint with Scalar UI

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Test Documentation Endpoints

**Step 1: Start the Encore development server**

Run:
```bash
cd /home/levireg/Documents/projects/Personal_Website/deren-cms-backend
encore run
```

Expected: Server starts on `http://localhost:4000`

**Step 2: Test `/openapi.json` endpoint**

Run (in another terminal):
```bash
curl http://localhost:4000/openapi.json | head -20
```

Expected: JSON response with OpenAPI spec

**Step 3: Test `/docs` endpoint**

Open in browser or run:
```bash
curl http://localhost:4000/docs
```

Expected: HTML response with Scalar script tags

**Step 4: Verify Scalar UI loads**

Open `http://localhost:4000/docs` in browser.

Expected: Scalar documentation UI renders with your API endpoints listed

---

## Task 5: Final Verification and Push

**Step 1: Run full test suite**

Run:
```bash
cd /home/levireg/Documents/projects/Personal_Website/deren-cms-backend
encore test ./...
```

Expected: All tests pass

**Step 2: Verify final state**

Run:
```bash
git status
git log --oneline -5
```

Expected: Clean working tree, all commits present

---

## Summary

After completion:
- `/openapi.json` - Serves the generated OpenAPI 3.0 spec
- `/docs` - Interactive Scalar documentation UI
- OpenAPI spec can be regenerated with: `encore gen client --lang=openapi > docs/openapi.json`
