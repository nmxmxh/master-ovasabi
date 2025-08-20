package crawler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	crawlerpb "github.com/nmxmxh/master-ovasabi/api/protos/crawler/v1"
	"github.com/nmxmxh/master-ovasabi/internal/service/crawler/workers"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
)

// --- Graceful Orchestration Adapter for EventEmitter (for DRY orchestration) ---

// --- Service and Dispatcher Types ---.
type Service struct {
	crawlerpb.UnimplementedCrawlerServiceServer
	log          *zap.Logger
	repo         *Repository
	cache        *redis.Cache
	eventEmitter events.EventEmitter
	eventEnabled bool
	dispatcher   *Dispatcher
	shutdown     chan struct{}
	handler      *graceful.Handler
}

type Dispatcher struct {
	log          *zap.Logger
	repo         *Repository
	workerMap    map[crawlerpb.TaskType]workers.Worker
	jobQueue     chan *crawlerpb.CrawlTask
	results      chan *crawlerpb.CrawlResult
	workerWg     sync.WaitGroup
	processorWg  sync.WaitGroup
	shutdownOnce sync.Once
}

type WorkerFactory func() workers.Worker

// --- Constants for worker pool ---.
const (
	DefaultWorkerCount = 4
	DefaultQueueSize   = 100
	RateLimit          = 100 * time.Millisecond
)

// --- CrawlerService gRPC Methods ---

// Ensure gRPC interface compliance.
var _ crawlerpb.CrawlerServiceServer = (*Service)(nil)

// SubmitTask submits a new crawl task and orchestrates via graceful.
func (s *Service) SubmitTask(ctx context.Context, req *crawlerpb.SubmitTaskRequest) (*crawlerpb.SubmitTaskResponse, error) {
	if req == nil || req.Task == nil {
		gErr := graceful.WrapErr(ctx, codes.InvalidArgument, "missing task in request", nil)
		s.handler.Error(ctx, "submit_task", codes.InvalidArgument, "missing task in request", gErr, nil, "")
		return nil, graceful.ToStatusError(gErr)
	}

	// Normalize and version metadata
	if req.Task.Metadata == nil {
		req.Task.Metadata = &commonpb.Metadata{}
	}
	// Optionally: set versioning fields here if needed

	req.Task.Status = crawlerpb.TaskStatus_TASK_STATUS_PENDING
	createdTask, err := s.repo.CreateCrawlTask(ctx, req.Task)
	if err != nil {
		s.log.Error("failed to create crawl task", zap.Error(err))
		gErr := graceful.WrapErr(ctx, codes.Internal, "failed to create crawl task", err)
		s.handler.Error(ctx, "submit_task", codes.Internal, "failed to create crawl task", gErr, req.Task.Metadata, req.Task.Uuid)
		return nil, graceful.ToStatusError(gErr)
	}

	// Enqueue the task for processing
	select {
	case s.dispatcher.jobQueue <- createdTask:
		// ok
	default:
		gErr := graceful.WrapErr(ctx, codes.ResourceExhausted, "task queue is full", nil)
		s.handler.Error(ctx, "submit_task", codes.ResourceExhausted, "task queue is full", gErr, createdTask.Metadata, createdTask.Uuid)
		return nil, graceful.ToStatusError(gErr)
	}

	resp := &crawlerpb.SubmitTaskResponse{
		Uuid:    createdTask.Uuid,
		Status:  createdTask.Status,
		Message: "Task submitted successfully",
	}
	s.handler.Success(ctx, "submit_task", codes.OK, "crawler task submitted", resp, createdTask.Metadata, createdTask.Uuid, &graceful.CacheInfo{Key: createdTask.Uuid, Value: createdTask, TTL: 10 * time.Minute})
	return resp, nil
}

// GetTaskStatus returns the current status of a crawl task.
func (s *Service) GetTaskStatus(ctx context.Context, req *crawlerpb.GetTaskStatusRequest) (*crawlerpb.CrawlTask, error) {
	if req == nil || req.Uuid == "" {
		gErr := graceful.WrapErr(ctx, codes.InvalidArgument, "missing uuid in request", nil)
		s.handler.Error(ctx, "get_task_status", codes.InvalidArgument, "missing uuid in request", gErr, nil, "")
		return nil, graceful.ToStatusError(gErr)
	}
	task, err := s.repo.GetCrawlTask(ctx, req.Uuid)
	if err != nil {
		s.log.Error("failed to get crawl task", zap.Error(err))
		gErr := graceful.WrapErr(ctx, codes.NotFound, "crawl task not found", err)
		s.handler.Error(ctx, "get_task_status", codes.NotFound, "crawl task not found", gErr, nil, req.Uuid)
		return nil, graceful.ToStatusError(gErr)
	}
	// Optionally orchestrate success (not always needed for read-only)
	return task, nil
}

// StreamResults streams crawl results for a given task UUID.
func (s *Service) StreamResults(req *crawlerpb.StreamResultsRequest, stream crawlerpb.CrawlerService_StreamResultsServer) error {
	ctx := stream.Context()
	if req == nil || req.TaskUuid == "" {
		gErr := graceful.WrapErr(ctx, codes.InvalidArgument, "missing task_uuid in request", nil)
		s.handler.Error(ctx, "stream_results", codes.InvalidArgument, "missing task_uuid in request", gErr, nil, "")
		return graceful.ToStatusError(gErr)
	}

	// For demo: just send the latest result (extend to real streaming if needed)
	result, err := s.repo.GetCrawlResult(ctx, req.TaskUuid)
	if err != nil {
		s.log.Error("failed to get crawl result", zap.Error(err))
		gErr := graceful.WrapErr(ctx, codes.NotFound, "crawl result not found", err)
		s.handler.Error(ctx, "stream_results", codes.NotFound, "crawl result not found", gErr, nil, req.TaskUuid)
		return graceful.ToStatusError(gErr)
	}

	if err := stream.Send(result); err != nil {
		s.log.Error("failed to stream result", zap.Error(err))
		gErr := graceful.WrapErr(ctx, codes.Internal, "failed to stream result", err)
		s.handler.Error(ctx, "stream_results", codes.Internal, "failed to stream result", gErr, nil, req.TaskUuid)
		return graceful.ToStatusError(gErr)
	}
	// Optionally orchestrate success (not always needed for streaming)
	return nil
}

// NewService creates a new crawler service with proper dependency injection.
func NewService(
	ctx context.Context,
	log *zap.Logger,
	repo *Repository,
	cache *redis.Cache,
	eventEmitter events.EventEmitter,
	eventEnabled bool,
	workerFactories map[crawlerpb.TaskType]WorkerFactory,
) (*Service, error) {
	// Reference unused ctx for diagnostics
	if ctx != nil && ctx.Err() != nil {
		log.Warn("Context error in NewService", zap.Error(ctx.Err()))
	}

	dispatcher, err := NewDispatcher(log, repo, workerFactories)
	if err != nil {
		return nil, fmt.Errorf("failed to create dispatcher: %w", err)
	}

	svc := &Service{
		log:          log,
		repo:         repo,
		cache:        cache,
		eventEmitter: eventEmitter,
		eventEnabled: eventEnabled,
		dispatcher:   dispatcher,
		shutdown:     make(chan struct{}),
		handler:      graceful.NewHandler(log, eventEmitter, cache, "crawler", "v1", eventEnabled),
	}

	dispatcher.Start(ctx)
	log.Info("crawler service started",
		zap.Int("workers", DefaultWorkerCount),
		zap.Bool("events_enabled", eventEnabled),
	)

	return svc, nil
}

// NewDispatcher creates and configures a new Dispatcher.
func NewDispatcher(
	log *zap.Logger,
	repo *Repository,
	workerFactories map[crawlerpb.TaskType]WorkerFactory,
) (*Dispatcher, error) {
	d := &Dispatcher{
		log:       log,
		repo:      repo,
		workerMap: make(map[crawlerpb.TaskType]workers.Worker),
		jobQueue:  make(chan *crawlerpb.CrawlTask, DefaultQueueSize),
		results:   make(chan *crawlerpb.CrawlResult, DefaultQueueSize),
	}

	// Register workers using factories
	for taskType, factory := range workerFactories {
		worker := factory()
		d.workerMap[taskType] = worker
		log.Info("registered worker",
			zap.String("type", taskType.String()),
		)
	}

	return d, nil
}

// Start launches the worker pool and the result processor.
func (d *Dispatcher) Start(ctx context.Context) {
	d.processorWg.Add(1)
	go d.resultProcessor(ctx)

	d.workerWg.Add(DefaultWorkerCount)
	for i := 0; i < DefaultWorkerCount; i++ {
		go d.worker(ctx, i)
	}
}

// worker processes tasks with context propagation and graceful error handling.
func (d *Dispatcher) worker(ctx context.Context, id int) {
	// Reference unused id for diagnostics
	d.log.Debug("Dispatcher.worker started", zap.Int("worker_id", id))
	defer d.workerWg.Done()
	ticker := time.NewTicker(RateLimit)
	defer ticker.Stop()

	for task := range d.jobQueue {
		<-ticker.C // Rate limiting

		// Create worker context with timeout, inheriting parent ctx
		workerCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)

		worker, exists := d.workerMap[task.Type]
		if !exists {
			errMsg := fmt.Sprintf("unsupported task type: %s", task.Type.String())
			d.log.Error("no worker found",
				zap.String("uuid", task.Uuid),
				zap.String("type", task.Type.String()),
				zap.Int("worker_id", id),
			)

			result := &crawlerpb.CrawlResult{
				TaskUuid:     task.Uuid,
				Status:       crawlerpb.TaskStatus_TASK_STATUS_FAILED,
				ErrorMessage: errMsg,
			}

			d.results <- result
			cancel()
			continue
		}

		// Update task status to processing
		task.Status = crawlerpb.TaskStatus_TASK_STATUS_PROCESSING
		if _, err := d.repo.UpdateCrawlTask(workerCtx, task); err != nil {
			d.log.Error("failed to update task status",
				zap.String("uuid", task.Uuid),
				zap.Error(err),
				zap.Int("worker_id", id),
			)
		}

		// Process task with context
		result, err := worker.Process(workerCtx, task)
		cancel()
		if err != nil {
			d.log.Error("task processing failed",
				zap.String("uuid", task.Uuid),
				zap.Error(err),
				zap.Int("worker_id", id),
			)

			result = &crawlerpb.CrawlResult{
				TaskUuid:     task.Uuid,
				Status:       crawlerpb.TaskStatus_TASK_STATUS_FAILED,
				ErrorMessage: err.Error(),
			}

		} else {
			result.Status = crawlerpb.TaskStatus_TASK_STATUS_COMPLETED
		}

		d.results <- result
		cancel()
	}
}

// resultProcessor handles results with proper context.
func (d *Dispatcher) resultProcessor(ctx context.Context) {
	defer d.processorWg.Done()

	for result := range d.results {
		if err := d.repo.StoreCrawlResult(ctx, result); err != nil {
			d.log.Error("failed to store result",
				zap.String("uuid", result.TaskUuid),
				zap.Error(err),
			)
			continue
		}

		task, err := d.repo.GetCrawlTask(ctx, result.TaskUuid)
		if err != nil {
			d.log.Error("failed to retrieve task",
				zap.String("uuid", result.TaskUuid),
				zap.Error(err),
			)
			continue
		}

		task.Status = result.Status

		if _, err := d.repo.UpdateCrawlTask(ctx, task); err != nil {
			d.log.Error("failed to update task status",
				zap.String("uuid", result.TaskUuid),
				zap.Error(err),
			)
		}
	}
}

// Stop gracefully shuts down the dispatcher with timeout.
func (d *Dispatcher) Stop(ctx context.Context) error {
	var shutdownErr error
	shutdownComplete := make(chan struct{})

	d.shutdownOnce.Do(func() {
		d.log.Info("shutting down dispatcher")
		close(d.jobQueue)

		go func() {
			d.workerWg.Wait()
			close(d.results)
			d.processorWg.Wait()
			for _, w := range d.workerMap {
				w.Cleanup()
			}
			close(shutdownComplete)
		}()

		select {
		case <-shutdownComplete:
			d.log.Info("dispatcher shut down complete")
		case <-ctx.Done():
			shutdownErr = ctx.Err()
			d.log.Error("dispatcher shutdown timed out", zap.Error(shutdownErr))
		}
	})

	return shutdownErr
}

// Shutdown gracefully stops the service.
func (s *Service) Shutdown(ctx context.Context) error {
	s.log.Info("shutting down crawler service")

	// Give dispatcher up to 30 seconds to shutdown
	dispatcherCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := s.dispatcher.Stop(dispatcherCtx); err != nil {
		s.log.Error("dispatcher shutdown failed", zap.Error(err))
	}

	close(s.shutdown)
	s.log.Info("crawler service shutdown complete")
	return nil
}

// RegisterGRPCServer registers the service.
func (s *Service) RegisterGRPCServer(server *grpc.Server) {
	crawlerpb.RegisterCrawlerServiceServer(server, s)
}
