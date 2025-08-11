package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
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
						)
					}
					return
				}

				// Create a child context with event metadata
				eventCtx := context.WithValue(ctx, requestIDKey, getRequestID(ctx))
				if event.EventId != "" {
					eventCtx = context.WithValue(eventCtx, eventIDKey, event.EventId)
				}
				if event.EventType != "" {
					eventCtx = context.WithValue(eventCtx, eventTypeKey, event.EventType)
				}

				// Call handler with enriched context
				handle(eventCtx, event)
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

	payloadBytes, err := json.Marshal(envelope)
	if err != nil {
		return "", fmt.Errorf("failed to marshal EventEnvelope: %w", err)
	}

	var payloadMap map[string]interface{}
	if err := json.Unmarshal(payloadBytes, &payloadMap); err != nil {
		return "", fmt.Errorf("failed to unmarshal EventEnvelope for structpb conversion: %w", err)
	}

	structData := metadata.NewStructFromMap(payloadMap, nil)

	req := &nexusv1.EventRequest{
		EventType: envelope.Type,
		EventId:   envelope.ID,
		Metadata:  envelope.Metadata,
		Payload:   &commonpb.Payload{Data: structData},
	}

	_, err = p.NexusClient.EmitEvent(ctx, req)
	if err != nil {
		return envelope.ID, fmt.Errorf("failed to emit EventEnvelope to Nexus: %w", err)
	}

	return envelope.ID, nil
}
