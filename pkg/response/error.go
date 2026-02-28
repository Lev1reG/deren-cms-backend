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

// WriteError writes a standardized error response to the HTTP response writer.
// It sets the Content-Type header to application/json, sets the appropriate
// HTTP status code based on the error code, and writes the error response as JSON.
//
// Example:
//
//	response.WriteError(w, errs.Unauthenticated, "Invalid email or password")
func WriteError(w http.ResponseWriter, code errs.ErrCode, message string) {
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
