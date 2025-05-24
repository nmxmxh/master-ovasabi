package search

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	repo "github.com/nmxmxh/master-ovasabi/internal/repository"
	"google.golang.org/protobuf/encoding/protojson"
)

// Result matches the proto definition.
type Result struct {
	ID         string             // id
	MasterID   int64              // master_id
	MasterUUID string             // master_uuid
	EntityType string             // entity_type
	Title      string             // title
	Snippet    string             // snippet
	Metadata   *commonpb.Metadata // metadata
	Score      float64            // score
}

type Repository struct {
	db         *sql.DB
	masterRepo repo.MasterRepository
}

func NewRepository(db *sql.DB, masterRepo repo.MasterRepository) *Repository {
	return &Repository{db: db, masterRepo: masterRepo}
}

// SearchEntities performs advanced full-text and fuzzy search on the master table.
// Supports filtering by entityType, query, masterID, fields, metadata, fuzzy, and language.
func (r *Repository) SearchEntities(
	ctx context.Context,
	entityType, query, masterID, masterUUID string,
	fields []string,
	metadata *commonpb.Metadata,
	page, pageSize int,
	fuzzy bool,
	language string,
) ([]*Result, int, error) {
	args := []interface{}{}
	where := []string{"is_active = TRUE"}
	argIdx := 1

	if entityType != "" {
		where = append(where, fmt.Sprintf("entity_type = $%d", argIdx))
		args = append(args, entityType)
		argIdx++
	}
	if query != "" {
		if fuzzy {
			// Use ILIKE for fuzzy search on title/snippet
			where = append(where, fmt.Sprintf("(title ILIKE $%d OR snippet ILIKE $%d)", argIdx, argIdx))
			args = append(args, "%"+query+"%")
			argIdx++
		} else {
			// Use full-text search
			lang := "english"
			if language != "" {
				lang = language
			}
			where = append(where, fmt.Sprintf("search_vector @@ plainto_tsquery('%s', $%d)", lang, argIdx))
			args = append(args, query)
			argIdx++
		}
	}
	if masterID != "" {
		where = append(where, fmt.Sprintf("master_id = $%d", argIdx))
		args = append(args, masterID)
		argIdx++
	}
	if masterUUID != "" {
		where = append(where, fmt.Sprintf("master_uuid = $%d", argIdx))
		args = append(args, masterUUID)
		argIdx++
	}
	if metadata != nil && metadata.ServiceSpecific != nil {
		for k, v := range metadata.ServiceSpecific.Fields {
			where = append(where, fmt.Sprintf("metadata->'service_specific'->>'%s' = $%d", k, argIdx))
			args = append(args, v.GetStringValue())
			argIdx++
		}
	}
	// Filter by fields (if specified, restrict columns returned)
	selectCols := "id, master_id, master_uuid, entity_type, title, snippet, metadata, score"
	if len(fields) > 0 {
		allowed := map[string]bool{"id": true, "master_id": true, "master_uuid": true, "entity_type": true, "title": true, "snippet": true, "metadata": true, "score": true}
		cols := []string{}
		for _, f := range fields {
			if allowed[f] {
				cols = append(cols, f)
			}
		}
		if len(cols) > 0 {
			selectCols = strings.Join(cols, ", ")
		}
	}
	if page < 0 {
		page = 0
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := page * pageSize
	args = append(args, pageSize, offset)
	//nolint:gosec // selectCols is a controlled variable, not user input, so this is safe
	baseQuery := fmt.Sprintf("SELECT %s FROM service_search_index", selectCols)
	if len(where) > 0 {
		baseQuery += " WHERE " + strings.Join(where, " AND ")
	}
	baseQuery += fmt.Sprintf(" ORDER BY score DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	rows, err := r.db.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	results := make([]*Result, 0, pageSize)
	for rows.Next() {
		// Always declare all possible fields
		var (
			id, masterID, masterUUID, entityType, title, snippet string
			metaRaw                                              []byte
			score                                                float64
		)
		meta := &commonpb.Metadata{}
		// Ensure all variables are initialized, even if not selected
		scanTargets := []interface{}{}
		colNames := strings.Split(selectCols, ",")
		for _, col := range colNames {
			col = strings.TrimSpace(col)
			switch col {
			case "id":
				scanTargets = append(scanTargets, &id)
			case "master_id":
				scanTargets = append(scanTargets, &masterID)
			case "master_uuid":
				scanTargets = append(scanTargets, &masterUUID)
			case "entity_type":
				scanTargets = append(scanTargets, &entityType)
			case "title":
				scanTargets = append(scanTargets, &title)
			case "snippet":
				scanTargets = append(scanTargets, &snippet)
			case "metadata":
				scanTargets = append(scanTargets, &metaRaw)
			case "score":
				scanTargets = append(scanTargets, &score)
			}
		}
		if err := rows.Scan(scanTargets...); err != nil {
			return nil, 0, err
		}
		if len(metaRaw) > 0 {
			if err := protojson.Unmarshal(metaRaw, meta); err != nil {
				return nil, 0, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}
		var masterIDInt int64
		if masterID != "" {
			var err error
			masterIDInt, err = strconv.ParseInt(masterID, 10, 64)
			if err != nil {
				return nil, 0, fmt.Errorf("invalid master_id: %w", err)
			}
		}
		results = append(results, &Result{
			ID:         id,
			MasterID:   masterIDInt,
			MasterUUID: masterUUID,
			EntityType: entityType,
			Title:      title,
			Snippet:    snippet,
			Metadata:   meta,
			Score:      score,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var total int
	countQuery := "SELECT COUNT(*) FROM service_search_index"
	if len(where) > 0 {
		countQuery += " WHERE " + strings.Join(where, " AND ")
	}
	countArgs := args[:len(args)-2]
	if err := r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		total = len(results)
	}
	return results, total, nil
}

// helper for entity search with FTS and metadata filtering.
func (r *Repository) searchEntity(
	ctx context.Context,
	_ string, // entity is unused
	sqlStr string,
	args []interface{},
) ([]*Result, error) {
	rows, err := r.db.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []*Result
	for rows.Next() {
		var (
			id, masterID, masterUUID, entityType, title, snippet string
			metaRaw                                              []byte
			score                                                float64
		)
		meta := &commonpb.Metadata{}
		if err := rows.Scan(&id, &masterID, &masterUUID, &entityType, &title, &snippet, &metaRaw, &score); err != nil {
			return nil, err
		}
		if len(metaRaw) > 0 {
			if err := protojson.Unmarshal(metaRaw, meta); err != nil {
				return nil, err
			}
		}
		var masterIDInt int64
		if masterID != "" {
			var err error
			masterIDInt, err = strconv.ParseInt(masterID, 10, 64)
			if err != nil {
				return nil, err
			}
		}
		results = append(results, &Result{
			ID:         id,
			MasterID:   masterIDInt,
			MasterUUID: masterUUID,
			EntityType: entityType,
			Title:      title,
			Snippet:    snippet,
			Metadata:   meta,
			Score:      score,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

// SearchAllEntities performs FTS and metadata filtering across multiple entity tables (content, campaign, user, talent).
// It merges and returns results in a unified format. The 'types' argument specifies which entity types to search.
func (r *Repository) SearchAllEntities(
	ctx context.Context,
	types []string,
	query string,
	metadata *commonpb.Metadata,
	campaignID int64,
	page, pageSize int,
) ([]*Result, int, error) {
	var allResults []*Result

	for _, entityType := range types {
		var (
			sqlStr     string
			args       []interface{}
			metaFilter string
		)
		if metadata != nil && metadata.ServiceSpecific != nil && len(metadata.ServiceSpecific.Fields) > 0 {
			filters := []string{}
			for k, v := range metadata.ServiceSpecific.Fields {
				filters = append(filters, fmt.Sprintf("metadata->'service_specific'->>'%s' = ?", k))
				args = append(args, v.GetStringValue())
			}
			if len(filters) > 0 {
				metaFilter = " AND " + strings.Join(filters, " AND ")
			}
		}
		switch entityType {
		case "content":
			sqlStr = `
				SELECT id::text, master_id::text, master_uuid::text, 'content' as entity_type, title, body, metadata, ts_rank(search_vector, plainto_tsquery($1)) as score
				FROM service_content_main
				WHERE search_vector @@ plainto_tsquery($1)` + metaFilter + `
				ORDER BY score DESC
				LIMIT $2 OFFSET $3
			`
			args = append([]interface{}{query, pageSize, page * pageSize}, args...)
		case "campaign":
			sqlStr = `
				SELECT id::text, master_id::text, master_uuid::text, 'campaign' as entity_type, name, description, metadata, ts_rank(search_vector, plainto_tsquery($1)) as score
				FROM service_campaign_main
				WHERE search_vector @@ plainto_tsquery($1)`
			baseArgs := []interface{}{query}
			if campaignID != 0 {
				sqlStr += " AND campaign_id = ?"
				baseArgs = append(baseArgs, campaignID)
			}
			sqlStr += metaFilter + `
				ORDER BY score DESC
				LIMIT $2 OFFSET $3
			`
			baseArgs = append(baseArgs, pageSize, page*pageSize)
			args = baseArgs
		case "user":
			sqlStr = `
				SELECT id::text, master_id::text, master_uuid::text, 'user' as entity_type, username, email, metadata, ts_rank(search_vector, plainto_tsquery($1)) as score
				FROM service_user_master
				WHERE search_vector @@ plainto_tsquery($1)` + metaFilter + `
				ORDER BY score DESC
				LIMIT $2 OFFSET $3
			`
			args = append([]interface{}{query, pageSize, page * pageSize}, args...)
		case "talent":
			sqlStr = `
				SELECT id::text, master_id::text, master_uuid::text, 'talent' as entity_type, display_name, bio, metadata, ts_rank(search_vector, plainto_tsquery($1)) as score
				FROM service_talent_profile
				WHERE search_vector @@ plainto_tsquery($1)` + metaFilter + `
				ORDER BY score DESC
				LIMIT $2 OFFSET $3
			`
			args = append([]interface{}{query, pageSize, page * pageSize}, args...)
		default:
			continue
		}
		results, err := r.searchEntity(ctx, entityType, sqlStr, args)
		if err != nil {
			return nil, 0, err
		}
		allResults = append(allResults, results...)
	}
	return allResults, len(allResults), nil
}
