package nexus

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid" // ...existing imports...

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"

	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/internal/nexus"
	"github.com/nmxmxh/master-ovasabi/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// MockEventRepository implements EventRepository for testing.
type MockEventRepository struct {
	events []*nexus.CanonicalEvent
	mutex  sync.RWMutex
}

func NewMockEventRepository() *MockEventRepository {
	return &MockEventRepository{
		events: make([]*nexus.CanonicalEvent, 0),
	}
}

func (m *MockEventRepository) SaveEvent(ctx context.Context, event *nexus.CanonicalEvent) error {
	// Use context for diagnostics (lint fix)
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.events = append(m.events, event)
	return nil
}

func (m *MockEventRepository) GetEvents() []*nexus.CanonicalEvent {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	// Return a copy to prevent race conditions
	result := make([]*nexus.CanonicalEvent, len(m.events))
	copy(result, m.events)
	return result
}

func (m *MockEventRepository) GetEvent(ctx context.Context, id uuid.UUID) (*nexus.CanonicalEvent, error) {
	// Use context for diagnostics (lint fix)
	if ctx != nil && ctx.Err() != nil {
		return nil, ctx.Err()
	}
	_ = id // Use id to avoid revive unused-parameter warning
	// Not implemented for this test
	return nil, fmt.Errorf("event not found")
}

func (m *MockEventRepository) ListEventsByMaster(ctx context.Context, masterID int64) ([]*nexus.CanonicalEvent, error) {
	// Use context for diagnostics (lint fix)
	if ctx != nil && ctx.Err() != nil {
		return nil, ctx.Err()
	}
	_ = masterID // Use masterID to avoid revive unused-parameter warning
	// Not implemented for this test
	return nil, fmt.Errorf("no events found for master")
}

func (m *MockEventRepository) ListPendingEvents(ctx context.Context, entityType repository.EntityType) ([]*nexus.CanonicalEvent, error) {
	// Use context for diagnostics (lint fix)
	if ctx != nil && ctx.Err() != nil {
		return nil, ctx.Err()
	}
	_ = entityType // Use entityType to avoid revive unused-parameter warning
	// Not implemented for this test
	return nil, fmt.Errorf("no pending events found")
}

func (m *MockEventRepository) UpdateEventStatus(ctx context.Context, id uuid.UUID, status string, errMsg *string) error {
	// Use context for diagnostics (lint fix)
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	_ = id     // Use id to avoid revive unused-parameter warning
	_ = status // Use status to avoid revive unused-parameter warning
	_ = errMsg // Use errMsg to avoid revive unused-parameter warning
	// Not implemented for this test
	return nil
}

func (m *MockEventRepository) ListEventsByPattern(ctx context.Context, patternID string) ([]*nexus.CanonicalEvent, error) {
	// Use context for diagnostics (lint fix)
	if ctx != nil && ctx.Err() != nil {
		return nil, ctx.Err()
	}
	_ = patternID // Use patternID to avoid revive unused-parameter warning
	// Not implemented for this test
	return nil, fmt.Errorf("no events found for pattern")
}

func TestEventOrdering(t *testing.T) {
	ctx := context.Background()
	mockRepo := NewMockEventRepository()
	logger := zap.NewNop()

	// Create a service with the mock repository
	svc := &Service{
		repo:          nil,
		eventRepo:     mockRepo,
		cache:         nil,
		log:           logger,
		eventBus:      nil,
		eventEnabled:  true,
		provider:      nil,
		subscribers:   make(map[string][]chan *nexusv1.EventResponse),
		eventSequence: 0,
		lastEventTime: time.Now(),
	}

	// Test sequential event emission
	t.Run("Sequential event emission", func(t *testing.T) {
		const numEvents = 5
		eventType := "test.sequential"
		entityID := "test-entity"

		for i := 0; i < numEvents; i++ {
			req := &nexusv1.EventRequest{
				EventType: eventType,
				EntityId:  entityID,
				Payload:   &commonpb.Payload{},
				Metadata:  &commonpb.Metadata{},
			}

			resp, err := svc.EmitEvent(ctx, req)
			require.NoError(t, err)
			assert.True(t, resp.Success)
		}

		// Verify sequence ordering
		events := mockRepo.GetEvents()
		require.Len(t, events, numEvents)

		for i, event := range events {
			require.NotNil(t, event.NexusSequence)
			idx := i
			var seq uint64
			if idx < 0 {
				seq = 1
			} else {
				seq = uint64(idx) + 1
			}
			assert.Equal(t, seq, *event.NexusSequence)
		}
	})

	// Reset the service for concurrent test
	svc.eventSequence = 0
	mockRepo.events = make([]*nexus.CanonicalEvent, 0)

	// Test concurrent event emission
	t.Run("Concurrent event emission", func(t *testing.T) {
		const numWorkers = 10
		const eventsPerWorker = 10
		totalEvents := numWorkers * eventsPerWorker

		var wg sync.WaitGroup
		wg.Add(numWorkers)

		// Start concurrent workers
		for worker := 0; worker < numWorkers; worker++ {
			go func(workerID int) {
				_ = workerID // Use workerID to avoid revive unused-parameter warning
				defer wg.Done()
				for i := 0; i < eventsPerWorker; i++ {
					req := &nexusv1.EventRequest{
						EventType: "test.concurrent",
						EntityId:  "test-entity",
						Payload:   &commonpb.Payload{},
						Metadata:  &commonpb.Metadata{},
					}

					resp, err := svc.EmitEvent(ctx, req)
					require.NoError(t, err)
					assert.True(t, resp.Success)
				}
			}(worker)
		}

		// Wait for all workers to complete
		wg.Wait()

		// Verify all events were saved
		events := mockRepo.GetEvents()
		require.Len(t, events, totalEvents)

		// Verify sequence numbers are unique and continuous
		sequences := make(map[uint64]bool)
		for _, event := range events {
			require.NotNil(t, event.NexusSequence)
			sequence := *event.NexusSequence
			assert.False(t, sequences[sequence], "Duplicate sequence number: %d", sequence)
			sequences[sequence] = true
		}

		// Verify we have all sequence numbers from 1 to totalEvents
		for i := 1; i <= totalEvents; i++ {
			// Ensure i is non-negative before conversion to uint64
			if i < 0 {
				t.Fatalf("Sequence index is negative: %d", i)
			}
			seq := uint64(i)
			assert.True(t, sequences[seq], "Missing sequence number: %d", i)
		}
	})
}

func TestTemporalConflictDetection(t *testing.T) {
	ctx := context.Background()
	mockRepo := NewMockEventRepository()
	logger := zap.NewNop()

	// Create a service with the mock repository
	svc := &Service{
		repo:          nil,
		eventRepo:     mockRepo,
		cache:         nil,
		log:           logger,
		eventBus:      nil,
		eventEnabled:  true,
		provider:      nil,
		subscribers:   make(map[string][]chan *nexusv1.EventResponse),
		eventSequence: 0,
		lastEventTime: time.Now(),
	}

	// Emit first event
	req1 := &nexusv1.EventRequest{
		EventType: "test.temporal",
		EntityId:  "test-entity",
		Payload:   &commonpb.Payload{},
		Metadata:  &commonpb.Metadata{},
	}

	resp1, err := svc.EmitEvent(ctx, req1)
	require.NoError(t, err)
	assert.True(t, resp1.Success)

	// Simulate time going backwards by manually setting lastEventTime to the future
	// This would happen if system clock is adjusted or events arrive out of order
	futureTime := time.Now().Add(1 * time.Hour)
	svc.lastEventTime = futureTime

	// Emit second event - this should trigger temporal conflict detection
	req2 := &nexusv1.EventRequest{
		EventType: "test.temporal",
		EntityId:  "test-entity",
		Payload:   &commonpb.Payload{},
		Metadata:  &commonpb.Metadata{},
	}

	resp2, err := svc.EmitEvent(ctx, req2)
	require.NoError(t, err)
	assert.True(t, resp2.Success)

	// Verify both events were saved despite temporal conflict
	events := mockRepo.GetEvents()
	require.Len(t, events, 2)

	// Verify sequence numbers are still correct
	assert.Equal(t, uint64(1), *events[0].NexusSequence)
	assert.Equal(t, uint64(2), *events[1].NexusSequence)
}

func TestEventMetadataEnrichment(t *testing.T) {
	ctx := context.Background()
	mockRepo := NewMockEventRepository()
	logger := zap.NewNop()

	// Create a service with the mock repository
	svc := &Service{
		repo:          nil,
		eventRepo:     mockRepo,
		cache:         nil,
		log:           logger,
		eventBus:      nil,
		eventEnabled:  true,
		provider:      nil,
		subscribers:   make(map[string][]chan *nexusv1.EventResponse),
		eventSequence: 0,
		lastEventTime: time.Now(),
	}

	req := &nexusv1.EventRequest{
		EventType: "test.metadata",
		EntityId:  "test-entity",
		Payload:   &commonpb.Payload{},
		Metadata:  &commonpb.Metadata{},
	}

	resp, err := svc.EmitEvent(ctx, req)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	// Verify metadata was enriched with sequence and timestamp
	events := mockRepo.GetEvents()
	require.Len(t, events, 1)

	event := events[0]
	require.NotNil(t, event.Metadata)
	require.NotNil(t, event.Metadata.ServiceSpecific)
	require.NotNil(t, event.Metadata.ServiceSpecific.Fields)

	// Check that sequence was added to metadata
	sequenceField, exists := event.Metadata.ServiceSpecific.Fields["nexus.sequence"]
	assert.True(t, exists)
	assert.Equal(t, "1", sequenceField.GetStringValue())

	// Check that timestamp was added to metadata
	timestampField, exists := event.Metadata.ServiceSpecific.Fields["nexus.emitter_timestamp"]
	assert.True(t, exists)
	assert.NotEmpty(t, timestampField.GetStringValue())

	// Verify timestamp is in correct format
	_, err = time.Parse(time.RFC3339Nano, timestampField.GetStringValue())
	assert.NoError(t, err)
}
