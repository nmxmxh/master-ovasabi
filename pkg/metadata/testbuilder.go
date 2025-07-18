package metadata

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/thecathasnoname"
	"google.golang.org/protobuf/types/known/structpb"
)

// TestMetadataBuilder builds metadata for tests or temporary/limited profiles.
type TestMetadataBuilder struct {
	fields map[string]interface{}
	mu     sync.Mutex
}

// TestScenario describes a scenario-driven test case.
type TestScenario struct {
	Name        string
	Description string
	InitialMeta *commonpb.Metadata
	Action      func(ctx context.Context, meta *commonpb.Metadata) error
	Before      func(ctx context.Context, meta *commonpb.Metadata)
	After       func(ctx context.Context, meta *commonpb.Metadata, err error)
	Params      map[string]interface{} // For parameterized/table-driven tests
}

// TestSuite groups scenarios and global hooks for orchestrated execution.
type TestSuite struct {
	Name        string
	Scenarios   []*TestScenario
	BeforeEach  func(ctx context.Context, scenario *TestScenario)
	AfterEach   func(ctx context.Context, scenario *TestScenario, err error)
	OnError     func(ctx context.Context, scenario *TestScenario, err error)
	Announcer   func(ctx context.Context, position, name string, meta *commonpb.Metadata, extra ...interface{})
	GhostLogger func(ctx context.Context, meta *commonpb.Metadata)
	Fuzz        func(scenario *TestScenario) []*TestScenario // For fuzzing/coverage
}

type contextKey string

// NewTestMetadataBuilder creates a new builder.
func NewTestMetadataBuilder() *TestMetadataBuilder {
	return &TestMetadataBuilder{fields: make(map[string]interface{})}
}

// Set sets a value at a dot-separated path (e.g., "user.user_id").
func (b *TestMetadataBuilder) Set(path string, value interface{}) *TestMetadataBuilder {
	b.mu.Lock()
	defer b.mu.Unlock()
	parts := strings.Split(path, ".")
	m := b.fields
	for i, part := range parts {
		if i == len(parts)-1 {
			m[part] = value
		} else {
			if _, ok := m[part]; !ok {
				m[part] = make(map[string]interface{})
			}
			if nextMap, ok := m[part].(map[string]interface{}); ok {
				m = nextMap
			} else {
				// If the value is not a map, create a new map and replace the existing value
				newMap := make(map[string]interface{})
				m[part] = newMap
				m = newMap
			}
		}
	}
	return b
}

// SetMap merges a nested map into the builder.
func (b *TestMetadataBuilder) SetMap(data map[string]interface{}) *TestMetadataBuilder {
	b.mu.Lock()
	defer b.mu.Unlock()
	mergeMaps(b.fields, data)
	return b
}

// Build returns a *commonpb.Metadata with all fields set and automatically enriches/hashes it.
func (b *TestMetadataBuilder) Build() *commonpb.Metadata {
	b.mu.Lock()
	defer b.mu.Unlock()
	meta := &commonpb.Metadata{}
	if tags, ok := b.fields["tags"]; ok {
		if arr, ok := tags.([]string); ok {
			meta.Tags = arr
		}
	}
	if features, ok := b.fields["features"]; ok {
		if arr, ok := features.([]string); ok {
			meta.Features = arr
		}
	}
	if audit, ok := b.fields["audit"]; ok {
		if m, ok := audit.(map[string]interface{}); ok {
			meta.Audit = MapToStruct(m)
		}
	}
	if custom, ok := b.fields["custom_rules"]; ok {
		if m, ok := custom.(map[string]interface{}); ok {
			meta.CustomRules = MapToStruct(m)
		}
	}
	if sched, ok := b.fields["scheduling"]; ok {
		if m, ok := sched.(map[string]interface{}); ok {
			meta.Scheduling = MapToStruct(m)
		}
	}
	// Defensive: ensure service_specific is always a map/object
	ss, ok := b.fields["service_specific"]
	if !ok || ss == nil {
		b.fields["service_specific"] = map[string]interface{}{}
	} else {
		if _, isMap := ss.(map[string]interface{}); !isMap {
			b.fields["service_specific"] = map[string]interface{}{}
		}
	}
	if ssMap, ok := b.fields["service_specific"].(map[string]interface{}); ok {
		// Defensive: ensure nexus.versioning is always present
		nexus, ok := ssMap["nexus"].(map[string]interface{})
		if !ok {
			nexus = map[string]interface{}{}
		}
		if _, vok := nexus["versioning"]; !vok {
			nexus["versioning"] = map[string]interface{}{
				"system_version":   "1.0.0",
				"service_version":  "1.0.0",
				"user_version":     "1.0.0",
				"environment":      "test",
				"user_type":        "ghost",
				"feature_flags":    []string{},
				"last_migrated_at": time.Now().Format(time.RFC3339),
			}
			ssMap["nexus"] = nexus
		}
		meta.ServiceSpecific = MapToStruct(ssMap)
	}
	return meta
}

// dummyStructpb recursively generates a dummy structpb.Struct with plausible keys/values for testing.
func dummyStructpb(fieldName string) *structpb.Struct {
	var m map[string]interface{}
	switch fieldName {
	case "scheduling":
		m = map[string]interface{}{"start_time": time.Now().Format(time.RFC3339), "end_time": time.Now().Add(24 * time.Hour).Format(time.RFC3339)}
	case "custom_rules":
		m = map[string]interface{}{"max": 10, "rule": "test_rule"}
	case "audit":
		m = map[string]interface{}{"created_by": "dummy_user", "history": []string{"created", "updated"}}
	case "knowledge_graph":
		m = map[string]interface{}{"dummy": "value"}
	case "service_specific":
		// Always include nexus namespace with actor and versioning
		m = map[string]interface{}{
			"nexus": map[string]interface{}{
				"actor": map[string]interface{}{
					"user_id": "ghost",
					"roles":   []string{"system", "admin"},
				},
				"versioning": map[string]interface{}{
					"system_version":   "1.0.0",
					"service_version":  "1.0.0",
					"user_version":     "1.0.0",
					"environment":      "test",
					"user_type":        "ghost",
					"feature_flags":    []string{},
					"last_migrated_at": time.Now().Format(time.RFC3339),
				},
			},
		}
	default:
		m = map[string]interface{}{fieldName + "_dummy": "value"}
	}
	s, err := structpb.NewStruct(m)
	if err != nil {
		// Return a default empty struct if creation fails
		return &structpb.Struct{
			Fields: make(map[string]*structpb.Value),
		}
	}
	return s
}

// AssignDummyValues uses reflection to assign dummy values for all fields in commonpb.Metadata, including nested structpb.Struct fields.
func (b *TestMetadataBuilder) AssignDummyValues() *TestMetadataBuilder {
	metaType := reflect.TypeOf(commonpb.Metadata{})
	for i := 0; i < metaType.NumField(); i++ {
		field := metaType.Field(i)
		name := field.Name
		switch name {
		case "Tags":
			b.Set("tags", []string{"test", "dummy"})
		case "Features":
			b.Set("features", []string{"feature1", "feature2"})
		case "Audit", "CustomRules", "Scheduling", "KnowledgeGraph", "ServiceSpecific":
			// Use dummyStructpb for all structpb.Struct fields
			b.SetMap(map[string]interface{}{toSnake(name): dummyStructpb(toSnake(name)).AsMap()})
		case "Taxation":
			b.SetMap(map[string]interface{}{"taxation": map[string]interface{}{
				"min_projects": 1,
				"max_projects": 10,
				"percentage":   0.1,
			}})
		case "Owner":
			b.SetMap(map[string]interface{}{"owner": map[string]interface{}{
				"id":     "owner_1",
				"wallet": "wallet_1",
			}})
		case "Referral":
			b.SetMap(map[string]interface{}{"referral": map[string]interface{}{
				"id":     "ref_1",
				"wallet": "wallet_1",
			}})
		}
	}
	return b
}

// toSnake converts CamelCase to snake_case for field names.
func toSnake(s string) string {
	// Pre-allocate with a reasonable capacity based on input length
	out := make([]rune, 0, len(s)*2)
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			out = append(out, '_')
		}
		out = append(out, r)
	}
	return strings.ToLower(string(out))
}

// RunScenario orchestrates a scenario-driven test, logging/announcing at each step.
func (s *TestScenario) RunScenario(ctx context.Context, announcer func(ctx context.Context, position, name string, meta *commonpb.Metadata, extra ...interface{}), ghostLogger func(ctx context.Context, meta *commonpb.Metadata)) error {
	// Detect guest/ghost
	isGuest := false
	userID := ""
	roles := []string{"tester"}
	guestInfo := map[string]interface{}{}
	versioning := map[string]interface{}{}
	if s.InitialMeta != nil && s.InitialMeta.ServiceSpecific != nil {
		ss := s.InitialMeta.ServiceSpecific.AsMap()
		if user, ok := ss["user"].(map[string]interface{}); ok {
			if guest, ok := user["guest"].(bool); ok && guest {
				isGuest = true
			}
			if id, ok := user["user_id"].(string); ok {
				userID = id
			}
			if ginfo, ok := user["guest_actor"].(map[string]interface{}); ok {
				guestInfo = ginfo
			}
			if v, ok := user["versioning"].(map[string]interface{}); ok {
				versioning = v
			}
		}
	}
	// Finalize metadata for emission (required for all events)
	if err := FinalizeMetadataForEmit(ctx, s.InitialMeta, isGuest, userID, roles, guestInfo, versioning); err != nil {
		if announcer != nil {
			announcer(ctx, "error", s.Name, s.InitialMeta, err)
		}
		return err
	}
	if s.Before != nil {
		s.Before(ctx, s.InitialMeta)
	}
	if ghostLogger != nil {
		ghostLogger(ctx, s.InitialMeta)
	}
	if announcer != nil {
		announcer(ctx, "start", s.Name, s.InitialMeta)
	}
	err := s.Action(ctx, s.InitialMeta)
	if announcer != nil {
		announcer(ctx, "after_action", s.Name, s.InitialMeta, err)
	}
	if ghostLogger != nil {
		ghostLogger(ctx, s.InitialMeta)
	}
	if s.After != nil {
		s.After(ctx, s.InitialMeta, err)
	}
	return err
}

// RunAll runs all scenarios in the suite with global hooks, announcer, and ghost support.
func (t *TestSuite) RunAll(ctx context.Context) {
	for _, scenario := range t.Scenarios {
		// Ensure cat announcer is present in context for all hooks
		cat, ok := scenario.Params["cat_announcer"].(*thecathasnoname.TheCatHasNoName)
		if !ok {
			continue
		}
		ctxWithCat := ctx
		if cat != nil {
			ctxWithCat = context.WithValue(ctx, contextKey("cat_announcer"), cat)
		}
		if err := t.runOne(ctxWithCat, scenario); err != nil {
			fmt.Printf("runOne failed for scenario %q: %v\n", scenario.Name, err)
		}
	}
}

func (t *TestSuite) runOne(ctx context.Context, scenario *TestScenario) error {
	cat, ok := scenario.Params["cat_announcer"].(*thecathasnoname.TheCatHasNoName)
	if !ok {
		return fmt.Errorf("invalid cat_announcer type")
	}
	ctxWithCat := ctx
	if cat != nil {
		ctxWithCat = context.WithValue(ctx, contextKey("cat_announcer"), cat)
	}
	// Use the cat for all hooks
	beforeEach := func(ctx context.Context, s *TestScenario) {
		if cat != nil {
			cat.AnnounceSystemEvent(ctx, "before_each", t.Name, s.Name, s.InitialMeta)
		}
	}
	afterEach := func(ctx context.Context, s *TestScenario, err error) {
		if cat != nil {
			summary := map[string]interface{}{
				"Scenario":    s.Name,
				"Description": s.Description,
				"Iterations":  s.Params["N"],
				"Error":       err,
			}
			if s.InitialMeta != nil {
				summary["Tags"] = s.InitialMeta.Tags
				summary["Features"] = s.InitialMeta.Features
			}
			cat.AnnounceSystemEvent(ctx, "after_each", "TestRunner", s.Name, s.InitialMeta, summary)
		}
	}
	onError := func(ctx context.Context, s *TestScenario, err error) {
		if cat != nil {
			cat.AnnounceSystemEvent(ctx, "error", t.Name, s.Name, s.InitialMeta, err)
		}
	}
	announcer := func(ctx context.Context, position, name string, meta *commonpb.Metadata, extra ...interface{}) {
		if cat != nil {
			cat.AnnounceSystemEvent(ctx, position, t.Name, name, meta, extra...)
		}
	}
	if t.BeforeEach != nil {
		beforeEach(ctxWithCat, scenario)
	}
	err := scenario.RunScenario(ctxWithCat, announcer, t.GhostLogger)
	if err != nil {
		onError(ctxWithCat, scenario, err)
	}
	if t.AfterEach != nil {
		afterEach(ctxWithCat, scenario, err)
	}
	return nil
}

// mergeMaps recursively merges src into dst.
func mergeMaps(dst, src map[string]interface{}) {
	for k, v := range src {
		if vmap, ok := v.(map[string]interface{}); ok {
			if dstmap, ok := dst[k].(map[string]interface{}); ok {
				mergeMaps(dstmap, vmap)
			} else {
				dst[k] = vmap
			}
		} else {
			dst[k] = v
		}
	}
}
