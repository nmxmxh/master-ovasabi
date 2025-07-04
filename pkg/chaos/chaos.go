package chaos

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	thecat "github.com/nmxmxh/master-ovasabi/pkg/thecathasnoname"
)

// RunChaosDemoWithProtos runs a demo and logs OrchestrationEvent and IntegrationPattern protos.
// RunChaosDemoWithProtos runs a demo, logs protos, and returns the OrchestrationEvent for gRPC emission.
func (c *ChaosOrchestrator) RunChaosDemoWithProtos(ctx context.Context, zaplog *zap.Logger) *commonpb.OrchestrationEvent {
	rand.Seed(time.Now().UnixNano())
	c.Cat.SendInstruction(ctx, "Starting Chaos Orchestration Demo (with protos)...")
	// Emit a random orchestration event
	if len(c.Services) == 0 {
		c.Cat.SendInstruction(ctx, "No services registered for chaos event.")
		return nil
	}
	service := c.Services[rand.Intn(len(c.Services))]
	method := ""
	if len(service.Methods) > 0 {
		method = service.Methods[rand.Intn(len(service.Methods))]
	}
	eventID := fmt.Sprintf("chaos_event_%d", rand.Intn(10000))
	now := time.Now()
	metaStruct, _ := structpb.NewStruct(map[string]interface{}{
		"event":           eventID,
		"method":          method,
		"service_version": service.Version,
	})
	orchestration := &commonpb.OrchestrationPayload{
		Code:          "CHAOS",
		Message:       "Chaos event emitted",
		Metadata:      &commonpb.Metadata{ServiceSpecific: metaStruct},
		YinYang:       "yin",
		CorrelationId: eventID,
		Service:       service.Name,
		EntityId:      eventID,
		Timestamp:     timestamppb.New(now),
		Environment:   "dev",
		ActorId:       "chaos-orchestrator",
		RequestId:     eventID,
		Tags:          []string{"chaos", "demo"},
	}
	orchestrationEvent := &commonpb.OrchestrationEvent{
		Type:          "orchestration.chaos",
		Orchestration: orchestration,
		Version:       "v1",
		Payload:       &commonpb.Payload{Data: metaStruct},
	}
	// Pretty log for OrchestrationEvent
	fmt.Println("\033[36m─────────────────────────────\033[0m")
	fmt.Printf("\033[36m[CHAOS] Emitting OrchestrationEvent\033[0m\n")
	fmt.Printf("\033[36m  • Type            :\033[0m \033[33m%-30s\033[0m\n", orchestrationEvent.Type)
	if orchestrationEvent.Orchestration != nil {
		o := orchestrationEvent.Orchestration
		fmt.Printf("\033[36m  • Service         :\033[0m \033[32m%-20s\033[0m\n", o.Service)
		fmt.Printf("\033[36m  • Method          :\033[0m \033[32m%-20s\033[0m\n", o.Message)
		fmt.Printf("\033[36m  • Correlation ID  :\033[0m \033[32m%-20s\033[0m\n", o.CorrelationId)
		fmt.Printf("\033[36m  • Actor ID        :\033[0m \033[32m%-20s\033[0m\n", o.ActorId)
		fmt.Printf("\033[36m  • Timestamp       :\033[0m \033[32m%-20v\033[0m\n", o.Timestamp)
	}
	fmt.Printf("\033[36m  • Version         :\033[0m \033[33m%-30s\033[0m\n", orchestrationEvent.Version)
	fmt.Println("\033[36m─────────────────────────────\033[0m")
	zaplog.Info("[CHAOS] Emitting OrchestrationEvent", zap.Any("event", orchestrationEvent))
	c.Logger.Printf("[CHAOS] Emitting OrchestrationEvent: %v", orchestrationEvent)

	// Emit a simple IntegrationPattern
	stepStruct, _ := structpb.NewStruct(map[string]interface{}{
		"service": service.Name,
		"method":  method,
	})
	pattern := &commonpb.IntegrationPattern{
		Id:          eventID,
		Version:     "v1",
		Description: "Chaos integration pattern demo",
		Steps: []*commonpb.PatternStep{{
			Type:       "chaos",
			Action:     "emit",
			Parameters: stepStruct,
		}},
		Metadata: &commonpb.Metadata{ServiceSpecific: metaStruct},
		Payload:  &commonpb.Payload{Data: metaStruct},
	}
	// Pretty log for IntegrationPattern
	fmt.Println("\033[35m─────────────────────────────\033[0m")
	fmt.Printf("\033[35m[CHAOS] Emitting IntegrationPattern\033[0m\n")
	fmt.Printf("\033[35m  • Pattern ID      :\033[0m \033[33m%-30s\033[0m\n", pattern.Id)
	fmt.Printf("\033[35m  • Service         :\033[0m \033[32m%-20s\033[0m\n", service.Name)
	fmt.Printf("\033[35m  • Method          :\033[0m \033[32m%-20s\033[0m\n", method)
	fmt.Printf("\033[35m  • Description     :\033[0m \033[32m%-40s\033[0m\n", pattern.Description)
	fmt.Printf("\033[35m  • Version         :\033[0m \033[33m%-30s\033[0m\n", pattern.Version)
	fmt.Println("\033[35m─────────────────────────────\033[0m")
	zaplog.Info("[CHAOS] Emitting IntegrationPattern", zap.Any("pattern", pattern))
	c.Logger.Printf("[CHAOS] Emitting IntegrationPattern: %v", pattern)

	c.Cat.SendInstruction(ctx, "Chaos Orchestration Demo (with protos) complete.")

	// Timing feedback
	elapsed := time.Since(now)
	fmt.Printf("\033[32m[CHAOS] Orchestration event and pattern emitted in %d ms\033[0m\n", elapsed.Milliseconds())
	if elapsed < 100 {
		fmt.Printf("\033[32m[CHAOS] Speed: 🚀 Fast (good for chaos/fuzzing)\033[0m\n")
	} else if elapsed < 500 {
		fmt.Printf("\033[33m[CHAOS] Speed: ⚡ Moderate\033[0m\n")
	} else {
		fmt.Printf("\033[31m[CHAOS] Speed: 🐢 Slow (consider optimizing)\033[0m\n")
	}
	return orchestrationEvent
}

// ServiceRegistration represents a registered service.
// ServiceRegistration represents a registered service and its methods.
type ServiceRegistration struct {
	Name    string
	Version string
	Methods []string
}

// ChaosOrchestrator coordinates chaos events between Nexus and services.
type ChaosOrchestrator struct {
	Logger      *log.Logger
	Cat         *thecat.TheCatHasNoName
	Services    []ServiceRegistration
	Concurrency int         // Number of concurrent events
	Provider    interface{} // Optional: for emitting events to Nexus (interface{} to avoid import cycle)
}

// NewChaosOrchestrator creates a new orchestrator. Provider is optional (for event emission).
func NewChaosOrchestrator(logger *log.Logger, cat *thecat.TheCatHasNoName, services []ServiceRegistration, concurrency int, provider interface{}) *ChaosOrchestrator {
	return &ChaosOrchestrator{
		Logger:      logger,
		Cat:         cat,
		Services:    services,
		Concurrency: concurrency,
		Provider:    provider,
	}
}

// LoadServiceRegistrationsFromJSON parses the service_registration.json and returns a list of ServiceRegistration.
func LoadServiceRegistrationsFromJSON(jsonData []byte) ([]ServiceRegistration, error) {
	type rawService struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Schema  struct {
			Methods []string `json:"methods"`
		} `json:"schema"`
	}
	var raw []rawService
	if err := json.Unmarshal(jsonData, &raw); err != nil {
		return nil, err
	}
	var regs []ServiceRegistration
	for _, s := range raw {
		regs = append(regs, ServiceRegistration{
			Name:    s.Name,
			Version: s.Version,
			Methods: s.Schema.Methods,
		})
	}
	return regs, nil
}

// EmitRandomEvent emits a random event to a random service.
func (c *ChaosOrchestrator) EmitRandomEvent(ctx context.Context) {
	if len(c.Services) == 0 {
		c.Cat.SendInstruction(ctx, "No services registered for chaos event.")
		return
	}
	service := c.Services[rand.Intn(len(c.Services))]
	method := ""
	if len(service.Methods) > 0 {
		method = service.Methods[rand.Intn(len(service.Methods))]
	}
	event := fmt.Sprintf("chaos_event_%d", rand.Intn(10000))
	meta := map[string]interface{}{"event": event, "method": method, "service_version": service.Version}
	c.Cat.AnnounceSystemEvent(ctx, "nexus", service.Name, "EmitRandomEvent", meta, nil)
	c.Logger.Printf("Nexus emitted event '%s' to service '%s' (method: '%s')", event, service.Name, method)
	// Simulate service rejection
	c.HandleServiceRejection(ctx, service, event)
}

// HandleServiceRejection simulates the service rejecting the event and sending it back to Nexus.
func (c *ChaosOrchestrator) HandleServiceRejection(ctx context.Context, service ServiceRegistration, event string) {
	c.Cat.LogDemoEvent(ctx, fmt.Sprintf("Service '%s' rejected event '%s'", service.Name, event))
	c.Logger.Printf("Service '%s' rejected event '%s', sending back to Nexus", service.Name, event)
	// Retry with another random service (excluding the rejecting one)
	var otherServices []ServiceRegistration
	for _, s := range c.Services {
		if s.Name != service.Name {
			otherServices = append(otherServices, s)
		}
	}
	if len(otherServices) == 0 {
		c.Cat.SendInstruction(ctx, "All services rejected the event. Chaos complete.")
		return
	}
	// Pick another random service
	newService := otherServices[rand.Intn(len(otherServices))]
	method := ""
	if len(newService.Methods) > 0 {
		method = newService.Methods[rand.Intn(len(newService.Methods))]
	}
	meta := map[string]interface{}{"event": event, "method": method, "service_version": newService.Version}
	c.Cat.AnnounceSystemEvent(ctx, "nexus", newService.Name, "RetryEvent", meta, nil)
	c.Logger.Printf("Nexus retried event '%s' with service '%s' (method: '%s')", event, newService.Name, method)
	// Simulate rejection again (for demo, could add acceptance logic)
	c.HandleServiceRejection(ctx, newService, event)
}

// EmitToMultipleServices emits an event to multiple services concurrently.
func (c *ChaosOrchestrator) EmitToMultipleServices(ctx context.Context) {
	if len(c.Services) == 0 {
		c.Cat.SendInstruction(ctx, "No services registered for chaos event.")
		return
	}
	event := fmt.Sprintf("chaos_event_%d", rand.Intn(10000))
	var wg sync.WaitGroup
	for i, service := range c.Services {
		if c.Concurrency > 0 && i >= c.Concurrency {
			break
		}
		wg.Add(1)
		go func(s ServiceRegistration) {
			defer wg.Done()
			method := ""
			if len(s.Methods) > 0 {
				method = s.Methods[rand.Intn(len(s.Methods))]
			}
			meta := map[string]interface{}{"event": event, "method": method, "service_version": s.Version}
			c.Cat.AnnounceSystemEvent(ctx, "nexus", s.Name, "EmitConcurrentEvent", meta, nil)
			c.Logger.Printf("Nexus emitted event '%s' to service '%s' (concurrent, method: '%s')", event, s.Name, method)
			// Simulate rejection
			c.HandleServiceRejection(ctx, s, event)
		}(service)
	}
	wg.Wait()
}

// RunChaosDemo runs a demo of the chaos orchestrator.
func (c *ChaosOrchestrator) RunChaosDemo(ctx context.Context) {
	rand.Seed(time.Now().UnixNano())
	c.Cat.SendInstruction(ctx, "Starting Chaos Orchestration Demo...")
	c.EmitRandomEvent(ctx)
	c.Cat.SendInstruction(ctx, "Now emitting to multiple services concurrently...")
	c.EmitToMultipleServices(ctx)
	c.Cat.SendInstruction(ctx, "Chaos Orchestration Demo complete.")
}
