package repository

import (
	"database/sql"

	"go.uber.org/zap"
)

// Provider manages repository instances.
type Provider struct {
	db         *sql.DB
	log        *zap.Logger
	masterRepo MasterRepository
}

// NewProvider creates a new repository provider.
func NewProvider(db *sql.DB, log *zap.Logger) *Provider {
	provider := &Provider{
		db:  db,
		log: log,
	}
	provider.masterRepo = NewRepository(db, log)
	return provider
}

// GetMasterRepository returns the master repository instance.
func (p *Provider) GetMasterRepository() MasterRepository {
	return p.masterRepo
}

// NewRepository creates a new repository instance.
func NewRepository(db *sql.DB, log *zap.Logger) MasterRepository {
	return NewMasterRepository(db, log)
}
