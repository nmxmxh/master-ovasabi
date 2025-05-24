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
	"fmt"

	schedulerpb "github.com/nmxmxh/master-ovasabi/api/protos/scheduler/v1"
	events "github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
	"go.uber.org/zap"
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
		// Emit failure event
		if s.eventEnabled && s.eventEmitter != nil {
			if errEmit := s.eventEmitter.EmitEvent(ctx, "scheduler.job_create_failed", "", req.Job.Metadata); errEmit != nil {
				s.log.Warn("Failed to emit scheduler.job_create_failed event", zap.Error(errEmit))
			}
		}
		return nil, err
	}
	// Emit scheduler.job_created event after successful creation
	if s.eventEnabled && s.eventEmitter != nil {
		_, ok := events.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "scheduler.job_created", job.Id, job.Metadata)
		if !ok {
			s.log.Warn("Failed to emit scheduler.job_created event")
		}
	}
	return &schedulerpb.CreateJobResponse{Job: job}, nil
}

// UpdateJob implements the gRPC UpdateJob endpoint.
func (s *Service) UpdateJob(ctx context.Context, req *schedulerpb.UpdateJobRequest) (*schedulerpb.UpdateJobResponse, error) {
	if req.Job != nil {
		req.Job.CampaignId = req.CampaignId
	}
	job, err := s.updateJobLogic(ctx, req.Job)
	if err != nil {
		if s.eventEnabled && s.eventEmitter != nil {
			if errEmit := s.eventEmitter.EmitEvent(ctx, "scheduler.job_update_failed", req.Job.Id, req.Job.Metadata); errEmit != nil {
				s.log.Warn("Failed to emit scheduler.job_update_failed event", zap.Error(errEmit))
			}
		}
		return nil, err
	}
	if s.eventEnabled && s.eventEmitter != nil {
		_, ok := events.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "scheduler.job_updated", job.Id, job.Metadata)
		if !ok {
			s.log.Warn("Failed to emit scheduler.job_updated event")
		}
	}
	return &schedulerpb.UpdateJobResponse{Job: job}, nil
}

// DeleteJob implements the gRPC DeleteJob endpoint.
func (s *Service) DeleteJob(ctx context.Context, req *schedulerpb.DeleteJobRequest) (*schedulerpb.DeleteJobResponse, error) {
	err := s.deleteJobLogic(ctx, req.JobId)
	if err != nil {
		if s.eventEnabled && s.eventEmitter != nil {
			if errEmit := s.eventEmitter.EmitEvent(ctx, "scheduler.job_delete_failed", req.JobId, nil); errEmit != nil {
				s.log.Warn("Failed to emit scheduler.job_delete_failed event", zap.Error(errEmit))
			}
		}
		return nil, err
	}
	if s.eventEnabled && s.eventEmitter != nil {
		if err := s.eventEmitter.EmitEvent(ctx, "scheduler.job_deleted", req.JobId, nil); err != nil {
			s.log.Warn("Failed to emit scheduler.job_deleted event", zap.Error(err))
		}
	}
	return &schedulerpb.DeleteJobResponse{}, nil
}

// GetJob implements the gRPC GetJob endpoint.
func (s *Service) GetJob(ctx context.Context, req *schedulerpb.GetJobRequest) (*schedulerpb.GetJobResponse, error) {
	job, err := s.getJobLogic(ctx, req.JobId, req.CampaignId)
	if err != nil {
		return nil, err
	}
	return &schedulerpb.GetJobResponse{Job: job}, nil
}

// ListJobs implements the gRPC ListJobs endpoint.
func (s *Service) ListJobs(ctx context.Context, req *schedulerpb.ListJobsRequest) (*schedulerpb.ListJobsResponse, error) {
	jobs, total, err := s.listJobsLogic(ctx, int(req.Page), int(req.PageSize), req.Status, req.CampaignId)
	if err != nil {
		return nil, err
	}
	total32 := utils.ToInt32(total)
	return &schedulerpb.ListJobsResponse{Jobs: jobs, TotalCount: total32}, nil
}

// RunJob implements the gRPC RunJob endpoint.
func (s *Service) RunJob(ctx context.Context, req *schedulerpb.RunJobRequest) (*schedulerpb.RunJobResponse, error) {
	run, err := s.runJobLogic(ctx, req.JobId, req.CampaignId)
	if err != nil {
		if s.eventEnabled && s.eventEmitter != nil {
			if errEmit := s.eventEmitter.EmitEvent(ctx, "scheduler.job_run_failed", req.JobId, nil); errEmit != nil {
				s.log.Warn("Failed to emit scheduler.job_run_failed event", zap.Error(errEmit))
			}
		}
		return nil, err
	}
	if s.eventEnabled && s.eventEmitter != nil {
		_, ok := events.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "scheduler.job_run", req.JobId, run.Metadata)
		if !ok {
			s.log.Warn("Failed to emit scheduler.job_run event")
		}
	}
	return &schedulerpb.RunJobResponse{Run: run}, nil
}

// ListJobRuns implements the gRPC ListJobRuns endpoint.
func (s *Service) ListJobRuns(ctx context.Context, req *schedulerpb.ListJobRunsRequest) (*schedulerpb.ListJobRunsResponse, error) {
	runs, total, err := s.listJobRunsLogic(ctx, req.JobId, int(req.Page), int(req.PageSize), req.CampaignId)
	if err != nil {
		return nil, err
	}
	total32 := utils.ToInt32(total)
	return &schedulerpb.ListJobRunsResponse{Runs: runs, TotalCount: total32}, nil
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
