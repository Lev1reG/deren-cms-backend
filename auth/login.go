package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

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

//encore:api public path=/auth/login method=POST
func Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Call Supabase Auth API
	httpClient := &http.Client{Timeout: 10 * time.Second}
	url := secrets.SupabaseURL + "/auth/v1/token?grant_type=password"

	body, err := json.Marshal(map[string]string{
		"email":    req.Email,
		"password": req.Password,
	})
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to encode request")
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to create request")
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("apikey", secrets.SupabaseAnonKey)

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to call Supabase auth")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusUnauthorized {
			return nil, &errs.Error{
				Code:    errs.Unauthenticated,
				Message: "Invalid email or password",
			}
		}
		return nil, &errs.Error{
			Code:    errs.Internal,
			Message: "Authentication service error",
		}
	}

	var authResp supabaseAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to decode response")
	}

	return &LoginResponse{
		UserID: authResp.User.ID,
		Email:  authResp.User.Email,
		Role:   authResp.User.Role,
	}, nil
}
