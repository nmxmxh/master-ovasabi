package crawler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	crawlerpb "github.com/nmxmxh/master-ovasabi/api/protos/crawler/v1"
	"github.com/nmxmxh/master-ovasabi/internal/service/crawler/workers"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
)

const (
	DefaultWorkerCount = 10
	DefaultQueueSize   = 100
	RateLimit          = 1 * time.Second
)

// Service implements the gRPC server for the crawler service.
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

// Dispatcher manages the worker pool and task distribution.
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

// WorkerFactory creates workers for specific task types
type WorkerFactory func() workers.Worker

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
		defer cancel()

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
