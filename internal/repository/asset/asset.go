package asset

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type StorageType string

const (
	StorageTypeLight StorageType = "light"
	StorageTypeHeavy StorageType = "heavy"
)

type AssetModel struct {
	ID        uuid.UUID   `db:"id"`
	UserID    uuid.UUID   `db:"user_id"`
	Type      StorageType `db:"type"`
	Name      string      `db:"name"`
	MimeType  string      `db:"mime_type"`
	Size      int64       `db:"size"`
	Data      []byte      `db:"data"`
	URL       string      `db:"url"`
	IsSystem  bool        `db:"is_system"`
	Checksum  string      `db:"checksum"`
	CreatedAt time.Time   `db:"created_at"`
	UpdatedAt time.Time   `db:"updated_at"`
	DeletedAt *time.Time  `db:"deleted_at"`
}

// AssetRepository defines the interface for asset operations
type AssetRepository interface {
	CreateAsset(ctx context.Context, asset *AssetModel) error
	GetAsset(ctx context.Context, id uuid.UUID) (*AssetModel, error)
	ListUserAssets(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*AssetModel, error)
	ListSystemAssets(ctx context.Context, limit, offset int) ([]*AssetModel, error)
	UpdateAsset(ctx context.Context, asset *AssetModel) error
	DeleteAsset(ctx context.Context, id uuid.UUID) error
}

// Repository implements AssetRepository
type Repository struct {
	db  *sql.DB
	log *zap.Logger
}

// InitRepository creates a new asset repository instance
func InitRepository(db *sql.DB, log *zap.Logger) *Repository {
	return &Repository{
		db:  db,
		log: log,
	}
}

// CreateAsset creates a new asset
func (r *Repository) CreateAsset(ctx context.Context, asset *AssetModel) error {
	query := `
		INSERT INTO assets (id, user_id, type, name, mime_type, size, data, url, checksum, is_system, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := r.db.ExecContext(ctx, query,
		asset.ID, asset.UserID, asset.Type, asset.Name, asset.MimeType,
		asset.Size, asset.Data, asset.URL, asset.Checksum, asset.IsSystem,
		asset.CreatedAt, asset.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create asset: %w", err)
	}
	return nil
}

// GetAsset retrieves an asset by ID
func (r *Repository) GetAsset(ctx context.Context, id uuid.UUID) (*AssetModel, error) {
	query := `
		SELECT id, user_id, type, name, mime_type, size, data, url, checksum, is_system,
			   created_at, updated_at, deleted_at
		FROM assets
		WHERE id = $1 AND deleted_at IS NULL
	`
	asset := &AssetModel{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&asset.ID, &asset.UserID, &asset.Type, &asset.Name, &asset.MimeType,
		&asset.Size, &asset.Data, &asset.URL, &asset.Checksum, &asset.IsSystem,
		&asset.CreatedAt, &asset.UpdatedAt, &asset.DeletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get asset: %w", err)
	}
	return asset, nil
}

// ListUserAssets retrieves assets for a user with pagination
func (r *Repository) ListUserAssets(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*AssetModel, error) {
	query := `
		SELECT id, user_id, type, name, mime_type, size, data, url, checksum, is_system, created_at, updated_at, deleted_at
		FROM assets
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query assets: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.log.Error("failed to close rows", zap.Error(err))
		}
	}()

	var assets []*AssetModel
	for rows.Next() {
		asset := &AssetModel{}
		err := rows.Scan(
			&asset.ID, &asset.UserID, &asset.Type, &asset.Name, &asset.MimeType,
			&asset.Size, &asset.Data, &asset.URL, &asset.Checksum, &asset.IsSystem,
			&asset.CreatedAt, &asset.UpdatedAt, &asset.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan asset: %w", err)
		}
		assets = append(assets, asset)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}
	return assets, nil
}

// ListSystemAssets retrieves system assets with pagination
func (r *Repository) ListSystemAssets(ctx context.Context, limit, offset int) ([]*AssetModel, error) {
	query := `
		SELECT id, user_id, type, name, mime_type, size, data, url, checksum, is_system,
			   created_at, updated_at, deleted_at
		FROM assets
		WHERE is_system = true AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query system assets: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.log.Error("failed to close rows", zap.Error(err))
		}
	}()

	var assets []*AssetModel
	for rows.Next() {
		asset := &AssetModel{}
		err := rows.Scan(
			&asset.ID, &asset.UserID, &asset.Type, &asset.Name, &asset.MimeType,
			&asset.Size, &asset.Data, &asset.URL, &asset.Checksum, &asset.IsSystem,
			&asset.CreatedAt, &asset.UpdatedAt, &asset.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan system asset: %w", err)
		}
		assets = append(assets, asset)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}
	return assets, nil
}

// UpdateAsset updates an existing asset
func (r *Repository) UpdateAsset(ctx context.Context, asset *AssetModel) error {
	query := `
		UPDATE assets
		SET type = $1, name = $2, mime_type = $3, size = $4, data = $5,
			url = $6, checksum = $7, is_system = $8, updated_at = $9
		WHERE id = $10 AND deleted_at IS NULL
	`
	result, err := r.db.ExecContext(ctx, query,
		asset.Type, asset.Name, asset.MimeType, asset.Size, asset.Data,
		asset.URL, asset.Checksum, asset.IsSystem, time.Now(), asset.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update asset: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("asset not found or already deleted")
	}
	return nil
}

// DeleteAsset soft deletes an asset
func (r *Repository) DeleteAsset(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE assets
		SET deleted_at = $1
		WHERE id = $2 AND deleted_at IS NULL
	`
	result, err := r.db.ExecContext(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to delete asset: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("asset not found or already deleted")
	}
	return nil
}
