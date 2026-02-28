// Package auth provides JWT validation against Supabase's JWKS endpoint.
package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"encore.dev/beta/auth"
	"encore.dev/beta/errs"
	"github.com/golang-jwt/jwt/v5"
)

// UserData represents the authenticated user's data extracted from JWT.
type UserData struct {
	UserID string
	Email  string
	Role   string
}

// AuthParams represents the authentication parameters extracted from the request.
// It supports both Authorization header and cookie-based authentication.
type AuthParams struct {
	// Token is the raw Authorization header value (without any parsing).
	Token string `header:"Authorization"`

	// AuthTokenCookie is the auth_token cookie value.
	AuthTokenCookie *http.Cookie `cookie:"auth_token"`
}

// secrets holds the auth-related secrets.
var secrets struct {
	SupabaseURL     string
	SupabaseJWKSURL string
	SupabaseAnonKey string
}

// jwksCache holds cached JWKS keys.
var jwksCache struct {
	sync.RWMutex
	keys     map[string]verifyingKey
	fetched  time.Time
	duration time.Duration
}

func init() {
	jwksCache.keys = make(map[string]verifyingKey)
	jwksCache.duration = 1 * time.Hour
}

// ValidateToken validates a JWT token against Supabase's JWKS.
// It returns the user data if valid, or an error if invalid.
func ValidateToken(ctx context.Context, tokenString string) (*UserData, error) {
	if tokenString == "" {
		return nil, fmt.Errorf("token is empty")
	}

	// Parse token without verifying to get the kid
	unverifiedToken, _, err := jwt.NewParser().ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	kid, ok := unverifiedToken.Header["kid"].(string)
	if !ok {
		return nil, fmt.Errorf("token missing kid header")
	}

	// Get the verifying key
	key, err := getVerifyingKey(ctx, kid)
	if err != nil {
		return nil, fmt.Errorf("failed to get verifying key: %w", err)
	}

	// Parse and verify the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Accept both RSA and EC signing methods
		switch token.Method.(type) {
		case *jwt.SigningMethodRSA, *jwt.SigningMethodECDSA:
			return key, nil
		default:
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
	})
	if err != nil {
		return nil, fmt.Errorf("failed to verify token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("token is invalid")
	}

	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("failed to extract claims")
	}

	userData := &UserData{}

	if sub, ok := claims["sub"].(string); ok {
		userData.UserID = sub
	}
	if email, ok := claims["email"].(string); ok {
		userData.Email = email
	}
	if role, ok := claims["role"].(string); ok {
		userData.Role = role
	}

	return userData, nil
}

// getVerifyingKey retrieves the public key for the given kid from JWKS cache or fetches it.
func getVerifyingKey(ctx context.Context, kid string) (verifyingKey, error) {
	// Check cache first
	jwksCache.RLock()
	if key, ok := jwksCache.keys[kid]; ok && time.Since(jwksCache.fetched) < jwksCache.duration {
		jwksCache.RUnlock()
		return key, nil
	}
	jwksCache.RUnlock()

	// Fetch new keys
	if err := fetchJWKS(ctx); err != nil {
		return nil, err
	}

	// Try cache again
	jwksCache.RLock()
	defer jwksCache.RUnlock()
	if key, ok := jwksCache.keys[kid]; ok {
		return key, nil
	}

	return nil, fmt.Errorf("key with kid %s not found", kid)
}

//encore:authhandler
func AuthHandler(ctx context.Context, p *AuthParams) (auth.UID, *UserData, error) {
	var token string

	// Try Authorization header first (remove "Bearer " prefix if present)
	if p.Token != "" {
		token = strings.TrimPrefix(p.Token, "Bearer ")
	}

	// Fall back to cookie
	if token == "" && p.AuthTokenCookie != nil {
		token = p.AuthTokenCookie.Value
	}

	if token == "" {
		return "", nil, &errs.Error{
			Code:    errs.Unauthenticated,
			Message: "invalid auth param",
		}
	}

	userData, err := ValidateToken(ctx, token)
	if err != nil {
		return "", nil, &errs.Error{
			Code:    errs.Unauthenticated,
			Message: err.Error(),
		}
	}
	return auth.UID(userData.UserID), userData, nil
}
