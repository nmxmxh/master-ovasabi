package schedulerrepo

import (
	"context"
	"errors"

	schedulerpb "github.com/nmxmxh/master-ovasabi/api/protos/scheduler/v1"
)

type Repository interface {
	CreateJob(ctx context.Context, job *schedulerpb.Job) (*schedulerpb.Job, error)
	UpdateJob(ctx context.Context, job *schedulerpb.Job) (*schedulerpb.Job, error)
	DeleteJob(ctx context.Context, jobID string) error
	GetJob(ctx context.Context, jobID string) (*schedulerpb.Job, error)
	ListJobs(ctx context.Context, page, pageSize int, status string) ([]*schedulerpb.Job, int, error)
	RunJob(ctx context.Context, jobID string) (*schedulerpb.JobRun, error)
	ListJobRuns(ctx context.Context, jobID string, page, pageSize int) ([]*schedulerpb.JobRun, int, error)
	// CDC event subscription (for event-driven jobs)
	SubscribeToCDCEvents(ctx context.Context, trigger *schedulerpb.CDCTrigger, handler func(event interface{}) error) error
}

type PostgresRepository struct {
	// db *sqlx.DB or pgxpool.Pool, add logger if needed
}

func NewPostgresRepository( /* db, logger */ ) *PostgresRepository {
	return &PostgresRepository{}
}

func (r *PostgresRepository) CreateJob(_ context.Context, _ *schedulerpb.Job) (*schedulerpb.Job, error) {
	// TODO: implement CreateJob logic
	return nil, errors.New("not implemented")
}

func (r *PostgresRepository) UpdateJob(_ context.Context, _ *schedulerpb.Job) (*schedulerpb.Job, error) {
	// TODO: implement UpdateJob logic
	return nil, errors.New("not implemented")
}

func (r *PostgresRepository) DeleteJob(_ context.Context, _ string) error {
	// TODO: implement DeleteJob logic
	return errors.New("not implemented")
}

func (r *PostgresRepository) GetJob(_ context.Context, _ string) (*schedulerpb.Job, error) {
	// TODO: implement GetJob logic
	return nil, errors.New("not implemented")
}

func (r *PostgresRepository) ListJobs(_ context.Context, _, _ int, _ string) ([]*schedulerpb.Job, int, error) {
	// TODO: implement ListJobs logic
	return nil, 0, errors.New("not implemented")
}

func (r *PostgresRepository) RunJob(_ context.Context, _ string) (*schedulerpb.JobRun, error) {
	// TODO: implement RunJob logic
	return nil, errors.New("not implemented")
}

func (r *PostgresRepository) ListJobRuns(_ context.Context, _ string, _, _ int) ([]*schedulerpb.JobRun, int, error) {
	// TODO: implement ListJobRuns logic
	return nil, 0, errors.New("not implemented")
}

func (r *PostgresRepository) SubscribeToCDCEvents(_ context.Context, _ *schedulerpb.CDCTrigger, _ func(event interface{}) error) error {
	// TODO: implement SubscribeToCDCEvents logic
	return errors.New("not implemented")
}
