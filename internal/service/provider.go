package service

import (
	"context"
	"database/sql"
	"fmt"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type Provider struct {
	Log           *zap.Logger
	DB            *sql.DB
	RedisProvider *redis.Provider
	NexusClient   nexusv1.NexusServiceClient
	Container     *di.Container
}

func NewProvider(log *zap.Logger, db *sql.DB, redisProvider *redis.Provider, nexusAddr string, container *di.Container) (*Provider, error) {
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
	}, nil
}

// EmitEvent emits an event to the Nexus event bus.
func (p *Provider) EmitEvent(ctx context.Context, eventType, entityID string, metadata *commonpb.Metadata) error {
	p.Log.Info("Emitting event to Nexus", zap.String("eventType", eventType), zap.String("entityId", entityID), zap.Any("metadata", metadata))
	_, err := p.NexusClient.EmitEvent(ctx, &nexusv1.EventRequest{
		EventType: eventType,
		EntityId:  entityID,
		Metadata:  metadata,
	})
	if err != nil {
		p.Log.Error("Failed to emit event", zap.Error(err))
	}
	return err
}

// SubscribeEvents subscribes to events from the Nexus event bus.
func (p *Provider) SubscribeEvents(ctx context.Context, eventTypes []string, metadata *commonpb.Metadata, handle func(*nexusv1.EventResponse)) error {
	stream, err := p.NexusClient.SubscribeEvents(ctx, &nexusv1.SubscribeRequest{
		EventTypes: eventTypes,
		Metadata:   metadata,
	})
	if err != nil {
		p.Log.Error("Failed to subscribe to events", zap.Error(err), zap.Strings("eventTypes", eventTypes))
		return err
	}
	p.Log.Info("Successfully subscribed to events from Nexus", zap.Strings("eventTypes", eventTypes), zap.Any("metadata", metadata))
	for {
		event, err := stream.Recv()
		if err != nil {
			st, ok := status.FromError(err)
			if ok && st.Code() == codes.Canceled {
				// Normal shutdown, do not log as error
				return err
			}
			p.Log.Error("Error receiving event from Nexus stream", zap.Error(err))
			return err
		}
		handle(event)
	}
}
