// OVASABI Scheduler Service (Canonical Pattern)
// --------------------------------------------
// Implements the canonical service pattern with robust metadata integration.
// See: docs/amadeus/amadeus_context.md, docs/services/metadata.md

// Provider/DI Registration Pattern (Modern, Extensible, DRY)
// ---------------------------------------------------------
//
// This file implements the centralized Provider pattern for service registration and dependency injection (DI) for the Scheduler service.
// It ensures the SchedulerService is registered, resolved, and composed in a DRY, maintainable, and extensible way.
//
// Key Features:
// - Centralized Service Registration: SchedulerService is registered with a DI container, ensuring single-point, modular registration and easy dependency management.
// - Repository & Cache Integration: The service specifies its repository constructor and cache name for Redis-backed caching.
// - Extensible Pattern: To add or update the service, define its repository and cache, then add a registration entry in the provider.
// - Consistent Error Handling: All registration errors are logged and wrapped for traceability.
// - Self-Documenting: The registration pattern is discoverable and enforced as a standard for all new service/provider files.
//
// Standard for New Service/Provider Files:
// 1. Document the registration pattern and DI approach at the top of the file.
// 2. Describe how to add new services, including repository, cache, and dependency resolution.
// 3. Note any special patterns for multi-dependency or cross-service orchestration.
// 4. Ensure all registration and error handling is consistent and logged.
// 5. Reference this comment as the standard for all new service/provider files.

package scheduler

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	schedulerpb "github.com/nmxmxh/master-ovasabi/api/protos/scheduler/v1"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	metadatautil "github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

// EventEmitterAdapter bridges any EventEmitter to the canonical interface.
type EventEmitterAdapter struct {
	Emitter EventEmitter
}

func (a *EventEmitterAdapter) EmitEventWithLogging(ctx context.Context, event interface{}, log *zap.Logger, eventType, eventID string, meta *commonpb.Metadata) (string, bool) {
	if a.Emitter != nil {
		return a.Emitter.EmitEventWithLogging(ctx, event, log, eventType, eventID, meta)
	}
	return "", false
}

func (a *EventEmitterAdapter) EmitRawEventWithLogging(ctx context.Context, log *zap.Logger, eventType, eventID string, payload []byte) (string, bool) {
	if emitter, ok := a.Emitter.(interface {
		EmitRawEventWithLogging(context.Context, *zap.Logger, string, string, []byte) (string, bool)
	}); ok {
		return emitter.EmitRawEventWithLogging(ctx, log, eventType, eventID, payload)
	}
	return "", false
}

// Handler registry for job execution.
var jobExecutionHandlers = map[string]func(ctx context.Context, provider *service.Provider, job *schedulerpb.Job, log *zap.Logger){
	"payday": HandlePaydayJob, // canonical handler for payday jobs
}

// Service implements the Scheduler business logic with rich metadata handling and gRPC server interface.
type Service struct {
	schedulerpb.UnimplementedSchedulerServiceServer // Embed for forward compatibility
	repo                                            RepositoryItf
	cache                                           *redis.Cache // Cache for future extensibility (can be nil)
	eventEmitter                                    EventEmitter
	eventEnabled                                    bool
	log                                             *zap.Logger
	provider                                        *service.Provider
	stopCleaner                                     chan struct{}
	cleanerWG                                       sync.WaitGroup
	cronScheduler                                   *cron.Cron
	stopScheduler                                   chan struct{}
}

// NewService constructs a new SchedulerService.
func NewService(ctx context.Context, log *zap.Logger, repo RepositoryItf, cache *redis.Cache, eventEmitter EventEmitter, eventEnabled bool, provider *service.Provider) *Service {
	svc := &Service{
		repo:          repo,
		cache:         cache,
		eventEmitter:  eventEmitter,
		eventEnabled:  eventEnabled,
		log:           log,
		provider:      provider,
		stopCleaner:   make(chan struct{}),
		cronScheduler: cron.New(cron.WithSeconds()),
		stopScheduler: make(chan struct{}),
	}
	svc.startCleanerLoop(ctx)
	svc.startAdvancedSchedulerLoop(ctx)
	svc.subscribeJobEvents()
	return svc
}

// Ensure Service implements schedulerpb.SchedulerServiceServer.
var _ schedulerpb.SchedulerServiceServer = (*Service)(nil)

// --- gRPC SchedulerServiceServer Implementation ---

// CreateJob implements the gRPC CreateJob endpoint.
func (s *Service) CreateJob(ctx context.Context, req *schedulerpb.CreateJobRequest) (*schedulerpb.CreateJobResponse, error) {
	if req.Job != nil {
		req.Job.CampaignId = req.CampaignId
	}
	job, err := s.createJobLogic(ctx, req.Job)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to create job", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "job created", job, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Cache: s.cache, CacheKey: job.Id, CacheValue: job, CacheTTL: 10 * time.Minute, Metadata: job.Metadata, EventEmitter: &EventEmitterAdapter{Emitter: s.eventEmitter}, EventEnabled: s.eventEnabled, EventType: "scheduler.job_created", EventID: job.Id, PatternType: "scheduler", PatternID: job.Id, PatternMeta: job.Metadata})
	return &schedulerpb.CreateJobResponse{Job: job}, nil
}

// UpdateJob implements the gRPC UpdateJob endpoint.
func (s *Service) UpdateJob(ctx context.Context, req *schedulerpb.UpdateJobRequest) (*schedulerpb.UpdateJobResponse, error) {
	if req.Job != nil {
		req.Job.CampaignId = req.CampaignId
	}
	job, err := s.updateJobLogic(ctx, req.Job)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to update job", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "job updated", job, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Cache: s.cache, CacheKey: job.Id, CacheValue: job, CacheTTL: 10 * time.Minute, Metadata: job.Metadata, EventEmitter: &EventEmitterAdapter{Emitter: s.eventEmitter}, EventEnabled: s.eventEnabled, EventType: "scheduler.job_updated", EventID: job.Id, PatternType: "scheduler", PatternID: job.Id, PatternMeta: job.Metadata})
	return &schedulerpb.UpdateJobResponse{Job: job}, nil
}

// DeleteJob implements the gRPC DeleteJob endpoint.
func (s *Service) DeleteJob(ctx context.Context, req *schedulerpb.DeleteJobRequest) (*schedulerpb.DeleteJobResponse, error) {
	err := s.deleteJobLogic(ctx, req.JobId)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to delete job", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "job deleted", req.JobId, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Cache: s.cache, CacheKey: req.JobId, CacheValue: req.JobId, CacheTTL: 10 * time.Minute, Metadata: nil, EventEmitter: &EventEmitterAdapter{Emitter: s.eventEmitter}, EventEnabled: s.eventEnabled, EventType: "scheduler.job_deleted", EventID: req.JobId, PatternType: "scheduler", PatternID: req.JobId, PatternMeta: nil})
	return &schedulerpb.DeleteJobResponse{}, nil
}

// GetJob implements the gRPC GetJob endpoint.
func (s *Service) GetJob(ctx context.Context, req *schedulerpb.GetJobRequest) (*schedulerpb.GetJobResponse, error) {
	job, err := s.getJobLogic(ctx, req.JobId, req.CampaignId)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to get job", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	resp := &schedulerpb.GetJobResponse{Job: job}
	success := graceful.WrapSuccess(ctx, codes.OK, "job fetched", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Cache: s.cache, CacheKey: job.Id, CacheValue: job, CacheTTL: 5 * time.Minute, Metadata: job.Metadata, EventEmitter: &EventEmitterAdapter{Emitter: s.eventEmitter}, EventEnabled: s.eventEnabled, EventType: "scheduler.job_fetched", EventID: job.Id, PatternType: "scheduler", PatternID: job.Id, PatternMeta: job.Metadata})
	return resp, nil
}

// ListJobs implements the gRPC ListJobs endpoint.
func (s *Service) ListJobs(ctx context.Context, req *schedulerpb.ListJobsRequest) (*schedulerpb.ListJobsResponse, error) {
	jobs, total, err := s.listJobsLogic(ctx, int(req.Page), int(req.PageSize), req.Status, req.CampaignId)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to list jobs", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	total32 := utils.ToInt32(total)
	resp := &schedulerpb.ListJobsResponse{Jobs: jobs, TotalCount: total32}
	success := graceful.WrapSuccess(ctx, codes.OK, "jobs listed", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Cache: s.cache, CacheKey: "scheduler:jobs", CacheValue: resp, CacheTTL: 5 * time.Minute, Metadata: nil, EventEmitter: &EventEmitterAdapter{Emitter: s.eventEmitter}, EventEnabled: s.eventEnabled, EventType: "scheduler.jobs_listed", EventID: "jobs", PatternType: "scheduler", PatternID: "jobs", PatternMeta: nil})
	return resp, nil
}

// RunJob implements the gRPC RunJob endpoint.
func (s *Service) RunJob(ctx context.Context, req *schedulerpb.RunJobRequest) (*schedulerpb.RunJobResponse, error) {
	run, err := s.runJobLogic(ctx, req.JobId, req.CampaignId)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to run job", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "job run", run, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Cache: s.cache, CacheKey: run.Id, CacheValue: run, CacheTTL: 10 * time.Minute, Metadata: run.Metadata, EventEmitter: &EventEmitterAdapter{Emitter: s.eventEmitter}, EventEnabled: s.eventEnabled, EventType: "scheduler.job_run", EventID: run.Id, PatternType: "scheduler", PatternID: run.Id, PatternMeta: run.Metadata})
	return &schedulerpb.RunJobResponse{Run: run}, nil
}

// ListJobRuns implements the gRPC ListJobRuns endpoint.
func (s *Service) ListJobRuns(ctx context.Context, req *schedulerpb.ListJobRunsRequest) (*schedulerpb.ListJobRunsResponse, error) {
	runs, total, err := s.listJobRunsLogic(ctx, req.JobId, int(req.Page), int(req.PageSize), req.CampaignId)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to list job runs", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	total32 := utils.ToInt32(total)
	resp := &schedulerpb.ListJobRunsResponse{Runs: runs, TotalCount: total32}
	success := graceful.WrapSuccess(ctx, codes.OK, "job runs listed", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Cache: s.cache, CacheKey: "scheduler:job_runs:" + req.JobId, CacheValue: resp, CacheTTL: 5 * time.Minute, Metadata: nil, EventEmitter: &EventEmitterAdapter{Emitter: s.eventEmitter}, EventEnabled: s.eventEnabled, EventType: "scheduler.job_runs_listed", EventID: req.JobId, PatternType: "scheduler", PatternID: req.JobId, PatternMeta: nil})
	return resp, nil
}

// --- Business Logic Methods (renamed to avoid conflicts) ---

// createJobLogic creates a new scheduler job with robust metadata validation and enrichment.
func (s *Service) createJobLogic(ctx context.Context, req *schedulerpb.Job) (*schedulerpb.Job, error) {
	meta, err := ExtractSchedulerMetadata(req.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to extract scheduler metadata: %w", err)
	}
	if err := ValidateSchedulerMetadata(meta); err != nil {
		return nil, fmt.Errorf("invalid scheduler metadata: %w", err)
	}
	if err := EnrichSchedulerMetadata(req.Metadata, meta); err != nil {
		return nil, fmt.Errorf("failed to enrich scheduler metadata: %w", err)
	}
	return s.repo.CreateJob(ctx, req)
}

// updateJobLogic updates an existing scheduler job with metadata validation.
func (s *Service) updateJobLogic(ctx context.Context, req *schedulerpb.Job) (*schedulerpb.Job, error) {
	meta, err := ExtractSchedulerMetadata(req.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to extract scheduler metadata: %w", err)
	}
	if err := ValidateSchedulerMetadata(meta); err != nil {
		return nil, fmt.Errorf("invalid scheduler metadata: %w", err)
	}
	if err := EnrichSchedulerMetadata(req.Metadata, meta); err != nil {
		return nil, fmt.Errorf("failed to enrich scheduler metadata: %w", err)
	}
	return s.repo.UpdateJob(ctx, req)
}

// deleteJobLogic deletes a scheduler job by ID.
func (s *Service) deleteJobLogic(ctx context.Context, jobID string) error {
	return s.repo.DeleteJob(ctx, jobID)
}

// getJobLogic retrieves a scheduler job by ID, with metadata extraction.
func (s *Service) getJobLogic(ctx context.Context, jobID string, campaignID int64) (*schedulerpb.Job, error) {
	job, err := s.repo.GetJob(ctx, jobID, campaignID)
	if err != nil {
		return nil, err
	}
	if _, err := ExtractSchedulerMetadata(job.Metadata); err != nil {
		return nil, fmt.Errorf("failed to extract scheduler metadata: %w", err)
	}
	return job, nil
}

// listJobsLogic lists scheduler jobs with metadata extraction.
func (s *Service) listJobsLogic(ctx context.Context, page, pageSize int, status string, campaignID int64) ([]*schedulerpb.Job, int, error) {
	jobs, total, err := s.repo.ListJobs(ctx, page, pageSize, status, campaignID)
	if err != nil {
		return nil, 0, err
	}
	for _, job := range jobs {
		if _, err := ExtractSchedulerMetadata(job.Metadata); err != nil {
			return nil, 0, fmt.Errorf("failed to extract scheduler metadata: %w", err)
		}
	}
	return jobs, total, nil
}

// runJobLogic executes a scheduler job and records the run with metadata.
func (s *Service) runJobLogic(ctx context.Context, jobID string, campaignID int64) (*schedulerpb.JobRun, error) {
	run, err := s.repo.RunJob(ctx, jobID, campaignID)
	if err != nil {
		return nil, err
	}
	if _, err := ExtractSchedulerMetadata(run.Metadata); err != nil {
		return nil, fmt.Errorf("failed to extract scheduler metadata: %w", err)
	}
	// --- Custom handler dispatch ---
	if run != nil && run.JobId != "" {
		job, err := s.repo.GetJob(ctx, run.JobId, campaignID)
		if err != nil {
			zap.L().Warn("Failed to get job", zap.Error(err))
		} else if job != nil {
			if job.Name != "" && len(job.Name) >= 16 && job.Name[:16] == "payday_monday_9am" {
				handler := jobExecutionHandlers["payday"]
				if handler != nil && s.provider != nil {
					handler(ctx, s.provider, job, s.log)
				}
			}
		}
	}
	return run, nil
}

// listJobRunsLogic lists job runs for a given job ID, with metadata extraction.
func (s *Service) listJobRunsLogic(ctx context.Context, jobID string, page, pageSize int, campaignID int64) ([]*schedulerpb.JobRun, int, error) {
	runs, total, err := s.repo.ListJobRuns(ctx, jobID, page, pageSize, campaignID)
	if err != nil {
		return nil, 0, err
	}
	for _, run := range runs {
		if _, err := ExtractSchedulerMetadata(run.Metadata); err != nil {
			return nil, 0, fmt.Errorf("failed to extract scheduler metadata: %w", err)
		}
	}
	return runs, total, nil
}

// startCleanerLoop launches the background cleaner goroutine.
func (s *Service) startCleanerLoop(ctx context.Context) {
	s.cleanerWG.Add(1)
	go func() {
		defer s.cleanerWG.Done()
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.runCleaner(ctx)
			case <-s.stopCleaner:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

// runCleaner scans for jobs to archive or delete based on metadata.
func (s *Service) runCleaner(ctx context.Context) {
	jobs, _, err := s.listJobsLogic(ctx, 1, 1000, "", 0)
	if err != nil {
		s.log.Warn("Cleaner: failed to list jobs", zap.Error(err))
		return
	}
	now := time.Now()
	for _, job := range jobs {
		// Extract scheduler service-specific variables from metadata
		vars := metadatautil.ExtractServiceVariables(job.Metadata, "scheduler")
		archiveAfter, err := getStringFromMap(vars, "archive_after")
		if err != nil {
			s.log.Warn("Cleaner: failed to get archive_after", zap.String("job_id", job.Id), zap.Error(err))
			continue
		}
		deleteAfter, err := getStringFromMap(vars, "delete_after")
		if err != nil {
			s.log.Warn("Cleaner: failed to get delete_after", zap.String("job_id", job.Id), zap.Error(err))
			continue
		}
		status := job.Status.String()
		var createdAt time.Time
		switch t := any(job.CreatedAt).(type) {
		case time.Time:
			createdAt = t
		case string:
			if t, err := time.Parse(time.RFC3339, t); err == nil {
				createdAt = t
			}
		case float64:
			createdAt = time.Unix(int64(t), 0)
		case int64:
			createdAt = time.Unix(t, 0)
		default:
			createdAt = time.Now()
		}
		if archiveAfter != "" && status == "COMPLETED" {
			if d, err := time.ParseDuration(archiveAfter); err == nil && now.After(createdAt.Add(d)) {
				err := s.repo.ArchiveJob(ctx, job.Id)
				if err != nil {
					s.log.Warn("Cleaner: failed to archive job", zap.String("job_id", job.Id), zap.Error(err))
				} else {
					s.log.Info("Cleaner: archived job", zap.String("job_id", job.Id))
				}
			}
		}
		if deleteAfter != "" {
			if d, err := time.ParseDuration(deleteAfter); err == nil && now.After(createdAt.Add(d)) {
				err := s.repo.DeleteJob(ctx, job.Id)
				if err != nil {
					s.log.Warn("Cleaner: failed to delete job", zap.String("job_id", job.Id), zap.Error(err))
				} else {
					s.log.Info("Cleaner: deleted job", zap.String("job_id", job.Id))
				}
			}
		}
	}
}

// StopCleaner gracefully stops the cleaner goroutine.
func (s *Service) StopCleaner() {
	close(s.stopCleaner)
	s.cleanerWG.Wait()
}

// startAdvancedSchedulerLoop launches the cron-based advanced scheduler goroutine.
func (s *Service) startAdvancedSchedulerLoop(ctx context.Context) {
	go s.runAdvancedScheduler(ctx)
}

// runAdvancedScheduler loads jobs with cron expressions and schedules them with time zone and retry/backoff support.
func (s *Service) runAdvancedScheduler(ctx context.Context) {
	jobs, _, err := s.listJobsLogic(ctx, 1, 1000, "ACTIVE", 0)
	if err != nil {
		s.log.Warn("AdvancedScheduler: failed to list jobs", zap.Error(err))
		return
	}
	for _, job := range jobs {
		meta := metadatautil.ProtoToMap(job.Metadata)
		// Parse cron expression and timezone from metadata.scheduling
		sched, ok := meta["scheduling"].(map[string]interface{})
		if !ok {
			continue
		}
		cronExpr, ok := sched["cron"].(string)
		if !ok || cronExpr == "" {
			continue
		}
		tz := "UTC"
		if tzVal, ok := sched["timezone"].(string); ok && tzVal != "" {
			tz = tzVal
		}
		loc, err := time.LoadLocation(tz)
		if err != nil {
			s.log.Warn("Invalid timezone in job metadata", zap.String("job_id", job.Id), zap.String("tz", tz), zap.Error(err))
			loc = time.UTC
		}
		// Schedule the job with cron
		jobID := job.Id
		if _, err := s.cronScheduler.AddFunc(cronExpr, func() {
			now := time.Now().In(loc)
			// Check window constraints
			windowOK := true
			if window, ok := sched["window"].(map[string]interface{}); ok {
				startStr, err := getStringFromMap(window, "start")
				if err != nil {
					s.log.Error("Failed to get window start time", zap.String("job_id", jobID), zap.Error(err))
					return
				}
				endStr, err := getStringFromMap(window, "end")
				if err != nil {
					s.log.Error("Failed to get window end time", zap.String("job_id", jobID), zap.Error(err))
					return
				}
				if startStr != "" && endStr != "" {
					start, err1 := time.Parse(time.RFC3339, startStr)
					end, err2 := time.Parse(time.RFC3339, endStr)
					if err1 == nil && err2 == nil {
						if now.Before(start) || now.After(end) {
							windowOK = false
						}
					}
				}
			}
			if !windowOK {
				s.log.Info("Job not run: outside window", zap.String("job_id", jobID))
				return
			}
			// Check dependencies (not implemented: stub)
			// TODO: Implement dependency check if needed
			// Retry/backoff policy
			operation := func() error {
				_, err := s.runJobLogic(ctx, jobID, job.CampaignId)
				return err
			}
			var bo backoff.BackOff = backoff.NewExponentialBackOff()
			maxAttempts := 3
			if retry, ok := sched["retry_policy"].(map[string]interface{}); ok {
				if ma, ok := retry["max_attempts"].(float64); ok {
					maxAttempts = int(ma)
				}
				if boType, ok := retry["backoff"].(string); ok && boType == "constant" {
					bo = backoff.NewConstantBackOff(5 * time.Second)
				}
			}
			attempts := 0
			err := backoff.Retry(func() error {
				if attempts >= maxAttempts {
					return nil // stop retrying
				}
				attempts++
				return operation()
			}, bo)
			if err != nil {
				s.log.Warn("Job failed after retries", zap.String("job_id", jobID), zap.Error(err))
			} else {
				s.log.Info("Job run succeeded", zap.String("job_id", jobID))
			}
		}); err != nil {
			s.log.Error("failed to add cron job", zap.Error(err), zap.String("cron_expr", cronExpr))
			return
		}
	}
	s.cronScheduler.Start()
	<-s.stopScheduler // Wait for stop signal
	s.cronScheduler.Stop()
}

// StopAdvancedScheduler gracefully stops the advanced scheduler goroutine.
func (s *Service) StopAdvancedScheduler() {
	close(s.stopScheduler)
}

// subscribeJobEvents subscribes to job CRUD events and updates the scheduler in real time.
func (s *Service) subscribeJobEvents() {
	go func() {
		// Pseudocode: Replace with actual event bus subscription logic
		for {
			// Wait for job event (created, updated, deleted)
			// event := <-eventBus.JobEvents
			// s.onJobChanged(event.Job, event.Action)
			// For demo, sleep
			time.Sleep(10 * time.Second)
		}
	}()
}

// GetJobStatus gets the status of a job.
func (s *Service) GetJobStatus(_ context.Context, _ string) (status string, err error) {
	if s.log != nil {
		s.log.Debug("Getting job status")
	}
	return "unknown", nil
}

// --- UI/Graph Endpoints (stubs) ---
// GetJobGraph returns the job dependency graph for UI visualization.
func (s *Service) GetJobGraph(_ context.Context) (nodes []string, edges [][2]string, err error) {
	// TODO: Implement real graph extraction from jobs/dependencies
	return nil, nil, nil
}

// RegisterJobPattern registers a job as a pattern in Nexus (stub).
func (s *Service) RegisterJobPattern(_ *schedulerpb.Job) error {
	// TODO: Implement pattern registration with Nexus
	return nil
}

// TriggerJobFromOrchestration triggers a job as part of an orchestration flow (stub).
func (s *Service) TriggerJobFromOrchestration(_ string, _ map[string]interface{}) error {
	// TODO: Implement orchestration-triggered job execution
	return nil
}

// Helper function to safely get string from map.
func getStringFromMap(m map[string]interface{}, key string) (string, error) {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str, nil
		}
		return "", fmt.Errorf("value for key %s is not a string", key)
	}
	return "", fmt.Errorf("key %s not found", key)
}
