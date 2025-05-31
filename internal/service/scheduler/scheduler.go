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
	"time"

	schedulerpb "github.com/nmxmxh/master-ovasabi/api/protos/scheduler/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

// Service implements the Scheduler business logic with rich metadata handling and gRPC server interface.
type Service struct {
	schedulerpb.UnimplementedSchedulerServiceServer // Embed for forward compatibility
	repo                                            RepositoryItf
	cache                                           *redis.Cache // Cache for future extensibility (can be nil)
	eventEmitter                                    EventEmitter
	eventEnabled                                    bool
	log                                             *zap.Logger
}

// NewService constructs a new SchedulerService.
func NewService(log *zap.Logger, repo RepositoryItf, cache *redis.Cache, eventEmitter EventEmitter, eventEnabled bool) *Service {
	return &Service{
		repo:         repo,
		cache:        cache,
		eventEmitter: eventEmitter,
		eventEnabled: eventEnabled,
		log:          log,
	}
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
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Cache: s.cache, CacheKey: job.Id, CacheValue: job, CacheTTL: 10 * time.Minute, Metadata: job.Metadata, EventEmitter: s.eventEmitter, EventEnabled: s.eventEnabled, EventType: "scheduler.job_created", EventID: job.Id, PatternType: "scheduler", PatternID: job.Id, PatternMeta: job.Metadata})
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
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Cache: s.cache, CacheKey: job.Id, CacheValue: job, CacheTTL: 10 * time.Minute, Metadata: job.Metadata, EventEmitter: s.eventEmitter, EventEnabled: s.eventEnabled, EventType: "scheduler.job_updated", EventID: job.Id, PatternType: "scheduler", PatternID: job.Id, PatternMeta: job.Metadata})
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
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Cache: s.cache, CacheKey: req.JobId, CacheValue: req.JobId, CacheTTL: 10 * time.Minute, Metadata: nil, EventEmitter: s.eventEmitter, EventEnabled: s.eventEnabled, EventType: "scheduler.job_deleted", EventID: req.JobId, PatternType: "scheduler", PatternID: req.JobId, PatternMeta: nil})
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
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Cache: s.cache, CacheKey: job.Id, CacheValue: job, CacheTTL: 5 * time.Minute, Metadata: job.Metadata, EventEmitter: s.eventEmitter, EventEnabled: s.eventEnabled, EventType: "scheduler.job_fetched", EventID: job.Id, PatternType: "scheduler", PatternID: job.Id, PatternMeta: job.Metadata})
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
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Cache: s.cache, CacheKey: "scheduler:jobs", CacheValue: resp, CacheTTL: 5 * time.Minute, Metadata: nil, EventEmitter: s.eventEmitter, EventEnabled: s.eventEnabled, EventType: "scheduler.jobs_listed", EventID: "jobs", PatternType: "scheduler", PatternID: "jobs", PatternMeta: nil})
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
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Cache: s.cache, CacheKey: run.Id, CacheValue: run, CacheTTL: 10 * time.Minute, Metadata: run.Metadata, EventEmitter: s.eventEmitter, EventEnabled: s.eventEnabled, EventType: "scheduler.job_run", EventID: run.Id, PatternType: "scheduler", PatternID: run.Id, PatternMeta: run.Metadata})
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
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Cache: s.cache, CacheKey: "scheduler:job_runs:" + req.JobId, CacheValue: resp, CacheTTL: 5 * time.Minute, Metadata: nil, EventEmitter: s.eventEmitter, EventEnabled: s.eventEnabled, EventType: "scheduler.job_runs_listed", EventID: req.JobId, PatternType: "scheduler", PatternID: req.JobId, PatternMeta: nil})
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
