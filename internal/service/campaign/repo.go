package campaign

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"strings"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"github.com/nmxmxh/master-ovasabi/internal/repository"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

var (
	ErrCampaignNotFound = errors.New("campaign not found")
	ErrCampaignExists   = errors.New("campaign already exists")
)

// Repository handles database operations for campaigns.
type Repository struct {
	*repository.BaseRepository
	master repository.MasterRepository
}

// NewRepository creates a new campaign repository instance.
func NewRepository(db *sql.DB, log *zap.Logger, master repository.MasterRepository) *Repository {
	return &Repository{
		BaseRepository: repository.NewBaseRepository(db, log),
		master:         master,
	}
}

// CreateWithTransaction creates a new campaign within a transaction.
func (r *Repository) CreateWithTransaction(ctx context.Context, tx *sql.Tx, campaign *Campaign) (*Campaign, error) {
	var metadataJSON []byte
	var err error
	if campaign.Metadata != nil {
		canonicalMeta, err := CanonicalizeFromProto(campaign.Metadata, campaign.Slug)
		if err != nil {
			return nil, err
		}
		metadataJSON, err = protojson.Marshal(ToProto(canonicalMeta))
		if err != nil {
			return nil, err
		}
	}
	query := `
		INSERT INTO service_campaign_main (
			master_id, master_uuid, slug, title, description,
			ranking_formula, metadata, start_date, end_date,
			created_at, updated_at, owner_id
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		) RETURNING id, created_at, updated_at`

	now := time.Now()
	row := tx.QueryRowContext(ctx, query,
		campaign.MasterID,
		campaign.MasterUUID,
		campaign.Slug,
		campaign.Title,
		campaign.Description,
		campaign.RankingFormula,
		metadataJSON,
		campaign.StartDate,
		campaign.EndDate,
		now,
		now,
		campaign.OwnerID,
	)
	var id int64
	var createdAt, updatedAt time.Time
	err = row.Scan(&id, &createdAt, &updatedAt)
	if err != nil {
		if err.Error() == "pq: duplicate key value violates unique constraint" {
			return nil, ErrCampaignExists
		}
		return nil, err
	}
	campaign.ID = id
	campaign.CreatedAt = createdAt
	campaign.UpdatedAt = updatedAt

	return campaign, nil
}

// GetBySlug retrieves a campaign by its slug.
func (r *Repository) GetBySlug(ctx context.Context, slug string) (*Campaign, error) {
	campaign := &Campaign{}
	query := `
		SELECT 
			id, master_id, master_uuid, slug, title, description,
			ranking_formula, metadata, start_date, end_date,
			created_at, updated_at, owner_id
		FROM service_campaign_main
		WHERE slug = $1`

	var metadataStr string
	err := r.GetDB().QueryRowContext(ctx, query, slug).Scan(
		&campaign.ID,
		&campaign.MasterID,
		&campaign.MasterUUID,
		&campaign.Slug,
		&campaign.Title,
		&campaign.Description,
		&campaign.RankingFormula,
		&metadataStr,
		&campaign.StartDate,
		&campaign.EndDate,
		&campaign.CreatedAt,
		&campaign.UpdatedAt,
		&campaign.OwnerID,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCampaignNotFound
	}

	if err != nil {
		return nil, err
	}

	// Always set serviceSpecific to campaign_id only, as err is always nil here
	serviceSpecific := map[string]interface{}{"campaign_id": campaign.ID}
	campaign.Metadata = &commonpb.Metadata{
		ServiceSpecific: metadata.NewStructFromMap(serviceSpecific, nil),
		Tags:            []string{},
		Features:        []string{},
	}
	if metadataStr != "" {
		err := protojson.Unmarshal([]byte(metadataStr), campaign.Metadata)
		if err != nil {
			r.GetLogger().Warn("failed to unmarshal campaign metadata", zap.Error(err))
			return nil, err
		}
	}
	// Canonicalize and validate metadata
	canonicalMeta, err := CanonicalizeFromProto(campaign.Metadata, campaign.Slug)
	if err != nil {
		r.GetLogger().Warn("campaign metadata is invalid", zap.Error(err))
		return nil, err
	}
	campaign.Metadata = ToProto(canonicalMeta)

	return campaign, nil
}

// Update updates an existing campaign.
func (r *Repository) Update(ctx context.Context, campaign *Campaign) error {
	var metadataJSON []byte
	var err error
	if campaign.Metadata != nil {
		canonicalMeta, err := CanonicalizeFromProto(campaign.Metadata, campaign.Slug)
		if err != nil {
			return err
		}
		metadataJSON, err = protojson.Marshal(ToProto(canonicalMeta))
		if err != nil {
			return err
		}
	}
	query := `
		UPDATE service_campaign_main SET
			title = $1,
			description = $2,
			ranking_formula = $3,
			metadata = $4,
			start_date = $5,
			end_date = $6,
			updated_at = $7,
			owner_id = $8
		WHERE id = $9`

	result, err := r.GetDB().ExecContext(ctx, query,
		campaign.Title,
		campaign.Description,
		campaign.RankingFormula,
		metadataJSON,
		campaign.StartDate,
		campaign.EndDate,
		time.Now(),
		campaign.OwnerID,
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

// Delete deletes a campaign by ID.
func (r *Repository) Delete(ctx context.Context, id int64) error {
	result, err := r.GetDB().ExecContext(ctx,
		"DELETE FROM service_campaign_main WHERE id = $1",
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

// List retrieves a paginated list of campaigns.
func (r *Repository) List(ctx context.Context, limit, offset int) ([]*Campaign, error) {
	query := `
		SELECT 
			id, master_id, master_uuid, slug, title, description,
			ranking_formula, metadata, start_date, end_date,
			created_at, updated_at, owner_id
		FROM service_campaign_main
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.GetDB().QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.GetLogger().Warn("error closing rows", zap.Error(err))
		}
	}()

	var campaigns []*Campaign
	for rows.Next() {
		campaign := &Campaign{}
		var metadataStr string
		err := rows.Scan(
			&campaign.ID,
			&campaign.MasterID,
			&campaign.MasterUUID,
			&campaign.Slug,
			&campaign.Title,
			&campaign.Description,
			&campaign.RankingFormula,
			&metadataStr,
			&campaign.StartDate,
			&campaign.EndDate,
			&campaign.CreatedAt,
			&campaign.UpdatedAt,
			&campaign.OwnerID,
		)
		if err != nil {
			return nil, err
		}
		// Always set serviceSpecific to campaign_id only, as err is always nil here
		serviceSpecific := map[string]interface{}{"campaign_id": campaign.ID}
		campaign.Metadata = &commonpb.Metadata{
			ServiceSpecific: metadata.NewStructFromMap(serviceSpecific, nil),
			Tags:            []string{},
			Features:        []string{},
		}
		if metadataStr != "" {
			err := protojson.Unmarshal([]byte(metadataStr), campaign.Metadata)
			if err != nil {
				r.GetLogger().Warn("failed to unmarshal campaign metadata", zap.Error(err))
				return nil, err
			}
		}
		// Canonicalize and validate metadata
		canonicalMeta, err := CanonicalizeFromProto(campaign.Metadata, campaign.Slug)
		if err != nil {
			r.GetLogger().Warn("campaign metadata is invalid", zap.Error(err))
			return nil, err
		}
		campaign.Metadata = ToProto(canonicalMeta)
		campaigns = append(campaigns, campaign)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return campaigns, nil
}

// ListActiveWithinWindow returns campaigns with status=active and now between start/end.
func (r *Repository) ListActiveWithinWindow(ctx context.Context, now time.Time) ([]*Campaign, error) {
	query := `
		SELECT id, master_id, master_uuid, slug, title, description, ranking_formula, metadata, start_date, end_date, created_at, updated_at
		FROM service_campaign_main
		WHERE (metadata->'service_specific'->'campaign'->>'status') = 'active'
		AND start_date <= $1 AND end_date >= $1
		ORDER BY created_at DESC`
	rows, err := r.GetDB().QueryContext(ctx, query, now)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.GetLogger().Warn("error closing rows", zap.Error(err))
		}
	}()

	var campaigns []*Campaign
	for rows.Next() {
		campaign := &Campaign{}
		var metadataStr string
		err := rows.Scan(
			&campaign.ID,
			&campaign.MasterID,
			&campaign.MasterUUID,
			&campaign.Slug,
			&campaign.Title,
			&campaign.Description,
			&campaign.RankingFormula,
			&metadataStr,
			&campaign.StartDate,
			&campaign.EndDate,
			&campaign.CreatedAt,
			&campaign.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		// Always set serviceSpecific to campaign_id only, as err is always nil here
		serviceSpecific := map[string]interface{}{"campaign_id": campaign.ID}
		campaign.Metadata = &commonpb.Metadata{
			ServiceSpecific: metadata.NewStructFromMap(serviceSpecific, nil),
			Tags:            []string{},
			Features:        []string{},
		}
		if metadataStr != "" {
			err := protojson.Unmarshal([]byte(metadataStr), campaign.Metadata)
			if err != nil {
				r.GetLogger().Warn("failed to unmarshal campaign metadata", zap.Error(err))
				return nil, err
			}
		}
		// Canonicalize and validate metadata
		canonicalMeta, err := CanonicalizeFromProto(campaign.Metadata, campaign.Slug)
		if err != nil {
			r.GetLogger().Warn("campaign metadata is invalid", zap.Error(err))
			return nil, err
		}
		campaign.Metadata = ToProto(canonicalMeta)
		campaigns = append(campaigns, campaign)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return campaigns, nil
}

// (move from shared repository types if needed).
type Campaign struct {
	ID             int64  `db:"id"`
	MasterID       int64  `db:"master_id"`
	MasterUUID     string `db:"master_uuid"`
	Slug           string `db:"slug"`
	Title          string `db:"title"`
	Description    string `db:"description"`
	RankingFormula string `db:"ranking_formula"`
	OwnerID        string `db:"owner_id" json:"owner_id"`
	Metadata       *commonpb.Metadata
	StartDate      time.Time `db:"start_date"`
	EndDate        time.Time `db:"end_date"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}

// LeaderboardEntry represents a single entry in the campaign leaderboard.
type LeaderboardEntry struct {
	Username      string
	ReferralCount int
	Variables     map[string]interface{}
	Score         float64
	Metadata      *commonpb.Metadata
}

// Example: "referral_count DESC, username ASC".
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

// validateRankingFormula parses and validates a ranking formula string for safety.
func validateRankingFormula(formula string) (*RankingFormula, error) {
	formula = strings.TrimSpace(formula)
	if formula == "" {
		return nil, errors.New("empty ranking formula")
	}
	columns := strings.Split(formula, ",")
	parsed := make([]RankingColumn, 0, len(columns))
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

// ToSQL returns the SQL ORDER BY clause for the validated formula.
func (rf *RankingFormula) ToSQL() string {
	parts := make([]string, 0, len(rf.Columns))
	for _, col := range rf.Columns {
		parts = append(parts, col.Name+" "+col.Direction)
	}
	return strings.Join(parts, ", ")
}

// FlattenMetadataToVars extracts primitive fields from campaign metadata into the variables map.
func FlattenMetadataToVars(meta *commonpb.Metadata, vars map[string]interface{}) {
	if meta == nil {
		return
	}
	// Example: flatten service_specific.campaign fields
	if meta.ServiceSpecific != nil {
		if campaignField, ok := meta.ServiceSpecific.Fields["campaign"]; ok {
			if campaignStruct := campaignField.GetStructValue(); campaignStruct != nil {
				for k, v := range campaignStruct.Fields {
					switch v.Kind.(type) {
					case *structpb.Value_NumberValue:
						vars[k] = v.GetNumberValue()
					case *structpb.Value_StringValue:
						vars[k] = v.GetStringValue()
					case *structpb.Value_BoolValue:
						vars[k] = v.GetBoolValue()
					}
				}
			}
		}
	}
	// Add more flattening as needed (e.g., audit, tags, etc.)
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
		FROM service_user_master u
		LEFT JOIN service_referral_main r ON u.id = r.referrer_id AND r.campaign_slug = $1
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
			r.GetLogger().Warn("error closing rows", zap.Error(err))
		}
	}()

	var leaderboard []LeaderboardEntry
	for rows.Next() {
		var entry LeaderboardEntry
		if err := rows.Scan(&entry.Username, &entry.ReferralCount); err != nil {
			return nil, err
		}
		entry.Variables = map[string]interface{}{"username": entry.Username, "referral_count": entry.ReferralCount}
		FlattenMetadataToVars(entry.Metadata, entry.Variables)
		leaderboard = append(leaderboard, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return leaderboard, nil
}
