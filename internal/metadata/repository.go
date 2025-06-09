package metadata

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
)

type Record struct {
	ID          uuid.UUID
	ParentID    *uuid.UUID
	MasterID    *uuid.UUID
	EntityID    *uuid.UUID
	Category    string
	Environment string
	Version     int
	Data        map[string]interface{}
	CreatedBy   *uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Repository struct {
	db *sql.DB
}

func NewMetadataRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// CreateMetadata creates a new metadata record.
func (r *Repository) CreateMetadata(ctx context.Context, parentID, masterID, entityID, createdBy *uuid.UUID, category, environment string, data map[string]interface{}) (*Record, error) {
	// Enrich and hash metadata before persisting
	meta := metadata.MapToProto(data)
	metadata.EnrichAndHashMetadata(meta, "repository.create")
	data = metadata.ProtoToMap(meta)
	id := uuid.New()
	version := 1
	if parentID != nil {
		// Get parent version
		var parentVersion int
		err := r.db.QueryRowContext(ctx, `SELECT version FROM _metadata_master WHERE id = $1`, *parentID).Scan(&parentVersion)
		if err != nil {
			return nil, err
		}
		version = parentVersion + 1
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO _metadata_master (id, parent_id, master_id, entity_id, category, environment, version, data, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9, now(), now())
	`, id, parentID, masterID, entityID, category, environment, version, data, createdBy)
	if err != nil {
		return nil, err
	}
	return &Record{
		ID:          id,
		ParentID:    parentID,
		MasterID:    masterID,
		EntityID:    entityID,
		Category:    category,
		Environment: environment,
		Version:     version,
		Data:        data,
		CreatedBy:   createdBy,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}

// GetLatestMetadata returns the latest metadata for a given master/entity/category/environment.
func (r *Repository) GetLatestMetadata(ctx context.Context, masterID, entityID *uuid.UUID, category, environment string) (*Record, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, parent_id, master_id, entity_id, category, environment, version, data, created_by, created_at, updated_at
		FROM _metadata_master
		WHERE master_id = $1 AND entity_id = $2 AND category = $3 AND environment = $4
		ORDER BY version DESC LIMIT 1
	`, masterID, entityID, category, environment)
	return scanMetadataRecord(row)
}

// UpdateMetadata creates a new version, with parent_id pointing to previous.
func (r *Repository) UpdateMetadata(ctx context.Context, parentID, masterID, entityID, createdBy *uuid.UUID, category, environment string, data map[string]interface{}) (*Record, error) {
	// Enrich and hash metadata before updating
	meta := metadata.MapToProto(data)
	metadata.EnrichAndHashMetadata(meta, "repository.update")
	data = metadata.ProtoToMap(meta)
	return r.CreateMetadata(ctx, parentID, masterID, entityID, createdBy, category, environment, data)
}

// GetMetadataLineage returns the full lineage (history) of a metadata object.
func (r *Repository) GetMetadataLineage(ctx context.Context, id uuid.UUID) ([]*Record, error) {
	var lineage []*Record
	currentID := &id
	for currentID != nil {
		row := r.db.QueryRowContext(ctx, `
			SELECT id, parent_id, master_id, entity_id, category, environment, version, data, created_by, created_at, updated_at
			FROM _metadata_master WHERE id = $1
		`, *currentID)
		rec, err := scanMetadataRecord(row)
		if err != nil {
			return nil, err
		}
		lineage = append([]*Record{rec}, lineage...)
		if rec.ParentID == nil {
			break
		}
		currentID = rec.ParentID
	}
	return lineage, nil
}

// GetPublicSummary returns the public summary for a given master/entity/category/environment.
func (r *Repository) GetPublicSummary(ctx context.Context, masterID, entityID *uuid.UUID, category, environment string) (map[string]interface{}, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT max(version), (data->>'score')::float, (data->>'compliance_status'), max(updated_at)
		FROM _metadata_master
		WHERE master_id = $1 AND entity_id = $2 AND category = $3 AND environment = $4
		GROUP BY master_id, entity_id, category, environment
	`, masterID, entityID, category, environment)
	var version int
	var score *float64
	var complianceStatus *string
	var lastUpdated time.Time
	if err := row.Scan(&version, &score, &complianceStatus, &lastUpdated); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"latest_version":    version,
		"latest_score":      score,
		"compliance_status": complianceStatus,
		"last_updated":      lastUpdated,
	}, nil
}

// Helper to scan a MetadataRecord from a sql.Row.
func scanMetadataRecord(row *sql.Row) (*Record, error) {
	var (
		id, parentID, masterID, entityID, createdBy sql.NullString
		category, environment                       string
		version                                     int
		dataBytes                                   []byte
		createdAt, updatedAt                        time.Time
	)
	err := row.Scan(&id, &parentID, &masterID, &entityID, &category, &environment, &version, &dataBytes, &createdBy, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	var data map[string]interface{}
	if err := json.Unmarshal(dataBytes, &data); err != nil {
		return nil, err
	}
	var pid, mid, eid, cbid *uuid.UUID
	if parentID.Valid {
		p, err := uuid.Parse(parentID.String)
		if err == nil {
			pid = &p
		}
	}
	if masterID.Valid {
		m, err := uuid.Parse(masterID.String)
		if err == nil {
			mid = &m
		}
	}
	if entityID.Valid {
		e, err := uuid.Parse(entityID.String)
		if err == nil {
			eid = &e
		}
	}
	if createdBy.Valid {
		c, err := uuid.Parse(createdBy.String)
		if err == nil {
			cbid = &c
		}
	}
	return &Record{
		ID:          uuid.MustParse(id.String),
		ParentID:    pid,
		MasterID:    mid,
		EntityID:    eid,
		Category:    category,
		Environment: environment,
		Version:     version,
		Data:        data,
		CreatedBy:   cbid,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}, nil
}
