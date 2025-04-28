package repository

import (
	"database/sql"
)

// repositoryProvider implements RepositoryProvider
type repositoryProvider struct {
	db         *sql.DB
	masterRepo MasterRepository
}

// NewRepositoryProvider creates a new repository provider
func NewRepositoryProvider(db *sql.DB) *repositoryProvider {
	provider := &repositoryProvider{db: db}
	provider.masterRepo = NewMasterRepository(db)
	return provider
}

func (p *repositoryProvider) Master() MasterRepository {
	return p.masterRepo
}
