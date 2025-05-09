package referral

import (
	"database/sql"
	"errors"
	"time"

	repository "github.com/nmxmxh/master-ovasabi/internal/repository"
)

var (
	ErrReferralNotFound = errors.New("referral not found")
	ErrReferralExists   = errors.New("referral already exists")
)

// Referral represents a referral record
type Referral struct {
	ID            int64     `json:"id"`
	MasterID      int64     `json:"master_id"`
	ReferrerID    int64     `json:"referrer_id"`
	RefereeID     int64     `json:"referee_id"`
	ReferralCode  string    `json:"referral_code"`
	Status        string    `json:"status"`
	RewardClaimed bool      `json:"reward_claimed"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ReferralStats represents referral statistics
type ReferralStats struct {
	TotalReferrals      int64 `json:"total_referrals"`
	SuccessfulReferrals int64 `json:"successful_referrals"`
	PendingReferrals    int64 `json:"pending_referrals"`
}

// ReferralRepository handles database operations for referrals
type ReferralRepository struct {
	*repository.BaseRepository
	master repository.MasterRepository
}

// NewReferralRepository creates a new referral repository instance
func NewReferralRepository(db *sql.DB, master repository.MasterRepository) *ReferralRepository {
	return &ReferralRepository{
		BaseRepository: repository.NewBaseRepository(db),
		master:         master,
	}
}

// Create inserts a new referral record
func (r *ReferralRepository) Create(referral *Referral) error {
	query := `
		INSERT INTO referrals (master_id, referrer_id, referee_id, referral_code, status, reward_claimed, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`

	err := r.GetDB().QueryRow(
		query,
		referral.MasterID,
		referral.ReferrerID,
		referral.RefereeID,
		referral.ReferralCode,
		referral.Status,
		referral.RewardClaimed,
		time.Now(),
		time.Now(),
	).Scan(&referral.ID)

	if err != nil {
		return err
	}

	return nil
}

// GetByID retrieves a referral by ID
func (r *ReferralRepository) GetByID(id int64) (*Referral, error) {
	referral := &Referral{}
	query := `
		SELECT id, master_id, referrer_id, referee_id, referral_code, status, reward_claimed, created_at, updated_at
		FROM referrals
		WHERE id = $1`

	err := r.GetDB().QueryRow(query, id).Scan(
		&referral.ID,
		&referral.MasterID,
		&referral.ReferrerID,
		&referral.RefereeID,
		&referral.ReferralCode,
		&referral.Status,
		&referral.RewardClaimed,
		&referral.CreatedAt,
		&referral.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrReferralNotFound
	}

	if err != nil {
		return nil, err
	}

	return referral, nil
}

// GetStats retrieves referral statistics for a user
func (r *ReferralRepository) GetStats(referrerID int64) (*ReferralStats, error) {
	stats := &ReferralStats{}
	query := `
		SELECT 
			COUNT(*) as total_referrals,
			COUNT(CASE WHEN status = 'completed' THEN 1 END) as successful_referrals,
			COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending_referrals
		FROM referrals
		WHERE referrer_id = $1`

	err := r.GetDB().QueryRow(query, referrerID).Scan(
		&stats.TotalReferrals,
		&stats.SuccessfulReferrals,
		&stats.PendingReferrals,
	)

	if err != nil {
		return nil, err
	}

	return stats, nil
}
