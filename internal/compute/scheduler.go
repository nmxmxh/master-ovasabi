
package compute

import (
	"context"
	"fmt"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

const (
	EventTaskRequested = "compute:task:v1:requested"
)

// Scheduler service for decomposing tasks.
type Scheduler struct {
	provider *service.Provider
	log      *zap.Logger
	store    Store
}

// NewScheduler creates a new Scheduler.
func NewScheduler(provider *service.Provider, log *zap.Logger, store Store) *Scheduler {
	return &Scheduler{
		provider: provider,
		log:      log,
		store:    store,
	}
}

// Start begins the scheduler's event processing loop.
func (s *Scheduler) Start(ctx context.Context) error {
	s.log.Info("Starting compute scheduler")
	err := s.provider.SubscribeEvents(ctx, []string{EventTaskRequested}, nil, s.handleTaskRequest)
	if err != nil {
		s.log.Error("Failed to subscribe to task requests", zap.Error(err))
		return err
	}
	<-ctx.Done()
	s.log.Info("Compute scheduler shutting down")
	return nil
}

func (s *Scheduler) handleTaskRequest(ctx context.Context, event *nexusv1.EventResponse) {
	var envelope commonpb.ComputeEnvelope
	if err := extractPayloadData(event.GetPayload().Data, &envelope); err != nil {
		s.log.Error("Failed to extract compute envelope from payload", zap.Error(err))
		return
	}

	s.log.Info("Received compute task request", zap.String("task_id", envelope.GetTaskId()))

	// Simple chunking strategy: 1 chunk for now
	numChunks := 1
	chunks := make([]*ChunkState, numChunks)
	for i := 0; i < numChunks; i++ {
		chunks[i] = &ChunkState{
			ID:     fmt.Sprintf("%s-chunk-%d", envelope.GetTaskId(), i),
			Index:  i,
			Status: "pending",
		}
	}

	task := &TaskState{
		ID:        envelope.GetTaskId(),
		Chunks:    chunks,
		CreatedAt: time.Now(),
	}
	if err := s.store.CreateTask(task); err != nil {
		s.log.Error("Failed to create task in store", zap.Error(err))
		return
	}

	for _, chunk := range chunks {
		chunkEnvelope := proto.Clone(&envelope).(*commonpb.ComputeEnvelope)
		chunkEnvelope.TaskId = chunk.ID

		// In a real implementation, you would modify the inputs for each chunk.
		// For now, we just forward the same envelope with a new task ID.

		assignedBody, err := proto.Marshal(chunkEnvelope)
		if err != nil {
			s.log.Error("Failed to create assigned payload", zap.Error(err), zap.String("task_id", chunk.ID))
			continue
		}
		assignedPayload := &commonpb.Payload{Data: assignedBody}

		canonicalAssigned := events.NewCanonicalEventEnvelope(
			EventComputeRequested,
			event.GetMetadata().GetGlobalContext().GetUserId(),
			event.GetMetadata().GetGlobalContext().GetCampaignId(),
			event.GetMetadata().GetGlobalContext().GetCorrelationId(),
			assignedPayload,
			nil,
		)
		assignedEnvelope := &events.EventEnvelope{
			ID:       canonicalAssigned.CorrelationID,
			Type:     canonicalAssigned.Type,
			Payload:  canonicalAssigned.Payload,
			Metadata: canonicalAssigned.Metadata,
		}
		if _, err := s.provider.EmitEventEnvelope(ctx, assignedEnvelope); err != nil {
			s.log.Error("Failed to emit chunk dispatch event", zap.Error(err), zap.String("task_id", chunk.ID))
		}
	}
}
