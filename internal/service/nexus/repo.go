// Package nexus provides the repository layer for the Nexus Service.
// See docs/services/nexus.md and api/protos/nexus/v1/nexus_service.proto for full context.
package nexus

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/internal/repository"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Repository handles pattern registration, orchestration, mining, and feedback.
type Repository struct {
	db         *sql.DB
	masterRepo repository.MasterRepository
}

// NewRepository creates a new Nexus repository instance.
func NewRepository(db *sql.DB, masterRepo repository.MasterRepository) *Repository {
	return &Repository{db: db, masterRepo: masterRepo}
}

// RegisterPattern inserts or updates a pattern in the database, with provenance.
func (r *Repository) RegisterPattern(ctx context.Context, req *nexusv1.RegisterPatternRequest, createdBy string, campaignID int64) error {
	meta, err := json.Marshal(req.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	def, err := json.Marshal(req.Definition)
	if err != nil {
		return fmt.Errorf("failed to marshal definition: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO service_nexus_pattern (pattern_id, pattern_type, version, origin, definition, metadata, created_by, campaign_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (pattern_id, campaign_id) DO UPDATE SET
		  pattern_type = EXCLUDED.pattern_type,
		  version = EXCLUDED.version,
		  origin = EXCLUDED.origin,
		  definition = EXCLUDED.definition,
		  metadata = EXCLUDED.metadata,
		  created_by = EXCLUDED.created_by
	`, req.PatternId, req.PatternType, req.Version, req.Origin, def, meta, createdBy, campaignID)
	if err != nil {
		return fmt.Errorf("failed to insert or update pattern: %w", err)
	}
	return nil
}

// ListPatterns returns all patterns, optionally filtered by type.
func (r *Repository) ListPatterns(ctx context.Context, patternType string, campaignID int64) ([]*nexusv1.Pattern, error) {
	q := `SELECT pattern_id, pattern_type, version, origin, definition, usage_count, last_used, metadata FROM service_nexus_pattern WHERE campaign_id = $1`
	args := []interface{}{campaignID}
	if patternType != "" {
		q += " AND pattern_type = $2"
		args = append(args, patternType)
	}
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var patterns []*nexusv1.Pattern
	for rows.Next() {
		var (
			pid, ptype, version, origin string
			def, meta                   []byte
			usage                       int64
			lastUsed                    sql.NullTime
		)
		pat := &nexusv1.Pattern{}
		if err := rows.Scan(&pid, &ptype, &version, &origin, &def, &usage, &lastUsed, &meta); err != nil {
			return nil, err
		}
		pat.PatternId = pid
		pat.PatternType = ptype
		pat.Version = version
		pat.Origin = origin
		pat.UsageCount = usage
		if lastUsed.Valid {
			pat.LastUsed = timestamppb.New(lastUsed.Time)
		}
		if err := json.Unmarshal(def, pat.Definition); err != nil {
			return nil, fmt.Errorf("failed to unmarshal pattern definition: %w", err)
		}
		if err := json.Unmarshal(meta, pat.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal pattern metadata: %w", err)
		}
		patterns = append(patterns, pat)
	}
	return patterns, rows.Err()
}

// GetPatternRequirements extracts requirements from a pattern's definition.
func (r *Repository) GetPatternRequirements(ctx context.Context, patternID string) ([]string, error) {
	row := r.db.QueryRowContext(ctx, `SELECT definition FROM service_nexus_pattern WHERE pattern_id = $1`, patternID)
	var defBytes []byte
	if err := row.Scan(&defBytes); err != nil {
		return nil, err
	}
	var def map[string]interface{}
	if err := json.Unmarshal(defBytes, &def); err != nil {
		return nil, err
	}
	reqs := []string{}
	if reqIface, ok := def["requirements"]; ok {
		if reqList, ok := reqIface.([]interface{}); ok {
			for _, r := range reqList {
				if s, ok := r.(string); ok {
					reqs = append(reqs, s)
				}
			}
		}
	}
	return reqs, nil
}

// ValidatePattern checks if input satisfies pattern requirements.
func (r *Repository) ValidatePattern(ctx context.Context, patternID string, input map[string]interface{}) ([]string, error) {
	reqs, err := r.GetPatternRequirements(ctx, patternID)
	if err != nil {
		return nil, err
	}
	missing := []string{}
	for _, req := range reqs {
		if _, ok := input[req]; !ok {
			missing = append(missing, req)
		}
	}
	return missing, nil
}

// Orchestrate executes a pattern and records the orchestration, with rollback on failure.
func (r *Repository) Orchestrate(ctx context.Context, req *nexusv1.OrchestrateRequest, createdBy string, _ int64) (string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer func() {
		if p := recover(); p != nil {
			if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
				panic(fmt.Errorf("rollback failed after panic: %w", err))
			}
			panic(p)
		}
	}()
	input, err := json.Marshal(req.Input)
	if err != nil {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			return "", fmt.Errorf("rollback failed: %w", err)
		}
		return "", fmt.Errorf("failed to marshal input: %w", err)
	}
	meta, err := json.Marshal(req.Metadata)
	if err != nil {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			return "", fmt.Errorf("rollback failed: %w", err)
		}
		return "", fmt.Errorf("failed to marshal metadata: %w", err)
	}
	var orchestrationID string
	err = tx.QueryRowContext(ctx, `
		INSERT INTO service_nexus_orchestration (pattern_id, input, metadata, created_by, started_at)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING orchestration_id
	`, req.PatternId, input, meta, createdBy).Scan(&orchestrationID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			return "", fmt.Errorf("rollback failed: %w", rbErr)
		}
		if logErr := r.logOrchestrationEvent(ctx, tx, orchestrationID, "rollback", fmt.Sprintf("insert failed: %v", err), nil); logErr != nil {
			return "", fmt.Errorf("failed to log rollback event: %w (original error: %w)", logErr, err)
		}
		return "", fmt.Errorf("failed to insert orchestration: %w", err)
	}
	// Example orchestration step: simulate a step that could fail
	// In real usage, replace this with actual orchestration logic (e.g., calling services, updating state, etc.)
	stepSucceeded := true // set to false to simulate failure
	if !stepSucceeded {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			return "", fmt.Errorf("rollback failed: %w", err)
		}
		if logErr := r.logOrchestrationEvent(ctx, tx, orchestrationID, "rollback", "orchestration step failed", nil); logErr != nil {
			return "", fmt.Errorf("failed to log rollback event: %w", logErr)
		}
		return "", fmt.Errorf("orchestration step failed, transaction rolled back")
	}
	if stepSucceeded {
		details := map[string]interface{}{"info": "step completed"}
		if err := r.AddTraceStep(ctx, tx, orchestrationID, "nexus", "example_step", sql.NullTime{}, details); err != nil {
			if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
				return "", fmt.Errorf("rollback failed: %w", rbErr)
			}
			return "", fmt.Errorf("failed to add trace step: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		if logErr := r.logOrchestrationEvent(ctx, tx, orchestrationID, "rollback", fmt.Sprintf("commit failed: %v", err), nil); logErr != nil {
			return "", fmt.Errorf("failed to log rollback event: %w (original error: %w)", logErr, err)
		}
		return "", err
	}
	return orchestrationID, nil
}

// logOrchestrationEvent logs rollback/audit events for orchestration.
func (r *Repository) logOrchestrationEvent(ctx context.Context, tx *sql.Tx, orchestrationID, eventType, message string, metadata map[string]interface{}) error {
	var metaBytes []byte
	if metadata != nil {
		var err error
		metaBytes, err = json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal orchestration event metadata: %w", err)
		}
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO service_nexus_orchestration_log (orchestration_id, event_type, message, metadata, created_at)
		VALUES ($1, $2, $3, $4, NOW())
	`, orchestrationID, eventType, message, metaBytes)
	return err
}

// TracePattern returns the trace for a given orchestration.
func (r *Repository) TracePattern(ctx context.Context, orchestrationID string) ([]*nexusv1.TraceStep, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT service, action, timestamp, details
		FROM service_nexus_trace
		WHERE orchestration_id = $1
		ORDER BY timestamp ASC
	`, orchestrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to query trace steps: %w", err)
	}
	defer rows.Close()
	var steps []*nexusv1.TraceStep
	for rows.Next() {
		var (
			service, action string
			timestamp       sql.NullTime
			details         []byte
		)
		step := &nexusv1.TraceStep{}
		if err := rows.Scan(&service, &action, &timestamp, &details); err != nil {
			return nil, fmt.Errorf("failed to scan trace step: %w", err)
		}
		step.Service = service
		step.Action = action
		if timestamp.Valid {
			step.Timestamp = timestamppb.New(timestamp.Time)
		}
		if err := json.Unmarshal(details, step.Details); err != nil {
			return nil, fmt.Errorf("failed to unmarshal trace step details: %w", err)
		}
		steps = append(steps, step)
	}
	return steps, rows.Err()
}

// MineAndStorePatterns analyzes trace data to discover frequent (service, action) patterns and stores them.
func (r *Repository) MineAndStorePatterns(ctx context.Context) error {
	// Find the most frequent (service, action) pairs in trace data
	rows, err := r.db.QueryContext(ctx, `
		SELECT service, action, COUNT(*) as support_count
		FROM service_nexus_trace
		GROUP BY service, action
		ORDER BY support_count DESC
		LIMIT 10
	`)
	if err != nil {
		return fmt.Errorf("failed to query trace data for mining: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			service, action string
			supportCount    int64
		)
		if err := rows.Scan(&service, &action, &supportCount); err != nil {
			return fmt.Errorf("failed to scan mining result: %w", err)
		}
		patternType := "trace_step"
		definition := map[string]interface{}{"steps": []map[string]string{{"service": service, "action": action}}}
		confidence := 1.0 // For now, set to 1.0; can be refined with more analytics
		metadata := map[string]interface{}{"mined_by": "trace_miner"}
		defBytes, err := json.Marshal(definition)
		if err != nil {
			return fmt.Errorf("failed to marshal mined pattern definition: %w", err)
		}
		metaBytes, err := json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal mined pattern metadata: %w", err)
		}
		_, err = r.db.ExecContext(ctx, `
			INSERT INTO service_nexus_mined_pattern (pattern_type, definition, support_count, confidence, metadata)
			VALUES ($1, $2, $3, $4, $5)
		`, patternType, defBytes, supportCount, confidence, metaBytes)
		if err != nil {
			return fmt.Errorf("failed to insert mined pattern: %w", err)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating mining results: %w", err)
	}
	return nil
}

// ListMinedPatterns returns all mined patterns.
func (r *Repository) ListMinedPatterns(ctx context.Context) ([]map[string]interface{}, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT mined_pattern_id, pattern_type, definition, support_count, confidence, mined_at, metadata
		FROM service_nexus_mined_pattern
		ORDER BY mined_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query mined patterns: %w", err)
	}
	defer rows.Close()
	results := make([]map[string]interface{}, 0, 10) // Preallocate with capacity 10 as a default
	for rows.Next() {
		var (
			minedPatternID      int64
			patternType         string
			defBytes, metaBytes []byte
			supportCount        sql.NullInt64
			confidence          sql.NullFloat64
			minedAt             sql.NullTime
		)
		if err := rows.Scan(&minedPatternID, &patternType, &defBytes, &supportCount, &confidence, &minedAt, &metaBytes); err != nil {
			return nil, fmt.Errorf("failed to scan mined pattern: %w", err)
		}
		var defVal, metaVal interface{}
		if err := json.Unmarshal(defBytes, &defVal); err != nil {
			return nil, fmt.Errorf("failed to unmarshal mined pattern definition: %w", err)
		}
		if err := json.Unmarshal(metaBytes, &metaVal); err != nil {
			return nil, fmt.Errorf("failed to unmarshal mined pattern metadata: %w", err)
		}
		result := map[string]interface{}{
			"mined_pattern_id": minedPatternID,
			"pattern_type":     patternType,
			"definition":       defVal,
			"support_count":    supportCount.Int64,
			"confidence":       confidence.Float64,
			"mined_at":         minedAt.Time,
			"metadata":         metaVal,
		}
		results = append(results, result)
	}
	return results, rows.Err()
}

// MinePatterns returns mined patterns if source == "mined", otherwise queries by origin.
func (r *Repository) MinePatterns(ctx context.Context, source string) ([]*nexusv1.Pattern, error) {
	if source == "mined" {
		rows, err := r.db.QueryContext(ctx, `
			SELECT mined_pattern_id, pattern_type, definition, support_count, confidence, mined_at, metadata
			FROM service_nexus_mined_pattern
			ORDER BY mined_at DESC
		`)
		if err != nil {
			return nil, fmt.Errorf("failed to query mined patterns: %w", err)
		}
		defer rows.Close()
		var patterns []*nexusv1.Pattern
		for rows.Next() {
			var (
				minedPatternID      int64
				patternType         string
				defBytes, metaBytes []byte
				supportCount        sql.NullInt64
				confidence          sql.NullFloat64
				minedAt             sql.NullTime
			)
			if err := rows.Scan(&minedPatternID, &patternType, &defBytes, &supportCount, &confidence, &minedAt, &metaBytes); err != nil {
				return nil, fmt.Errorf("failed to scan mined pattern: %w", err)
			}
			pat := &nexusv1.Pattern{
				PatternId:   fmt.Sprintf("mined_%d", minedPatternID),
				PatternType: patternType,
				UsageCount:  supportCount.Int64,
			}
			if err := json.Unmarshal(defBytes, pat.Definition); err != nil {
				return nil, fmt.Errorf("failed to unmarshal mined pattern definition: %w", err)
			}
			if err := json.Unmarshal(metaBytes, pat.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal mined pattern metadata: %w", err)
			}
			patterns = append(patterns, pat)
		}
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("error iterating mined patterns: %w", err)
		}
		return patterns, nil
	}
	// Fallback: query by origin
	rows, err := r.db.QueryContext(ctx, `
		SELECT pattern_id, pattern_type, version, origin, definition, usage_count, last_used, metadata
		FROM service_nexus_pattern
		WHERE origin = $1
	`, source)
	if err != nil {
		return nil, fmt.Errorf("failed to query mined patterns: %w", err)
	}
	defer rows.Close()
	var patterns []*nexusv1.Pattern
	for rows.Next() {
		var (
			pid, ptype, version, origin string
			def, meta                   []byte
			usage                       int64
			lastUsed                    sql.NullTime
		)
		pat := &nexusv1.Pattern{}
		if err := rows.Scan(&pid, &ptype, &version, &origin, &def, &usage, &lastUsed, &meta); err != nil {
			return nil, fmt.Errorf("failed to scan mined pattern: %w", err)
		}
		pat.PatternId = pid
		pat.PatternType = ptype
		pat.Version = version
		pat.Origin = origin
		pat.UsageCount = usage
		if lastUsed.Valid {
			pat.LastUsed = timestamppb.New(lastUsed.Time)
		}
		if err := json.Unmarshal(def, pat.Definition); err != nil {
			return nil, fmt.Errorf("failed to unmarshal pattern definition: %w", err)
		}
		if err := json.Unmarshal(meta, pat.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal pattern metadata: %w", err)
		}
		patterns = append(patterns, pat)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating mined patterns: %w", err)
	}
	return patterns, nil
}

// Feedback records feedback for a pattern.
func (r *Repository) Feedback(ctx context.Context, req *nexusv1.FeedbackRequest) error {
	meta, err := json.Marshal(req.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal feedback metadata: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO service_nexus_feedback (pattern_id, score, comments, metadata, created_at)
		VALUES ($1, $2, $3, $4, NOW())
	`, req.PatternId, req.Score, req.Comments, meta)
	if err != nil {
		return fmt.Errorf("failed to insert feedback: %w", err)
	}
	return nil
}

// SearchPatternsByMetadata finds patterns matching a metadata key path and value.
// Supports top-level and service_specific metadata (e.g., "tags", "service_specific.content.editor_mode").
func (r *Repository) SearchPatternsByMetadata(ctx context.Context, keyPath []string, value string) ([]*nexusv1.Pattern, error) {
	var q string
	var args []interface{}
	switch len(keyPath) {
	case 1:
		// Top-level key (e.g., tags)
		q = `SELECT pattern_id, pattern_type, version, origin, definition, usage_count, last_used, metadata FROM service_nexus_pattern WHERE metadata->>? = to_jsonb(?::text)`
		args = append(args, keyPath[0], value)
	case 2:
		// e.g., service_specific.content
		q = `SELECT pattern_id, pattern_type, version, origin, definition, usage_count, last_used, metadata FROM service_nexus_pattern WHERE metadata->?->>? = to_jsonb(?::text)`
		args = append(args, keyPath[0], keyPath[1], value)
	case 3:
		// e.g., service_specific.content.editor_mode
		q = `SELECT pattern_id, pattern_type, version, origin, definition, usage_count, last_used, metadata FROM service_nexus_pattern WHERE metadata->?->?->>? = to_jsonb(?::text)`
		args = append(args, keyPath[0], keyPath[1], keyPath[2], value)
	default:
		return nil, fmt.Errorf("unsupported key path depth")
	}
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var patterns []*nexusv1.Pattern
	for rows.Next() {
		var (
			pid, ptype, version, origin string
			def, meta                   []byte
			usage                       int64
			lastUsed                    sql.NullTime
		)
		pat := &nexusv1.Pattern{}
		if err := rows.Scan(&pid, &ptype, &version, &origin, &def, &usage, &lastUsed, &meta); err != nil {
			return nil, err
		}
		pat.PatternId = pid
		pat.PatternType = ptype
		pat.Version = version
		pat.Origin = origin
		pat.UsageCount = usage
		if lastUsed.Valid {
			pat.LastUsed = timestamppb.New(lastUsed.Time)
		}
		if err := json.Unmarshal(def, pat.Definition); err != nil {
			return nil, fmt.Errorf("failed to unmarshal pattern definition: %w", err)
		}
		if err := json.Unmarshal(meta, pat.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal pattern metadata: %w", err)
		}
		patterns = append(patterns, pat)
	}
	return patterns, rows.Err()
}

// SearchOrchestrationsByMetadata finds orchestrations matching a metadata key path and value.
func (r *Repository) SearchOrchestrationsByMetadata(ctx context.Context, keyPath []string, value string) ([]map[string]interface{}, error) {
	// Only supports up to 2-level paths for now
	var q string
	var args []interface{}
	switch len(keyPath) {
	case 1:
		q = `SELECT orchestration_id, pattern_id, input, metadata, created_by, started_at FROM service_nexus_orchestration WHERE metadata->>? = to_jsonb(?::text)`
		args = append(args, keyPath[0], value)
	case 2:
		q = `SELECT orchestration_id, pattern_id, input, metadata, created_by, started_at FROM service_nexus_orchestration WHERE metadata->?->>? = to_jsonb(?::text)`
		args = append(args, keyPath[0], keyPath[1], value)
	case 3:
		q = `SELECT orchestration_id, pattern_id, input, metadata, created_by, started_at FROM service_nexus_orchestration WHERE metadata->?->?->>? = to_jsonb(?::text)`
		args = append(args, keyPath[0], keyPath[1], keyPath[2], value)
	default:
		return nil, fmt.Errorf("unsupported key path depth")
	}
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []map[string]interface{}
	for rows.Next() {
		var (
			orchestrationID, patternID, createdBy string
			input, meta                           []byte
			startedAt                             sql.NullTime
		)
		if err := rows.Scan(&orchestrationID, &patternID, &input, &meta, &createdBy, &startedAt); err != nil {
			return nil, err
		}
		var inputVal, metaVal interface{}
		if err := json.Unmarshal(input, &inputVal); err != nil {
			return nil, fmt.Errorf("failed to unmarshal orchestration input: %w", err)
		}
		if err := json.Unmarshal(meta, &metaVal); err != nil {
			return nil, fmt.Errorf("failed to unmarshal orchestration metadata: %w", err)
		}
		result := map[string]interface{}{
			"orchestration_id": orchestrationID,
			"pattern_id":       patternID,
			"created_by":       createdBy,
			"started_at":       startedAt.Time,
			"input":            inputVal,
			"metadata":         metaVal,
		}
		results = append(results, result)
	}
	return results, rows.Err()
}

// AddTraceStep inserts a trace step for an orchestration.
func (r *Repository) AddTraceStep(ctx context.Context, tx *sql.Tx, orchestrationID, service, action string, timestamp sql.NullTime, details map[string]interface{}) error {
	var detailsBytes []byte
	var err error
	if details != nil {
		detailsBytes, err = json.Marshal(details)
		if err != nil {
			return fmt.Errorf("failed to marshal trace step details: %w", err)
		}
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO service_nexus_trace (orchestration_id, service, action, timestamp, details)
		VALUES ($1, $2, $3, COALESCE($4, NOW()), $5)
	`, orchestrationID, service, action, timestamp, detailsBytes)
	if err != nil {
		return fmt.Errorf("failed to insert trace step: %w", err)
	}
	return nil
}

// SearchPatternsByJSONPath supports arbitrary-depth metadata search using JSONPath (Postgres 12+).
func (r *Repository) SearchPatternsByJSONPath(ctx context.Context, jsonPath, value string) ([]*nexusv1.Pattern, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT pattern_id, pattern_type, version, origin, definition, usage_count, last_used, metadata
		FROM service_nexus_pattern
		WHERE jsonb_path_exists(metadata, $1, '{"val": "' || $2 || '"}')
	`, jsonPath, value)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var patterns []*nexusv1.Pattern
	for rows.Next() {
		var (
			pid, ptype, version, origin string
			def, meta                   []byte
			usage                       int64
			lastUsed                    sql.NullTime
		)
		pat := &nexusv1.Pattern{}
		if err := rows.Scan(&pid, &ptype, &version, &origin, &def, &usage, &lastUsed, &meta); err != nil {
			return nil, err
		}
		pat.PatternId = pid
		pat.PatternType = ptype
		pat.Version = version
		pat.Origin = origin
		pat.UsageCount = usage
		if lastUsed.Valid {
			pat.LastUsed = timestamppb.New(lastUsed.Time)
		}
		if err := json.Unmarshal(def, pat.Definition); err != nil {
			return nil, fmt.Errorf("failed to unmarshal pattern definition: %w", err)
		}
		if err := json.Unmarshal(meta, pat.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal pattern metadata: %w", err)
		}
		patterns = append(patterns, pat)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating patterns: %w", err)
	}
	return patterns, nil
}

// FacetPatternTags returns a count of patterns per tag for faceted search.
func (r *Repository) FacetPatternTags(ctx context.Context) (map[string]int, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT tag, COUNT(*) FROM (
			SELECT jsonb_array_elements_text(metadata->'tags') AS tag FROM service_nexus_pattern
		) t
		GROUP BY tag
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]int)
	for rows.Next() {
		var tag string
		var count int
		if err := rows.Scan(&tag, &count); err != nil {
			return nil, err
		}
		result[tag] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tag facets: %w", err)
	}
	return result, nil
}

// SearchPatternsExplainable returns patterns and the matching metadata fields for explainability.
func (r *Repository) SearchPatternsExplainable(ctx context.Context, keyPath []string, value string) ([]map[string]interface{}, error) {
	patterns, err := r.SearchPatternsByMetadata(ctx, keyPath, value)
	if err != nil {
		return nil, err
	}
	results := make([]map[string]interface{}, 0, 10)
	for _, pat := range patterns {
		matchedFields := []string{}
		// Traverse metadata to find which fields matched
		metaBytes, err := json.Marshal(pat.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata for explainability: %w", err)
		}
		var metaMap map[string]interface{}
		if err := json.Unmarshal(metaBytes, &metaMap); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata for explainability: %w", err)
		}
		// Simple check: if the value exists at the keyPath, add to matchedFields
		current := metaMap
		for i, key := range keyPath {
			if i == len(keyPath)-1 {
				if v, ok := current[key]; ok && fmt.Sprintf("%v", v) == value {
					matchedFields = append(matchedFields, key)
				}
			} else {
				next, ok := current[key].(map[string]interface{})
				if !ok {
					break
				}
				current = next
			}
		}
		results = append(results, map[string]interface{}{
			"pattern":       pat,
			"matchedFields": matchedFields,
		})
	}
	return results, nil
}

// InsertEvent persists an event to the service_nexus_event table.
func (r *Repository) InsertEvent(ctx context.Context, eventType, entityID string, metadata *commonpb.Metadata) error {
	metaJSON, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO service_nexus_event (event_type, entity_id, metadata) VALUES ($1, $2, $3)`,
		eventType, entityID, metaJSON,
	)
	return err
}
