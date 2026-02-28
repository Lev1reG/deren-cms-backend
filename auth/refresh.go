package auth

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"
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
		http.Error(w, "Refresh token not found", http.StatusUnauthorized)
		return
	}

	if refreshCookie.Value == "" {
		http.Error(w, "Refresh token is empty", http.StatusUnauthorized)
		return
	}

	// Call Supabase Auth API to refresh
	httpClient := &http.Client{Timeout: 10 * time.Second}
	apiURL := secrets.SupabaseURL + "/auth/v1/token?grant_type=refresh_token"

	formData := url.Values{}
	formData.Set("refresh_token", refreshCookie.Value)

	httpReq, err := http.NewRequest("POST", apiURL, strings.NewReader(formData.Encode()))
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpReq.Header.Set("apikey", secrets.SupabaseAnonKey)

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, "Invalid or expired refresh token", http.StatusUnauthorized)
		return
	}

	var authResp supabaseAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
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
