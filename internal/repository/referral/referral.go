package referral

import (
	"database/sql"
	"errors"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	repository "github.com/nmxmxh/master-ovasabi/internal/repository"
	metadatautil "github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"google.golang.org/protobuf/encoding/protojson"
)

var (
	ErrReferralNotFound = errors.New("referral not found")
	ErrReferralExists   = errors.New("referral already exists")
)

// Referral represents a referral record.
type Referral struct {
	ID               int64              `json:"id"`
	ReferrerMasterID string             `json:"referrer_master_id"`
	ReferredMasterID sql.NullString     `json:"referred_master_id"`
	CampaignID       sql.NullInt64      `json:"campaign_id"`
	DeviceHash       sql.NullString     `json:"device_hash"`
	ReferralCode     string             `json:"referral_code"`
	Successful       bool               `json:"successful"`
	CreatedAt        time.Time          `json:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at"`
	Metadata         *commonpb.Metadata `json:"metadata"`
}

// Stats represents referral statistics.
type Stats struct {
	TotalReferrals      int64 `json:"total_referrals"`
	SuccessfulReferrals int64 `json:"successful_referrals"`
}

// Repository handles database operations for referrals.
type Repository struct {
	*repository.BaseRepository
	master repository.MasterRepository
}

// NewReferralRepository creates a new referral repository instance.
func NewReferralRepository(db *sql.DB, master repository.MasterRepository) *Repository {
	return &Repository{
		BaseRepository: repository.NewBaseRepository(db),
		master:         master,
	}
}

// Create inserts a new referral record.
func (r *Repository) Create(referral *Referral) error {
	err := metadatautil.ValidateMetadata(referral.Metadata)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO service_referral_main (referrer_master_id, referred_master_id, campaign_id, device_hash, referral_code, successful, created_at, updated_at, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`

	metadataJSON, err := protojson.Marshal(referral.Metadata)
	if err != nil {
		return err
	}

	err = r.GetDB().QueryRow(
		query,
		referral.ReferrerMasterID,
		referral.ReferredMasterID,
		referral.CampaignID,
		referral.DeviceHash,
		referral.ReferralCode,
		referral.Successful,
		time.Now(),
		time.Now(),
		metadataJSON,
	).Scan(&referral.ID)
	if err != nil {
		return err
	}

	return nil
}

// GetByID retrieves a referral by ID.
func (r *Repository) GetByID(id int64) (*Referral, error) {
	referral := &Referral{}
	var metadataJSON []byte
	query := `
		SELECT id, referrer_master_id, referred_master_id, campaign_id, device_hash, referral_code, successful, created_at, updated_at, metadata
		FROM service_referral_main
		WHERE id = $1`

	err := r.GetDB().QueryRow(query, id).Scan(
		&referral.ID,
		&referral.ReferrerMasterID,
		&referral.ReferredMasterID,
		&referral.CampaignID,
		&referral.DeviceHash,
		&referral.ReferralCode,
		&referral.Successful,
		&referral.CreatedAt,
		&referral.UpdatedAt,
		&metadataJSON,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrReferralNotFound
	}
	if err != nil {
		return nil, err
	}
	meta := &commonpb.Metadata{}
	if err := protojson.Unmarshal(metadataJSON, meta); err != nil {
		return nil, err
	}
	referral.Metadata = meta
	return referral, nil
}

// GetStats retrieves referral statistics for a user.
func (r *Repository) GetStats(referrerMasterID string) (*Stats, error) {
	stats := &Stats{}
	query := `
		SELECT 
			COUNT(*) as total_referrals,
			COUNT(CASE WHEN successful THEN 1 END) as successful_referrals
		FROM service_referral_main
		WHERE referrer_master_id = $1`

	err := r.GetDB().QueryRow(query, referrerMasterID).Scan(
		&stats.TotalReferrals,
		&stats.SuccessfulReferrals,
	)
	if err != nil {
		return nil, err
	}
	return stats, nil
}

// GetByCode retrieves a referral by referral_code.
func (r *Repository) GetByCode(referralCode string) (*Referral, error) {
	referral := &Referral{}
	var metadataJSON []byte
	query := `
		SELECT id, referrer_master_id, referred_master_id, campaign_id, device_hash, referral_code, successful, created_at, updated_at, metadata
		FROM service_referral_main
		WHERE referral_code = $1`

	err := r.GetDB().QueryRow(query, referralCode).Scan(
		&referral.ID,
		&referral.ReferrerMasterID,
		&referral.ReferredMasterID,
		&referral.CampaignID,
		&referral.DeviceHash,
		&referral.ReferralCode,
		&referral.Successful,
		&referral.CreatedAt,
		&referral.UpdatedAt,
		&metadataJSON,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrReferralNotFound
	}
	if err != nil {
		return nil, err
	}
	meta := &commonpb.Metadata{}
	if err := protojson.Unmarshal(metadataJSON, meta); err != nil {
		return nil, err
	}
	referral.Metadata = meta
	return referral, nil
}
