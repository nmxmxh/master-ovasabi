package scheduler

import (
	"context"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	schedulerpb "github.com/nmxmxh/master-ovasabi/api/protos/scheduler/v1"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/structpb"
)

type EventHandlerFunc func(ctx context.Context, provider *service.Provider, event *nexusv1.EventResponse, log *zap.Logger)

type EventSubscription struct {
	EventTypes []string
	Handler    EventHandlerFunc
}

type EventRegistry []EventSubscription

// Handler for user.created event: schedules a recurring Monday 9am Africa/Lagos payday job for the user.
func handleUserCreated(ctx context.Context, provider *service.Provider, event *nexusv1.EventResponse, log *zap.Logger) {
	if event == nil || event.Metadata == nil {
		log.Warn("user.created event missing metadata")
		return
	}
	userVars := metadata.ExtractServiceVariables(event.Metadata, "user")
	userID, ok := userVars["id"].(string)
	if !ok {
		log.Warn("Failed to assert userID as string")
		return
	}
	if userID == "" && event.Payload != nil && event.Payload.Data != nil {
		if idField, ok := event.Payload.Data.Fields["user_id"]; ok {
			userID = idField.GetStringValue()
		}
	}
	if userID == "" {
		log.Warn("user.created event missing user_id in service_specific and payload.data")
		return
	}
	cronExpr := "0 9 * * 1" // 9am every Monday
	jobName := "payday_monday_9am_" + userID
	payload := "{\"user_id\":\"" + userID + "\"}"
	metaMap := map[string]interface{}{
		"scheduling": map[string]interface{}{
			"timezone": "Africa/Lagos",
		},
	}
	metaStruct, err := structpb.NewStruct(metaMap)
	if err != nil {
		log.Error("Failed to create metadata struct", zap.Error(err))
		return
	}
	job := &schedulerpb.Job{
		Name:     jobName,
		Schedule: cronExpr,
		Payload:  payload,
		JobType:  schedulerpb.JobType_JOB_TYPE_CUSTOM,
		Metadata: &commonpb.Metadata{ServiceSpecific: metaStruct},
	}
	// Resolve SchedulerServiceClient from DI container
	var schedulerClient schedulerpb.SchedulerServiceClient
	err = provider.Container.Resolve(&schedulerClient)
	if err != nil {
		log.Error("Failed to resolve SchedulerServiceClient", zap.Error(err))
		return
	}
	_, err = schedulerClient.CreateJob(ctx, &schedulerpb.CreateJobRequest{Job: job})
	if err != nil {
		log.Error("Failed to create payday job", zap.String("user_id", userID), zap.Error(err))
		return
	}
	log.Info("Scheduled payday job (CreateJob called)", zap.String("user_id", userID), zap.String("cron", cronExpr), zap.String("job_name", jobName), zap.String("payload", payload))
}

// Handler for payday job execution: emits a payday.triggered event for the user.
func HandlePaydayJob(ctx context.Context, provider *service.Provider, job *schedulerpb.Job, log *zap.Logger) {
	// Resolve the canonical *Service from the DI container
	var schedulerService *Service
	if err := provider.Container.Resolve(&schedulerService); err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to resolve scheduler service for payday job", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return
	}
	userID := ""
	if job != nil && job.Payload != "" {
		// Simple JSON extraction (for demo; use a real JSON parser in production)
		payload := job.Payload
		if len(payload) > 12 {
			userID = payload[11 : len(payload)-2] // assumes {"user_id":"..."}
		}
	}
	if userID == "" {
		log.Warn("payday job missing user_id in payload")
		return
	}
	// Emit payday.triggered event using the service's event emitter if needed
	err := provider.EmitEvent(ctx, "payday.triggered", userID, job.Metadata)
	if err != nil {
		log.Error("Failed to emit payday.triggered event", zap.String("user_id", userID), zap.Error(err))
		return
	}
	log.Info("Emitted payday.triggered event", zap.String("user_id", userID))
}

var SchedulerEventRegistry = EventRegistry{
	{
		EventTypes: []string{"user.created"},
		Handler:    handleUserCreated,
	},
}

func StartEventSubscribers(ctx context.Context, provider *service.Provider, log *zap.Logger) {
	for _, sub := range SchedulerEventRegistry {
		sub := sub // capture range var
		go func() {
			err := provider.SubscribeEvents(ctx, sub.EventTypes, nil, func(ctx context.Context, event *nexusv1.EventResponse) {
				sub.Handler(ctx, provider, event, log)
			})
			if err != nil {
				log.Error("Failed to subscribe to events", zap.Strings("eventTypes", sub.EventTypes), zap.Error(err))
			}
		}()
	}
}
