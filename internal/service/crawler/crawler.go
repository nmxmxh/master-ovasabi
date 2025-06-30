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
type EventEmitterAdapter struct {
	Emitter events.EventEmitter
}

func (a *EventEmitterAdapter) EmitRawEventWithLogging(ctx context.Context, log *zap.Logger, eventType, eventID string, payload []byte) (string, bool) {
	if a.Emitter == nil {
		log.Warn("Event emitter not configured", zap.String("event_type", eventType))
		return "", false
	}
	return a.Emitter.EmitRawEventWithLogging(ctx, log, eventType, eventID, payload)
}

func (a *EventEmitterAdapter) EmitEventWithLogging(ctx context.Context, event interface{}, log *zap.Logger, eventType, eventID string, meta *commonpb.Metadata) (string, bool) {
	if a.Emitter == nil {
		log.Warn("Event emitter not configured", zap.String("event_type", eventType))
		return "", false
	}
	return a.Emitter.EmitEventWithLogging(ctx, event, log, eventType, eventID, meta)
}

// --- Service and Dispatcher Types ---
type Service struct {
	crawlerpb.UnimplementedCrawlerServiceServer
	log          *zap.Logger
	repo         *Repository
	cache        *redis.Cache
	eventEmitter events.EventEmitter
	eventEnabled bool
	dispatcher   *Dispatcher
	shutdown     chan struct{}
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

// --- Constants for worker pool ---
const (
	DefaultWorkerCount = 4
	DefaultQueueSize   = 100
	RateLimit          = 100 * time.Millisecond
)

// --- CrawlerService gRPC Methods ---

// Ensure gRPC interface compliance
var _ crawlerpb.CrawlerServiceServer = (*Service)(nil)

// SubmitTask submits a new crawl task and orchestrates via graceful
func (s *Service) SubmitTask(ctx context.Context, req *crawlerpb.SubmitTaskRequest) (*crawlerpb.SubmitTaskResponse, error) {
	if req == nil || req.Task == nil {
		errCtx := graceful.WrapErr(ctx, codes.InvalidArgument, "missing task in request", nil)
		errCtx.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{
			Log:          s.log,
			Context:      ctx,
			EventEmitter: &EventEmitterAdapter{Emitter: s.eventEmitter},
			EventEnabled: s.eventEnabled,
			EventType:    "crawler.task_submit_error",
			PatternType:  "crawler_task",
		})
		return nil, graceful.ToStatusError(errCtx)
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
		errCtx := graceful.WrapErr(ctx, codes.Internal, "failed to create crawl task", err)
		errCtx.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{
			Log:          s.log,
			Context:      ctx,
			Metadata:     req.Task.Metadata,
			EventEmitter: &EventEmitterAdapter{Emitter: s.eventEmitter},
			EventEnabled: s.eventEnabled,
			EventType:    "crawler.task_submit_error",
			EventID:      req.Task.Uuid,
			PatternType:  "crawler_task",
			PatternID:    req.Task.Uuid,
			PatternMeta:  req.Task.Metadata,
		})
		return nil, graceful.ToStatusError(errCtx)
	}

	// Enqueue the task for processing
	select {
	case s.dispatcher.jobQueue <- createdTask:
		// ok
	default:
		errCtx := graceful.WrapErr(ctx, codes.ResourceExhausted, "task queue is full", nil)
		errCtx.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{
			Log:          s.log,
			Context:      ctx,
			Metadata:     createdTask.Metadata,
			EventEmitter: &EventEmitterAdapter{Emitter: s.eventEmitter},
			EventEnabled: s.eventEnabled,
			EventType:    "crawler.task_queue_full",
			EventID:      createdTask.Uuid,
			PatternType:  "crawler_task",
			PatternID:    createdTask.Uuid,
			PatternMeta:  createdTask.Metadata,
		})
		return nil, graceful.ToStatusError(errCtx)
	}

	resp := &crawlerpb.SubmitTaskResponse{
		Uuid:    createdTask.Uuid,
		Status:  createdTask.Status,
		Message: "Task submitted successfully",
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "crawler task submitted", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          s.log,
		Cache:        s.cache,
		CacheKey:     createdTask.Uuid,
		CacheValue:   createdTask,
		CacheTTL:     10 * time.Minute,
		Metadata:     createdTask.Metadata,
		EventEmitter: &EventEmitterAdapter{Emitter: s.eventEmitter},
		EventEnabled: s.eventEnabled,
		EventType:    "crawler.task_submitted",
		EventID:      createdTask.Uuid,
		PatternType:  "crawler_task",
		PatternID:    createdTask.Uuid,
		PatternMeta:  createdTask.Metadata,
	})
	return resp, nil
}

// GetTaskStatus returns the current status of a crawl task
func (s *Service) GetTaskStatus(ctx context.Context, req *crawlerpb.GetTaskStatusRequest) (*crawlerpb.CrawlTask, error) {
	if req == nil || req.Uuid == "" {
		errCtx := graceful.WrapErr(ctx, codes.InvalidArgument, "missing uuid in request", nil)
		errCtx.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{
			Log:          s.log,
			Context:      ctx,
			EventEmitter: &EventEmitterAdapter{Emitter: s.eventEmitter},
			EventEnabled: s.eventEnabled,
			EventType:    "crawler.task_status_error",
			PatternType:  "crawler_task",
		})
		return nil, graceful.ToStatusError(errCtx)
	}
	task, err := s.repo.GetCrawlTask(ctx, req.Uuid)
	if err != nil {
		s.log.Error("failed to get crawl task", zap.Error(err))
		errCtx := graceful.WrapErr(ctx, codes.NotFound, "crawl task not found", err)
		errCtx.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{
			Log:          s.log,
			Context:      ctx,
			EventEmitter: &EventEmitterAdapter{Emitter: s.eventEmitter},
			EventEnabled: s.eventEnabled,
			EventType:    "crawler.task_status_error",
			EventID:      req.Uuid,
			PatternType:  "crawler_task",
			PatternID:    req.Uuid,
		})
		return nil, graceful.ToStatusError(errCtx)
	}
	// Optionally orchestrate success (not always needed for read-only)
	return task, nil
}

// StreamResults streams crawl results for a given task UUID
func (s *Service) StreamResults(req *crawlerpb.StreamResultsRequest, stream crawlerpb.CrawlerService_StreamResultsServer) error {
	ctx := stream.Context()
	if req == nil || req.TaskUuid == "" {
		errCtx := graceful.WrapErr(ctx, codes.InvalidArgument, "missing task_uuid in request", nil)
		errCtx.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{
			Log:          s.log,
			Context:      ctx,
			EventEmitter: &EventEmitterAdapter{Emitter: s.eventEmitter},
			EventEnabled: s.eventEnabled,
			EventType:    "crawler.stream_results_error",
			PatternType:  "crawler_task",
		})
		return graceful.ToStatusError(errCtx)
	}

	// For demo: just send the latest result (extend to real streaming if needed)
	result, err := s.repo.GetCrawlResult(ctx, req.TaskUuid)
	if err != nil {
		s.log.Error("failed to get crawl result", zap.Error(err))
		errCtx := graceful.WrapErr(ctx, codes.NotFound, "crawl result not found", err)
		errCtx.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{
			Log:          s.log,
			Context:      ctx,
			EventEmitter: &EventEmitterAdapter{Emitter: s.eventEmitter},
			EventEnabled: s.eventEnabled,
			EventType:    "crawler.stream_results_error",
			EventID:      req.TaskUuid,
			PatternType:  "crawler_task",
			PatternID:    req.TaskUuid,
		})
		return graceful.ToStatusError(errCtx)
	}

	if err := stream.Send(result); err != nil {
		s.log.Error("failed to stream result", zap.Error(err))
		errCtx := graceful.WrapErr(ctx, codes.Internal, "failed to stream result", err)
		errCtx.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{
			Log:          s.log,
			Context:      ctx,
			EventEmitter: &EventEmitterAdapter{Emitter: s.eventEmitter},
			EventEnabled: s.eventEnabled,
			EventType:    "crawler.stream_results_error",
			EventID:      req.TaskUuid,
			PatternType:  "crawler_task",
			PatternID:    req.TaskUuid,
		})
		return graceful.ToStatusError(errCtx)
	}
	// Optionally orchestrate success (not always needed for streaming)
	return nil
}

// NewService creates a new crawler service with proper dependency injection
func NewService(
	ctx context.Context,
	log *zap.Logger,
	repo *Repository,
	cache *redis.Cache,
	eventEmitter events.EventEmitter,
	eventEnabled bool,
	workerFactories map[crawlerpb.TaskType]WorkerFactory,
) (*Service, error) {
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
	}

	dispatcher.Start()
	log.Info("crawler service started",
		zap.Int("workers", DefaultWorkerCount),
		zap.Bool("events_enabled", eventEnabled),
	)

	return svc, nil
}

// NewDispatcher creates and configures a new Dispatcher
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

// Start launches the worker pool and the result processor
func (d *Dispatcher) Start() {
	d.processorWg.Add(1)
	go d.resultProcessor()

	d.workerWg.Add(DefaultWorkerCount)
	for i := 0; i < DefaultWorkerCount; i++ {
		go d.worker(i)
	}
}

// worker processes tasks with context propagation and graceful error handling
func (d *Dispatcher) worker(id int) {
	defer d.workerWg.Done()
	ticker := time.NewTicker(RateLimit)
	defer ticker.Stop()

	for task := range d.jobQueue {
		<-ticker.C // Rate limiting

		// Create worker context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)

		worker, exists := d.workerMap[task.Type]
		if !exists {
			errMsg := fmt.Sprintf("unsupported task type: %s", task.Type.String())
			d.log.Error("no worker found",
				zap.String("uuid", task.Uuid),
				zap.String("type", task.Type.String()),
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
		if _, err := d.repo.UpdateCrawlTask(ctx, task); err != nil {
			d.log.Error("failed to update task status",
				zap.String("uuid", task.Uuid),
				zap.Error(err),
			)
		}

		// Process task with context
		result, err := worker.Process(ctx, task)
		if err != nil {
			d.log.Error("task processing failed",
				zap.String("uuid", task.Uuid),
				zap.Error(err),
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

// resultProcessor handles results with proper context
func (d *Dispatcher) resultProcessor() {
	defer d.processorWg.Done()

	for result := range d.results {
		ctx := context.Background()

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

// Stop gracefully shuts down the dispatcher with timeout
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

// Shutdown gracefully stops the service
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

// RegisterGRPCServer registers the service
func (s *Service) RegisterGRPCServer(server *grpc.Server) {
	crawlerpb.RegisterCrawlerServiceServer(server, s)
}
