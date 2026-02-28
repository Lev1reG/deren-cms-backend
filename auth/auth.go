// Package auth provides JWT validation against Supabase's JWKS endpoint.
package auth

import (
	"context"
	"crypto/rsa"
	"fmt"
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

// secrets holds the auth-related secrets.
var secrets struct {
	SupabaseURL     string
	SupabaseJWKSURL string
	SupabaseAnonKey string
}

// jwksCache holds cached JWKS keys.
var jwksCache struct {
	sync.RWMutex
	keys     map[string]*rsa.PublicKey
	fetched  time.Time
	duration time.Duration
}

func init() {
	jwksCache.keys = make(map[string]*rsa.PublicKey)
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
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return key, nil
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

// getVerifyingKey retrieves the RSA public key for the given kid from JWKS cache or fetches it.
func getVerifyingKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
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
func AuthHandler(ctx context.Context, token string) (auth.UID, *UserData, error) {
	userData, err := ValidateToken(ctx, token)
	if err != nil {
		return "", nil, &errs.Error{
			Code:    errs.Unauthenticated,
			Message: err.Error(),
		}
	}
	return auth.UID(userData.UserID), userData, nil
}
