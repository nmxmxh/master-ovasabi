package utils

import (
	"context"
	"sync"
	"time"

	"github.com/nmxmxh/master-ovasabi/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

// Task represents a unit of work to be processed
type Task interface {
	Process(ctx context.Context) error
}

// WorkerPool manages a pool of workers for processing tasks
type WorkerPool struct {
	numWorkers int
	tasks      chan Task
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	errors     chan error
	metrics    *workerPoolMetrics
}

type workerPoolMetrics struct {
	activeWorkers  prometheus.Gauge
	queuedTasks    prometheus.Gauge
	processedTasks prometheus.Counter
	taskErrors     prometheus.Counter
	processingTime prometheus.Histogram
}

func newWorkerPoolMetrics(poolName string) *workerPoolMetrics {
	return &workerPoolMetrics{
		activeWorkers:  metrics.WorkerPoolGauges.WithLabelValues(poolName, "active_workers"),
		queuedTasks:    metrics.WorkerPoolGauges.WithLabelValues(poolName, "queued_tasks"),
		processedTasks: metrics.WorkerPoolCounters.WithLabelValues(poolName, "processed_tasks"),
		taskErrors:     metrics.WorkerPoolCounters.WithLabelValues(poolName, "task_errors"),
		processingTime: metrics.WorkerPoolHistograms.WithLabelValues(poolName).(prometheus.Histogram),
	}
}

// NewWorkerPool creates a new worker pool with the specified number of workers
func NewWorkerPool(numWorkers int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerPool{
		numWorkers: numWorkers,
		tasks:      make(chan Task, numWorkers*2), // Buffer size is 2x number of workers
		ctx:        ctx,
		cancel:     cancel,
		errors:     make(chan error, numWorkers),
		metrics:    newWorkerPoolMetrics("default"),
	}
}

// Start initializes and starts the worker pool
func (p *WorkerPool) Start() {
	for i := 0; i < p.numWorkers; i++ {
		p.wg.Add(1)
		go p.worker(i)
		p.metrics.activeWorkers.Inc()
	}
}

// Stop gracefully shuts down the worker pool
func (p *WorkerPool) Stop() {
	p.cancel()
	close(p.tasks)
	p.wg.Wait()
	close(p.errors)
	p.metrics.activeWorkers.Set(0)
}

// Submit adds a task to the pool
func (p *WorkerPool) Submit(task Task) error {
	select {
	case p.tasks <- task:
		p.metrics.queuedTasks.Inc()
		return nil
	case <-p.ctx.Done():
		return p.ctx.Err()
	}
}

// Errors returns a channel that receives task processing errors
func (p *WorkerPool) Errors() <-chan error {
	return p.errors
}

// worker processes tasks from the pool
func (p *WorkerPool) worker(_ int) {
	defer func() {
		p.wg.Done()
		p.metrics.activeWorkers.Dec()
	}()

	for {
		select {
		case task, ok := <-p.tasks:
			if !ok {
				return
			}
			p.metrics.queuedTasks.Dec()
			start := time.Now()

			if err := task.Process(p.ctx); err != nil {
				p.metrics.taskErrors.Inc()
				select {
				case p.errors <- err:
				default:
					// Error channel is full, log or handle accordingly
				}
			}

			p.metrics.processedTasks.Inc()
			p.metrics.processingTime.Observe(time.Since(start).Seconds())

		case <-p.ctx.Done():
			return
		}
	}
}
