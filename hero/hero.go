// Package hero provides operations for the hero section content.
package hero

import (
	"context"
	"time"

	"encore.app/pkg/database"
	"encore.dev/beta/errs"
)

// Hero represents the hero section content.
type Hero struct {
	ID          string   `json:"id"`
	Phrases     []string `json:"phrases"`
	Description string   `json:"description"`
	UpdatedAt   string   `json:"updated_at"`
}

// UpdateRequest is the request body for updating hero content.
type UpdateRequest struct {
	Phrases     []string `json:"phrases"`
	Description string   `json:"description"`
}

// dbToHero converts a database Hero to API Hero.
func dbToHero(db *database.Hero) *Hero {
	return &Hero{
		ID:          db.ID,
		Phrases:     db.Phrases,
		Description: db.Description,
		UpdatedAt:   db.UpdatedAt.Format(time.RFC3339),
	}
}

//encore:api public path=/hero method=GET
func Get(ctx context.Context) (*Hero, error) {
	pool, err := database.Get(ctx)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to get database connection")
	}

	query := `
		SELECT id, phrases, description, updated_at
		FROM hero
		WHERE id = '00000000-0000-0000-0000-000000000001'::UUID
	`

	var db database.Hero

	err = pool.QueryRow(ctx, query).Scan(
		&db.ID,
		&db.Phrases,
		&db.Description,
		&db.UpdatedAt,
	)

	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to get hero content")
	}

	return dbToHero(&db), nil
}

//encore:api auth path=/hero method=PUT
func Update(ctx context.Context, req *UpdateRequest) (*Hero, error) {
	if req.Description == "" {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "description is required"}
	}
	if req.Phrases == nil {
		req.Phrases = []string{}
	}

	pool, err := database.Get(ctx)
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to get database connection")
	}

	query := `
		UPDATE hero
		SET phrases = $1, description = $2, updated_at = NOW()
		WHERE id = '00000000-0000-0000-0000-000000000001'::UUID
		RETURNING id, phrases, description, updated_at
	`

	var db database.Hero

	err = pool.QueryRow(ctx, query, req.Phrases, req.Description).Scan(
		&db.ID,
		&db.Phrases,
		&db.Description,
		&db.UpdatedAt,
	)

	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "failed to update hero content")
	}

	return dbToHero(&db), nil
}
