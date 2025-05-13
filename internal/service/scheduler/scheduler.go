// Scheduler Service Integration Pattern
// -------------------------------------
//
// After scheduling or updating a job, use the following pattern:
//
// if job.Metadata != nil && job.Metadata.Scheduling != nil {
//     _ = pattern.RegisterSchedule(ctx, "job", job.Id, job.Metadata)
// }
// Optionally cache job metadata, enrich knowledge graph, and register with Nexus:
// if s.Cache != nil && job.Metadata != nil {
//     _ = pattern.CacheMetadata(ctx, s.Cache, "job", job.Id, job.Metadata, 10*time.Minute)
// }
// _ = pattern.EnrichKnowledgeGraph(ctx, "job", job.Id, job.Metadata)
// _ = pattern.RegisterWithNexus(ctx, "job", job.Metadata)

package scheduler

import (
	"context"
	"time"

	schedulerpb "github.com/nmxmxh/master-ovasabi/api/protos/scheduler/v1"
	schedulerrepo "github.com/nmxmxh/master-ovasabi/internal/repository/scheduler"
	pattern "github.com/nmxmxh/master-ovasabi/internal/service/pattern"
	metadatautil "github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service struct {
	Repo  schedulerrepo.Repository
	Cache *redis.Cache // optional, can be nil
	log   *zap.Logger
}

func NewService(repo schedulerrepo.Repository, cache *redis.Cache, log *zap.Logger) *Service {
	s := &Service{
		Repo:  repo,
		Cache: cache,
		log:   log,
	}
	// Register the service in the knowledge graph at startup
	if err := pattern.RegisterWithNexus(context.Background(), log, "scheduler", nil); err != nil {
		log.Error("RegisterWithNexus failed in NewService (scheduler)", zap.Error(err))
	}
	return s
}

// CreateJob creates a new scheduled job.
func (s *Service) CreateJob(ctx context.Context, req *schedulerpb.CreateJobRequest) (*schedulerpb.CreateJobResponse, error) {
	// 1. Validate metadata (if present)
	if err := metadatautil.ValidateMetadata(req.Job.Metadata); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid metadata: %v", err)
	}
	// 2. Store metadata as *common.Metadata in Postgres (jsonb) via Repo.CreateJob
	job, err := s.Repo.CreateJob(ctx, req.Job)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create job: %v", err)
	}
	// 3. Integration points
	if s.Cache != nil && job.Metadata != nil {
		if err := pattern.CacheMetadata(ctx, s.log, s.Cache, "job", job.Id, job.Metadata, 10*time.Minute); err != nil {
			s.log.Error("failed to cache metadata", zap.Error(err))
		}
	}
	if err := pattern.RegisterSchedule(ctx, s.log, "job", job.Id, job.Metadata); err != nil {
		s.log.Error("failed to register schedule", zap.Error(err))
	}
	if err := pattern.EnrichKnowledgeGraph(ctx, s.log, "job", job.Id, job.Metadata); err != nil {
		s.log.Error("failed to enrich knowledge graph", zap.Error(err))
	}
	if err := pattern.RegisterWithNexus(ctx, s.log, "job", job.Metadata); err != nil {
		s.log.Error("failed to register with nexus", zap.Error(err))
	}
	return &schedulerpb.CreateJobResponse{Job: job}, nil
}

// UpdateJob updates an existing scheduled job.
func (s *Service) UpdateJob(ctx context.Context, req *schedulerpb.UpdateJobRequest) (*schedulerpb.UpdateJobResponse, error) {
	if err := metadatautil.ValidateMetadata(req.Job.Metadata); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid metadata: %v", err)
	}
	job, err := s.Repo.UpdateJob(ctx, req.Job)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update job: %v", err)
	}
	if s.Cache != nil && job.Metadata != nil {
		if err := pattern.CacheMetadata(ctx, s.log, s.Cache, "job", job.Id, job.Metadata, 10*time.Minute); err != nil {
			s.log.Error("failed to cache metadata", zap.Error(err))
		}
	}
	if err := pattern.RegisterSchedule(ctx, s.log, "job", job.Id, job.Metadata); err != nil {
		s.log.Error("failed to register schedule", zap.Error(err))
	}
	if err := pattern.EnrichKnowledgeGraph(ctx, s.log, "job", job.Id, job.Metadata); err != nil {
		s.log.Error("failed to enrich knowledge graph", zap.Error(err))
	}
	if err := pattern.RegisterWithNexus(ctx, s.log, "job", job.Metadata); err != nil {
		s.log.Error("failed to register with nexus", zap.Error(err))
	}
	return &schedulerpb.UpdateJobResponse{Job: job}, nil
}

// DeleteJob deletes a scheduled job.
func (s *Service) DeleteJob(_ context.Context, _ *schedulerpb.DeleteJobRequest) (*schedulerpb.DeleteJobResponse, error) {
	// TODO (Amadeus Context): Implement DeleteJob following the canonical metadata pattern.
	// Reference: docs/amadeus/amadeus_context.md, section 'Canonical Metadata Integration Pattern (System-Wide)'.
	// Steps: Delete job via Repo.DeleteJob, update orchestration if metadata present, log errors.
	return nil, status.Error(codes.Unimplemented, "DeleteJob not yet implemented")
}

// ListJobs lists scheduled jobs.
func (s *Service) ListJobs(_ context.Context, _ *schedulerpb.ListJobsRequest) (*schedulerpb.ListJobsResponse, error) {
	// TODO (Amadeus Context): Implement ListJobs following the canonical metadata pattern.
	// Reference: docs/amadeus/amadeus_context.md, section 'Canonical Metadata Integration Pattern (System-Wide)'.
	// Steps: List jobs via Repo.ListJobs, include metadata, log errors.
	return nil, status.Error(codes.Unimplemented, "ListJobs not yet implemented")
}

// GetJob retrieves a scheduled job by ID.
func (s *Service) GetJob(_ context.Context, _ *schedulerpb.GetJobRequest) (*schedulerpb.GetJobResponse, error) {
	// TODO (Amadeus Context): Implement GetJob following the canonical metadata pattern.
	// Reference: docs/amadeus/amadeus_context.md, section 'Canonical Metadata Integration Pattern (System-Wide)'.
	// Steps: Get job via Repo.GetJob, include metadata, log errors.
	return nil, status.Error(codes.Unimplemented, "GetJob not yet implemented")
}

// RunJob triggers a job to run immediately.
func (s *Service) RunJob(_ context.Context, _ *schedulerpb.RunJobRequest) (*schedulerpb.RunJobResponse, error) {
	// TODO (Amadeus Context): Implement RunJob following the canonical metadata pattern.
	// Reference: docs/amadeus/amadeus_context.md, section 'Canonical Metadata Integration Pattern (System-Wide)'.
	// Steps: Run job via Repo.RunJob, include metadata, log errors.
	return nil, status.Error(codes.Unimplemented, "RunJob not yet implemented")
}

// ListJobRuns lists runs for a given job.
func (s *Service) ListJobRuns(_ context.Context, _ *schedulerpb.ListJobRunsRequest) (*schedulerpb.ListJobRunsResponse, error) {
	// TODO (Amadeus Context): Implement ListJobRuns following the canonical metadata pattern.
	// Reference: docs/amadeus/amadeus_context.md, section 'Canonical Metadata Integration Pattern (System-Wide)'.
	// Steps: List job runs via Repo.ListJobRuns, include metadata, log errors.
	return nil, status.Error(codes.Unimplemented, "ListJobRuns not yet implemented")
}
