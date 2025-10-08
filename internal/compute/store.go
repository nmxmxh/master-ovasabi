
package compute

import (
	"sync"
	"time"
)

// TaskState represents the state of a distributed compute task.
type TaskState struct {
	ID          string
	Chunks      []*ChunkState
	ResultURIs  []string
	Completed   bool
	CreatedAt   time.Time
	CompletedAt time.Time
}

// ChunkState represents the state of a single chunk of a task.
type ChunkState struct {
	ID        string
	Index     int
	Status    string // pending, in-progress, completed, failed
	ResultURI string
}

// Store is an interface for storing and retrieving the state of compute tasks.
type Store interface {
	CreateTask(task *TaskState) error
	GetTask(taskID string) (*TaskState, error)
	UpdateChunk(taskID string, chunk *ChunkState) error
	CompleteTask(taskID string, resultURIs []string) error
}

// InMemoryStore is an in-memory implementation of the Store interface.
type InMemoryStore struct {
	tasks sync.Map
}

// NewInMemoryStore creates a new InMemoryStore.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{}
}

// CreateTask stores a new task.
func (s *InMemoryStore) CreateTask(task *TaskState) error {
	s.tasks.Store(task.ID, task)
	return nil
}

// GetTask retrieves a task by its ID.
func (s *InMemoryStore) GetTask(taskID string) (*TaskState, error) {
	task, ok := s.tasks.Load(taskID)
	if !ok {
		return nil, nil
	}
	return task.(*TaskState), nil
}

// UpdateChunk updates the state of a chunk within a task.
func (s *InMemoryStore) UpdateChunk(taskID string, chunk *ChunkState) error {
	task, err := s.GetTask(taskID)
	if err != nil {
		return err
	}
	if task == nil {
		return nil
	}
	task.Chunks[chunk.Index] = chunk
	s.tasks.Store(taskID, task)
	return nil
}

// CompleteTask marks a task as complete and stores the result URIs.
func (s *InMemoryStore) CompleteTask(taskID string, resultURIs []string) error {
	task, err := s.GetTask(taskID)
	if err != nil {
		return err
	}
	if task == nil {
		return nil
	}
	task.Completed = true
	task.ResultURIs = resultURIs
	task.CompletedAt = time.Now()
	s.tasks.Store(taskID, task)
	return nil
}
