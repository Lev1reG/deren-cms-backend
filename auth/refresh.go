package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"encore.app/pkg/response"
	"encore.dev/beta/errs"
)

// RefreshResponse is the response for successful token refresh.
type RefreshResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

//encore:api public raw path=/auth/refresh method=POST
func Refresh(w http.ResponseWriter, r *http.Request) {
	// Get refresh token from cookie
	refreshCookie, err := r.Cookie("refresh_token")
	if err != nil {
		response.WriteError(w, errs.Unauthenticated, "Refresh token not found")
		return
	}

	if refreshCookie.Value == "" {
		response.WriteError(w, errs.Unauthenticated, "Refresh token is empty")
		return
	}

	// Call Supabase Auth API to refresh
	httpClient := &http.Client{Timeout: 10 * time.Second}
	apiURL := secrets.SupabaseURL + "/auth/v1/token?grant_type=refresh_token"

	body, err := json.Marshal(map[string]string{
		"refresh_token": refreshCookie.Value,
	})
	if err != nil {
		response.WriteError(w, errs.Internal, "Internal error")
		return
	}

	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewReader(body))
	if err != nil {
		response.WriteError(w, errs.Internal, "Internal error")
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("apikey", secrets.SupabaseAnonKey)

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		response.WriteError(w, errs.Internal, "Internal error")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		response.WriteError(w, errs.Unauthenticated, "Invalid or expired refresh token")
		return
	}

	var authResp supabaseAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		response.WriteError(w, errs.Internal, "Internal error")
		return
	}

	// Update access token cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    authResp.AccessToken,
		Expires:  time.Now().Add(time.Duration(authResp.ExpiresIn) * time.Second),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})

	// Update refresh token cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    authResp.RefreshToken,
		Expires:  time.Now().Add(30 * 24 * time.Hour), // 30 days
		HttpOnly: false,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})

	// Return new access token
	respData := RefreshResponse{
		AccessToken: authResp.AccessToken,
		ExpiresIn:   authResp.ExpiresIn,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(respData)
}
