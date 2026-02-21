// Package work provides CRUD operations for work experience content.
package work

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
	// WorkExperience represents a work experience entry in API responses.
	WorkExperience struct {
		ID           string  `json:"id"`
		Company      string  `json:"company"`
		Position     string  `json:"position"`
		Date         string  `json:"date"`
		Description  string  `json:"description"`
		Href         *string `json:"href,omitempty"`
		Type         *string `json:"type,omitempty"`
		DisplayOrder int     `json:"display_order"`
		CreatedAt    string  `json:"created_at"`
		UpdatedAt    string  `json:"updated_at"`
	}

	// ListResponse is the response for listing work experiences.
	ListResponse struct {
		WorkExperiences []*WorkExperience `json:"work_experiences"`
	}

	// CreateRequest is the request body for creating a work experience.
	CreateRequest struct {
		Company      string  `json:"company"`
		Position     string  `json:"position"`
		Date         string  `json:"date"`
		Description  string  `json:"description"`
		Href         *string `json:"href,omitempty"`
		Type         *string `json:"type,omitempty"`
		DisplayOrder int     `json:"display_order"`
	}

	// UpdateRequest is the request body for updating a work experience.
	UpdateRequest struct {
		ID           string  `json:"-"`
		Company      *string `json:"company,omitempty"`
		Position     *string `json:"position,omitempty"`
		Date         *string `json:"date,omitempty"`
		Description  *string `json:"description,omitempty"`
		Href         *string `json:"href,omitempty"`
		Type         *string `json:"type,omitempty"`
		DisplayOrder *int    `json:"display_order,omitempty"`
	}
)

// validWorkTypes are the allowed values for work type.
var validWorkTypes = map[string]bool{
	"Freelance":  true,
	"Internship": true,
	"Contract":   true,
	"Part-Time":  true,
	"Full-Time":  true,
}

// dbToWork converts a database WorkExperience to API WorkExperience.
func dbToWork(db *database.WorkExperience) *WorkExperience {
	return &WorkExperience{
		ID:           db.ID,
		Company:      db.Company,
		Position:     db.Position,
		Date:         db.Date,
		Description:  db.Description,
		Href:         db.Href,
		Type:         db.Type,
		DisplayOrder: db.DisplayOrder,
		CreatedAt:    db.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    db.UpdatedAt.Format(time.RFC3339),
	}
}

//encore:api auth path=/work method=GET
func List(ctx context.Context) (*ListResponse, error) {
	pool, err := database.Get(ctx)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to get database connection")
	}

	query := `
		SELECT id, company, position, date, description, href, type, display_order, created_at, updated_at
		FROM work_experience
		WHERE deleted_at IS NULL
		ORDER BY display_order ASC, created_at DESC
	`

	rows, err := pool.Query(ctx, query)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to query work experiences")
	}
	defer rows.Close()

	var workExperiences []*WorkExperience
	for rows.Next() {
		var db database.WorkExperience
		err := rows.Scan(
			&db.ID,
			&db.Company,
			&db.Position,
			&db.Date,
			&db.Description,
			&db.Href,
			&db.Type,
			&db.DisplayOrder,
			&db.CreatedAt,
			&db.UpdatedAt,
		)
		if err != nil {
			return nil, errs.WrapCode(err, errs.Internal, "failed to scan work experience row")
		}
		workExperiences = append(workExperiences, dbToWork(&db))
	}

	if err := rows.Err(); err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "error iterating work experience rows")
	}

	if workExperiences == nil {
		workExperiences = []*WorkExperience{}
	}

	return &ListResponse{WorkExperiences: workExperiences}, nil
}

//encore:api auth path=/work method=POST
func Create(ctx context.Context, req *CreateRequest) (*WorkExperience, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	pool, err := database.Get(ctx)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to get database connection")
	}

	query := `
		INSERT INTO work_experience (company, position, date, description, href, type, display_order)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, company, position, date, description, href, type, display_order, created_at, updated_at
	`

	var db database.WorkExperience
	err = pool.QueryRow(ctx, query,
		req.Company,
		req.Position,
		req.Date,
		req.Description,
		req.Href,
		req.Type,
		req.DisplayOrder,
	).Scan(
		&db.ID,
		&db.Company,
		&db.Position,
		&db.Date,
		&db.Description,
		&db.Href,
		&db.Type,
		&db.DisplayOrder,
		&db.CreatedAt,
		&db.UpdatedAt,
	)

	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to create work experience")
	}

	return dbToWork(&db), nil
}

func (r *CreateRequest) Validate() error {
	if r.Company == "" {
		return &errs.Error{Code: errs.InvalidArgument, Message: "company is required"}
	}
	if r.Position == "" {
		return &errs.Error{Code: errs.InvalidArgument, Message: "position is required"}
	}
	if r.Date == "" {
		return &errs.Error{Code: errs.InvalidArgument, Message: "date is required"}
	}
	if r.Description == "" {
		return &errs.Error{Code: errs.InvalidArgument, Message: "description is required"}
	}
	if r.Type != nil && !validWorkTypes[*r.Type] {
		return &errs.Error{Code: errs.InvalidArgument, Message: "invalid work type"}
	}
	return nil
}

//encore:api auth path=/work/:id method=PUT
func Update(ctx context.Context, id string, req *UpdateRequest) (*WorkExperience, error) {
	req.ID = id

	pool, err := database.Get(ctx)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to get database connection")
	}

	// Validate type if provided
	if req.Type != nil && !validWorkTypes[*req.Type] {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "invalid work type"}
	}

	// Build dynamic update query
	updates := []string{}
	args := []interface{}{}
	argNum := 1

	if req.Company != nil {
		updates = append(updates, fmt.Sprintf("company = $%d", argNum))
		args = append(args, *req.Company)
		argNum++
	}
	if req.Position != nil {
		updates = append(updates, fmt.Sprintf("position = $%d", argNum))
		args = append(args, *req.Position)
		argNum++
	}
	if req.Date != nil {
		updates = append(updates, fmt.Sprintf("date = $%d", argNum))
		args = append(args, *req.Date)
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
	if req.Type != nil {
		updates = append(updates, fmt.Sprintf("type = $%d", argNum))
		args = append(args, *req.Type)
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

	args = append(args, req.ID)

	query := fmt.Sprintf(`
		UPDATE work_experience
		SET %s
		WHERE id = $%d AND deleted_at IS NULL
		RETURNING id, company, position, date, description, href, type, display_order, created_at, updated_at
	`, joinUpdates(updates), argNum)

	var db database.WorkExperience
	err = pool.QueryRow(ctx, query, args...).Scan(
		&db.ID,
		&db.Company,
		&db.Position,
		&db.Date,
		&db.Description,
		&db.Href,
		&db.Type,
		&db.DisplayOrder,
		&db.CreatedAt,
		&db.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, &errs.Error{Code: errs.NotFound, Message: "work experience not found"}
	}
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to update work experience")
	}

	return dbToWork(&db), nil
}

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

//encore:api auth path=/work/:id method=DELETE
func Delete(ctx context.Context, id string) error {
	pool, err := database.Get(ctx)
	if err != nil {
		return errs.WrapCode(err, errs.Internal, "failed to get database connection")
	}

	query := `
		UPDATE work_experience
		SET deleted_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := pool.Exec(ctx, query, id)
	if err != nil {
		return errs.WrapCode(err, errs.Internal, "failed to delete work experience")
	}

	if result.RowsAffected() == 0 {
		return &errs.Error{Code: errs.NotFound, Message: "work experience not found"}
	}

	return nil
}
