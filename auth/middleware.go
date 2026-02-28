package auth

import (
	"net/http"
	"strings"

	"encore.dev/beta/auth"
)

// CookieAuth extracts JWT from cookie and sets it in the auth context.
// This is used as middleware for endpoints that support cookie auth.
func CookieAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to get token from Authorization header first
		authHeader := r.Header.Get("Authorization")
		var token string

		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
				token = parts[1]
			}
		}

		// If no token in header, try cookie
		if token == "" {
			if cookie, err := r.Cookie("auth_token"); err == nil && cookie.Value != "" {
				token = cookie.Value
			}
		}

		// If we have a token, validate and set auth context
		if token != "" {
			userData, err := ValidateToken(r.Context(), token)
			if err == nil {
				ctx := auth.WithContext(r.Context(), auth.UID(userData.UserID), userData)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		// No valid token, continue without auth context
		// Protected endpoints will fail with 401
		next.ServeHTTP(w, r)
	})
}
