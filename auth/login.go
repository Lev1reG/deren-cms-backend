package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"encore.app/pkg/response"
	"encore.dev/beta/errs"
)

// LoginRequest is the request body for login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse is the response for successful login.
type LoginResponse struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
}

// supabaseAuthResponse represents Supabase's auth API response.
type supabaseAuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	User         struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Role  string `json:"role"`
	} `json:"user"`
}

// Validate validates the login request.
func (r *LoginRequest) Validate() error {
	if r.Email == "" {
		return &errs.Error{Code: errs.InvalidArgument, Message: "email is required"}
	}
	if r.Password == "" {
		return &errs.Error{Code: errs.InvalidArgument, Message: "password is required"}
	}
	return nil
}

//encore:api public raw path=/auth/login method=POST
func Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, errs.InvalidArgument, err.Error())
		return
	}
	if err := req.Validate(); err != nil {
		var eErr *errs.Error
		if errors.As(err, &eErr) {
			response.WriteError(w, eErr.Code, eErr.Message)
		} else {
			response.WriteError(w, errs.InvalidArgument, err.Error())
		}
		return
	}

	// Call Supabase Auth API
	httpClient := &http.Client{Timeout: 10 * time.Second}
	url := secrets.SupabaseURL + "/auth/v1/token?grant_type=password"

	body, err := json.Marshal(map[string]string{
		"email":    req.Email,
		"password": req.Password,
	})
	if err != nil {
		response.WriteError(w, errs.Internal, "Internal error")
		return
	}

	httpReq, err := http.NewRequestWithContext(r.Context(), "POST", url, bytes.NewReader(body))
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
		if resp.StatusCode == http.StatusBadRequest {
			response.WriteError(w, errs.Unauthenticated, "Invalid email or password")
			return
		}
		response.WriteError(w, errs.Internal, "Authentication service error")
		return
	}

	var authResp supabaseAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		response.WriteError(w, errs.Internal, "Internal error")
		return
	}

	// Set httpOnly cookie for access token
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    authResp.AccessToken,
		Expires:  time.Now().Add(time.Duration(authResp.ExpiresIn) * time.Second),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})

	// Set refresh token cookie (non-httpOnly for frontend to read)
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    authResp.RefreshToken,
		Expires:  time.Now().Add(30 * 24 * time.Hour), // 30 days
		HttpOnly: false,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})

	// Return user data
	respData := LoginResponse{
		UserID: authResp.User.ID,
		Email:  authResp.User.Email,
		Role:   authResp.User.Role,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(respData)
}
