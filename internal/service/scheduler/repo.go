package scheduler

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/nmxmxh/master-ovasabi/internal/repository"
	protojson "google.golang.org/protobuf/encoding/protojson"
	structpb "google.golang.org/protobuf/types/known/structpb"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	schedulerpb "github.com/nmxmxh/master-ovasabi/api/protos/scheduler/v1"
)

type RepositoryItf interface {
	CreateJob(ctx context.Context, job *schedulerpb.Job) (*schedulerpb.Job, error)
	UpdateJob(ctx context.Context, job *schedulerpb.Job) (*schedulerpb.Job, error)
	DeleteJob(ctx context.Context, jobID string) error
	GetJob(ctx context.Context, jobID string, campaignID int64) (*schedulerpb.Job, error)
	ListJobs(ctx context.Context, page, pageSize int, status string, campaignID int64) ([]*schedulerpb.Job, int, error)
	RunJob(ctx context.Context, jobID string, campaignID int64) (*schedulerpb.JobRun, error)
	ListJobRuns(ctx context.Context, jobID string, page, pageSize int, campaignID int64) ([]*schedulerpb.JobRun, int, error)
	// CDC event subscription (for event-driven jobs)
	SubscribeToCDCEvents(ctx context.Context, trigger *schedulerpb.CDCTrigger, handler func(event interface{}) error) error
}

type Repository struct {
	db         *sql.DB
	masterRepo repository.MasterRepository
	dsn        string // Store the DSN for CDC listeners
}

// NewRepository creates a new scheduler repository instance.
func NewRepository(db *sql.DB, masterRepo repository.MasterRepository, dsn string) *Repository {
	return &Repository{db: db, masterRepo: masterRepo, dsn: dsn}
}

func safeInt16(v int32) (int16, error) {
	if v < math.MinInt16 || v > math.MaxInt16 {
		return 0, fmt.Errorf("value %d out of int16 range", v)
	}
	return int16(v), nil
}

func (r *Repository) CreateJob(ctx context.Context, job *schedulerpb.Job) (*schedulerpb.Job, error) {
	if job == nil {
		return nil, fmt.Errorf("job is nil")
	}
	id := uuid.New()
	masterID, err := uuid.Parse(job.GetId())
	if err != nil || masterID == uuid.Nil {
		masterID = uuid.New()
	}
	metaBytes, err := marshalMetadata(job.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}
	status, err := safeInt16(int32(job.GetStatus()))
	if err != nil {
		return nil, err
	}
	triggerType, err := safeInt16(int32(job.GetTriggerType()))
	if err != nil {
		return nil, err
	}
	jobType, err := safeInt16(int32(job.GetJobType()))
	if err != nil {
		return nil, err
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO service_scheduler_job (
			id, master_id, name, schedule, payload, status, metadata, last_run_id, trigger_type, job_type, owner, next_run_time, labels, created_at, updated_at, campaign_id
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, NULL, $8, $9, $10, $11, $12, NOW(), NOW(), $13
		)
	`,
		id, masterID, job.GetName(), job.GetSchedule(), job.GetPayload(), status, metaBytes,
		triggerType, jobType, job.GetOwner(), job.GetNextRunTime(), marshalLabels(job.Labels), job.GetCampaignId(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert job: %w", err)
	}
	job.Id = id.String()
	return job, nil
}

func (r *Repository) UpdateJob(ctx context.Context, job *schedulerpb.Job) (*schedulerpb.Job, error) {
	if job == nil || job.GetId() == "" {
		return nil, fmt.Errorf("job or job id is nil")
	}
	id, err := uuid.Parse(job.GetId())
	if err != nil {
		return nil, fmt.Errorf("invalid job id: %w", err)
	}
	metaBytes, err := marshalMetadata(job.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}
	status, err := safeInt16(int32(job.GetStatus()))
	if err != nil {
		return nil, err
	}
	triggerType, err := safeInt16(int32(job.GetTriggerType()))
	if err != nil {
		return nil, err
	}
	jobType, err := safeInt16(int32(job.GetJobType()))
	if err != nil {
		return nil, err
	}
	_, err = r.db.ExecContext(ctx, `
		UPDATE service_scheduler_job SET
			name = $1, schedule = $2, payload = $3, status = $4, metadata = $5, trigger_type = $6, job_type = $7, owner = $8, next_run_time = $9, labels = $10, updated_at = NOW(), campaign_id = $11
		WHERE id = $12
	`,
		job.GetName(), job.GetSchedule(), job.GetPayload(), status, metaBytes,
		triggerType, jobType, job.GetOwner(), job.GetNextRunTime(), marshalLabels(job.Labels), job.GetCampaignId(), id,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update job: %w", err)
	}
	return job, nil
}

func (r *Repository) DeleteJob(ctx context.Context, jobID string) error {
	id, err := uuid.Parse(jobID)
	if err != nil {
		return fmt.Errorf("invalid job id: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `DELETE FROM service_scheduler_job WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete job: %w", err)
	}
	return nil
}

func (r *Repository) GetJob(ctx context.Context, jobID string, campaignID int64) (*schedulerpb.Job, error) {
	id, err := uuid.Parse(jobID)
	if err != nil {
		return nil, fmt.Errorf("invalid job id: %w", err)
	}
	var row *sql.Row
	if campaignID == 0 {
		row = r.db.QueryRowContext(ctx, `
			SELECT id, master_id, name, schedule, payload, status, metadata, last_run_id, trigger_type, job_type, owner, next_run_time, labels, created_at, updated_at, campaign_id
			FROM service_scheduler_job WHERE id = $1
		`, id)
	} else {
		row = r.db.QueryRowContext(ctx, `
			SELECT id, master_id, name, schedule, payload, status, metadata, last_run_id, trigger_type, job_type, owner, next_run_time, labels, created_at, updated_at, campaign_id
			FROM service_scheduler_job WHERE id = $1 AND campaign_id = $2
		`, id, campaignID)
	}
	var (
		jid, masterID                  uuid.UUID
		name, schedule, payload, owner string
		status, triggerType, jobType   int16
		metaBytes, labelsBytes         []byte
		lastRunID                      *uuid.UUID
		nextRunTime                    *int64
		createdAt, updatedAt           time.Time
		dbCampaignID                   int64
	)
	err = row.Scan(&jid, &masterID, &name, &schedule, &payload, &status, &metaBytes, &lastRunID, &triggerType, &jobType, &owner, &nextRunTime, &labelsBytes, &createdAt, &updatedAt, &dbCampaignID)
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}
	meta, err := unmarshalMetadata(metaBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}
	labels := unmarshalLabels(labelsBytes)
	return &schedulerpb.Job{
		Id:          jid.String(),
		Name:        name,
		Schedule:    schedule,
		Payload:     payload,
		Status:      schedulerpb.JobStatus(status),
		Metadata:    meta,
		LastRunId:   uuidPtrToString(lastRunID),
		TriggerType: schedulerpb.TriggerType(triggerType),
		JobType:     schedulerpb.JobType(jobType),
		Owner:       owner,
		NextRunTime: derefInt64(nextRunTime),
		Labels:      labels,
		CampaignId:  dbCampaignID,
	}, nil
}

func (r *Repository) ListJobs(ctx context.Context, page, pageSize int, status string, campaignID int64) ([]*schedulerpb.Job, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	var (
		jobs   []*schedulerpb.Job
		offset = (page - 1) * pageSize
		rows   *sql.Rows
		err    error
	)
	if status != "" {
		if campaignID == 0 {
			rows, err = r.db.QueryContext(ctx, `
				SELECT id, master_id, name, schedule, payload, status, metadata, last_run_id, trigger_type, job_type, owner, next_run_time, labels, created_at, updated_at, campaign_id
				FROM service_scheduler_job WHERE status = $1
				ORDER BY created_at DESC LIMIT $2 OFFSET $3
			`, status, pageSize, offset)
		} else {
			rows, err = r.db.QueryContext(ctx, `
				SELECT id, master_id, name, schedule, payload, status, metadata, last_run_id, trigger_type, job_type, owner, next_run_time, labels, created_at, updated_at, campaign_id
				FROM service_scheduler_job WHERE status = $1 AND campaign_id = $2
				ORDER BY created_at DESC LIMIT $3 OFFSET $4
			`, status, campaignID, pageSize, offset)
		}
	} else {
		if campaignID == 0 {
			rows, err = r.db.QueryContext(ctx, `
				SELECT id, master_id, name, schedule, payload, status, metadata, last_run_id, trigger_type, job_type, owner, next_run_time, labels, created_at, updated_at, campaign_id
				FROM service_scheduler_job
				ORDER BY created_at DESC LIMIT $1 OFFSET $2
			`, pageSize, offset)
		} else {
			rows, err = r.db.QueryContext(ctx, `
				SELECT id, master_id, name, schedule, payload, status, metadata, last_run_id, trigger_type, job_type, owner, next_run_time, labels, created_at, updated_at, campaign_id
				FROM service_scheduler_job WHERE campaign_id = $1
				ORDER BY created_at DESC LIMIT $2 OFFSET $3
			`, campaignID, pageSize, offset)
		}
	}
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list jobs: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			jid, masterID                  uuid.UUID
			name, schedule, payload, owner string
			status, triggerType, jobType   int16
			metaBytes, labelsBytes         []byte
			lastRunID                      *uuid.UUID
			nextRunTime                    *int64
			createdAt, updatedAt           time.Time
			campaignID                     int64
		)
		err = rows.Scan(&jid, &masterID, &name, &schedule, &payload, &status, &metaBytes, &lastRunID, &triggerType, &jobType, &owner, &nextRunTime, &labelsBytes, &createdAt, &updatedAt, &campaignID)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan job: %w", err)
		}
		meta, err := unmarshalMetadata(metaBytes)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
		labels := unmarshalLabels(labelsBytes)
		jobs = append(jobs, &schedulerpb.Job{
			Id:          jid.String(),
			Name:        name,
			Schedule:    schedule,
			Payload:     payload,
			Status:      schedulerpb.JobStatus(status),
			Metadata:    meta,
			LastRunId:   uuidPtrToString(lastRunID),
			TriggerType: schedulerpb.TriggerType(triggerType),
			JobType:     schedulerpb.JobType(jobType),
			Owner:       owner,
			NextRunTime: derefInt64(nextRunTime),
			Labels:      labels,
			CampaignId:  campaignID,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows iteration error: %w", err)
	}
	// Get total count
	var total int
	countRow := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM service_scheduler_job`)
	if err := countRow.Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to scan total count: %w", err)
	}
	return jobs, total, nil
}

func (r *Repository) RunJob(ctx context.Context, jobID string, campaignID int64) (*schedulerpb.JobRun, error) {
	// 1. Fetch the job
	job, err := r.GetJob(ctx, jobID, campaignID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch job: %w", err)
	}

	// 2. Simulate job execution (in real code, you'd dispatch to a worker or handler)
	startedAt := time.Now().Unix()
	// Simulate some work
	time.Sleep(100 * time.Millisecond)
	finishedAt := time.Now().Unix()
	status := "success"
	result := "Job executed successfully"
	runError := ""

	// 3. Insert job run record
	runID := uuid.New()
	metaBytes, err := marshalMetadata(job.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO service_scheduler_job_run (
			id, job_id, started_at, finished_at, status, result, error, metadata, created_at, campaign_id
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, NOW(), $9
		)
	`, runID, jobID, startedAt, finishedAt, status, result, runError, metaBytes, campaignID)
	if err != nil {
		return nil, fmt.Errorf("failed to insert job run: %w", err)
	}

	// 4. Update job's last_run_id
	_, err = r.db.ExecContext(ctx, `
		UPDATE service_scheduler_job SET last_run_id = $1 WHERE id = $2
	`, runID, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to update job last_run_id: %w", err)
	}

	// 5. Return the JobRun proto
	return &schedulerpb.JobRun{
		Id:         runID.String(),
		JobId:      jobID,
		StartedAt:  startedAt,
		FinishedAt: finishedAt,
		Status:     status,
		Result:     result,
		Error:      runError,
		Metadata:   job.Metadata,
	}, nil
}

func (r *Repository) ListJobRuns(ctx context.Context, jobID string, page, pageSize int, campaignID int64) ([]*schedulerpb.JobRun, int, error) {
	id, err := uuid.Parse(jobID)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid job id: %w", err)
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize
	var rows *sql.Rows
	if campaignID == 0 {
		rows, err = r.db.QueryContext(ctx, `
			SELECT id, job_id, started_at, finished_at, status, result, error, metadata, created_at
			FROM service_scheduler_job_run WHERE job_id = $1
			ORDER BY created_at DESC LIMIT $2 OFFSET $3
		`, id, pageSize, offset)
	} else {
		rows, err = r.db.QueryContext(ctx, `
			SELECT id, job_id, started_at, finished_at, status, result, error, metadata, created_at
			FROM service_scheduler_job_run WHERE job_id = $1 AND campaign_id = $2
			ORDER BY created_at DESC LIMIT $3 OFFSET $4
		`, id, campaignID, pageSize, offset)
	}
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list job runs: %w", err)
	}
	defer rows.Close()
	var runs []*schedulerpb.JobRun
	for rows.Next() {
		var (
			runID, jobID           uuid.UUID
			startedAt, finishedAt  *int64
			status, result, errStr string
			metaBytes              []byte
			createdAt              time.Time
		)
		err = rows.Scan(&runID, &jobID, &startedAt, &finishedAt, &status, &result, &errStr, &metaBytes, &createdAt)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan job run: %w", err)
		}
		meta, err := unmarshalMetadata(metaBytes)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
		runs = append(runs, &schedulerpb.JobRun{
			Id:         runID.String(),
			JobId:      jobID.String(),
			StartedAt:  derefInt64(startedAt),
			FinishedAt: derefInt64(finishedAt),
			Status:     status,
			Result:     result,
			Error:      errStr,
			Metadata:   meta,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows iteration error: %w", err)
	}
	// Get total count
	var total int
	countRow := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM service_scheduler_job_run WHERE job_id = $1`, id)
	if err := countRow.Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to scan total count: %w", err)
	}
	return runs, total, nil
}

// SubscribeToCDCEvents subscribes to CDC events on the master table using PostgreSQL LISTEN/NOTIFY.
// The handler receives the JSON payload as a string (with id and event_type).
func (r *Repository) SubscribeToCDCEvents(ctx context.Context, trigger *schedulerpb.CDCTrigger, handler func(event interface{}) error) error {
	if trigger == nil || trigger.Table != "master" || trigger.EventType == "" {
		return fmt.Errorf("invalid CDC trigger: only master table supported and event_type required")
	}
	channel := "cdc_master_" + trigger.EventType

	// Use the stored DSN for a dedicated listener connection.
	dsn := r.dsn
	listener := pq.NewListener(dsn, 10*time.Second, time.Minute, nil)
	if err := listener.Listen(channel); err != nil {
		return fmt.Errorf("failed to listen on channel %s: %w", channel, err)
	}
	go func() {
		for {
			select {
			case n := <-listener.Notify:
				if n == nil {
					continue
				}
				if err := handler(n.Extra); err != nil {
					log.Printf("CDC handler error on channel %s: %v", channel, err)
				}
			case <-ctx.Done():
				if err := listener.UnlistenAll(); err != nil {
					log.Printf("CDC listener.UnlistenAll error on channel %s: %v", channel, err)
				}
				return
			}
		}
	}()
	return nil
}

// --- Helpers ---

// marshalMetadata marshals *commonpb.Metadata to JSONB for Postgres.
func marshalMetadata(meta *commonpb.Metadata) ([]byte, error) {
	if meta == nil {
		return []byte("{}"), nil
	}
	return protojson.Marshal(meta)
}

// unmarshalMetadata unmarshals JSONB to *commonpb.Metadata.
func unmarshalMetadata(b []byte) (*commonpb.Metadata, error) {
	if len(b) == 0 {
		return &commonpb.Metadata{}, nil
	}
	meta := &commonpb.Metadata{}
	if err := protojson.Unmarshal(b, meta); err != nil {
		return nil, err
	}
	return meta, nil
}

// marshalLabels marshals a map[string]string to JSONB.
func marshalLabels(labels map[string]string) []byte {
	if labels == nil {
		return []byte("{}")
	}
	b, err := protojson.Marshal(&structpb.Struct{Fields: toStructFields(labels)})
	if err != nil {
		return []byte("{}")
	}
	return b
}

// unmarshalLabels unmarshals JSONB to map[string]string.
func unmarshalLabels(b []byte) map[string]string {
	if len(b) == 0 {
		return map[string]string{}
	}
	var s structpb.Struct
	if err := protojson.Unmarshal(b, &s); err != nil {
		return map[string]string{}
	}
	m := map[string]string{}
	for k, v := range s.Fields {
		m[k] = v.GetStringValue()
	}
	return m
}

// toStructFields converts map[string]string to map[string]*structpb.Value.
func toStructFields(m map[string]string) map[string]*structpb.Value {
	fields := make(map[string]*structpb.Value, len(m))
	for k, v := range m {
		fields[k] = structpb.NewStringValue(v)
	}
	return fields
}

func uuidPtrToString(u *uuid.UUID) string {
	if u == nil {
		return ""
	}
	return u.String()
}

func derefInt64(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}
