// Package database provides a connection pool to the Supabase PostgreSQL database.
package database

import (
	"time"
)

// Project represents a project entry in the database.
type Project struct {
	ID           string     `json:"id" db:"id"`
	Title        string     `json:"title" db:"title"`
	Description  string     `json:"description" db:"description"`
	Href         *string    `json:"href" db:"href"`
	Technologies []string   `json:"technologies" db:"technologies"`
	DisplayOrder int        `json:"display_order" db:"display_order"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at" db:"deleted_at"`
}

// Hero represents the hero section content (single row).
type Hero struct {
	ID          string    `json:"id" db:"id"`
	Phrases     []string  `json:"phrases" db:"phrases"`
	Description string    `json:"description" db:"description"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// WorkExperience represents a work experience entry.
type WorkExperience struct {
	ID           string     `json:"id" db:"id"`
	Company      string     `json:"company" db:"company"`
	Position     string     `json:"position" db:"position"`
	Date         string     `json:"date" db:"date"`
	Description  string     `json:"description" db:"description"`
	Href         *string    `json:"href" db:"href"`
	Type         *string    `json:"type" db:"type"`
	DisplayOrder int        `json:"display_order" db:"display_order"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at" db:"deleted_at"`
}
