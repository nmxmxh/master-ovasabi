package pattern

import (
	"context"
	"fmt"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	schedulerpb "github.com/nmxmxh/master-ovasabi/api/protos/scheduler/v1"
	kgserver "github.com/nmxmxh/master-ovasabi/internal/server/kg"
	"github.com/nmxmxh/master-ovasabi/pkg/contextx"
	"go.uber.org/zap"
)

// Helper to extract schedule string from metadata.
func extractScheduleFromMetadata(meta *commonpb.Metadata) (string, error) {
	if meta == nil || meta.ServiceSpecific == nil {
		return "", fmt.Errorf("metadata or service_specific is nil")
	}
	ss := meta.ServiceSpecific.AsMap()
	sched, ok := ss["scheduling"]
	if !ok {
		return "", fmt.Errorf("no scheduling section in metadata")
	}
	schedMap, ok := sched.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("scheduling is not a map")
	}
	if cron, ok := schedMap["cron"].(string); ok && cron != "" {
		return cron, nil
	}
	if interval, ok := schedMap["interval"].(string); ok && interval != "" {
		return interval, nil
	}
	return "", fmt.Errorf("no cron or interval found in scheduling metadata")
}

// EnrichKnowledgeGraph connects to the KGService and publishes an update using DI.
func EnrichKnowledgeGraph(ctx context.Context, log *zap.Logger, patternType, patternID string, meta *commonpb.Metadata) error {
	container := contextx.DI(ctx)
	if container == nil {
		log.Error("DIContainer not found in context")
		return fmt.Errorf("DIContainer not found in context")
	}
	var kgService *kgserver.KGService
	if err := container.Resolve(&kgService); err != nil || kgService == nil {
		log.Error("Failed to resolve KGService from DI", zap.Error(err))
		return fmt.Errorf("KGService not found in DI: %w", err)
	}
	node := map[string]interface{}{
		"id":           patternID,
		"type":         patternType,
		"metadata":     meta,
		"last_updated": time.Now().UTC(),
	}
	update := &kgserver.KGUpdate{
		ID:        patternID,
		Type:      kgserver.PatternDetection,
		ServiceID: patternID,
		Payload:   node,
		Timestamp: time.Now(),
		Version:   "1.0",
	}
	if err := kgService.PublishUpdate(ctx, update); err != nil {
		log.Error("Failed to enrich KG", zap.Error(err))
		return err
	}
	log.Info("Enriched KG", zap.String("type", patternType), zap.String("id", patternID))
	return nil
}

// RegisterSchedule connects to the SchedulerService and registers a job using DI.
func RegisterSchedule(ctx context.Context, log *zap.Logger, patternType, patternID string, meta *commonpb.Metadata) error {
	container := contextx.DI(ctx)
	if container == nil {
		log.Error("DIContainer not found in context")
		return fmt.Errorf("DIContainer not found in context")
	}
	var schedulerClient schedulerpb.SchedulerServiceClient
	if err := container.Resolve(&schedulerClient); err != nil || schedulerClient == nil {
		log.Error("Failed to resolve SchedulerServiceClient from DI", zap.Error(err))
		return fmt.Errorf("SchedulerServiceClient not found in DI: %w", err)
	}
	schedule, err := extractScheduleFromMetadata(meta)
	if err != nil {
		log.Error("Failed to extract schedule from metadata", zap.Error(err))
		return err
	}
	job := &schedulerpb.Job{
		Name:        patternID,
		Schedule:    schedule,
		Payload:     "", // Optionally serialize business data
		Status:      schedulerpb.JobStatus_JOB_STATUS_ACTIVE,
		Metadata:    meta,
		TriggerType: schedulerpb.TriggerType_TRIGGER_TYPE_CRON,
		JobType:     schedulerpb.JobType_JOB_TYPE_CUSTOM,
	}
	req := &schedulerpb.CreateJobRequest{Job: job}
	_, err = schedulerClient.CreateJob(ctx, req)
	if err != nil {
		log.Error("Failed to register job with scheduler", zap.Error(err))
		return err
	}
	log.Info("Registered schedule", zap.String("type", patternType), zap.String("id", patternID))
	return nil
}
