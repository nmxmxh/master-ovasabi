package security

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	repo "github.com/nmxmxh/master-ovasabi/internal/repository"
)

type Master struct {
	ID        string          `json:"id"`
	UUID      string          `json:"uuid"`
	Type      string          `json:"type"`
	Status    string          `json:"status"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
	DeletedAt *time.Time      `json:"deleted_at,omitempty"`
	Metadata  json.RawMessage `json:"metadata"`
}

type Identity struct {
	ID                 int64           `json:"id"`
	MasterID           int64           `json:"master_id"`
	MasterUUID         string          `json:"master_uuid"`
	IdentityType       string          `json:"identity_type"`
	Identifier         string          `json:"identifier"`
	Credentials        json.RawMessage `json:"credentials"`
	Attributes         json.RawMessage `json:"attributes"`
	LastAuthentication *time.Time      `json:"last_authentication,omitempty"`
	RiskScore          float64         `json:"risk_score"`
}

type Pattern struct {
	ID             int64           `json:"id"`
	MasterID       int64           `json:"master_id"`
	MasterUUID     string          `json:"master_uuid"`
	PatternName    string          `json:"pattern_name"`
	Description    string          `json:"description"`
	Vertices       json.RawMessage `json:"vertices"`
	Edges          json.RawMessage `json:"edges"`
	Constraints    json.RawMessage `json:"constraints"`
	RiskAssessment json.RawMessage `json:"risk_assessment"`
}

type Incident struct {
	ID             int64           `json:"id"`
	MasterID       int64           `json:"master_id"`
	MasterUUID     string          `json:"master_uuid"`
	IncidentType   string          `json:"incident_type"`
	Severity       string          `json:"severity"`
	Description    string          `json:"description"`
	DetectionTime  time.Time       `json:"detection_time"`
	ResolutionTime *time.Time      `json:"resolution_time,omitempty"`
	Context        json.RawMessage `json:"context"`
	RiskAssessment json.RawMessage `json:"risk_assessment"`
}

type Event struct {
	ID         string          `json:"id"`
	MasterID   int64           `json:"master_id"`
	EventType  string          `json:"event_type"`
	Principal  string          `json:"principal"`
	Details    json.RawMessage `json:"details"`
	OccurredAt time.Time       `json:"occurred_at"`
	Metadata   json.RawMessage `json:"metadata"`
}

type RepositoryItf interface {
	// Master
	CreateMaster(ctx context.Context, master *Master) (id string, uuid string, err error)
	GetMaster(ctx context.Context, id string) (*Master, error)
	UpdateMaster(ctx context.Context, master *Master) error

	// Identity
	CreateIdentity(ctx context.Context, identity *Identity) (int64, error)
	GetIdentity(ctx context.Context, identityType, identifier string) (*Identity, error)
	UpdateIdentityRiskScore(ctx context.Context, id int64, score float64) error

	// Pattern
	RegisterPattern(ctx context.Context, pattern *Pattern) (int64, error)
	GetPattern(ctx context.Context, patternName string) (*Pattern, error)
	ListPatterns(ctx context.Context, filter map[string]interface{}) ([]*Pattern, error)

	// Incident
	RecordIncident(ctx context.Context, incident *Incident) (int64, error)
	GetIncident(ctx context.Context, id int64) (*Incident, error)
	ListIncidents(ctx context.Context, filter map[string]interface{}) ([]*Incident, error)
	UpdateIncidentResolution(ctx context.Context, id int64, resolutionTime *time.Time) error

	// Event
	RecordEvent(ctx context.Context, event *Event) (string, error)
	GetEvents(ctx context.Context, filter map[string]interface{}) ([]*Event, error)

	// Analytics
	GetSecurityMetrics(ctx context.Context, filter map[string]interface{}) (map[string]interface{}, error)
	GetRiskAssessment(ctx context.Context, resourceID string) (map[string]interface{}, error)
}

type Repository struct {
	db         *sql.DB
	masterRepo repo.MasterRepository
}

// Compile-time check.
var _ RepositoryItf = (*Repository)(nil)

func NewRepository(db *sql.DB, masterRepo repo.MasterRepository) *Repository {
	return &Repository{db: db, masterRepo: masterRepo}
}

// Master.
func (r *Repository) CreateMaster(ctx context.Context, master *Master) (id, uuid string, err error) {
	data, err := json.Marshal(master.Metadata)
	if err != nil {
		return "", "", fmt.Errorf("marshal metadata: %w", err)
	}
	err = r.db.QueryRowContext(ctx, `
		INSERT INTO service_security_master (type, status, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		RETURNING id, uuid
	`, master.Type, master.Status, data).Scan(&id, &uuid)
	return id, uuid, err
}

func (r *Repository) GetMaster(ctx context.Context, id string) (*Master, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, uuid, type, status, created_at, updated_at, deleted_at, metadata
		FROM service_security_master WHERE id = $1
	`, id)
	var m Master
	var meta []byte
	err := row.Scan(&m.ID, &m.UUID, &m.Type, &m.Status, &m.CreatedAt, &m.UpdatedAt, &m.DeletedAt, &meta)
	if err != nil {
		return nil, err
	}
	m.Metadata = meta
	return &m, nil
}

func (r *Repository) UpdateMaster(ctx context.Context, master *Master) error {
	data, err := json.Marshal(master.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `
		UPDATE service_security_master SET type=$1, status=$2, metadata=$3, updated_at=NOW()
		WHERE id=$4
	`, master.Type, master.Status, data, master.ID)
	return err
}

// Identity.
func (r *Repository) CreateIdentity(ctx context.Context, identity *Identity) (int64, error) {
	cred, err := json.Marshal(identity.Credentials)
	if err != nil {
		return 0, fmt.Errorf("marshal credentials: %w", err)
	}
	attr, err := json.Marshal(identity.Attributes)
	if err != nil {
		return 0, fmt.Errorf("marshal attributes: %w", err)
	}
	var id int64
	err = r.db.QueryRowContext(ctx, `
		INSERT INTO service_security_identity (master_id, master_uuid, identity_type, identifier, credentials, attributes, risk_score)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, identity.MasterID, identity.MasterUUID, identity.IdentityType, identity.Identifier, cred, attr, identity.RiskScore).Scan(&id)
	return id, err
}

func (r *Repository) GetIdentity(ctx context.Context, identityType, identifier string) (*Identity, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, master_id, master_uuid, identity_type, identifier, credentials, attributes, last_authentication, risk_score
		FROM service_security_identity WHERE identity_type = $1 AND identifier = $2
	`, identityType, identifier)
	var i Identity
	var cred, attr []byte
	err := row.Scan(&i.ID, &i.MasterID, &i.MasterUUID, &i.IdentityType, &i.Identifier, &cred, &attr, &i.LastAuthentication, &i.RiskScore)
	if err != nil {
		return nil, err
	}
	i.Credentials = cred
	i.Attributes = attr
	return &i, nil
}

func (r *Repository) UpdateIdentityRiskScore(ctx context.Context, id int64, score float64) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE service_security_identity SET risk_score = $1 WHERE id = $2
	`, score, id)
	return err
}

// Pattern.
func (r *Repository) RegisterPattern(ctx context.Context, pattern *Pattern) (int64, error) {
	vertices, err := json.Marshal(pattern.Vertices)
	if err != nil {
		return 0, fmt.Errorf("marshal vertices: %w", err)
	}
	edges, err := json.Marshal(pattern.Edges)
	if err != nil {
		return 0, fmt.Errorf("marshal edges: %w", err)
	}
	constraints, err := json.Marshal(pattern.Constraints)
	if err != nil {
		return 0, fmt.Errorf("marshal constraints: %w", err)
	}
	risk, err := json.Marshal(pattern.RiskAssessment)
	if err != nil {
		return 0, fmt.Errorf("marshal risk: %w", err)
	}
	var id int64
	err = r.db.QueryRowContext(ctx, `
		INSERT INTO service_security_pattern (master_id, master_uuid, pattern_name, description, vertices, edges, constraints, risk_assessment)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`, pattern.MasterID, pattern.MasterUUID, pattern.PatternName, pattern.Description, vertices, edges, constraints, risk).Scan(&id)
	return id, err
}

func (r *Repository) GetPattern(ctx context.Context, patternName string) (*Pattern, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, master_id, master_uuid, pattern_name, description, vertices, edges, constraints, risk_assessment
		FROM service_security_pattern WHERE pattern_name = $1
	`, patternName)
	var p Pattern
	var vertices, edges, constraints, risk []byte
	err := row.Scan(&p.ID, &p.MasterID, &p.MasterUUID, &p.PatternName, &p.Description, &vertices, &edges, &constraints, &risk)
	if err != nil {
		return nil, err
	}
	p.Vertices = vertices
	p.Edges = edges
	p.Constraints = constraints
	p.RiskAssessment = risk
	return &p, nil
}

func (r *Repository) ListPatterns(ctx context.Context, filter map[string]interface{}) ([]*Pattern, error) {
	q := `SELECT id, master_id, master_uuid, pattern_name, description, vertices, edges, constraints, risk_assessment FROM service_security_pattern WHERE 1=1`
	args := []interface{}{}
	if v, ok := filter["pattern_name"]; ok {
		q += " AND pattern_name = $1"
		args = append(args, v)
	}
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var patterns []*Pattern
	for rows.Next() {
		var p Pattern
		var vertices, edges, constraints, risk []byte
		err := rows.Scan(&p.ID, &p.MasterID, &p.MasterUUID, &p.PatternName, &p.Description, &vertices, &edges, &constraints, &risk)
		if err != nil {
			return nil, err
		}
		p.Vertices = vertices
		p.Edges = edges
		p.Constraints = constraints
		p.RiskAssessment = risk
		patterns = append(patterns, &p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return patterns, nil
}

// Incident.
func (r *Repository) RecordIncident(ctx context.Context, incident *Incident) (int64, error) {
	ctxData, err := json.Marshal(incident.Context)
	if err != nil {
		return 0, fmt.Errorf("marshal context: %w", err)
	}
	risk, err := json.Marshal(incident.RiskAssessment)
	if err != nil {
		return 0, fmt.Errorf("marshal risk: %w", err)
	}
	var id int64
	err = r.db.QueryRowContext(ctx, `
		INSERT INTO service_security_incident (master_id, master_uuid, incident_type, severity, description, detection_time, resolution_time, context, risk_assessment)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`, incident.MasterID, incident.MasterUUID, incident.IncidentType, incident.Severity, incident.Description, incident.DetectionTime, incident.ResolutionTime, ctxData, risk).Scan(&id)
	return id, err
}

func (r *Repository) GetIncident(ctx context.Context, id int64) (*Incident, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, master_id, master_uuid, incident_type, severity, description, detection_time, resolution_time, context, risk_assessment
		FROM service_security_incident WHERE id = $1
	`, id)
	var inc Incident
	var ctxData, risk []byte
	err := row.Scan(&inc.ID, &inc.MasterID, &inc.MasterUUID, &inc.IncidentType, &inc.Severity, &inc.Description, &inc.DetectionTime, &inc.ResolutionTime, &ctxData, &risk)
	if err != nil {
		return nil, err
	}
	inc.Context = ctxData
	inc.RiskAssessment = risk
	return &inc, nil
}

func (r *Repository) ListIncidents(ctx context.Context, filter map[string]interface{}) ([]*Incident, error) {
	q := `SELECT id, master_id, master_uuid, incident_type, severity, description, detection_time, resolution_time, context, risk_assessment FROM service_security_incident WHERE 1=1`
	args := []interface{}{}
	if v, ok := filter["incident_type"]; ok {
		q += " AND incident_type = $1"
		args = append(args, v)
	}
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var incidents []*Incident
	for rows.Next() {
		var inc Incident
		var ctxData, risk []byte
		err := rows.Scan(&inc.ID, &inc.MasterID, &inc.MasterUUID, &inc.IncidentType, &inc.Severity, &inc.Description, &inc.DetectionTime, &inc.ResolutionTime, &ctxData, &risk)
		if err != nil {
			return nil, err
		}
		inc.Context = ctxData
		inc.RiskAssessment = risk
		incidents = append(incidents, &inc)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return incidents, nil
}

func (r *Repository) UpdateIncidentResolution(ctx context.Context, id int64, resolutionTime *time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE service_security_incident SET resolution_time = $1 WHERE id = $2
	`, resolutionTime, id)
	return err
}

// Event.
func (r *Repository) RecordEvent(ctx context.Context, event *Event) (string, error) {
	if len(event.Details) == 0 || string(event.Details) == "" {
		event.Details = []byte("{}")
	}
	if len(event.Metadata) == 0 || string(event.Metadata) == "" {
		event.Metadata = []byte("{}")
	}
	var id string
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO service_security_event (master_id, event_type, principal, details, occurred_at, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, event.MasterID, event.EventType, event.Principal, event.Details, event.OccurredAt, event.Metadata).Scan(&id)
	return id, err
}

func (r *Repository) GetEvents(ctx context.Context, filter map[string]interface{}) ([]*Event, error) {
	q := `SELECT id, master_id, event_type, principal, details, occurred_at, metadata FROM service_security_event WHERE 1=1`
	args := []interface{}{}
	if v, ok := filter["event_type"]; ok {
		q += " AND event_type = $1"
		args = append(args, v)
	}
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []*Event
	for rows.Next() {
		var e Event
		var detailsRaw, metaRaw []byte
		err := rows.Scan(&e.ID, &e.MasterID, &e.EventType, &e.Principal, &detailsRaw, &e.OccurredAt, &metaRaw)
		if err != nil {
			return nil, err
		}
		e.Details = detailsRaw
		e.Metadata = metaRaw
		events = append(events, &e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

// Analytics.
func (r *Repository) GetSecurityMetrics(ctx context.Context, _ map[string]interface{}) (map[string]interface{}, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT event_type, COUNT(*) FROM service_security_event GROUP BY event_type
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]interface{})
	for rows.Next() {
		var eventType string
		var count int
		if err := rows.Scan(&eventType, &count); err != nil {
			return nil, err
		}
		result[eventType] = count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *Repository) GetRiskAssessment(ctx context.Context, resourceID string) (map[string]interface{}, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT risk_score FROM service_security_identity WHERE identifier = $1
	`, resourceID)
	var score float64
	if err := row.Scan(&score); err != nil {
		return nil, err
	}
	return map[string]interface{}{"risk_score": score}, nil
}
