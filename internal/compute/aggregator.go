
package compute

import (
	"context"
	"strings"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

const (
	EventTaskSuccess = "compute:task:v1:success"
)

// Aggregator service for collecting and combining results.
type Aggregator struct {
	provider *service.Provider
	log      *zap.Logger
	store    Store
}

// NewAggregator creates a new Aggregator.
func NewAggregator(provider *service.Provider, log *zap.Logger, store Store) *Aggregator {
	return &Aggregator{
		provider: provider,
		log:      log,
		store:    store,
	}
}

// Start begins the aggregator's event processing loop.
func (a *Aggregator) Start(ctx context.Context) error {
	a.log.Info("Starting compute aggregator")
	err := a.provider.SubscribeEvents(ctx, []string{EventComputeSuccess}, nil, a.handleChunkSuccess)
	if err != nil {
		a.log.Error("Failed to subscribe to chunk success events", zap.Error(err))
		return err
	}
	<-ctx.Done()
	a.log.Info("Compute aggregator shutting down")
	return nil
}

func (a *Aggregator) handleChunkSuccess(ctx context.Context, event *nexusv1.EventResponse) {
	var result commonpb.ComputeResult
	if err := extractPayloadData(event.GetPayload().Data, &result); err != nil {
		a.log.Error("Failed to extract compute result from payload", zap.Error(err))
		return
	}

	chunkID := result.GetTaskId()
	parentTaskID := getParentTaskID(chunkID)

	a.log.Info("Received chunk success event", zap.String("chunk_id", chunkID), zap.String("parent_task_id", parentTaskID))

	task, err := a.store.GetTask(parentTaskID)
	if err != nil {
		a.log.Error("Failed to get task from store", zap.Error(err))
		return
	}
	if task == nil {
		a.log.Warn("Received chunk success for unknown task", zap.String("parent_task_id", parentTaskID))
		return
	}

	chunkIndex := -1
	for i, chunk := range task.Chunks {
		if chunk.ID == chunkID {
			chunkIndex = i
			break
		}
	}

	if chunkIndex == -1 {
		a.log.Warn("Received chunk success for unknown chunk", zap.String("chunk_id", chunkID))
		return
	}

	task.Chunks[chunkIndex].Status = "completed"
	if len(result.GetOutputs()) > 0 {
		task.Chunks[chunkIndex].ResultURI = result.GetOutputs()[0].GetUri()
	}

	if err := a.store.UpdateChunk(parentTaskID, task.Chunks[chunkIndex]); err != nil {
		a.log.Error("Failed to update chunk in store", zap.Error(err))
		return
	}

	// Check if all chunks are complete
	allComplete := true
	resultURIs := make([]string, len(task.Chunks))
	for i, chunk := range task.Chunks {
		if chunk.Status != "completed" {
			allComplete = false
			break
		}
		resultURIs[i] = chunk.ResultURI
	}

	if allComplete {
		a.log.Info("All chunks completed for task", zap.String("parent_task_id", parentTaskID))

		if err := a.store.CompleteTask(parentTaskID, resultURIs); err != nil {
			a.log.Error("Failed to complete task in store", zap.Error(err))
			return
		}

		// For now, just create a simple result with the list of URIs.
		finalResult := &commonpb.ComputeResult{
			TaskId: parentTaskID,
			Outputs: []*commonpb.DataRef{
				{
					Name: "aggregated_results",
					Body: &commonpb.DataRef_InlineJson{
						InlineJson: &commonpb.Struct{
							Fields: map[string]*commonpb.Value{
								"result_uris": {
									Kind: &commonpb.Value_ListValue{
										ListValue: &commonpb.ListValue{
											Values: urisToValues(resultURIs),
										},
									},
								},
							},
						},
					},
				},
			},
		}

		resultBody, err := proto.Marshal(finalResult)
		if err != nil {
			a.log.Error("Failed to marshal final result", zap.Error(err))
			return
		}
		resultPayload := &commonpb.Payload{Data: resultBody}

		canonicalResult := events.NewCanonicalEventEnvelope(
			EventTaskSuccess,
			event.GetMetadata().GetGlobalContext().GetUserId(),
			event.GetMetadata().GetGlobalContext().GetCampaignId(),
			event.GetMetadata().GetGlobalContext().GetCorrelationId(),
			resultPayload,
			nil,
		)
		resultEnvelope := &events.EventEnvelope{
			ID:       canonicalResult.CorrelationID,
			Type:     canonicalResult.Type,
			Payload:  canonicalResult.Payload,
			Metadata: canonicalResult.Metadata,
		}
		if _, err := a.provider.EmitEventEnvelope(ctx, resultEnvelope); err != nil {
			a.log.Error("Failed to emit final task success event", zap.Error(err))
		}
	}
}

func getParentTaskID(chunkID string) string {
	parts := strings.Split(chunkID, "-chunk-")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

func urisToValues(uris []string) []*commonpb.Value {
	values := make([]*commonpb.Value, len(uris))
	for i, uri := range uris {
		values[i] = &commonpb.Value{
			Kind: &commonpb.Value_StringValue{
				StringValue: uri,
			},
		}
	}
	return values
}
