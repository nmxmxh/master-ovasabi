package repository

import (
	"context"
	"database/sql"
	"time"
)

// DB interface for database operations (for mocking/testing).
type DB interface {
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

type Campaign struct {
	ID             int
	MasterID       int
	Slug           string
	Title          string
	Description    string
	RankingFormula string
	StartDate      *time.Time
	EndDate        *time.Time
	Metadata       string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type CampaignRepository struct {
	db DB
}

func NewCampaignRepository(db DB) *CampaignRepository {
	return &CampaignRepository{db: db}
}

func (r *CampaignRepository) Create(ctx context.Context, c *Campaign) (*Campaign, error) {
	query := `INSERT INTO service_campaign (master_id, slug, title, description, ranking_formula, start_date, end_date, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW()) RETURNING id, created_at, updated_at`
	row := r.db.QueryRowContext(ctx, query, c.MasterID, c.Slug, c.Title, c.Description, c.RankingFormula, c.StartDate, c.EndDate, c.Metadata)
	var created Campaign
	if err := row.Scan(&created.ID, &created.CreatedAt, &created.UpdatedAt); err != nil {
		return nil, err
	}
	created.MasterID = c.MasterID
	created.Slug = c.Slug
	created.Title = c.Title
	created.Description = c.Description
	created.RankingFormula = c.RankingFormula
	created.StartDate = c.StartDate
	created.EndDate = c.EndDate
	created.Metadata = c.Metadata
	return &created, nil
}

func (r *CampaignRepository) GetBySlug(ctx context.Context, slug string) (*Campaign, error) {
	query := `SELECT id, master_id, slug, title, description, ranking_formula, start_date, end_date, metadata, created_at, updated_at FROM service_campaign WHERE slug = $1`
	row := r.db.QueryRowContext(ctx, query, slug)
	var c Campaign
	if err := row.Scan(&c.ID, &c.MasterID, &c.Slug, &c.Title, &c.Description, &c.RankingFormula, &c.StartDate, &c.EndDate, &c.Metadata, &c.CreatedAt, &c.UpdatedAt); err != nil {
		return nil, err
	}
	return &c, nil
}
