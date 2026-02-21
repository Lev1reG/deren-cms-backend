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
