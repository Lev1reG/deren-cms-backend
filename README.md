# Deren CMS Backend

A RESTful API backend for managing personal website content. Built with Encore.go and PostgreSQL.

## Features

- **Projects Management** - CRUD operations for portfolio projects
- **Work Experience** - Manage professional history with work types
- **Hero Section** - Dynamic hero content management
- **API Documentation** - Interactive API docs with Scalar UI
- **JWT Authentication** - Supabase-backed JWT validation with JWKS caching
- **Webhook Support** - Trigger external webhooks (e.g., Netlify rebuilds)
- **Soft Deletes** - Data retention with deleted_at timestamps

## API Endpoints

### Public Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/projects` | List all projects |
| GET | `/work` | List work experiences |
| GET | `/hero` | Get hero section content |
| GET | `/docs` | Interactive API documentation |
| GET | `/openapi.json` | OpenAPI specification |

### Authenticated Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/projects` | Create a project |
| PUT | `/projects/:id` | Update a project |
| DELETE | `/projects/:id` | Delete a project |
| POST | `/work` | Create work experience |
| PUT | `/work/:id` | Update work experience |
| DELETE | `/work/:id` | Delete work experience |
| PUT | `/hero` | Update hero section |

### Webhooks

| Method | Path | Description |
|--------|------|-------------|
| POST | `/webhook/rebuild` | Trigger external build (requires Bearer token) |

## Prerequisites

**Install Encore:**
- **macOS:** `brew install encoredev/tap/encore`
- **Linux:** `curl -L https://encore.dev/install.sh | bash`
- **Windows:** `iwr https://encore.dev/install.ps1 | iex`

**Docker** is required for running PostgreSQL locally.

## Setup

### Clone the repository

```bash
git clone https://github.com/Lev1reG/deren-cms-backend.git
cd deren-cms-backend
```

### Configure Secrets

Create a `.secrets.local.cue` file in the project root:

```cue
// Supabase JWT validation
SupabaseJWKSURL: "https://<your-project>.supabase.co/auth/v1/.well-known/jwks.json"

// Webhook configuration (optional)
NetlifyBuildHook: "https://api.netlify.com/build_hooks/<hook-id>"
WebhookSecret: "your-webhook-secret-token"
```

### Run locally

```bash
encore run
```

The API will be available at `http://localhost:4000`

### Local Development Dashboard

While `encore run` is running, open [http://localhost:9400/](http://localhost:9400/) to access Encore's [local developer dashboard](https://encore.dev/docs/go/observability/dev-dash).

Here you can view:
- Request traces
- Architecture diagram
- Service catalog

## API Documentation

Visit `/docs` on your running instance for interactive API documentation powered by Scalar.

## Services

### Projects Service

Manages portfolio projects with the following fields:
- `title` - Project title
- `description` - Project description
- `href` - Optional project link
- `technologies` - Array of technology tags
- `display_order` - Ordering priority

### Work Service

Manages work experience entries with the following fields:
- `company` - Company name
- `position` - Position held
- `date` - Employment date range
- `description` - Job description
- `href` - Optional company link
- `type` - Work type (Freelance, Internship, Contract, Part-Time, Full-Time)
- `display_order` - Ordering priority

### Hero Service

Manages the website's hero section:
- `phrases` - Array of rotating phrases
- `description` - Hero description text

### Auth Service

Provides JWT authentication using Supabase's JWKS endpoint:
- Validates JWT tokens against Supabase's public keys
- Caches JWKS keys for 1 hour
- Extracts user data (userID, email, role) from claims

### Webhook Service

Allows triggering external webhooks:
- Netlify build hook integration
- Bearer token authentication

### Docs Service

Serves API documentation:
- OpenAPI specification at `/openapi.json`
- Interactive documentation at `/docs` using Scalar UI

## Database Schema

The application uses PostgreSQL with the following tables:

- `projects` - Portfolio projects
- `hero` - Single-row hero content
- `work_experience` - Professional work history

All tables include:
- UUID primary keys
- `created_at` and `updated_at` timestamps
- Soft delete support (`deleted_at`)

## Authentication

The API uses JWT authentication backed by Supabase:

1. Obtain a JWT token from Supabase Auth
2. Include the token in the `Authorization` header as `Bearer <token>`
3. The auth service validates against Supabase's JWKS endpoint

## Testing

```bash
encore test ./...
```

## Deployment

### Encore Cloud Platform

Deploy to Encore's development cloud:

```bash
git add -A .
git commit -m 'Commit message'
git push encore
```

### Self-hosting

Build a Docker image:

```bash
encore build docker
```

See the [self-hosting instructions](https://encore.dev/docs/go/self-host/docker-build) for more details.

## Link to GitHub

Follow these steps to link your app to GitHub:

1. Create a GitHub repo, commit and push the app.
2. Open your app in the [Cloud Dashboard](https://app.encore.dev).
3. Go to **Settings ➔ GitHub** and click on **Link app to GitHub**.
4. Configure automatic deploys for specific branches.

## Learn More

- [Encore Documentation](https://encore.dev/docs)
- [Encore GitHub](https://github.com/encoredev/encore)
