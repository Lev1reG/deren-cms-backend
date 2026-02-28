package auth

import (
	"context"

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
	return nil, &errs.Error{Code: errs.Unimplemented, Message: "not implemented"}
}
