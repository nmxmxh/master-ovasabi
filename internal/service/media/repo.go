package media

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/logger"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

type StorageType string

const (
	StorageTypeLight StorageType = "light"
	StorageTypeHeavy StorageType = "heavy"
)

var ErrMediaNotFound = errors.New("media not found")

type Model struct {
	ID         uuid.UUID   `db:"id"`
	MasterID   string      `db:"master_id"`
	MasterUUID string      `db:"master_uuid"`
	UserID     uuid.UUID   `db:"user_id"`
	Type       StorageType `db:"type"`
	Name       string      `db:"name"`
	MimeType   string      `db:"mime_type"`
	Size       int64       `db:"size"`
	URL        string      `db:"url"`
	IsSystem   bool        `db:"is_system"`
	Checksum   string      `db:"checksum"`
	Metadata   *commonpb.Metadata
	CreatedAt  time.Time  `db:"created_at"`
	UpdatedAt  time.Time  `db:"updated_at"`
	DeletedAt  *time.Time `db:"deleted_at"`
	// NFT/Authenticity fields
	AuthenticityHash string `db:"authenticity_hash"`
	R2Key            string `db:"r2_key"`
	Signature        string `db:"signature"`
}

// Repository defines the interface for media operations.
type Repository interface {
	CreateMedia(ctx context.Context, media *Model) error
	GetMedia(ctx context.Context, id uuid.UUID) (*Model, error)
	ListUserMedia(ctx context.Context, userID uuid.UUID, masterID string, limit, offset int) ([]*Model, error)
	ListSystemMedia(ctx context.Context, masterID string, limit, offset int) ([]*Model, error)
	UpdateMedia(ctx context.Context, media *Model) error
	DeleteMedia(ctx context.Context, id uuid.UUID) error
}

// Repo implements Repository.
type Repo struct {
	db  *sql.DB
	log *zap.Logger
}

var logInstance logger.Logger

func init() {
	var err error
	logInstance, err = logger.NewDefault()
	if err != nil {
		panic(fmt.Sprintf("failed to initialize logger: %v", err))
	}
}

// InitRepository creates a new media repository instance.
func InitRepository(db *sql.DB, log *zap.Logger) *Repo {
	return &Repo{
		db:  db,
		log: log,
	}
}

// CreateMedia creates a new media.
func (r *Repo) CreateMedia(ctx context.Context, media *Model) error {
	var metadataJSON []byte
	var err error
	if media.Metadata != nil {
		metadataJSON, err = protojson.Marshal(media.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}
	query := `
		INSERT INTO service_media_main (id, master_id, user_id, type, name, mime_type, size, url, checksum, is_system, created_at, updated_at, authenticity_hash, r2_key, signature)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`
	_, err = r.db.ExecContext(ctx, query,
		media.ID, media.MasterID, media.UserID, media.Type, media.Name, media.MimeType,
		media.Size, media.URL, media.Checksum, media.IsSystem,
		media.CreatedAt, media.UpdatedAt,
		media.AuthenticityHash, media.R2Key, media.Signature,
		metadataJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to create media: %w", err)
	}
	return nil
}

// GetMedia retrieves media by ID.
func (r *Repo) GetMedia(ctx context.Context, id uuid.UUID) (*Model, error) {
	query := `
		SELECT id, master_id, user_id, type, name, mime_type, size, url, checksum, is_system,
			   created_at, updated_at, deleted_at, authenticity_hash, r2_key, signature
		FROM service_media_main
		WHERE id = $1 AND deleted_at IS NULL
	`
	media := &Model{}
	var metadataStr string
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&media.ID, &media.MasterID, &media.UserID, &media.Type, &media.Name, &media.MimeType,
		&media.Size, &media.URL, &media.Checksum, &media.IsSystem,
		&media.CreatedAt, &media.UpdatedAt, &media.DeletedAt,
		&media.AuthenticityHash, &media.R2Key, &media.Signature,
		&metadataStr,
	)
	if err == sql.ErrNoRows {
		return nil, ErrMediaNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get media: %w", err)
	}
	media.Metadata = &commonpb.Metadata{
		ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{"error": err.Error(), "media_id": media.ID},
			func() *structpb.Struct {
				if media != nil && media.Metadata != nil {
					return media.Metadata.ServiceSpecific
				}
				return nil
			}()),
		Tags:     []string{},
		Features: []string{},
	}
	if metadataStr != "" {
		err := protojson.Unmarshal([]byte(metadataStr), media.Metadata)
		if err != nil {
			logInstance.Warn("failed to unmarshal media metadata", zap.Error(err))
			return nil, err
		}
	}
	return media, nil
}

// ListUserMedia retrieves media for a user with pagination and optional master_id filter.
func (r *Repo) ListUserMedia(ctx context.Context, userID uuid.UUID, masterID string, limit, offset int) ([]*Model, error) {
	query := `
		SELECT id, master_id, user_id, type, name, mime_type, size, url, checksum, is_system, created_at, updated_at, deleted_at, authenticity_hash, r2_key, signature
		FROM service_media_main
		WHERE user_id = $1 AND deleted_at IS NULL`
	args := []interface{}{userID}
	if masterID != "" {
		query += " AND master_id = $2"
		args = append(args, masterID)
	}
	query += " ORDER BY created_at DESC LIMIT $3 OFFSET $4"
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query media: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.log.Error("failed to close rows", zap.Error(err))
		}
	}()
	var mediaList []*Model
	for rows.Next() {
		media := &Model{}
		var metadataStr string
		err := rows.Scan(
			&media.ID, &media.MasterID, &media.UserID, &media.Type, &media.Name, &media.MimeType,
			&media.Size, &media.URL, &media.Checksum, &media.IsSystem,
			&media.CreatedAt, &media.UpdatedAt, &media.DeletedAt,
			&media.AuthenticityHash, &media.R2Key, &media.Signature,
			&metadataStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan media: %w", err)
		}
		media.Metadata = &commonpb.Metadata{
			ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{"error": err.Error(), "media_id": media.ID},
				func() *structpb.Struct {
					if media != nil && media.Metadata != nil {
						return media.Metadata.ServiceSpecific
					}
					return nil
				}()),
			Tags:     []string{},
			Features: []string{},
		}
		if metadataStr != "" {
			err := protojson.Unmarshal([]byte(metadataStr), media.Metadata)
			if err != nil {
				logInstance.Warn("failed to unmarshal media metadata", zap.Error(err))
				return nil, err
			}
		}
		mediaList = append(mediaList, media)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}
	return mediaList, nil
}

// ListSystemMedia retrieves system media with pagination and optional master_id filter.
func (r *Repo) ListSystemMedia(ctx context.Context, masterID string, limit, offset int) ([]*Model, error) {
	query := `
		SELECT id, master_id, user_id, type, name, mime_type, size, url, checksum, is_system,
			   created_at, updated_at, deleted_at, authenticity_hash, r2_key, signature
		FROM service_media_main
		WHERE is_system = true AND deleted_at IS NULL`
	args := []interface{}{}
	if masterID != "" {
		query += " AND master_id = $1"
		args = append(args, masterID)
	}
	query += " ORDER BY created_at DESC LIMIT $2 OFFSET $3"
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query system media: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.log.Error("failed to close rows", zap.Error(err))
		}
	}()
	var mediaList []*Model
	for rows.Next() {
		media := &Model{}
		var metadataStr string
		err := rows.Scan(
			&media.ID, &media.MasterID, &media.UserID, &media.Type, &media.Name, &media.MimeType,
			&media.Size, &media.URL, &media.Checksum, &media.IsSystem,
			&media.CreatedAt, &media.UpdatedAt, &media.DeletedAt,
			&media.AuthenticityHash, &media.R2Key, &media.Signature,
			&metadataStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan system media: %w", err)
		}
		media.Metadata = &commonpb.Metadata{
			ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{"error": err.Error(), "media_id": media.ID},
				func() *structpb.Struct {
					if media != nil && media.Metadata != nil {
						return media.Metadata.ServiceSpecific
					}
					return nil
				}()),
			Tags:     []string{},
			Features: []string{},
		}
		if metadataStr != "" {
			err := protojson.Unmarshal([]byte(metadataStr), media.Metadata)
			if err != nil {
				logInstance.Warn("failed to unmarshal media metadata", zap.Error(err))
				return nil, err
			}
		}
		mediaList = append(mediaList, media)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}
	return mediaList, nil
}

// UpdateMedia updates an existing media.
func (r *Repo) UpdateMedia(ctx context.Context, media *Model) error {
	var metadataJSON []byte
	var err error
	if media.Metadata != nil {
		metadataJSON, err = protojson.Marshal(media.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}
	query := `
		UPDATE service_media_main
		SET type = $1, name = $2, mime_type = $3, size = $4, url = $5, checksum = $6, is_system = $7, updated_at = $8, authenticity_hash = $9, r2_key = $10, signature = $11
		WHERE id = $12 AND deleted_at IS NULL
	`
	result, err := r.db.ExecContext(ctx, query,
		media.Type, media.Name, media.MimeType, media.Size, media.URL,
		media.Checksum, media.IsSystem, time.Now(),
		media.AuthenticityHash, media.R2Key, media.Signature,
		media.ID,
		metadataJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to update media: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("media not found or already deleted")
	}
	return nil
}

// DeleteMedia deletes media by ID.
func (r *Repo) DeleteMedia(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE service_media_main
		SET deleted_at = $1
		WHERE id = $2 AND deleted_at IS NULL
	`
	result, err := r.db.ExecContext(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to delete media: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("media not found or already deleted")
	}
	return nil
}
