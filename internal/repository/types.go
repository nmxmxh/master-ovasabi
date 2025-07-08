package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// Repository defines the common interface for all repositories.
type Repository interface {
	// GetDB returns the database connection
	GetDB() *sql.DB
	// GetContext returns a new context with transaction if in transaction
	GetContext(ctx context.Context) context.Context
	// WithTx wraps the repository with a transaction
	WithTx(tx *sql.Tx) Repository
}

// Only define Provider once, and update all references from RepositoryProvider to Provider.

// EntityType represents the type of entity in the master table.
type EntityType string

const (
	EntityTypeUser         EntityType = "user"
	EntityTypeNotification EntityType = "notification"
	EntityTypeBroadcast    EntityType = "broadcast"
	EntityTypeCampaign     EntityType = "campaign"
	EntityTypeQuote        EntityType = "quote"
	EntityTypeI18n         EntityType = "i18n"
	EntityTypeReferral     EntityType = "referral"
	EntityTypeAuth         EntityType = "auth"
	EntityTypeFinance      EntityType = "finance"
	EntityTypeRole         EntityType = "role"
)

// Master represents the core entity in the master table.
type Master struct {
	ID          int64      `db:"id"`
	UUID        uuid.UUID  `db:"uuid"`
	Name        string     `db:"name"`
	Type        EntityType `db:"type"`
	Description string     `db:"description"`
	Version     string     `db:"version"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
}

// Remove duplicate MasterRepository interface definition from here.

// Remove Campaign struct and CampaignRepository interface

// Remove Translation struct and I18nRepository interface

// MasterRepository defines the interface for master entity operations, including caching and search.
type MasterRepository interface {
	CreateMasterRecord(ctx context.Context, entityType, name string) (int64, string, error)            // Non-transactional convenience
	Create(ctx context.Context, tx *sql.Tx, entityType EntityType, name string) (int64, string, error) // Transactional
	Get(ctx context.Context, id int64) (*Master, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, limit, offset int) ([]*Master, error)
	GetByUUID(ctx context.Context, id uuid.UUID) (*Master, error)
	Update(ctx context.Context, master *Master) error
	SearchByPattern(ctx context.Context, pattern string, entityType EntityType, limit int) ([]*SearchResult, error)
	SearchByPatternAcrossTypes(ctx context.Context, pattern string, limit int) ([]*SearchResult, error)
	QuickSearch(ctx context.Context, pattern string) ([]*SearchResult, error)
	WithLock(ctx context.Context, entityType EntityType, id interface{}, ttl time.Duration, fn func() error) error
}
