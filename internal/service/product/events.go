package product

import (
	"context"
	"strings"

	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	productpb "github.com/nmxmxh/master-ovasabi/api/protos/product/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
)

// CanonicalEventTypeRegistry provides lookup and validation for canonical event types.
var CanonicalEventTypeRegistry map[string]string

func InitCanonicalEventTypeRegistry() {
	CanonicalEventTypeRegistry = make(map[string]string)
	evts := loadProductEvents()
	for _, evt := range evts {
		parts := strings.Split(evt, ":")
		if len(parts) >= 4 {
			key := parts[1] + ":" + parts[3]
			CanonicalEventTypeRegistry[key] = evt
		}
	}
}

func GetCanonicalEventType(action, state string) string {
	if CanonicalEventTypeRegistry == nil {
		InitCanonicalEventTypeRegistry()
	}
	key := action + ":" + state
	if evt, ok := CanonicalEventTypeRegistry[key]; ok {
		return evt
	}
	return ""
}

func loadProductEvents() []string {
	return events.LoadCanonicalEvents("product")
}

// ActionHandlerFunc defines the signature for business logic handlers for each action.
type ActionHandlerFunc func(ctx context.Context, s *Service, event *nexusv1.EventResponse)

var actionHandlers = map[string]ActionHandlerFunc{}

func RegisterActionHandler(action string, handler ActionHandlerFunc) {
	actionHandlers[action] = handler
}

func parseActionAndState(eventType string) (action, state string) {
	parts := strings.Split(eventType, ":")
	if len(parts) >= 4 {
		return parts[1], parts[3]
	}
	return "", ""
}

// Generic event handler for all product service actions
func HandleProductServiceEvent(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
	eventType := event.GetEventType()
	action, _ := parseActionAndState(eventType)
	handler, ok := actionHandlers[action]
	if !ok {
		if s.log != nil {
			s.log.Warn("No handler for action", zap.String("action", action), zap.String("event_type", eventType))
		}
		return
	}
	expectedPrefix := "product:" + action + ":"
	if !strings.HasPrefix(eventType, expectedPrefix) {
		if s.log != nil {
			s.log.Warn("Event type does not match handler action, ignoring", zap.String("event_type", eventType), zap.String("expected_prefix", expectedPrefix))
		}
		return
	}
	handler(ctx, s, event)
}

// Register all canonical event types to the generic handler
var eventTypeToHandler = func() map[string]ActionHandlerFunc {
	InitCanonicalEventTypeRegistry()
	m := make(map[string]ActionHandlerFunc)
	for _, evt := range loadProductEvents() {
		m[evt] = HandleProductServiceEvent
	}
	return m
}()

// Canonical product event handlers (cover all product actions from actions.txt)
func handleCreateProduct(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
	var req productpb.CreateProductRequest
	if event.Payload != nil && event.Payload.Data != nil {
		b, err := protojson.Marshal(event.Payload.Data)
		if err == nil {
			err = protojson.Unmarshal(b, &req)
		}
		if err != nil {
			s.log.Error("Failed to unmarshal CreateProductRequest payload", zap.Error(err))
			return
		}
	}
	s.CreateProduct(ctx, &req)
}

func handleUpdateProduct(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
	var req productpb.UpdateProductRequest
	if event.Payload != nil && event.Payload.Data != nil {
		b, err := protojson.Marshal(event.Payload.Data)
		if err == nil {
			err = protojson.Unmarshal(b, &req)
		}
		if err != nil {
			s.log.Error("Failed to unmarshal UpdateProductRequest payload", zap.Error(err))
			return
		}
	}
	s.UpdateProduct(ctx, &req)
}

func handleUpdateInventory(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
	var req productpb.UpdateInventoryRequest
	if event.Payload != nil && event.Payload.Data != nil {
		b, err := protojson.Marshal(event.Payload.Data)
		if err == nil {
			err = protojson.Unmarshal(b, &req)
		}
		if err != nil {
			s.log.Error("Failed to unmarshal UpdateInventoryRequest payload", zap.Error(err))
			return
		}
	}
	s.UpdateInventory(ctx, &req)
}

func handleDeleteProduct(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
	var req productpb.DeleteProductRequest
	if event.Payload != nil && event.Payload.Data != nil {
		b, err := protojson.Marshal(event.Payload.Data)
		if err == nil {
			err = protojson.Unmarshal(b, &req)
		}
		if err != nil {
			s.log.Error("Failed to unmarshal DeleteProductRequest payload", zap.Error(err))
			return
		}
	}
	s.DeleteProduct(ctx, &req)
}

func handleListProducts(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
	var req productpb.ListProductsRequest
	if event.Payload != nil && event.Payload.Data != nil {
		b, err := protojson.Marshal(event.Payload.Data)
		if err == nil {
			err = protojson.Unmarshal(b, &req)
		}
		if err != nil {
			s.log.Error("Failed to unmarshal ListProductsRequest payload", zap.Error(err))
			return
		}
	}
	s.ListProducts(ctx, &req)
}

func handleSearchProducts(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
	var req productpb.SearchProductsRequest
	if event.Payload != nil && event.Payload.Data != nil {
		b, err := protojson.Marshal(event.Payload.Data)
		if err == nil {
			err = protojson.Unmarshal(b, &req)
		}
		if err != nil {
			s.log.Error("Failed to unmarshal SearchProductsRequest payload", zap.Error(err))
			return
		}
	}
	s.SearchProducts(ctx, &req)
}

// Register all product action handlers (from actions.txt)
func init() {
	RegisterActionHandler("create_product", handleCreateProduct)
	RegisterActionHandler("update_product", handleUpdateProduct)
	RegisterActionHandler("delete_product", handleDeleteProduct)
	RegisterActionHandler("list_products", handleListProducts)
	RegisterActionHandler("search_products", handleSearchProducts)
	RegisterActionHandler("update_inventory", handleUpdateInventory)
	// Add more handlers here for full coverage, matching the explicit messaging pattern
}
