package workers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"

	crawlerpb "github.com/nmxmxh/master-ovasabi/api/protos/crawler/v1"
)

var (
	// ErrUnsupportedArchive is returned when an archive format cannot be processed.
	ErrUnsupportedArchive = errors.New("unsupported archive format")
	// ErrMaxDepthExceeded is returned when a recursive worker exceeds its depth limit.
	ErrMaxDepthExceeded = errors.New("maximum recursion depth exceeded")
)

// Worker interface all devourers implement
type Worker interface {
	Process(ctx context.Context, task *crawlerpb.CrawlTask) (*crawlerpb.CrawlResult, error)
	WorkerType() crawlerpb.TaskType
	Cleanup()
}

// BaseWorker provides common functionality
type BaseWorker struct {
	mu          sync.Mutex
	timeout     time.Duration
	contentSize int64 // Max in bytes
}

func (b *BaseWorker) WithTimeout(d time.Duration) *BaseWorker {
	b.timeout = d
	return b
}

// Worker registry and dispatcher
type WorkerDispatcher struct {
	workers map[crawlerpb.TaskType]Worker
}

func NewDispatcher() *WorkerDispatcher {
	return &WorkerDispatcher{
		workers: make(map[crawlerpb.TaskType]Worker),
	}
}

func (d *WorkerDispatcher) Register(w Worker) {
	d.workers[w.WorkerType()] = w
}

func (d *WorkerDispatcher) ProcessTask(ctx context.Context, task *crawlerpb.CrawlTask) (*crawlerpb.CrawlResult, error) {
	if worker, exists := d.workers[task.Type]; exists {
		return worker.Process(ctx, task)
	}
	return nil, fmt.Errorf("no worker for type: %v", task.Type)
}
