package nexus

import (
	"context"
	"testing"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
)

// BenchmarkEmitEchoEvents benchmarks emitting 1,000,000 echo events through the Nexus service.
func BenchmarkEmitEchoEvents(b *testing.B) {
	ctx := context.Background()

	// Minimal mock Nexus service (no DB, no cache, no event repo)
	svc := &Service{
		repo:         nil,
		eventRepo:    nil,
		cache:        nil,
		log:          nil,
		eventBus:     nil,
		eventEnabled: true,
		provider:     nil,
		subscribers:  make(map[string][]chan *nexusv1.EventResponse),
	}

	eventType := "nexus.echo"
	entityID := "echo"
	payload := &commonpb.Payload{}
	meta := &commonpb.Metadata{}

	b.ResetTimer()
	start := time.Now()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := &nexusv1.EventRequest{
				EventType: eventType,
				EntityId:  entityID,
				Payload:   payload,
				Metadata:  meta,
			}
			resp, err := svc.EmitEvent(ctx, req)
			if err != nil {
				b.Errorf("Failed to emit event: %v", err)
				return
			}
			if !resp.Success {
				b.Errorf("Event emission failed: %s", resp.Message)
				return
			}
		}
	})
	dur := time.Since(start)
	events := b.N
	throughput := float64(events) / dur.Seconds()
	b.Logf("Pushed %d echo events in %v (%.2f events/sec)", events, dur, throughput)
}
