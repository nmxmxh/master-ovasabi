
package compute

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
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

	// --- Smart Chunking Logic ---
	var chunkableInputs []*commonpb.DataRef
	var taskContext *structpb.Struct
	chunkingHint := "auto" // default

	// Safely extract hints and context from the raw payload data
	if event.GetPayload() != nil {
		if data := event.GetPayload().GetData(); data != nil && data.Fields != nil {
			if ctxVal, ok := data.Fields["task_context"]; ok {
				if sc, ok := ctxVal.GetKind().(*structpb.Value_StructValue); ok {
					taskContext = sc.StructValue
				}
			}
		}
	}
	if meta := event.GetMetadata(); meta != nil && meta.GetServiceSpecific() != nil {
		if hintVal, ok := meta.GetServiceSpecific().Fields["chunking_hint"]; ok {
			if str, ok := hintVal.GetKind().(*structpb.Value_StringValue); ok {
				chunkingHint = str.StringValue
			}
		}
	}

	// Determine if the task should be chunked based on hint and input structure
	if chunkingHint != "none" {
		if len(envelope.GetInputs()) > 1 {
			// Strategy 1: Multiple top-level inputs. Treat each as a chunk.
			chunkableInputs = envelope.GetInputs()
		} else if len(envelope.GetInputs()) == 1 {
			input := envelope.GetInputs()[0]
			// Strategy 2: Single input that is a list. Break down the list into chunks.
			if ij := input.GetInlineJson(); ij != nil && ij.Fields != nil {
				if itemsVal, ok := ij.Fields["items"]; ok {
					if lv, ok := itemsVal.GetKind().(*structpb.Value_ListValue); ok {
						// Input is a JSON object with an "items" list, e.g., {"items": [...]}
						items := lv.ListValue.GetValues()
						for _, item := range items {
							chunkableInputs = append(chunkableInputs, &commonpb.DataRef{
								Body: &commonpb.DataRef_InlineJson{
									InlineJson: &structpb.Struct{
										Fields: map[string]*structpb.Value{"item": item},
									},
								},
							})
						}
					}
				}
			}
		}
	}

	isChunking := len(chunkableInputs) > 0
	numChunks := 1
	if isChunking {
		numChunks = len(chunkableInputs)
	}

	if numChunks == 1 && len(envelope.GetInputs()) == 0 {
		s.log.Warn("Task has no inputs to process, creating one default chunk.", zap.String("task_id", envelope.GetTaskId()))
	}

	chunks := make([]*Chunk, numChunks)
	for i := 0; i < numChunks; i++ {
		chunks[i] = &Chunk{
			ID:     fmt.Sprintf("%s-chunk-%d", envelope.GetTaskId(), i),
			Status: "pending",
		}
	}

	task := &Task{
		ID:          envelope.GetTaskId(),
		TotalChunks: numChunks,
		Chunks:      chunks,
		Status:      "in-progress",
	}
	if err := s.store.CreateTask(task); err != nil {
		s.log.Error("Failed to create task in store", zap.Error(err))
		return
	}

	for i, chunk := range chunks {
		chunkEnvelope := proto.Clone(&envelope).(*commonpb.ComputeEnvelope)
		chunkEnvelope.TaskId = chunk.ID

		// Set the specific input(s) for this chunk
		if isChunking {
			chunkEnvelope.Inputs = []*commonpb.DataRef{chunkableInputs[i]}
		} else {
			// Not chunking, so the single chunk gets all original inputs
			chunkEnvelope.Inputs = envelope.GetInputs()
		}

		// Marshal chunk envelope to structpb.Struct for Payload.Data
		jsonBytes, err := protojson.Marshal(chunkEnvelope)
		if err != nil {
			s.log.Error("Failed to marshal chunk envelope to JSON", zap.Error(err), zap.String("task_id", chunk.ID))
			continue
		}
		dataStruct := &structpb.Struct{}
		if err := protojson.Unmarshal(jsonBytes, dataStruct); err != nil {
			s.log.Error("Failed to unmarshal JSON to structpb.Struct", zap.Error(err), zap.String("task_id", chunk.ID))
			continue
		}

		// Add the shared task context back into the final payload struct for the worker
		if taskContext != nil {
			if dataStruct.Fields == nil {
				dataStruct.Fields = make(map[string]*structpb.Value)
			}
			dataStruct.Fields["task_context"] = structpb.NewStructValue(taskContext)
		}
		assignedPayload := &commonpb.Payload{Data: dataStruct}

		if event.GetMetadata() == nil || event.GetMetadata().GetGlobalContext() == nil {
			s.log.Error("Event is missing metadata, cannot emit chunk", zap.String("task_id", chunk.ID))
			continue
		}

		globalCtx := event.GetMetadata().GetGlobalContext()
		canonicalAssigned := events.NewCanonicalEventEnvelope(
			EventComputeRequested, // Use the constant from coordinator.go
			globalCtx.GetSource(),
			globalCtx.GetCampaignId(),
			globalCtx.GetCorrelationId(),
			assignedPayload,
			nil,
		)
		assignedEnvelope := &events.EventEnvelope{
			ID:       uuid.New().String(),
			Type:     canonicalAssigned.Type,
			Payload:  canonicalAssigned.Payload,
			Metadata: canonicalAssigned.Metadata,
		}
		if _, err := s.provider.EmitEventEnvelope(ctx, assignedEnvelope); err != nil {
			s.log.Error("Failed to emit chunk dispatch event", zap.Error(err), zap.String("task_id", chunk.ID))
		}
	}
}
