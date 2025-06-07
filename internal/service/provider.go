package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
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
	Container     *di.Container
	JWTSecret     string
	ctx           context.Context
	cancel        context.CancelFunc
	services      map[string]interface{}
	eventHandlers map[string]func(context.Context, interface{}) error
}

func NewProvider(log *zap.Logger, db *sql.DB, redisProvider *redis.Provider, nexusAddr string, container *di.Container, jwtSecret string) (*Provider, error) {
	conn, err := grpc.DialContext(context.Background(), nexusAddr, grpc.WithTransportCredentials(insecure.NewCredentials())) //nolint:staticcheck // grpc.DialContext is required until generated client supports NewClient API
	if err != nil {
		log.Error("Failed to connect to Nexus event bus", zap.Error(err))
		return nil, fmt.Errorf("failed to dial nexus: %w", err)
	}
	log.Info("Connected to Nexus event bus", zap.String("address", nexusAddr))
	nexusClient := nexusv1.NewNexusServiceClient(conn)
	return &Provider{
		Log:           log,
		DB:            db,
		RedisProvider: redisProvider,
		NexusClient:   nexusClient,
		Container:     container,
		JWTSecret:     jwtSecret,
		services:      make(map[string]interface{}),
		eventHandlers: make(map[string]func(context.Context, interface{}) error),
	}, nil
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
		if p.Log != nil {
			p.Log.Error("NexusClient is nil, cannot emit event",
				zap.String("eventType", eventType),
				zap.String("entityID", entityID),
			)
		}
		return graceful.WrapErr(ctx, codes.Internal, "NexusClient is nil", nil)
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
		return graceful.WrapErr(ctx, codes.Internal, "failed to emit event to Nexus", err)
	}
	if p.Log != nil {
		p.Log.Info("Event emitted to Nexus",
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

// EmitEventWithLogging emits an event to Nexus and logs the outcome, orchestrating errors with graceful.
func (p *Provider) EmitEventWithLogging(ctx context.Context, _ interface{}, log *zap.Logger, eventType, eventID string, meta *commonpb.Metadata) (string, bool) {
	if p.NexusClient == nil {
		if log != nil {
			log.Error("NexusClient is nil, cannot emit event",
				zap.String("eventType", eventType),
				zap.String("eventID", eventID),
			)
		}
		return "", false
	}

	// Use the canonical EmitEvent method
	err := p.EmitEvent(ctx, eventType, eventID, meta)
	if err != nil {
		if log != nil {
			log.Error("Failed to emit event to Nexus",
				zap.String("eventType", eventType),
				zap.String("eventID", eventID),
				zap.Error(err),
			)
		}
		graceful.WrapErr(ctx, codes.Internal, "failed to emit event to Nexus", err).StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{
			Log:     log,
			Context: ctx,
		})
		return "", false
	}
	if log != nil {
		log.Info("Event emitted to Nexus",
			zap.String("eventType", eventType),
			zap.String("eventID", eventID),
			zap.String("nexus_message", "event emitted"),
		)
	}
	return eventID, true
}

// EmitRawEventWithLogging emits a raw event (with payload) to Nexus and logs the outcome.
func (p *Provider) EmitRawEventWithLogging(_ context.Context, log *zap.Logger, eventType, eventID string, payload []byte) (string, bool) {
	if p.NexusClient == nil {
		if log != nil {
			log.Error("NexusClient is nil, cannot emit raw event",
				zap.String("eventType", eventType),
				zap.String("eventID", eventID),
			)
		}
		return "", false
	}
	// For demonstration, just log the payload and event info. In production, you would unmarshal and send as needed.
	if log != nil {
		log.Info("EmitRawEventWithLogging called",
			zap.String("eventType", eventType),
			zap.String("eventID", eventID),
			zap.ByteString("payload", payload),
		)
	}
	// Optionally, you could unmarshal payload and send to NexusClient.EmitEvent if needed.
	return eventID, true
}

// EmitEchoEvent emits a canonical 'echo' event to Nexus for testing and onboarding.
func (p *Provider) EmitEchoEvent(ctx context.Context, serviceName, message string) (string, bool) {
	meta := &commonpb.Metadata{
		ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{
			"echo": map[string]interface{}{
				"service":   serviceName,
				"message":   message,
				"timestamp": time.Now().Format(time.RFC3339),
			},
			// Ensure required nexus.actor field for validation
			"nexus": map[string]interface{}{
				"actor": map[string]interface{}{
					"user_id": serviceName,
					"roles":   []string{"system"},
				},
			},
		}, nil),
	}
	return p.EmitEventWithLogging(ctx, nil, p.Log, "echo", serviceName, meta)
}

// StartEchoLoop starts a background goroutine that emits an echo event every 15 seconds.
func (p *Provider) StartEchoLoop(ctx context.Context, serviceName, message string) {
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				p.Log.Info("Echo loop stopped", zap.String("service", serviceName))
				return
			case <-ticker.C:
				msg, ok := p.EmitEchoEvent(ctx, serviceName, message)
				if ok {
					p.Log.Info("Echo event emitted", zap.String("service", serviceName), zap.String("msg", msg))
				} else {
					p.Log.Warn("Failed to emit echo event", zap.String("service", serviceName))
				}
			}
		}
	}()
}

// Update all logging calls to use the helper function.
func (p *Provider) RegisterService(name string, service interface{}) error {
	p.Log.Info("Registering service",
		zap.String("service", name),
		zap.String("request_id", getRequestID(p.ctx)),
	)
	p.services[name] = service
	return nil
}

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

func (p *Provider) Shutdown() {
	p.Log.Info("Shutting down provider",
		zap.String("request_id", getRequestID(p.ctx)),
	)
	p.cancel()
}
