// Package projects provides CRUD operations for project content.
package projects

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"encore.app/pkg/database"
	"encore.dev/beta/errs"
)

// Response types for API
type (
	// Project represents a project in API responses.
	Project struct {
		ID           string   `json:"id"`
		Title        string   `json:"title"`
		Description  string   `json:"description"`
		Href         *string  `json:"href,omitempty"`
		Technologies []string `json:"technologies"`
		DisplayOrder int      `json:"display_order"`
		CreatedAt    string   `json:"created_at"`
		UpdatedAt    string   `json:"updated_at"`
	}

	// ListResponse is the response for listing projects.
	ListResponse struct {
		Projects []*Project `json:"projects"`
	}

	// CreateRequest is the request body for creating a project.
	CreateRequest struct {
		Title        string   `json:"title"`
		Description  string   `json:"description"`
		Href         *string  `json:"href,omitempty"`
		Technologies []string `json:"technologies"`
		DisplayOrder int      `json:"display_order"`
	}

	// UpdateRequest is the request body for updating a project.
	// ID is populated from the path parameter.
	UpdateRequest struct {
		ID           string   `json:"-"`
		Title        *string  `json:"title,omitempty"`
		Description  *string  `json:"description,omitempty"`
		Href         *string  `json:"href,omitempty"`
		Technologies []string `json:"technologies,omitempty"`
		DisplayOrder *int     `json:"display_order,omitempty"`
	}
)

// dbToProject converts a database Project to API Project.
func dbToProject(db *database.Project) *Project {
	return &Project{
		ID:           db.ID,
		Title:        db.Title,
		Description:  db.Description,
		Href:         db.Href,
		Technologies: db.Technologies,
		DisplayOrder: db.DisplayOrder,
		CreatedAt:    db.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    db.UpdatedAt.Format(time.RFC3339),
	}
}

//encore:api public path=/projects method=GET
func List(ctx context.Context) (*ListResponse, error) {
	pool, err := database.Get(ctx)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to get database connection")
	}

	query := `
		SELECT id, title, description, href, technologies, display_order, created_at, updated_at
		FROM projects
		WHERE deleted_at IS NULL
		ORDER BY display_order ASC, created_at DESC
	`

	rows, err := pool.Query(ctx, query)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to query projects")
	}
	defer rows.Close()

	var projects []*Project
	for rows.Next() {
		var db database.Project

		err := rows.Scan(
			&db.ID,
			&db.Title,
			&db.Description,
			&db.Href,
			&db.Technologies,
			&db.DisplayOrder,
			&db.CreatedAt,
			&db.UpdatedAt,
		)
		if err != nil {
			return nil, errs.WrapCode(err, errs.Internal, "failed to scan project row")
		}

		projects = append(projects, dbToProject(&db))
	}

	if err := rows.Err(); err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "error iterating project rows")
	}

	if projects == nil {
		projects = []*Project{}
	}

	return &ListResponse{Projects: projects}, nil
}

//encore:api auth path=/projects method=POST
func Create(ctx context.Context, req *CreateRequest) (*Project, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	pool, err := database.Get(ctx)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to get database connection")
	}

	query := `
		INSERT INTO projects (title, description, href, technologies, display_order)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, title, description, href, technologies, display_order, created_at, updated_at
	`

	var db database.Project

	err = pool.QueryRow(ctx, query,
		req.Title,
		req.Description,
		req.Href,
		req.Technologies,
		req.DisplayOrder,
	).Scan(
		&db.ID,
		&db.Title,
		&db.Description,
		&db.Href,
		&db.Technologies,
		&db.DisplayOrder,
		&db.CreatedAt,
		&db.UpdatedAt,
	)

	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to create project")
	}

	return dbToProject(&db), nil
}

func (r *CreateRequest) Validate() error {
	if r.Title == "" {
		return &errs.Error{Code: errs.InvalidArgument, Message: "title is required"}
	}
	if r.Description == "" {
		return &errs.Error{Code: errs.InvalidArgument, Message: "description is required"}
	}
	if r.Technologies == nil {
		r.Technologies = []string{}
	}
	return nil
}

//encore:api auth path=/projects/:id method=PUT
func Update(ctx context.Context, id string, req *UpdateRequest) (*Project, error) {
	req.ID = id

	pool, err := database.Get(ctx)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to get database connection")
	}

	// Build dynamic update query
	updates := []string{}
	args := []interface{}{}
	argNum := 1

	if req.Title != nil {
		updates = append(updates, fmt.Sprintf("title = $%d", argNum))
		args = append(args, *req.Title)
		argNum++
	}
	if req.Description != nil {
		updates = append(updates, fmt.Sprintf("description = $%d", argNum))
		args = append(args, *req.Description)
		argNum++
	}
	if req.Href != nil {
		updates = append(updates, fmt.Sprintf("href = $%d", argNum))
		args = append(args, *req.Href)
		argNum++
	}
	if req.Technologies != nil {
		updates = append(updates, fmt.Sprintf("technologies = $%d", argNum))
		args = append(args, req.Technologies)
		argNum++
	}
	if req.DisplayOrder != nil {
		updates = append(updates, fmt.Sprintf("display_order = $%d", argNum))
		args = append(args, *req.DisplayOrder)
		argNum++
	}

	if len(updates) == 0 {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "no fields to update"}
	}

	// Add ID as last argument
	args = append(args, req.ID)

	query := fmt.Sprintf(`
		UPDATE projects
		SET %s
		WHERE id = $%d AND deleted_at IS NULL
		RETURNING id, title, description, href, technologies, display_order, created_at, updated_at
	`, joinUpdates(updates), argNum)

	var db database.Project

	err = pool.QueryRow(ctx, query, args...).Scan(
		&db.ID,
		&db.Title,
		&db.Description,
		&db.Href,
		&db.Technologies,
		&db.DisplayOrder,
		&db.CreatedAt,
		&db.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, &errs.Error{Code: errs.NotFound, Message: "project not found"}
	}
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to update project")
	}

	return dbToProject(&db), nil
}

// joinUpdates joins update clauses with commas.
func joinUpdates(updates []string) string {
	result := ""
	for i, u := range updates {
		if i > 0 {
			result += ", "
		}
		result += u
	}
	return result
}

//encore:api auth path=/projects/:id method=DELETE
func Delete(ctx context.Context, id string) error {
	pool, err := database.Get(ctx)
	if err != nil {
		return errs.WrapCode(err, errs.Internal, "failed to get database connection")
	}

	query := `
		UPDATE projects
		SET deleted_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := pool.Exec(ctx, query, id)
	if err != nil {
		return errs.WrapCode(err, errs.Internal, "failed to delete project")
	}

	if result.RowsAffected() == 0 {
		return &errs.Error{Code: errs.NotFound, Message: "project not found"}
	}

	return nil
}
