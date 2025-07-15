// Package graceful provides a streamlined handler for orchestrating success and error events.
// It offers a minimal, symmetrical implementation for handling success and error conditions,
// including cache management and event emission, without extensive data or metadata manipulation.
package graceful

import (
	"context"
	"fmt"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

// Cache defines the interface for a cache, allowing for simple key-value storage with a TTL.
type Cache interface {
	Set(ctx context.Context, key, field string, value interface{}, ttl time.Duration) error
}

// EventEmitter defines the interface for emitting raw events, consistent with other services.
type EventEmitter interface {
	EmitEventEnvelope(ctx context.Context, envelope *events.EventEnvelope) (string, error)
}

// Handler provides a simplified, linear approach to handling service events.
// It is responsible for logging, wrapping, and emitting orchestration events.
type Handler struct {
	Log          *zap.Logger
	EventEmitter EventEmitter
	Cache        Cache
	EventEnabled bool
	Service      string // The name of the service using this handler.
	Version      string // The version of the service.
}

// NewHandler creates a new Handler with the required dependencies.
func NewHandler(log *zap.Logger, eventEmitter EventEmitter, cache Cache, service, version string, eventEnabled bool) *Handler {
	return &Handler{
		Log:          log,
		EventEmitter: eventEmitter,
		Cache:        cache,
		Service:      service,
		Version:      version,
		EventEnabled: eventEnabled,
	}
}

// Success handles a successful operation by logging, caching, and emitting an event.
func (h *Handler) Success(ctx context.Context, action string, code codes.Code, msg string, result interface{}, metaVal interface{}, entityID string, cacheInfo *CacheInfo) *SuccessContext {
	sCtx := &SuccessContext{
		Code:      code,
		Message:   msg,
		Result:    result,
		Timestamp: time.Now().UTC(),
	}

	if h.Log != nil {
		h.Log.Info(msg,
			zap.String("service", h.Service),
			zap.String("entityID", entityID),
			zap.String("code", code.String()),
		)
	}

	// Caching logic
	if cacheInfo != nil && h.Cache != nil {
		// Use a default TTL of 5 minutes for now.
		if err := h.Cache.Set(ctx, cacheInfo.Key, "", result, 5*time.Minute); err != nil {
			h.Log.Warn("Failed to cache successful response", zap.String("key", cacheInfo.Key), zap.Error(err))
		}
	}

	if h.EventEnabled && h.EventEmitter != nil {
		eventType := fmt.Sprintf("%s:%s:%s:%s", h.Service, action, h.Version, "success")
		eventID := entityID
		timestamp := sCtx.Timestamp.Format(time.RFC3339)
		payloadMap := map[string]interface{}{
			"code":           code.String(),
			"message":        msg,
			"result":         result,
			"yin_yang":       "yang",
			"correlation_id": utils.GetStringFromContext(ctx, "correlation_id"),
			"service":        h.Service,
			"actor_id":       utils.GetStringFromContext(ctx, "actor_id"),
			"request_id":     utils.GetStringFromContext(ctx, "request_id"),
			"timestamp":      timestamp,
		}
		// Convert payload to *structpb.Struct
		payloadStruct := metadata.NewStructFromMap(payloadMap, nil)
		var meta *commonpb.Metadata
		if m, ok := metaVal.(*commonpb.Metadata); ok {
			meta = m
		}
		var ts int64
		t, err := time.Parse(time.RFC3339, timestamp)
		if err == nil {
			ts = t.Unix()
		} else {
			ts = time.Now().Unix()
		}
		envelope := &events.EventEnvelope{
			ID:        eventID,
			Type:      eventType,
			Payload:   &commonpb.Payload{Data: payloadStruct},
			Metadata:  meta,
			Timestamp: ts,
		}
		eventID, emitErr := h.EventEmitter.EmitEventEnvelope(ctx, envelope)
		if emitErr != nil && h.Log != nil {
			h.Log.Warn("Failed to emit success event envelope", zap.String("event_id", eventID), zap.Error(emitErr))
		}
	}

	return sCtx
}

// Error handles a failed operation by logging and emitting an event.
func (h *Handler) Error(ctx context.Context, action string, code codes.Code, msg string, cause error, metaVal interface{}, entityID string) *ContextError {
	errCtx := &ContextError{
		Code:    code,
		Message: msg,
		Cause:   cause,
	}

	if h.Log != nil {
		h.Log.Error(msg,
			zap.String("service", h.Service),
			zap.String("entityID", entityID),
			zap.String("code", code.String()),
			zap.Error(cause),
		)
	}

	if h.EventEnabled && h.EventEmitter != nil {
		eventType := fmt.Sprintf("%s:%s:%s:%s", h.Service, action, h.Version, "failed")
		eventID := entityID
		timestamp := time.Now().UTC().Format(time.RFC3339)
		payloadMap := map[string]interface{}{
			"code":           code.String(),
			"message":        msg,
			"result":         map[string]string{"error": errCtx.Error()},
			"yin_yang":       "yin",
			"correlation_id": utils.GetStringFromContext(ctx, "correlation_id"),
			"service":        h.Service,
			"actor_id":       utils.GetStringFromContext(ctx, "actor_id"),
			"request_id":     utils.GetStringFromContext(ctx, "request_id"),
			"timestamp":      timestamp,
		}
		payloadStruct := metadata.NewStructFromMap(payloadMap, nil)
		var meta *commonpb.Metadata
		if m, ok := metaVal.(*commonpb.Metadata); ok {
			meta = m
		}
		var ts int64
		t, err := time.Parse(time.RFC3339, timestamp)
		if err == nil {
			ts = t.Unix()
		} else {
			ts = time.Now().Unix()
		}
		envelope := &events.EventEnvelope{
			ID:        eventID,
			Type:      eventType,
			Payload:   &commonpb.Payload{Data: payloadStruct},
			Metadata:  meta,
			Timestamp: ts,
		}
		eventID, emitErr := h.EventEmitter.EmitEventEnvelope(ctx, envelope)
		if emitErr != nil && h.Log != nil {
			h.Log.Warn("Failed to emit error event envelope", zap.String("event_id", eventID), zap.Error(emitErr))
		}
	}

	return errCtx
}

// CacheInfo holds the information needed to cache a value.
type CacheInfo struct {
	Key   string
	Value interface{}
	TTL   time.Duration
}
