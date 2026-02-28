# Authentication with HttpOnly Cookie Design

**Date:** 2026-02-28
**Status:** Approved

## Overview

Implement secure cookie-based authentication for the admin panel using Supabase Auth. The backend acts as an auth proxy, managing httpOnly cookies for access tokens while handling login, logout, and token refresh.

## Architecture

### Backend (deren-cms-backend)

**New endpoints in `auth` package:**
- `POST /auth/login` — authenticate user, set httpOnly cookie
- `POST /auth/logout` — clear httpOnly cookie
- `POST /auth/refresh` — exchange refresh token for new access token

**Existing:**
- `AuthHandler` — updated to read JWT from httpOnly cookie (fallback to Bearer header)

**Supabase Integration:**
- Use Supabase Auth REST API (anon key) for all auth operations
- JWKS-based JWT validation (already implemented)

### Frontend (deren-cms)

**New components:**
- `src/lib/auth-client.ts` — API client wrapper with 401 retry logic
- Login page with React Hook Form + Zod validation

**Changes:**
- Remove localStorage JWT handling
- AuthProvider simplifies to track `isAuthenticated` state
- API client uses `credentials: include` for cookie requests

## Components

### Backend Components

**`auth/login.go`**
- Request: `{email, password}`
- Response: `{user_id, email, role}`
- Sets `auth_token` httpOnly cookie with access JWT
- Sets `refresh_token` cookie (non-httpOnly for frontend read)

**`auth/logout.go`**
- Clears both cookies with `Max-Age=-1`

**`auth/refresh.go`**
- Reads refresh token from cookie
- Calls Supabase to get new access token
- Updates `auth_token` cookie

**`auth/auth.go`** (update)
- Reads JWT from `Cookie` header first, then `Authorization` header

### Frontend Components

**`src/lib/auth-client.ts`**
- Wraps fetch with credentials include
- On 401: calls `/auth/refresh`, then retries original request

**Login Page**
- Form with email/password fields
- React Hook Form + Zod validation
- Calls `/auth/login` on submit

## Data Flow

### Login Flow
```
User enters email/password
    ↓
Frontend POST /auth/login
    ↓
Backend calls Supabase /auth/v1/token?grant_type=password
    ↓
Supabase returns {access_token, refresh_token, user}
    ↓
Backend sets httpOnly cookie: auth_token=access_token
Backend sets refresh cookie: refresh_token=refresh_token
Backend returns {user_id, email, role}
    ↓
Frontend sets auth state = authenticated
Redirects to dashboard
```

### Authenticated Request Flow
```
Frontend API call (with credentials: include)
    ↓
Browser sends auth_token cookie automatically
    ↓
Backend AuthHandler reads JWT from cookie
    ↓
Backend validates JWT via JWKS
    ↓
Request proceeds with user context
```

### 401 Refresh Flow
```
API call returns 401
    ↓
Frontend detects 401
    ↓
POST /auth/refresh with refresh_token cookie
    ↓
Backend calls Supabase /auth/v1/token?grant_type=refresh_token
    ↓
Supabase returns new {access_token, refresh_token}
    ↓
Backend updates both cookies
    ↓
Frontend retries original request
```

### Logout Flow
```
Frontend POST /auth/logout
    ↓
Backend clears both cookies (Max-Age=-1)
    ↓
Frontend clears auth state
    ↓
Redirect to login page
```

## Error Handling

### Backend Errors
- `400 InvalidArgument`: Invalid email/password format, missing fields
- `401 Unauthenticated`: Invalid credentials, invalid/expired refresh token
- `429 TooManyRequests`: Supabase rate limit (forward from Supabase)
- `500 Internal`: Supabase API failure, network error

### Frontend Errors
- Login: Display user-friendly messages (e.g., "Invalid email or password")
- Refresh: If refresh fails, redirect to login and clear auth state
- API: Show toast notification for 4xx/5xx errors

### Security Considerations
- Validate all inputs on backend
- Never return raw Supabase errors to frontend (log, return generic message)
- Cookies: `Secure` flag for HTTPS, `SameSite=Strict` to prevent CSRF
- Relies on Supabase for rate limiting

## Cookie Configuration

| Cookie | Type | Secure | SameSite | Path | Max-Age | Purpose |
|--------|------|--------|----------|------|---------|---------|
| `auth_token` | httpOnly | Yes | Strict | / | 3600s (1hr) | Access JWT |
| `refresh_token` | Not httpOnly | Yes | Strict | / | 2592000s (30days) | Refresh token |

**Note:** Refresh token is non-httpOnly to allow frontend to read it for the 401 refresh flow. This is a trade-off for simplicity. Alternative: store in memory only, lose session on page refresh.

## Environment Variables

### Backend (.secrets.local.cue)
```cue
SupabaseURL:       "https://<project>.supabase.co"      // existing
SupabaseAnonKey:   "<anon-key>"                           // new
SupabaseJWKSURL:   "https://<project>.supabase.co/auth/v1/jwks"  // existing
```

### Frontend (.env)
```bash
VITE_ENCORE_API_URL="https://your-app.encr.app"  // existing
# Supabase URL/AnonKey no longer needed
```

## Testing

### Backend Tests
- `auth/login_test.go`:
  - Valid credentials → returns user, sets cookie header
  - Invalid credentials → 401 error
  - Missing fields → 400 error
  - Supabase API failure → 500 error
- `auth/logout_test.go`:
  - Valid logout → clears cookies
- `auth/refresh_test.go`:
  - Valid refresh token → returns new token, updates cookie
  - Invalid/expired refresh token → 401 error
- `auth/auth_test.go` (update):
  - AuthHandler reads from Cookie header
  - Falls back to Authorization header

### Frontend Tests
- Auth client: 401 triggers refresh → retry flow
- Login form: validation, submission, error display
- AuthProvider: state updates on login/logout

### Integration
- Full flow: login → authenticated request → refresh → logout
- Mock Supabase API in backend tests

## Migration Path

1. Add new backend endpoints
2. Update `AuthHandler` to read cookies
3. Implement frontend auth client and login page
4. Update frontend API calls to use credentials
5. Remove localStorage JWT handling
6. Deploy and test

**Backward compatibility:** Existing Bearer header support retained in `AuthHandler` for smooth transition.

## No Database Changes Required

Supabase manages user authentication — no migrations needed.
