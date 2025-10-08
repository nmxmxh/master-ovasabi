package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"github.com/nmxmxh/master-ovasabi/pkg/lifecycle"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"
)

// Context key types for type-safe context values.
type contextKey string

const (
	requestIDKey contextKey = "request_id"
	eventIDKey   contextKey = "event_id"
	eventTypeKey contextKey = "event_type"
)

type Provider struct {
	Log           *zap.Logger
	DB            *sql.DB
	RedisProvider *redis.Provider
	NexusClient   nexusv1.NexusServiceClient
	nexusConn     *grpc.ClientConn // Add connection reference for cleanup
	Container     *di.Container
	JWTSecret     string
	ctx           context.Context
	cancel        context.CancelFunc
	services      map[string]interface{}
	eventHandlers map[string]func(context.Context, interface{}) error
	EventEmitter  events.EventEmitter // Canonical event emitter for envelope emission
}

func NewProvider(log *zap.Logger, db *sql.DB, redisProvider *redis.Provider, nexusAddr string, container *di.Container, jwtSecret string) (*Provider, error) {
	conn, err := grpc.NewClient(nexusAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Error("Failed to connect to Nexus event bus", zap.Error(err))
		return nil, fmt.Errorf("failed to connect to nexus: %w", err)
	}
	log.Info("Connected to Nexus event bus", zap.String("address", nexusAddr))
	nexusClient := nexusv1.NewNexusServiceClient(conn)

	// Create context with cancellation for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())

	provider := &Provider{
		Log:           log,
		DB:            db,
		RedisProvider: redisProvider,
		NexusClient:   nexusClient,
		nexusConn:     conn,
		Container:     container,
		JWTSecret:     jwtSecret,
		ctx:           ctx,
		cancel:        cancel,
		services:      make(map[string]interface{}),
		eventHandlers: make(map[string]func(context.Context, interface{}) error),
	}

	// Register provider cleanup with lifecycle management if available
	lifecycle.RegisterCleanup(container, "service-provider", func() error {
		log.Info("Shutting down service provider")
		provider.Shutdown()
		return nil
	})

	return provider, nil
}

// Helper function to safely get request ID from context.
func getRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value("request_id").(string); ok {
		return requestID
	}
	return ""
}

// EmitEvent emits an event to the Nexus event bus.
func (p *Provider) EmitEvent(ctx context.Context, eventType, entityID string, meta *commonpb.Metadata) error {
	if p.NexusClient == nil {
		err := fmt.Errorf("nexus client is not initialized")
		if p.Log != nil {
			p.Log.Error("NexusClient is nil, cannot emit event",
				zap.String("eventType", eventType),
				zap.String("entityID", entityID),
				zap.Error(err),
			)
		}
		return graceful.WrapErr(ctx, codes.Internal, "nexus client not initialized", err)
	}

	req := &nexusv1.EventRequest{
		EventType: eventType,
		EntityId:  entityID,
		Metadata:  meta,
	}

	_, err := p.NexusClient.EmitEvent(ctx, req)
	if err != nil {
		if p.Log != nil {
			p.Log.Error("Failed to emit event to Nexus",
				zap.String("eventType", eventType),
				zap.String("entityID", entityID),
				zap.Error(err),
			)
		}
		return graceful.WrapErr(ctx, codes.Internal, "failed to emit event to nexus", err)
	}

	if p.Log != nil {
		p.Log.Debug("Event emitted to Nexus",
			zap.String("eventType", eventType),
			zap.String("entityID", entityID),
		)
	}
	return nil
}

// SubscribeEvents subscribes to events from Nexus and handles them with the provided handler.
// This method now includes automatic event lifecycle orchestration (requested -> started -> success/failed).
func (p *Provider) SubscribeEvents(ctx context.Context, eventTypes []string, meta *commonpb.Metadata, handle func(context.Context, *nexusv1.EventResponse)) error {
	req := &nexusv1.SubscribeRequest{
		EventTypes: eventTypes,
		Metadata:   meta,
	}
	stream, err := p.NexusClient.SubscribeEvents(ctx, req)
	if err != nil {
		if p.Log != nil {
			p.Log.Error("Failed to subscribe to events from Nexus",
				zap.Strings("eventTypes", eventTypes),
				zap.Error(err),
				zap.String("request_id", getRequestID(ctx)),
			)
		}
		return graceful.WrapErr(ctx, codes.Internal, "failed to subscribe to events from Nexus", err)
	}

	// Run the receive loop in a goroutine so it doesn't block the caller
	go func() {
		retryCount := 0
		maxRetries := 5
		baseDelay := time.Second

		for {
			select {
			case <-ctx.Done():
				return
			default:
				event, err := stream.Recv()
				if err != nil {
					if p.Log != nil {
						p.Log.Error("Error receiving event from Nexus stream",
							zap.Strings("eventTypes", eventTypes),
							zap.Error(err),
							zap.String("request_id", getRequestID(ctx)),
							zap.Int("retry_count", retryCount),
						)
					}

					// Check if this is a recoverable error and we haven't exceeded max retries
					if retryCount < maxRetries && (errors.Is(err, io.EOF) || strings.Contains(err.Error(), "EOF") || strings.Contains(err.Error(), "connection reset")) {
						retryCount++
						delay := time.Duration(retryCount) * baseDelay
						if p.Log != nil {
							p.Log.Info("Attempting to reconnect to Nexus stream",
								zap.Strings("eventTypes", eventTypes),
								zap.Int("retry_count", retryCount),
								zap.Duration("delay", delay),
							)
						}

						// Wait before retrying
						select {
						case <-time.After(delay):
							// Try to create a new stream
							newStream, streamErr := p.NexusClient.SubscribeEvents(ctx, req)
							if streamErr != nil {
								if p.Log != nil {
									p.Log.Error("Failed to recreate Nexus stream",
										zap.Strings("eventTypes", eventTypes),
										zap.Error(streamErr),
										zap.Int("retry_count", retryCount),
									)
								}
								continue
							}
							stream = newStream
							continue
						case <-ctx.Done():
							return
						}
					}
					return
				}

				// Reset retry count on successful receive
				retryCount = 0

				// Create a child context with event metadata
				eventCtx := context.WithValue(ctx, requestIDKey, getRequestID(ctx))
				if event.EventId != "" {
					eventCtx = context.WithValue(eventCtx, eventIDKey, event.EventId)
				}
				if event.EventType != "" {
					eventCtx = context.WithValue(eventCtx, eventTypeKey, event.EventType)
				}

				// Enhanced event handling with automatic lifecycle orchestration
				p.handleEventWithLifecycle(eventCtx, event, handle)
			}
		}
	}()

	if p.Log != nil {
		p.Log.Info("Subscribed to events from Nexus",
			zap.Strings("eventTypes", eventTypes),
			zap.String("request_id", getRequestID(ctx)),
		)
	}
	return nil
}

// EmitEchoEvent emits a canonical 'echo' event to Nexus for testing and onboarding.
func (p *Provider) EmitEchoEvent(ctx context.Context, serviceName, message string) (string, bool) {
	envelope := &events.EventEnvelope{
		Type: "echo",
		ID:   serviceName,
		Metadata: &commonpb.Metadata{
			ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{
				"echo": map[string]interface{}{
					"service":   serviceName,
					"message":   message,
					"timestamp": time.Now().Format(time.RFC3339),
				},
				"nexus": map[string]interface{}{
					"actor": map[string]interface{}{
						"user_id": serviceName,
						"roles":   []string{"system"},
					},
				},
			}, nil),
		},
	}
	id, err := p.EmitEventEnvelope(ctx, envelope)
	return id, err == nil
}

// RegisterService registers a service with the provider.
func (p *Provider) RegisterService(name string, service interface{}) error {
	p.Log.Info("Registering service",
		zap.String("service", name),
		zap.String("request_id", getRequestID(p.ctx)),
	)
	p.services[name] = service
	return nil
}

// GetService retrieves a service from the provider.
func (p *Provider) GetService(name string) (interface{}, error) {
	p.Log.Info("Getting service",
		zap.String("service", name),
		zap.String("request_id", getRequestID(p.ctx)),
	)
	if service, ok := p.services[name]; ok {
		return service, nil
	}
	return nil, fmt.Errorf("service %s not found", name)
}

// HandleEvent processes events using registered handlers.
func (p *Provider) HandleEvent(eventType string, payload interface{}) error {
	p.Log.Info("Handling event",
		zap.String("event_type", eventType),
		zap.String("request_id", getRequestID(p.ctx)),
	)
	if handler, ok := p.eventHandlers[eventType]; ok {
		return handler(p.ctx, payload)
	}
	return fmt.Errorf("no handler registered for event type %s", eventType)
}

// Shutdown gracefully shuts down the provider.
func (p *Provider) Shutdown() {
	p.Log.Info("Shutting down provider",
		zap.String("request_id", getRequestID(p.ctx)),
	)

	// Cancel context to stop all operations
	if p.cancel != nil {
		p.cancel()
	}

	// Close gRPC connection
	if p.nexusConn != nil {
		if err := p.nexusConn.Close(); err != nil {
			p.Log.Warn("Failed to close Nexus connection", zap.Error(err))
		}
	}
}

// EmitEventEnvelope emits a canonical EventEnvelope to Nexus and logs the outcome. This is the preferred event emission method.
func (p *Provider) EmitEventEnvelope(ctx context.Context, envelope *events.EventEnvelope) (string, error) {
	if p.NexusClient == nil {
		return "", fmt.Errorf("nexus client is not initialized")
	}

	// Use the envelope's payload directly instead of wrapping it in another payload
	req := &nexusv1.EventRequest{
		EventType: envelope.Type,
		EventId:   envelope.ID,
		Metadata:  envelope.Metadata,
		Payload:   envelope.Payload, // Use the envelope's payload directly
	}

	_, err := p.NexusClient.EmitEvent(ctx, req)
	if err != nil {
		return envelope.ID, fmt.Errorf("failed to emit EventEnvelope to Nexus: %w", err)
	}

	return envelope.ID, nil
}

// handleEventWithLifecycle processes events with automatic lifecycle orchestration.
func (p *Provider) handleEventWithLifecycle(ctx context.Context, event *nexusv1.EventResponse, handler func(context.Context, *nexusv1.EventResponse)) {
	eventType := event.GetEventType()
	eventID := event.EventId

	// Check if this is a streaming event
	if p.isStreamingEvent(eventType) {
		p.handleStreamingEvent(ctx, event, handler)
		return
	}

	// Only process requested events for lifecycle orchestration
	if !strings.HasSuffix(eventType, ":requested") {
		// For non-requested events, just call the handler directly
		handler(ctx, event)
		return
	}

	// Extract service and action from event type
	parts := strings.Split(eventType, ":")
	if len(parts) < 2 {
		if p.Log != nil {
			p.Log.Warn("Invalid event type format", zap.String("event_type", eventType))
		}
		handler(ctx, event)
		return
	}

	serviceName := parts[0]
	action := parts[1]

	// Emit started event
	startedEventType := fmt.Sprintf("%s:%s:v1:started", serviceName, action)
	startedEvent := &nexusv1.EventRequest{
		EventType: startedEventType,
		EventId:   eventID + ":started",
		Metadata:  event.Metadata,
		Payload:   event.Payload,
	}

	if p.NexusClient != nil {
		if _, err := p.NexusClient.EmitEvent(ctx, startedEvent); err != nil && p.Log != nil {
			p.Log.Warn("Failed to emit started event",
				zap.String("event_type", startedEventType),
				zap.Error(err))
		}
	}

	// Process the business logic with panic recovery
	var processErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				if p.Log != nil {
					p.Log.Error("Panic in event handler",
						zap.String("event_type", eventType),
						zap.String("event_id", eventID),
						zap.Any("panic", r))
				}
				processErr = fmt.Errorf("panic in handler: %v", r)
			}
		}()

		handler(ctx, event)
	}()

	// Emit success or failure event based on result
	successEventType := fmt.Sprintf("%s:%s:v1:success", serviceName, action)
	failedEventType := fmt.Sprintf("%s:%s:v1:failed", serviceName, action)

	var resultEvent *nexusv1.EventRequest
	if processErr != nil {
		// Create failure payload
		failurePayload := map[string]interface{}{
			"action":            action,
			"error":             processErr.Error(),
			"failed_at":         time.Now().UTC().Format(time.RFC3339),
			"original_event_id": eventID,
		}

		// Merge with original payload if it exists
		if event.Payload != nil && event.Payload.Data != nil {
			originalData := event.Payload.Data.AsMap()
			for k, v := range originalData {
				failurePayload[k] = v
			}
		}

		payloadStruct, err := structpb.NewStruct(failurePayload)
		if err != nil {
			payloadStruct = &structpb.Struct{}
		}

		resultEvent = &nexusv1.EventRequest{
			EventType: failedEventType,
			EventId:   eventID, // Keep original EventID for correlation tracking
			Metadata:  event.Metadata,
			Payload: &commonpb.Payload{
				Data: payloadStruct,
			},
		}

		if p.Log != nil {
			p.Log.Error("Event processing failed",
				zap.String("action", action),
				zap.String("event_id", eventID),
				zap.Error(processErr))
		}
	} else {
		// Create success payload with timing information
		successPayload := map[string]interface{}{
			"action":            action,
			"completed_at":      time.Now().UTC().Format(time.RFC3339),
			"original_event_id": eventID,
		}

		// Merge with original payload if it exists
		if event.Payload != nil && event.Payload.Data != nil {
			originalData := event.Payload.Data.AsMap()
			for k, v := range originalData {
				successPayload[k] = v
			}
		}

		payloadStruct, err := structpb.NewStruct(successPayload)
		if err != nil {
			payloadStruct = &structpb.Struct{}
		}

		resultEvent = &nexusv1.EventRequest{
			EventType: successEventType,
			EventId:   eventID, // Keep original EventID for correlation tracking
			Metadata:  event.Metadata,
			Payload: &commonpb.Payload{
				Data: payloadStruct,
			},
		}

		if p.Log != nil {
			p.Log.Info("Event processing succeeded",
				zap.String("action", action),
				zap.String("event_id", eventID))
		}
	}

	// Emit the result event
	if p.NexusClient != nil && resultEvent != nil {
		if _, err := p.NexusClient.EmitEvent(ctx, resultEvent); err != nil && p.Log != nil {
			p.Log.Warn("Failed to emit result event",
				zap.String("event_type", resultEvent.EventType),
				zap.Error(err))
		}
	}
}

// isStreamingEvent determines if an event type should be handled as a streaming event.
func (p *Provider) isStreamingEvent(eventType string) bool {
	// Define streaming event patterns
	streamingPatterns := []string{
		"stream_",    // Events that start with "stream_"
		":stream:",   // Events with ":stream:" in the action
		":typing:",   // Typing events
		":presence:", // Presence events
		":chunks:",   // Chunk streaming events
		":live:",     // Live events
		":realtime:", // Real-time events
	}

	for _, pattern := range streamingPatterns {
		if strings.Contains(eventType, pattern) {
			return true
		}
	}

	// Check for specific streaming event types
	streamingEventTypes := map[string]bool{
		"messaging:stream_messages:v1:requested":        true,
		"messaging:stream_typing:v1:requested":          true,
		"messaging:stream_presence:v1:requested":        true,
		"notification:stream_asset_chunks:v1:requested": true,
		"media:stream_media_content:v1:requested":       true,
		"search:stream_results:v1:requested":            true,
		"analytics:stream_metrics:v1:requested":         true,
		// Campaign state streaming
		"campaign:stream_state:v1:requested":     true,
		"campaign:stream_analytics:v1:requested": true,
		"campaign:stream_events:v1:requested":    true,
		// Media streaming integration
		"media:stream_webrtc:v1:requested": true,
		"media:stream_rooms:v1:requested":  true,
		"media:stream_peers:v1:requested":  true,
	}

	return streamingEventTypes[eventType]
}

// handleStreamingEvent processes streaming events with continuous event emission.
func (p *Provider) handleStreamingEvent(ctx context.Context, event *nexusv1.EventResponse, handler func(context.Context, *nexusv1.EventResponse)) {
	eventType := event.GetEventType()
	eventID := event.EventId

	if p.Log != nil {
		p.Log.Info("Processing streaming event",
			zap.String("event_type", eventType),
			zap.String("event_id", eventID))
	}

	// Extract service and action from event type
	parts := strings.Split(eventType, ":")
	if len(parts) < 2 {
		if p.Log != nil {
			p.Log.Warn("Invalid streaming event type format", zap.String("event_type", eventType))
		}
		handler(ctx, event)
		return
	}

	serviceName := parts[0]
	action := parts[1]

	// Emit stream started event
	streamStartedType := fmt.Sprintf("%s:%s:v1:started", serviceName, action)
	streamStartedEvent := &nexusv1.EventRequest{
		EventType: streamStartedType,
		EventId:   eventID + ":stream_started",
		Metadata:  event.Metadata,
		Payload:   event.Payload,
	}

	if p.NexusClient != nil {
		if _, err := p.NexusClient.EmitEvent(ctx, streamStartedEvent); err != nil && p.Log != nil {
			p.Log.Warn("Failed to emit stream started event",
				zap.String("event_type", streamStartedType),
				zap.Error(err))
		}
	}

	// Create a context with timeout for streaming (configurable)
	streamCtx, cancel := context.WithTimeout(ctx, 30*time.Minute) // Default 30 min timeout
	defer cancel()

	// For streaming events, we need to call the handler and let it handle the streaming logic
	// The handler should be designed to work with streaming events
	handler(streamCtx, event)

	// Emit stream success event
	successEventType := fmt.Sprintf("%s:%s:v1:success", serviceName, action)
	p.emitStreamResult(ctx, successEventType, eventID+":stream_success", event.Metadata, map[string]interface{}{
		"action":            action,
		"completed_at":      time.Now().UTC().Format(time.RFC3339),
		"original_event_id": eventID,
	})
}

// emitStreamResult emits the final result of a streaming operation.
func (p *Provider) emitStreamResult(ctx context.Context, eventType, eventID string, metadata *commonpb.Metadata, payload map[string]interface{}) {
	// Merge with original payload if it exists
	if metadata != nil && metadata.ServiceSpecific != nil {
		originalData := metadata.ServiceSpecific.AsMap()
		for k, v := range originalData {
			payload[k] = v
		}
	}

	payloadStruct, err := structpb.NewStruct(payload)
	if err != nil {
		payloadStruct = &structpb.Struct{}
	}

	resultEvent := &nexusv1.EventRequest{
		EventType: eventType,
		EventId:   eventID,
		Metadata:  metadata,
		Payload: &commonpb.Payload{
			Data: payloadStruct,
		},
	}

	if p.NexusClient != nil {
		if _, err := p.NexusClient.EmitEvent(ctx, resultEvent); err != nil && p.Log != nil {
			p.Log.Warn("Failed to emit stream result event",
				zap.String("event_type", eventType),
				zap.Error(err))
		}
	}
}
