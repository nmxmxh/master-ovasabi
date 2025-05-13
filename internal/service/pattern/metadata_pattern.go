// Metadata Integration Pattern (Redis, Scheduler, Knowledge Graph, Nexus)
// -------------------------------------------------------------------
//
// This file provides reusable helpers and documentation for integrating
// Redis caching, Scheduler orchestration, Knowledge Graph enrichment,
// and Nexus orchestration with the canonical *commonpb.Metadata pattern.
//
// Usage:
// - Import and use these helpers in your service layer.
// - Register integration points in your service's Provider/DI setup.
// - Document any service-specific metadata fields in your proto and onboarding docs.
//
// Example: See the Content service for a reference implementation.

package pattern

import (
	"context"
	"fmt"
	"time"

	kg "github.com/nmxmxh/master-ovasabi/amadeus/pkg/kg"
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
)

// Redis Integration ---------------------------------------------------
// Cache or retrieve metadata for an entity.
func CacheMetadata(ctx context.Context, logger *zap.Logger, cache *redis.Cache, entityType, id string, meta *commonpb.Metadata, ttl time.Duration) error {
	if meta == nil {
		return nil
	}
	key := fmt.Sprintf("service:%s:%s:metadata", entityType, id)
	err := cache.Set(ctx, key, "", meta, ttl)
	if err != nil {
		logger.Warn("CacheMetadata failed", zap.Error(err), zap.String("entityType", entityType), zap.String("id", id))
		return fmt.Errorf("CacheMetadata failed: %w", err)
	}
	return nil
}

func GetCachedMetadata(ctx context.Context, logger *zap.Logger, cache *redis.Cache, entityType, id string) (*commonpb.Metadata, error) {
	key := fmt.Sprintf("service:%s:%s:metadata", entityType, id)
	var meta commonpb.Metadata
	err := cache.Get(ctx, key, "", &meta)
	if err != nil {
		logger.Warn("GetCachedMetadata failed", zap.Error(err), zap.String("entityType", entityType), zap.String("id", id))
		return nil, fmt.Errorf("GetCachedMetadata failed: %w", err)
	}
	return &meta, nil
}

// Scheduler Integration ------------------------------------------------
// Extract scheduling info and register a job.
func RegisterSchedule(_ context.Context, _ *zap.Logger, _, _ string, meta *commonpb.Metadata) error {
	if meta == nil || meta.Scheduling == nil {
		return nil
	}
	// Example: extract start_time, end_time, cron, etc. from meta.Scheduling.Fields
	// and register with your Scheduler service.
	// This is a stub; implement actual scheduler registration as needed.
	// If error occurs in real implementation:
	// logger.Warn("RegisterSchedule failed", zap.Error(err))
	// return fmt.Errorf("RegisterSchedule failed: %w", err)
	return nil
}

// Knowledge Graph Enrichment -------------------------------------------
// Enrich the knowledge graph with metadata.
func EnrichKnowledgeGraph(_ context.Context, logger *zap.Logger, entityType, id string, meta *commonpb.Metadata) error {
	if meta == nil {
		return nil
	}
	kgInstance := kg.DefaultKnowledgeGraph()
	serviceInfo := map[string]interface{}{
		"id":       id,
		"type":     entityType,
		"metadata": meta,
	}
	err := kgInstance.AddService(entityType, id, serviceInfo)
	if err != nil {
		logger.Warn("EnrichKnowledgeGraph failed", zap.Error(err), zap.String("entityType", entityType), zap.String("id", id))
		return fmt.Errorf("EnrichKnowledgeGraph failed: %w", err)
	}
	return nil
}

// Nexus Orchestration --------------------------------------------------
// Register service pattern and metadata schema with Nexus.
func RegisterWithNexus(_ context.Context, _ *zap.Logger, _ string, _ interface{}) error {
	// Example: Register service and metadata schema with Nexus for orchestration.
	// This is a stub; implement actual Nexus registration as needed.
	// If error occurs in real implementation:
	// logger.Warn("RegisterWithNexus failed", zap.Error(err))
	// return fmt.Errorf("RegisterWithNexus failed: %w", err)
	return nil
}

// Example Usage: Content Service ---------------------------------------
//
// In your content service, after creating or updating content:
//
//   err := pattern.CacheMetadata(ctx, cache, "content", content.Id, content.Metadata, 10*time.Minute)
//   if err != nil { ... }
//   _ = pattern.RegisterSchedule(ctx, "content", content.Id, content.Metadata)
//   _ = pattern.EnrichKnowledgeGraph(ctx, "content", content.Id, content.Metadata)
//   _ = pattern.RegisterWithNexus(ctx, "content", content.Metadata)
//
// See the Content service for a full reference implementation.
