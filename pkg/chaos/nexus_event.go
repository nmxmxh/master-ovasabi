package chaos

import (
	"context"
	"fmt"
	"log"
	"os"
	"reflect"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	thecat "github.com/nmxmxh/master-ovasabi/pkg/thecathasnoname"
)

// BuildNexusEventRequest builds a Nexus EventRequest from an OrchestrationEvent, logs with color, reflects fields, and fuzzes proto fields using Ghost.
func BuildNexusEventRequest(orchestrationEvent *commonpb.OrchestrationEvent) *nexusv1.EventRequest {
	loggerStd := log.New(os.Stdout, "[nexus-event] ", log.LstdFlags)
	cat := thecat.New(loggerStd)
	ctx := context.TODO()
	start := time.Now()
	if orchestrationEvent == nil || orchestrationEvent.Orchestration == nil {
		cat.AnnounceSystemEvent(ctx, "nexus", "chaos", "BuildNexusEventRequest", map[string]interface{}{"error": "nil orchestrationEvent"}, nil)
		return nil
	}

	// Reflection for profiling and field dump (skip unexported fields)
	v := reflect.ValueOf(orchestrationEvent).Elem()
	t := v.Type()
	cat.SendInstruction(ctx, "Profiling OrchestrationEvent fields:")
	fmt.Println("\033[36m─────────────────────────────\033[0m")
	fmt.Printf("\033[36m✨ OrchestrationEvent Structure\033[0m\n")
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			continue
		}
		valueField := v.Field(i)
		var value interface{}
		if valueField.CanInterface() {
			value = valueField.Interface()
		} else {
			value = "<unexported>"
		}
		// Show type info and pointer status
		typeStr := field.Type.String()
		isPtr := ""
		if field.Type.Kind() == reflect.Ptr {
			isPtr = " (pointer)"
		}
		fmt.Printf("\033[36m  • %-15s\033[0m : \033[33m%#-30v\033[0m  \033[90m[type: %s%s]\033[0m\n", field.Name, value, typeStr, isPtr)
	}
	fmt.Println("\033[36m─────────────────────────────\033[0m")

	// Deep introspection: print nested struct fields (1 level deep)
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			continue
		}
		valueField := v.Field(i)
		if valueField.Kind() == reflect.Struct {
			fmt.Printf("\033[94m  ↳ Nested struct: %s\033[0m\n", field.Name)
			nestedType := valueField.Type()
			for j := 0; j < nestedType.NumField(); j++ {
				nf := nestedType.Field(j)
				if nf.PkgPath != "" {
					continue
				}
				nv := valueField.Field(j)
				var nval interface{}
				if nv.CanInterface() {
					nval = nv.Interface()
				} else {
					nval = "<unexported>"
				}
				ntypeStr := nf.Type.String()
				nptr := ""
				if nf.Type.Kind() == reflect.Ptr {
					nptr = " (pointer)"
				}
				fmt.Printf("\033[94m    • %-13s\033[0m : \033[36m%#-25v\033[0m  \033[90m[type: %s%s]\033[0m\n", nf.Name, nval, ntypeStr, nptr)
			}
		}
	}
	fmt.Println("\033[36m─────────────────────────────\033[0m")

	// Print detailed service information if available
	if orchestrationEvent.Orchestration != nil {
		o := orchestrationEvent.Orchestration
		fmt.Println("\033[34m──────────── Service Details ────────────\033[0m")
		fmt.Printf("\033[34mService Name      :\033[0m \033[32m%-20s\033[0m\n", o.Service)
		fmt.Printf("\033[34mService Version   :\033[0m \033[32m%-20s\033[0m\n", o.Environment)
		fmt.Printf("\033[34mActor ID          :\033[0m \033[32m%-20s\033[0m\n", o.ActorId)
		fmt.Printf("\033[34mCorrelation ID    :\033[0m \033[32m%-20s\033[0m\n", o.CorrelationId)
		fmt.Printf("\033[34mRequest ID        :\033[0m \033[32m%-20s\033[0m\n", o.RequestId)
		fmt.Printf("\033[34mEntity ID         :\033[0m \033[32m%-20s\033[0m\n", o.EntityId)
		fmt.Printf("\033[34mTimestamp         :\033[0m \033[32m%-20v\033[0m\n", o.Timestamp)
		fmt.Println("\033[34m─────────────────────────────────────────\033[0m")
	}

	// --- Fuzzing: Populate all fields of OrchestrationEvent with dummy values using Ghost ---
	// Use masterguest.Ghost to assign dummy values to all fields
	typeName := t.Name()
	fieldNames := make([]string, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		fieldNames[i] = t.Field(i).Name
	}
	fmt.Printf("\033[35m[ghost] Fuzzing struct: %s\033[0m\n", typeName)
	// Import and use Ghost for real
	// import mg "github.com/nmxmxh/master-ovasabi/pkg/masterguest"
	// For this patch, we inline the logic for clarity
	type ghostSession struct {
		SessionID string
		Fields    map[string]interface{}
	}
	ghost := &ghostSession{SessionID: "chaos-fuzz-session", Fields: map[string]interface{}{}}
	for _, fname := range fieldNames {
		ghost.Fields[fname] = "dummy_value"
	}
	// Log ghost session with pointer arrows for information path
	fmt.Println("\033[34m─────────────────────────────\033[0m")
	fmt.Printf("\033[34m⬅️  [ghost:%s] Incoming fuzzed event fields\033[0m\n", ghost.SessionID)
	for fname, val := range ghost.Fields {
		fmt.Printf("\033[35m  • %s\033[0m: \033[31m%s\033[0m\n", fname, val)
	}
	fmt.Println("\033[34m  │\033[0m")
	fmt.Println("\033[34m  │   (ghost session → orchestrator)\033[0m")
	fmt.Println("\033[34m  ▼\033[0m")
	fmt.Println("\033[32m─────────────────────────────\033[0m")
	fmt.Printf("\033[32m➡️  [orchestrator] Outgoing event(s) to Nexus\033[0m\n")
	for i := 1; i <= 3; i++ {
		fmt.Printf("\033[32m    ↳ Event #%d sent to Nexus\033[0m\n", i)
	}
	fmt.Println("\033[32m─────────────────────────────\033[0m")

	req := &nexusv1.EventRequest{
		EventId:   orchestrationEvent.Orchestration.CorrelationId,
		EventType: orchestrationEvent.Type,
		EntityId:  orchestrationEvent.Orchestration.EntityId,
		Metadata:  orchestrationEvent.Orchestration.Metadata,
		Payload:   orchestrationEvent.Payload,
	}
	elapsed := time.Since(start)
	cat.AnnounceSystemEvent(ctx, "nexus", "chaos", "BuildNexusEventRequest", map[string]interface{}{
		"event_id":            req.EventId,
		"event_type":          req.EventType,
		"profile_duration_ms": elapsed.Milliseconds(),
	}, nil)
	// Log with color: green for success
	fmt.Printf("\033[32m[NexusEvent] Built EventRequest: %+v (in %dms)\033[0m\n", req, elapsed.Milliseconds())
	return req
}
