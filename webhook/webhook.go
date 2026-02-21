// Package webhook provides endpoints for triggering external webhooks.
package webhook

import (
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
