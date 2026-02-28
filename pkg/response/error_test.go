package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"encore.dev/beta/errs"
	"testing"
)

// testErrorResponse is a helper struct for testing that decodes the code as a string
// since errs.ErrCode marshals to a string but doesn't unmarshal from a string.
type testErrorResponse struct {
	Code    string          `json:"code"`
	Message string          `json:"message"`
	Details json.RawMessage `json:"details"`
}

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

			var resp testErrorResponse
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if resp.Code != tt.code.String() {
				t.Errorf("expected code %s, got %s", tt.code.String(), resp.Code)
			}

			if resp.Message != tt.message {
				t.Errorf("expected message %q, got %q", tt.message, resp.Message)
			}

			var details interface{}
			if len(resp.Details) > 0 && resp.Details[0] != 'n' { // not "null"
				if err := json.Unmarshal(resp.Details, &details); err != nil {
					t.Fatalf("failed to decode details: %v", err)
				}
				if details != nil {
					t.Errorf("expected details to be null, got %v", details)
				}
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
