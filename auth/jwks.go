package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"time"
)

// jwksResponse represents the JWKS endpoint response.
type jwksResponse struct {
	Keys []jwksKey `json:"keys"`
}

type jwksKey struct {
	Kty string   `json:"kty"`
	Kid string   `json:"kid"`
	Alg string   `json:"alg"`
	Use string   `json:"use"`
	Crv string   `json:"crv,omitempty"`
	N   string   `json:"n,omitempty"`
	E   string   `json:"e,omitempty"`
	X   string   `json:"x,omitempty"`
	Y   string   `json:"y,omitempty"`
	X5c []string `json:"x5c,omitempty"`
}

// verifyingKey represents any type of public key that can verify JWTs.
// We use interface{} to support both RSA and EC keys.
type verifyingKey interface{}

// fetchJWKS fetches JWKS from Supabase and caches keys.
func fetchJWKS(ctx context.Context) error {
	if secrets.SupabaseJWKSURL == "" {
		return fmt.Errorf("SupabaseJWKSURL secret not configured")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, secrets.SupabaseJWKSURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("JWKS endpoint returned status %d", resp.StatusCode)
	}

	var jwks jwksResponse
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("failed to decode JWKS: %w", err)
	}

	// Convert to verifying keys - support both RSA and EC
	keys := make(map[string]verifyingKey)
	for _, key := range jwks.Keys {
		var vk verifyingKey
		var err error

		switch key.Kty {
		case "RSA":
			vk, err = jwkToRSA(key)
		case "EC":
			vk, err = jwkToEC(key)
		default:
			continue // Skip unsupported key types
		}

		if err != nil {
			continue // Skip invalid keys
		}

		keys[key.Kid] = vk
	}

	// Update cache
	jwksCache.Lock()
	jwksCache.keys = keys
	jwksCache.fetched = time.Now()
	jwksCache.Unlock()

	return nil
}

// jwkToRSA converts a JWK to an RSA public key.
func jwkToRSA(jwk jwksKey) (*rsa.PublicKey, error) {
	if jwk.N == "" || jwk.E == "" {
		return nil, fmt.Errorf("missing RSA key parameters")
	}

	nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("failed to decode n: %w", err)
	}

	eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("failed to decode e: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)

	return &rsa.PublicKey{
		N: n,
		E: int(e.Int64()),
	}, nil
}

// jwkToEC converts a JWK to an ECDSA public key.
func jwkToEC(jwk jwksKey) (*ecdsa.PublicKey, error) {
	if jwk.X == "" || jwk.Y == "" {
		return nil, fmt.Errorf("missing EC key parameters")
	}

	xBytes, err := base64.RawURLEncoding.DecodeString(jwk.X)
	if err != nil {
		return nil, fmt.Errorf("failed to decode x: %w", err)
	}

	yBytes, err := base64.RawURLEncoding.DecodeString(jwk.Y)
	if err != nil {
		return nil, fmt.Errorf("failed to decode y: %w", err)
	}

	x := new(big.Int).SetBytes(xBytes)
	y := new(big.Int).SetBytes(yBytes)

	var curve elliptic.Curve
	switch jwk.Crv {
	case "P-256":
		curve = elliptic.P256()
	case "P-384":
		curve = elliptic.P384()
	case "P-521":
		curve = elliptic.P521()
	default:
		return nil, fmt.Errorf("unsupported EC curve: %s", jwk.Crv)
	}

	return &ecdsa.PublicKey{
		Curve: curve,
		X:     x,
		Y:     y,
	}, nil
}
