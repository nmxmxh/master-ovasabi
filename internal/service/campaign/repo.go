package campaign

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
	"time"

	campaignpb "github.com/nmxmxh/master-ovasabi/api/protos/campaign/v1"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	mediapb "github.com/nmxmxh/master-ovasabi/api/protos/media/v1"
	"github.com/nmxmxh/master-ovasabi/internal/repository"
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
	// master is the MasterRepository, which is a dependency for creating master entries.
	// It's part of the internal/repository package.
	master repository.MasterRepository
	// End of metadata block, do not close function here
}

// SaveBroadcastEvent persists a broadcast event for audit/recovery, including media links.
func (r *Repository) SaveBroadcastEvent(ctx context.Context, event *campaignpb.Broadcast) error {
	query := `
		INSERT INTO service_campaign_broadcast_event (
			campaign_id, timestamp, event_type, user_id, metadata, json_payload, binary_payload, media_links
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)`

	var metadataJSON []byte
	if event.Metadata != nil {
		var errMetaMarshal error
		metadataJSON, errMetaMarshal = json.Marshal(event.Metadata)
		if errMetaMarshal != nil {
			return fmt.Errorf("failed to marshal event metadata: %w", errMetaMarshal)
		}
	}

	var jsonPayload, binaryPayload interface{}
	if event.GetJsonPayload() != "" {
		jsonPayload = event.GetJsonPayload()
	}
	if len(event.GetBinaryPayload()) > 0 {
		binaryPayload = event.GetBinaryPayload()
	}

	// Persist media links as JSON array of URLs
	var mediaLinksJSON []byte
	if len(event.Media) > 0 {
		var links []string
		for _, m := range event.Media {
			if m != nil && m.Url != "" {
				links = append(links, m.Url)
			}
		}
		var errMarshal error
		mediaLinksJSON, errMarshal = json.Marshal(links)
		if errMarshal != nil {
			return fmt.Errorf("failed to marshal media links: %w", errMarshal)
		}
	}

	_, err := r.GetDB().ExecContext(ctx, query,
		event.CampaignId,
		event.Timestamp,
		event.EventType,
		event.UserId,
		metadataJSON,
		jsonPayload,
		binaryPayload,
		mediaLinksJSON,
	)
	return err
}

// ListBroadcastEvents retrieves broadcast events for a campaign (for audit/history), including media links.
func (r *Repository) ListBroadcastEvents(ctx context.Context, campaignID string, limit, offset int) ([]*campaignpb.Broadcast, error) {
	query := `
		SELECT campaign_id, timestamp, event_type, user_id, metadata, json_payload, binary_payload, media_links
		FROM service_campaign_broadcast_event
		WHERE campaign_id = $1
		ORDER BY timestamp DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.GetDB().QueryContext(ctx, query, campaignID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*campaignpb.Broadcast
	for rows.Next() {
		var (
			campaignID, timestamp, eventType, userID string
			metadataStr, jsonPayload, mediaLinksStr  string
			binaryPayload                            []byte
		)
		err := rows.Scan(&campaignID, &timestamp, &eventType, &userID, &metadataStr, &jsonPayload, &binaryPayload, &mediaLinksStr)
		if err != nil {
			return nil, err
		}
		broadcast := &campaignpb.Broadcast{
			CampaignId: campaignID,
			Timestamp:  timestamp,
			EventType:  eventType,
			UserId:     userID,
		}
		if metadataStr != "" {
			meta := &commonpb.Metadata{}
			if json.Unmarshal([]byte(metadataStr), meta) == nil {
				broadcast.Metadata = meta
			}
		}
		if jsonPayload != "" {
			broadcast.Payload = &campaignpb.Broadcast_JsonPayload{JsonPayload: jsonPayload}
		}
		if len(binaryPayload) > 0 {
			broadcast.Payload = &campaignpb.Broadcast_BinaryPayload{BinaryPayload: binaryPayload}
		}
		// Load media links from JSON array
		if mediaLinksStr != "" {
			var links []string
			if json.Unmarshal([]byte(mediaLinksStr), &links) == nil {
				for _, url := range links {
					broadcast.Media = append(broadcast.Media, &mediapb.Media{Url: url})
				}
			}
		}
		events = append(events, broadcast)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

// NewRepository creates a new campaign repository instance.
func NewRepository(db *sql.DB, log *zap.Logger, master repository.MasterRepository) *Repository {
	return &Repository{
		BaseRepository: repository.NewBaseRepository(db, log),
		master:         master,
	}
}

// CreateWithTransaction creates a new campaign within a transaction, including its master record.
func (r *Repository) CreateWithTransaction(ctx context.Context, tx *sql.Tx, campaign *Campaign) (*Campaign, error) {
	// Create master entry first, as it's a core part of creating a campaign entity.
	// This ensures every campaign is registered in the master table for cross-service orchestration.
	masterID, masterUUID, err := r.master.Create(ctx, tx, repository.EntityType("campaign"), campaign.Slug)
	if err != nil {
		return nil, fmt.Errorf("failed to create master entry for campaign: %w", err)
	}
	campaign.MasterID = masterID
	campaign.MasterUUID = masterUUID

	var metadataJSON []byte
	if campaign.Metadata != nil {
		canonicalMeta, err := CanonicalizeFromProto(campaign.Metadata, campaign.Slug)
		if err != nil {
			return nil, err
		}
		// Directly marshal the canonical Go struct to JSON.
		// This avoids the complex and error-prone round-trip back to a proto before marshaling.
		metadataJSON, err = json.Marshal(canonicalMeta)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal canonical metadata to JSON: %w", err)
		}
	}

	query := `
		INSERT INTO service_campaign_main (
			master_id, master_uuid, slug, title, description,
			ranking_formula, status, metadata, start_date, end_date,
			created_at, updated_at, owner_id
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		) RETURNING id, created_at, updated_at`

	now := time.Now()
	row := tx.QueryRowContext(ctx, query,
		campaign.MasterID,
		campaign.MasterUUID,
		campaign.Slug,
		campaign.Title,
		campaign.Description,
		campaign.RankingFormula,
		campaign.Status,
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
		// Check for unique constraint violation on slug
		if strings.Contains(err.Error(), "service_campaign_main_slug_key") || strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return nil, ErrCampaignExists
		}
		return nil, err
	}
	campaign.ID = strconv.FormatInt(id, 10)
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
			ranking_formula, status, metadata, start_date, end_date,
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
		&campaign.Status,
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

	campaign.Metadata = &commonpb.Metadata{} // Initialize metadata
	if metadataStr != "" {
		err := protojson.Unmarshal([]byte(metadataStr), campaign.Metadata)
		if err != nil {
			r.GetLogger().Warn("failed to unmarshal campaign metadata", zap.Error(err))
			return nil, err
		}
	}
	// Ensure ServiceSpecific exists and add/update campaign_id
	if campaign.Metadata.ServiceSpecific == nil {
		campaign.Metadata.ServiceSpecific = &structpb.Struct{Fields: make(map[string]*structpb.Value)}
	}
	// Convert string ID to number for campaign_id
	if campaignID, err := strconv.ParseInt(campaign.ID, 10, 64); err == nil {
		campaign.Metadata.ServiceSpecific.Fields["campaign_id"] = structpb.NewNumberValue(float64(campaignID))
	} else {
		// Fallback: use hash of ID as number
		hash := fnv.New32a()
		hash.Write([]byte(campaign.ID))
		campaign.Metadata.ServiceSpecific.Fields["campaign_id"] = structpb.NewNumberValue(float64(hash.Sum32()))
	}

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
		// Directly marshal the canonical Go struct to JSON.
		metadataJSON, err = json.Marshal(canonicalMeta)
		if err != nil {
			return fmt.Errorf("failed to marshal canonical metadata to JSON: %w", err)
		}
	}
	query := `
		UPDATE service_campaign_main SET
			title = $1,
			description = $2,
			ranking_formula = $3,
			status = $4,
			metadata = $5,
			start_date = $6,
			end_date = $7,
			updated_at = $8,
			owner_id = $9
		WHERE id = $10`

	result, err := r.GetDB().ExecContext(ctx, query,
		campaign.Title,
		campaign.Description,
		campaign.RankingFormula,
		campaign.Status,
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
			ranking_formula, status, metadata, start_date, end_date,
			created_at, updated_at, owner_id
		FROM service_campaign_main
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	// r.GetLogger().Info("Fetching campaigns from database",
	// 	zap.Int("limit", limit),
	// 	zap.Int("offset", offset),
	// 	zap.String("query", query))

	rows, err := r.GetDB().QueryContext(ctx, query, limit, offset)
	if err != nil {
		r.GetLogger().Error("Failed to query campaigns from database", zap.Error(err))
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
			&campaign.Status,
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
		campaign.Metadata = &commonpb.Metadata{} // Initialize metadata
		if metadataStr != "" {
			err := protojson.Unmarshal([]byte(metadataStr), campaign.Metadata)
			if err != nil {
				r.GetLogger().Warn("failed to unmarshal campaign metadata", zap.Error(err))
				return nil, err
			}
		}
		// Ensure ServiceSpecific exists and add/update campaign_id
		if campaign.Metadata.ServiceSpecific == nil {
			campaign.Metadata.ServiceSpecific = &structpb.Struct{Fields: make(map[string]*structpb.Value)}
		}
		// Convert UUID string to number for campaign_id
		if campaignID, err := strconv.ParseInt(campaign.ID, 10, 64); err == nil {
			campaign.Metadata.ServiceSpecific.Fields["campaign_id"] = structpb.NewNumberValue(float64(campaignID))
		} else {
			// Fallback: use hash of UUID as number
			hash := fnv.New32a()
			hash.Write([]byte(campaign.ID))
			campaign.Metadata.ServiceSpecific.Fields["campaign_id"] = structpb.NewNumberValue(float64(hash.Sum32()))
		}
		campaigns = append(campaigns, campaign)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// r.GetLogger().Info("Successfully fetched campaigns from database",
	// 	zap.Int("count", len(campaigns)),
	// 	zap.Any("campaign_names", func() []string {
	// 		names := make([]string, len(campaigns))
	// 		for i, c := range campaigns {
	// 			names[i] = c.Slug
	// 		}
	// 		return names
	// 	}()))

	return campaigns, nil
}

// ListActiveWithinWindow returns campaigns with status=active and now between start/end.
func (r *Repository) ListActiveWithinWindow(ctx context.Context, now time.Time) ([]*Campaign, error) {
	query := `
		SELECT id, master_id, master_uuid, slug, title, description, ranking_formula, status, metadata, start_date, end_date, created_at, updated_at, owner_id
		FROM service_campaign_main
		WHERE status = 'active'
		AND start_date <= $1 AND end_date >= $1
		ORDER BY created_at DESC`
	rows, err := r.GetDB().QueryContext(ctx, query, now)
	if err != nil {
		r.GetLogger().Error("Failed to query active campaigns", zap.Error(err))
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
			&campaign.Status,
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
		campaign.Metadata = &commonpb.Metadata{} // Initialize metadata
		if metadataStr != "" {
			err := protojson.Unmarshal([]byte(metadataStr), campaign.Metadata)
			if err != nil {
				r.GetLogger().Warn("failed to unmarshal campaign metadata", zap.Error(err))
				return nil, err
			}
		}
		// Ensure ServiceSpecific exists and add/update campaign_id
		if campaign.Metadata.ServiceSpecific == nil {
			campaign.Metadata.ServiceSpecific = &structpb.Struct{Fields: make(map[string]*structpb.Value)}
		}
		// Convert UUID string to number for campaign_id
		if campaignID, err := strconv.ParseInt(campaign.ID, 10, 64); err == nil {
			campaign.Metadata.ServiceSpecific.Fields["campaign_id"] = structpb.NewNumberValue(float64(campaignID))
		} else {
			// Fallback: use hash of UUID as number
			hash := fnv.New32a()
			hash.Write([]byte(campaign.ID))
			campaign.Metadata.ServiceSpecific.Fields["campaign_id"] = structpb.NewNumberValue(float64(hash.Sum32()))
		}
		campaigns = append(campaigns, campaign)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return campaigns, nil
}

// (move from shared repository types if needed).
type Campaign struct {
	ID             string `db:"id"` // UUID in database
	MasterID       int64  `db:"master_id"`
	MasterUUID     string `db:"master_uuid"`
	Slug           string `db:"slug"`  // Maps to proto slug field
	Title          string `db:"title"` // Maps to proto title field
	Description    string `db:"description"`
	RankingFormula string `db:"ranking_formula"` // Maps to proto ranking_formula field
	Status         string `db:"status"`          // Maps to proto status field
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
func (r *Repository) GetLeaderboard(ctx context.Context, campaignSlug string, limit int) ([]LeaderboardEntry, error) {
	// The SQL query fetches raw data; scoring and sorting by formula happen in Go using 'expr'.
	query := `
		SELECT u.username, COUNT(r.id) AS referral_count
		FROM service_user_master u
		LEFT JOIN service_referral_main r ON u.id = r.referrer_id AND r.campaign_slug = $1
		GROUP BY u.id
		ORDER BY referral_count DESC, u.username ASC -- Fixed order for initial fetch, actual scoring/sorting in Go
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
