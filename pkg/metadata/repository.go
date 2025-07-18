package metadata

import (
	"context"
	"time"

	"database/sql"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
)

// Repository provides metadata update methods and depends on a masterRepo for routing.
type Repository struct {
	DB         DBTX             // DBTX is an interface satisfied by *sql.DB and *sql.Tx
	masterRepo MasterRepository // MasterRepository is the interface for master table lookups
}

// DBTX is an interface satisfied by *sql.DB and *sql.Tx for database operations.
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

// MasterRepository is the interface for master table lookups.
type MasterRepository interface {
	LookupByUUIDOrID(ctx context.Context, uuidOrID string, log *zap.Logger) (*MasterRecord, error)
}

// MasterRecord represents a resolved entity from the master table.
type MasterRecord struct {
	TableName  string
	LocalID    string
	EntityType string
}

// UpdateEntityMetadataFromEvent now uses the master table's LookupByUUIDOrID for routing and update.
func (r *Repository) UpdateEntityMetadataFromEvent(ctx context.Context, uuidOrID string, newMeta *commonpb.Metadata, log *zap.Logger) error {
	// 1. Lookup master table for routing info (try as UUID, then as local ID)
	rec, err := r.masterRepo.LookupByUUIDOrID(ctx, uuidOrID, log)
	if err != nil || rec == nil {
		log.Warn("Could not resolve entity in master table for metadata update", zap.String("uuid_or_id", uuidOrID))
		return err
	}
	table := rec.TableName
	entityID := rec.LocalID
	entityType := rec.EntityType

	// 2. Fetch old metadata for hooks
	var oldMeta *commonpb.Metadata
	var oldMetaJSON string
	query := "SELECT metadata FROM " + table + " WHERE id = $1"
	err = r.DB.QueryRowContext(ctx, query, entityID).Scan(&oldMetaJSON)
	if err != nil {
		log.Error("Failed to fetch old metadata", zap.Error(err))
		oldMeta = nil
	} else {
		oldMeta = &commonpb.Metadata{}
		if err := UnmarshalJSONToProto(oldMetaJSON, oldMeta); err != nil {
			log.Error("Failed to unmarshal old metadata", zap.Error(err))
			oldMeta = nil
		}
	}

	// 4. Update metadata in DB
	metaJSON, err := MarshalProtoToJSON(newMeta)
	if err != nil {
		log.Error("Failed to marshal new metadata", zap.Error(err))
		return err
	}
	updateQuery := "UPDATE " + table + " SET metadata = $1, updated_at = NOW() WHERE id = $2"
	_, err = r.DB.ExecContext(ctx, updateQuery, metaJSON, entityID)
	if err != nil {
		log.Error("Failed to update metadata in DB", zap.Error(err))
		return err
	}
	log.Info("Updated entity metadata in DB", zap.String("table", table), zap.String("entity_id", entityID))

	// 6. Audit (append audit entry)
	Handler{}.AppendAudit(ProtoToMap(newMeta), map[string]interface{}{
		"action":      "update_metadata",
		"entity_type": entityType,
		"entity_id":   entityID,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
	})

	return nil
}

// MarshalProtoToJSON marshals a proto message to JSON string using protojson.
func MarshalProtoToJSON(meta *commonpb.Metadata) (string, error) {
	data, err := protojson.Marshal(meta)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// UnmarshalJSONToProto unmarshals a JSON string to a proto message using protojson.
func UnmarshalJSONToProto(jsonStr string, meta *commonpb.Metadata) error {
	return protojson.Unmarshal([]byte(jsonStr), meta)
}
