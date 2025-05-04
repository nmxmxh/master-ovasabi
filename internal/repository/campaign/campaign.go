package campaign

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/nmxmxh/master-ovasabi/internal/repository"
	"go.uber.org/zap"
)

var (
	ErrCampaignNotFound = errors.New("campaign not found")
	ErrCampaignExists   = errors.New("campaign already exists")
)

var log *zap.Logger

func SetLogger(l *zap.Logger) {
	log = l
}

// Repository handles database operations for campaigns
type Repository struct {
	*repository.BaseRepository
	master repository.MasterRepository
}

// NewRepository creates a new campaign repository instance
func NewRepository(db *sql.DB, master repository.MasterRepository) *Repository {
	return &Repository{
		BaseRepository: repository.NewBaseRepository(db),
		master:         master,
	}
}

// CreateWithTransaction creates a new campaign within a transaction
func (r *Repository) CreateWithTransaction(ctx context.Context, tx *sql.Tx, campaign *Campaign) (*Campaign, error) {
	query := `
		INSERT INTO service_campaign (
			master_id, slug, title, description,
			ranking_formula, metadata, start_date, end_date,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		) RETURNING id, created_at, updated_at`

	now := time.Now()
	err := tx.QueryRowContext(ctx, query,
		campaign.MasterID,
		campaign.Slug,
		campaign.Title,
		campaign.Description,
		campaign.RankingFormula,
		campaign.Metadata,
		campaign.StartDate,
		campaign.EndDate,
		now,
		now,
	).Scan(&campaign.ID, &campaign.CreatedAt, &campaign.UpdatedAt)

	if err != nil {
		if err.Error() == "pq: duplicate key value violates unique constraint" {
			return nil, ErrCampaignExists
		}
		return nil, err
	}

	return campaign, nil
}

// GetBySlug retrieves a campaign by its slug
func (r *Repository) GetBySlug(ctx context.Context, slug string) (*Campaign, error) {
	campaign := &Campaign{}
	query := `
		SELECT 
			id, master_id, slug, title, description,
			ranking_formula, metadata, start_date, end_date,
			created_at, updated_at
		FROM service_campaign
		WHERE slug = $1`

	err := r.GetDB().QueryRowContext(ctx, query, slug).Scan(
		&campaign.ID,
		&campaign.MasterID,
		&campaign.Slug,
		&campaign.Title,
		&campaign.Description,
		&campaign.RankingFormula,
		&campaign.Metadata,
		&campaign.StartDate,
		&campaign.EndDate,
		&campaign.CreatedAt,
		&campaign.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrCampaignNotFound
	}

	if err != nil {
		return nil, err
	}

	return campaign, nil
}

// Update updates an existing campaign
func (r *Repository) Update(ctx context.Context, campaign *Campaign) error {
	query := `
		UPDATE service_campaign SET
			title = $1,
			description = $2,
			ranking_formula = $3,
			metadata = $4,
			start_date = $5,
			end_date = $6,
			updated_at = $7
		WHERE id = $8`

	result, err := r.GetDB().ExecContext(ctx, query,
		campaign.Title,
		campaign.Description,
		campaign.RankingFormula,
		campaign.Metadata,
		campaign.StartDate,
		campaign.EndDate,
		time.Now(),
		campaign.ID,
	)

	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return ErrCampaignNotFound
	}

	return nil
}

// Delete deletes a campaign by ID
func (r *Repository) Delete(ctx context.Context, id int64) error {
	result, err := r.GetDB().ExecContext(ctx,
		"DELETE FROM service_campaign WHERE id = $1",
		id,
	)

	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return ErrCampaignNotFound
	}

	return nil
}

// List retrieves a paginated list of campaigns
func (r *Repository) List(ctx context.Context, limit, offset int) ([]*Campaign, error) {
	query := `
		SELECT 
			id, master_id, slug, title, description,
			ranking_formula, metadata, start_date, end_date,
			created_at, updated_at
		FROM service_campaign
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.GetDB().QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			fmt.Printf("error closing rows: %v\n", err)
		}
	}()

	var campaigns []*Campaign
	for rows.Next() {
		campaign := &Campaign{}
		err := rows.Scan(
			&campaign.ID,
			&campaign.MasterID,
			&campaign.Slug,
			&campaign.Title,
			&campaign.Description,
			&campaign.RankingFormula,
			&campaign.Metadata,
			&campaign.StartDate,
			&campaign.EndDate,
			&campaign.CreatedAt,
			&campaign.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		campaigns = append(campaigns, campaign)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return campaigns, nil
}

// Define the Campaign struct here
// Campaign represents a campaign entity
// (move from shared repository types if needed)
type Campaign struct {
	ID             int64     `db:"id"`
	MasterID       int64     `db:"master_id"`
	Slug           string    `db:"slug"`
	Title          string    `db:"title"`
	Description    string    `db:"description"`
	RankingFormula string    `db:"ranking_formula"`
	Metadata       string    `db:"metadata"`
	StartDate      time.Time `db:"start_date"`
	EndDate        time.Time `db:"end_date"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}

// LeaderboardEntry represents a leaderboard row
// (move to a shared types file if needed)
type LeaderboardEntry struct {
	Username      string
	ReferralCount int
}

// RankingFormula represents a parsed and validated ranking formula
// Supports multiple columns and directions, and can be extended for expressions
// Example: "referral_count DESC, username ASC"
type RankingFormula struct {
	Columns []RankingColumn
}

type RankingColumn struct {
	Name      string
	Direction string // "ASC" or "DESC"
}

var allowedColumns = map[string]bool{
	"referral_count": true,
	"username":       true,
}
var allowedDirections = map[string]bool{
	"ASC":  true,
	"DESC": true,
}

// validateRankingFormula parses and validates a ranking formula string for safety
func validateRankingFormula(formula string) (*RankingFormula, error) {
	formula = strings.TrimSpace(formula)
	if formula == "" {
		return nil, errors.New("empty ranking formula")
	}
	columns := strings.Split(formula, ",")
	var parsed []RankingColumn
	for _, col := range columns {
		col = strings.TrimSpace(col)
		// Use regex to match: column_name [ASC|DESC]
		re := regexp.MustCompile(`^([a-zA-Z0-9_]+)(?:\s+(ASC|DESC))?$`)
		matches := re.FindStringSubmatch(col)
		if len(matches) == 0 {
			return nil, errors.New("invalid ranking formula syntax")
		}
		name := matches[1]
		dir := "DESC" // Default direction
		if len(matches) > 2 && matches[2] != "" {
			dir = strings.ToUpper(matches[2])
		}
		if !allowedColumns[name] {
			return nil, errors.New("column " + name + " not allowed in ranking formula")
		}
		if !allowedDirections[dir] {
			return nil, errors.New("direction " + dir + " not allowed in ranking formula")
		}
		parsed = append(parsed, RankingColumn{Name: name, Direction: dir})
	}
	return &RankingFormula{Columns: parsed}, nil
}

// ToSQL returns the SQL ORDER BY clause for the validated formula
func (rf *RankingFormula) ToSQL() string {
	var parts []string
	for _, col := range rf.Columns {
		parts = append(parts, col.Name+" "+col.Direction)
	}
	return strings.Join(parts, ", ")
}

// GetLeaderboard returns the leaderboard for a campaign, applying the ranking formula.
func (r *Repository) GetLeaderboard(ctx context.Context, campaignSlug, rankingFormula string, limit int) ([]LeaderboardEntry, error) {
	rf, err := validateRankingFormula(rankingFormula)
	if err != nil {
		return nil, err
	}
	orderBy := rf.ToSQL()
	query := `
		SELECT u.username, COUNT(r.id) AS referral_count
		FROM users u
		LEFT JOIN referrals r ON u.id = r.referrer_id AND r.campaign_slug = $1
		GROUP BY u.id
		ORDER BY ` + orderBy + `
		LIMIT $2
	`
	rows, err := r.GetDB().QueryContext(ctx, query, campaignSlug, limit)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			fmt.Printf("error closing rows: %v\n", err)
		}
	}()

	var leaderboard []LeaderboardEntry
	for rows.Next() {
		var entry LeaderboardEntry
		if err := rows.Scan(&entry.Username, &entry.ReferralCount); err != nil {
			return nil, err
		}
		leaderboard = append(leaderboard, entry)
	}
	return leaderboard, nil
}
