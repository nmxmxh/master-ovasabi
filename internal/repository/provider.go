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
	provider.masterRepo = NewMasterRepository(db, log)
	return provider
}

// GetMasterRepository returns the master repository instance.
func (p *Provider) GetMasterRepository() MasterRepository {
	return p.masterRepo
}
