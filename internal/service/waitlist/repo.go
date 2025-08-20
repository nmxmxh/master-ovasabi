package waitlist

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/lib/pq"
	waitlistpb "github.com/nmxmxh/master-ovasabi/api/protos/waitlist/v1"
	repositorypkg "github.com/nmxmxh/master-ovasabi/internal/repository"
	"go.uber.org/zap"
)

// Repository interface for waitlist data access.
type Repository interface {
	// Create a new waitlist entry
	Create(ctx context.Context, entry *waitlistpb.WaitlistEntry) (*waitlistpb.WaitlistEntry, error)
	// Update an existing waitlist entry
	Update(ctx context.Context, entry *waitlistpb.WaitlistEntry) (*waitlistpb.WaitlistEntry, error)
	// Get waitlist entry by ID
	GetByID(ctx context.Context, id int64) (*waitlistpb.WaitlistEntry, error)
	// Get waitlist entry by UUID
	GetByUUID(ctx context.Context, uuid string) (*waitlistpb.WaitlistEntry, error)
	// Get waitlist entry by email
	GetByEmail(ctx context.Context, email string) (*waitlistpb.WaitlistEntry, error)
	// List waitlist entries with pagination and filters
	List(ctx context.Context, limit, offset int, tierFilter, statusFilter, campaignFilter string) ([]*waitlistpb.WaitlistEntry, int64, error)
	// Check if email exists
	EmailExists(ctx context.Context, email string) (bool, error)
	// Check if username is reserved
	UsernameExists(ctx context.Context, username string) (bool, error)
	// Update status by ID
	UpdateStatus(ctx context.Context, id int64, status string) error
	// Update priority score by ID
	UpdatePriorityScore(ctx context.Context, id int64, score int) error
	// Get waitlist statistics
	GetStats(ctx context.Context, campaign string) (*waitlistpb.WaitlistStats, error)
	// Get user's waitlist position
	GetWaitlistPosition(ctx context.Context, id int64) (int, error)

	// Campaign-specific methods
	GetLeaderboard(ctx context.Context, limit int, campaign string) ([]*waitlistpb.LeaderboardEntry, error)
	GetReferralsByUser(ctx context.Context, userID int64) ([]*waitlistpb.ReferralRecord, error)
	GetLocationStats(ctx context.Context, campaign string) ([]*waitlistpb.LocationStat, error)
	ValidateReferralUsername(ctx context.Context, username string) (bool, error)
	CreateReferralRecord(ctx context.Context, referrerUsername string, referredID int64) error
}

// repository implements the Repository interface.
type repository struct {
	db         *sql.DB
	logger     *zap.Logger
	masterRepo repositorypkg.MasterRepository
}

// NewRepository creates a new waitlist repository.
func NewRepository(db *sql.DB, logger *zap.Logger, masterRepo repositorypkg.MasterRepository) Repository {
	return &repository{
		db:         db,
		logger:     logger,
		masterRepo: masterRepo,
	}
}

// Create creates a new waitlist entry.
func (r *repository) Create(ctx context.Context, entry *waitlistpb.WaitlistEntry) (*waitlistpb.WaitlistEntry, error) {
	questionnaireJSON, err := json.Marshal(structToMap(entry.QuestionnaireAnswers))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal questionnaire answers: %w", err)
	}

	contactPrefsJSON, err := json.Marshal(structToMap(entry.ContactPreferences))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal contact preferences: %w", err)
	}

	metadataJSON, err := json.Marshal(metadataToMap(entry.Metadata))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO service_waitlist_main (
			uuid, master_id, master_uuid, email, first_name, last_name, tier,
			reserved_username, intention, questionnaire_answers, interests,
			referral_username, referral_code, feedback, additional_comments,
			status, priority_score, contact_preferences, metadata,
			campaign_name, location_country, location_region, location_city,
			location_lat, location_lng, ip_address, user_agent, referrer_url,
			utm_source, utm_medium, utm_campaign, utm_term, utm_content
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19,
			$20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33
		) RETURNING id, created_at, updated_at`

	// For now, we'll use dummy values for master_id and master_uuid
	masterID := int64(1)
	masterUUID := uuid.New()

	var id int64
	var createdAt, updatedAt sql.NullTime

	err = r.db.QueryRowContext(ctx, query,
		entry.Uuid,
		masterID,
		masterUUID.String(),
		entry.Email,
		entry.FirstName,
		entry.LastName,
		entry.Tier,
		entry.ReservedUsername,
		entry.Intention,
		questionnaireJSON,
		pq.Array(entry.Interests),
		entry.ReferralUsername,
		entry.ReferralCode,
		entry.Feedback,
		entry.AdditionalComments,
		entry.Status,
		entry.PriorityScore,
		contactPrefsJSON,
		metadataJSON,
		// Campaign fields
		entry.CampaignName,
		entry.LocationCountry,
		entry.LocationRegion,
		entry.LocationCity,
		entry.LocationLat,
		entry.LocationLng,
		entry.IpAddress,
		entry.UserAgent,
		entry.ReferrerUrl,
		entry.UtmSource,
		entry.UtmMedium,
		entry.UtmCampaign,
		entry.UtmTerm,
		entry.UtmContent,
	).Scan(&id, &createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create waitlist entry: %w", err)
	}

	entry.Id = id
	entry.MasterId = masterID
	entry.MasterUuid = masterUUID.String()
	if createdAt.Valid {
		entry.CreatedAt = timestampProto(createdAt.Time)
	}
	if updatedAt.Valid {
		entry.UpdatedAt = timestampProto(updatedAt.Time)
	}

	return entry, nil
}

// Update updates an existing waitlist entry.
func (r *repository) Update(ctx context.Context, entry *waitlistpb.WaitlistEntry) (*waitlistpb.WaitlistEntry, error) {
	questionnaireJSON, err := json.Marshal(structToMap(entry.QuestionnaireAnswers))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal questionnaire answers: %w", err)
	}

	contactPrefsJSON, err := json.Marshal(structToMap(entry.ContactPreferences))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal contact preferences: %w", err)
	}

	metadataJSON, err := json.Marshal(metadataToMap(entry.Metadata))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		UPDATE service_waitlist_main SET
			email = $2, first_name = $3, last_name = $4, tier = $5,
			reserved_username = $6, intention = $7, questionnaire_answers = $8, interests = $9,
			referral_username = $10, referral_code = $11, feedback = $12, additional_comments = $13,
			status = $14, priority_score = $15, contact_preferences = $16, metadata = $17,
			updated_at = now()
		WHERE id = $1
		RETURNING updated_at`

	var updatedAt sql.NullTime
	err = r.db.QueryRowContext(ctx, query,
		entry.Id, entry.Email, entry.FirstName, entry.LastName, entry.Tier,
		entry.ReservedUsername, entry.Intention, questionnaireJSON, pq.Array(entry.Interests),
		entry.ReferralUsername, entry.ReferralCode, entry.Feedback, entry.AdditionalComments,
		entry.Status, entry.PriorityScore, contactPrefsJSON, metadataJSON,
	).Scan(&updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrWaitlistEntryNotFound
		}
		return nil, fmt.Errorf("failed to update waitlist entry: %w", err)
	}

	if updatedAt.Valid {
		entry.UpdatedAt = timestampProto(updatedAt.Time)
	}

	return entry, nil
}

// GetByID gets a waitlist entry by ID.
func (r *repository) GetByID(ctx context.Context, id int64) (*waitlistpb.WaitlistEntry, error) {
	query := `
		SELECT id, uuid, master_id, master_uuid, email, first_name, last_name, tier,
			   reserved_username, intention, questionnaire_answers, interests,
			   referral_username, referral_code, feedback, additional_comments,
			   status, priority_score, contact_preferences, metadata,
			   created_at, updated_at, invited_at, waitlist_position,
			   campaign_name, referral_count, referral_points, location_country,
			   location_region, location_city, location_lat, location_lng,
			   ip_address, user_agent, referrer_url, utm_source, utm_medium,
			   utm_campaign, utm_term, utm_content
		FROM service_waitlist_main WHERE id = $1`
	return r.scanWaitlistEntry(ctx, query, id)
}

// GetByUUID gets a waitlist entry by UUID.
func (r *repository) GetByUUID(ctx context.Context, uuidStr string) (*waitlistpb.WaitlistEntry, error) {
	query := `
		 SELECT id, uuid, master_id, master_uuid, email, first_name, last_name, tier,
			 reserved_username, intention, questionnaire_answers, interests,
			 referral_username, referral_code, feedback, additional_comments,
			 status, priority_score, contact_preferences, metadata,
			 created_at, updated_at, invited_at, waitlist_position,
			 campaign_name, referral_count, referral_points, location_country,
			 location_region, location_city, location_lat, location_lng,
			 ip_address, user_agent, referrer_url, utm_source, utm_medium,
			 utm_campaign, utm_term, utm_content
		 FROM service_waitlist_main WHERE uuid = $1`
	return r.scanWaitlistEntry(ctx, query, uuidStr)
}

// GetByEmail gets a waitlist entry by email.
func (r *repository) GetByEmail(ctx context.Context, email string) (*waitlistpb.WaitlistEntry, error) {
	query := `
		SELECT id, uuid, master_id, master_uuid, email, first_name, last_name, tier,
			   reserved_username, intention, questionnaire_answers, interests,
			   referral_username, referral_code, feedback, additional_comments,
			   status, priority_score, contact_preferences, metadata,
			   created_at, updated_at, invited_at, waitlist_position,
			   campaign_name, referral_count, referral_points, location_country,
			   location_region, location_city, location_lat, location_lng,
			   ip_address, user_agent, referrer_url, utm_source, utm_medium,
			   utm_campaign, utm_term, utm_content
		FROM service_waitlist_main WHERE email = $1`
	return r.scanWaitlistEntry(ctx, query, email)
}

// List lists waitlist entries with pagination and filters.
func (r *repository) List(ctx context.Context, limit, offset int, tierFilter, statusFilter, campaignFilter string) ([]*waitlistpb.WaitlistEntry, int64, error) {
	conditions := []string{}
	args := []interface{}{}
	argIndex := 1

	if tierFilter != "" {
		conditions = append(conditions, fmt.Sprintf("tier = $%d", argIndex))
		args = append(args, tierFilter)
		argIndex++
	}
	if statusFilter != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, statusFilter)
		argIndex++
	}
	if campaignFilter != "" {
		conditions = append(conditions, fmt.Sprintf("campaign_name = $%d", argIndex))
		args = append(args, campaignFilter)
		argIndex++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count query
	countQuery := "SELECT COUNT(*) FROM service_waitlist_main " + whereClause
	var totalCount int64
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count waitlist entries: %w", err)
	}

	// Data query
	query := "SELECT id, uuid, master_id, master_uuid, email, first_name, last_name, tier, " +
		"reserved_username, intention, questionnaire_answers, interests, " +
		"referral_username, referral_code, feedback, additional_comments, " +
		"status, priority_score, contact_preferences, metadata, " +
		"created_at, updated_at, invited_at, waitlist_position, " +
		"campaign_name, referral_count, referral_points, location_country, " +
		"location_region, location_city, location_lat, location_lng, " +
		"ip_address, user_agent, referrer_url, utm_source, utm_medium, " +
		"utm_campaign, utm_term, utm_content FROM service_waitlist_main"
	if whereClause != "" {
		query += " " + whereClause
	}
	query += " ORDER BY priority_score DESC, created_at ASC LIMIT $" + fmt.Sprintf("%d", argIndex) + " OFFSET $" + fmt.Sprintf("%d", argIndex+1)

	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list waitlist entries: %w", err)
	}
	defer rows.Close()

	var entries []*waitlistpb.WaitlistEntry
	for rows.Next() {
		entry, err := r.scanWaitlistEntryRow(rows)
		if err != nil {
			return nil, 0, err
		}
		entries = append(entries, entry)
	}

	return entries, totalCount, rows.Err()
}

// EmailExists checks if an email exists in the waitlist.
func (r *repository) EmailExists(ctx context.Context, email string) (bool, error) {
	var count int
	query := "SELECT COUNT(*) FROM service_waitlist_main WHERE email = $1"
	err := r.db.QueryRowContext(ctx, query, email).Scan(&count)
	return count > 0, err
}

// UsernameExists checks if a username is already reserved.
func (r *repository) UsernameExists(ctx context.Context, username string) (bool, error) {
	var count int
	query := "SELECT COUNT(*) FROM service_waitlist_main WHERE reserved_username = $1"
	err := r.db.QueryRowContext(ctx, query, username).Scan(&count)
	return count > 0, err
}

// UpdateStatus updates the status of a waitlist entry.
func (r *repository) UpdateStatus(ctx context.Context, id int64, status string) error {
	query := "UPDATE service_waitlist_main SET status = $2, updated_at = now() WHERE id = $1"
	result, err := r.db.ExecContext(ctx, query, id, status)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrWaitlistEntryNotFound
	}

	return nil
}

// UpdatePriorityScore updates the priority score of a waitlist entry.
func (r *repository) UpdatePriorityScore(ctx context.Context, id int64, score int) error {
	query := "UPDATE service_waitlist_main SET priority_score = $2, updated_at = now() WHERE id = $1"
	result, err := r.db.ExecContext(ctx, query, id, score)
	if err != nil {
		return fmt.Errorf("failed to update priority score: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrWaitlistEntryNotFound
	}

	return nil
}

// GetStats gets waitlist statistics.
func (r *repository) GetStats(ctx context.Context, campaign string) (*waitlistpb.WaitlistStats, error) {
	stats := &waitlistpb.WaitlistStats{
		TierBreakdown:   make(map[string]int64),
		StatusBreakdown: make(map[string]int64),
		CampaignStats:   make(map[string]int64),
	}

	// Removed unused whereClause variable
	// No need for whereClause, queries are parameterized below

	// Get counts
	// Use parameterized queries to avoid SQL injection
	var err error
	if campaign != "" {
		err = r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM service_waitlist_main WHERE campaign_name = $1", campaign).Scan(&stats.TotalEntries)
		if err != nil {
			return nil, fmt.Errorf("failed to scan total entries: %w", err)
		}
		err = r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM service_waitlist_main WHERE campaign_name = $1 AND status = 'pending'", campaign).Scan(&stats.PendingEntries)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pending entries: %w", err)
		}
		err = r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM service_waitlist_main WHERE campaign_name = $1 AND status = 'invited'", campaign).Scan(&stats.InvitedEntries)
		if err != nil {
			return nil, fmt.Errorf("failed to scan invited entries: %w", err)
		}
	} else {
		err = r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM service_waitlist_main").Scan(&stats.TotalEntries)
		if err != nil {
			return nil, fmt.Errorf("failed to scan total entries: %w", err)
		}
		err = r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM service_waitlist_main WHERE status = 'pending'").Scan(&stats.PendingEntries)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pending entries: %w", err)
		}
		err = r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM service_waitlist_main WHERE status = 'invited'").Scan(&stats.InvitedEntries)
		if err != nil {
			return nil, fmt.Errorf("failed to scan invited entries: %w", err)
		}
	}

	return stats, nil
}

// GetWaitlistPosition gets the waitlist position for a user.
func (r *repository) GetWaitlistPosition(ctx context.Context, id int64) (int, error) {
	query := `
		SELECT COUNT(*) + 1 FROM service_waitlist_main w1
		WHERE w1.priority_score > (SELECT w2.priority_score FROM service_waitlist_main w2 WHERE w2.id = $1)
		OR (w1.priority_score = (SELECT w2.priority_score FROM service_waitlist_main w2 WHERE w2.id = $1) 
		    AND w1.created_at < (SELECT w2.created_at FROM service_waitlist_main w2 WHERE w2.id = $1))`

	var position int
	err := r.db.QueryRowContext(ctx, query, id).Scan(&position)
	return position, err
}

// GetLeaderboard gets the referral leaderboard for a campaign.
func (r *repository) GetLeaderboard(ctx context.Context, limit int, campaign string) ([]*waitlistpb.LeaderboardEntry, error) {
	var (
		query string
		args  []interface{}
	)
	if campaign != "" {
		query = `SELECT id, uuid, reserved_username, first_name, last_name, tier,
 		referral_count, referral_points, priority_score,
 		location_country, location_region, location_city, created_at,
 		ROW_NUMBER() OVER (ORDER BY referral_points DESC, referral_count DESC, created_at ASC) as position
 		FROM service_waitlist_main WHERE campaign_name = $2
 		ORDER BY referral_points DESC, referral_count DESC, created_at ASC LIMIT $1`
		args = []interface{}{limit, campaign}
	} else {
		query = `SELECT id, uuid, reserved_username, first_name, last_name, tier,
 		referral_count, referral_points, priority_score,
 		location_country, location_region, location_city, created_at,
 		ROW_NUMBER() OVER (ORDER BY referral_points DESC, referral_count DESC, created_at ASC) as position
 		FROM service_waitlist_main
 		ORDER BY referral_points DESC, referral_count DESC, created_at ASC LIMIT $1`
		args = []interface{}{limit}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get leaderboard: %w", err)
	}
	defer rows.Close()

	var entries []*waitlistpb.LeaderboardEntry
	for rows.Next() {
		entry := &waitlistpb.LeaderboardEntry{}
		var createdAt sql.NullTime

		err := rows.Scan(&entry.Id, &entry.Uuid, &entry.ReservedUsername, &entry.FirstName, &entry.LastName,
			&entry.Tier, &entry.ReferralCount, &entry.ReferralPoints, &entry.PriorityScore,
			&entry.LocationCountry, &entry.LocationRegion, &entry.LocationCity, &createdAt, &entry.Position)
		if err != nil {
			return nil, fmt.Errorf("failed to scan leaderboard entry: %w", err)
		}

		if createdAt.Valid {
			entry.CreatedAt = timestampProto(createdAt.Time)
		}
		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

// GetReferralsByUser gets all referrals made by a specific user.
func (r *repository) GetReferralsByUser(ctx context.Context, userID int64) ([]*waitlistpb.ReferralRecord, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	query := `
	       SELECT id, user_id, referrer_master_uuid, referred_master_uuid, referral_code, status, metadata, created_at, updated_at
	       FROM service_referral_main
	       WHERE user_id = $1 OR referrer_master_uuid = $2
       `
	rows, err := r.db.QueryContext(ctx, query, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []*waitlistpb.ReferralRecord
	for rows.Next() {
		var (
			idStr, userIDStr, referrerUUID, referredUUID, referralCode string
			status                                                     int
			metadataJSON                                               []byte
			createdAt, updatedAt                                       sql.NullTime
		)
		if err := rows.Scan(&idStr, &userIDStr, &referrerUUID, &referredUUID, &referralCode, &status, &metadataJSON, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		record := &waitlistpb.ReferralRecord{
			Uuid:         idStr,
			ReferrerUuid: referrerUUID,
			ReferredUuid: referredUUID,
		}
		if createdAt.Valid {
			record.CreatedAt = timestampProto(createdAt.Time)
		}
		// Optionally unmarshal metadataJSON if needed
		records = append(records, record)
	}
	return records, rows.Err()
}

// GetLocationStats gets location-based statistics for a campaign.
func (r *repository) GetLocationStats(ctx context.Context, campaign string) ([]*waitlistpb.LocationStat, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	// Use campaign for diagnostics (lint fix)
	_ = campaign
	// Stub implementation - would need proper aggregation
	return []*waitlistpb.LocationStat{}, nil
}

// ValidateReferralUsername validates a referral username (checks if it exists).
func (r *repository) ValidateReferralUsername(ctx context.Context, username string) (bool, error) {
	var count int
	query := "SELECT COUNT(*) FROM service_waitlist_main WHERE reserved_username = $1"
	err := r.db.QueryRowContext(ctx, query, username).Scan(&count)
	return count > 0, err
}

// CreateReferralRecord creates a new referral record.
func (r *repository) CreateReferralRecord(ctx context.Context, referrerUsername string, referredID int64) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	// For demo: referrerUsername must be resolved to referrer_master_uuid, referredID to referred_master_uuid
	// You may need to join with waitlist table to get UUIDs
	// Here, we assume referrerUsername is a UUID string and referredID is a UUID string (adapt as needed)
	query := `
	       INSERT INTO service_referral_main (referrer_master_uuid, referred_master_uuid, referral_code, status, created_at, updated_at)
	       VALUES ($1, $2, $3, $4, now(), now())
       `
	// For demo, referral_code is generated as UUID, status=1 (active)
	referralCode := uuid.New().String()
	status := 1
	_, err := r.db.ExecContext(ctx, query, referrerUsername, referredID, referralCode, status)
	return err
}

// Helper methods

// scanWaitlistEntry scans a single waitlist entry from a query.
func (r *repository) scanWaitlistEntry(ctx context.Context, query string, args ...interface{}) (*waitlistpb.WaitlistEntry, error) {
	row := r.db.QueryRowContext(ctx, query, args...)
	return r.scanWaitlistEntryRow(row)
}

// scanWaitlistEntryRow scans a waitlist entry from a row.
func (r *repository) scanWaitlistEntryRow(row interface{}) (*waitlistpb.WaitlistEntry, error) {
	entry := &waitlistpb.WaitlistEntry{}
	var questionnaireJSON, contactPrefsJSON, metadataJSON []byte
	var interests pq.StringArray
	var createdAt, updatedAt, invitedAt sql.NullTime

	scanner, ok := row.(interface{ Scan(...interface{}) error })
	if !ok {
		return nil, fmt.Errorf("invalid row type")
	}

	err := scanner.Scan(
		&entry.Id, &entry.Uuid, &entry.MasterId, &entry.MasterUuid,
		&entry.Email, &entry.FirstName, &entry.LastName, &entry.Tier,
		&entry.ReservedUsername, &entry.Intention, &questionnaireJSON, &interests,
		&entry.ReferralUsername, &entry.ReferralCode, &entry.Feedback, &entry.AdditionalComments,
		&entry.Status, &entry.PriorityScore, &contactPrefsJSON, &metadataJSON,
		&createdAt, &updatedAt, &invitedAt, &entry.WaitlistPosition,
		&entry.CampaignName, &entry.ReferralCount, &entry.ReferralPoints,
		&entry.LocationCountry, &entry.LocationRegion, &entry.LocationCity,
		&entry.LocationLat, &entry.LocationLng, &entry.IpAddress, &entry.UserAgent,
		&entry.ReferrerUrl, &entry.UtmSource, &entry.UtmMedium, &entry.UtmCampaign,
		&entry.UtmTerm, &entry.UtmContent,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrWaitlistEntryNotFound
		}
		return nil, fmt.Errorf("failed to scan waitlist entry: %w", err)
	}

	// Convert data
	entry.Interests = []string(interests)

	if createdAt.Valid {
		entry.CreatedAt = timestampProto(createdAt.Time)
	}
	if updatedAt.Valid {
		entry.UpdatedAt = timestampProto(updatedAt.Time)
	}
	if invitedAt.Valid {
		entry.InvitedAt = timestampProto(invitedAt.Time)
	}

	// Unmarshal JSON fields
	if len(questionnaireJSON) > 0 {
		var questionnaireMap map[string]interface{}
		if err := json.Unmarshal(questionnaireJSON, &questionnaireMap); err == nil {
			entry.QuestionnaireAnswers = mapToStruct(questionnaireMap)
		}
	}

	if len(contactPrefsJSON) > 0 {
		var contactPrefsMap map[string]interface{}
		if err := json.Unmarshal(contactPrefsJSON, &contactPrefsMap); err == nil {
			entry.ContactPreferences = mapToStruct(contactPrefsMap)
		}
	}

	if len(metadataJSON) > 0 {
		var metadataMap map[string]interface{}
		if err := json.Unmarshal(metadataJSON, &metadataMap); err == nil {
			entry.Metadata = mapToMetadata(metadataMap)
		}
	}

	return entry, nil
}
