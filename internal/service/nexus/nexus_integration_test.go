//go:build integration
// +build integration

package nexus

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/thecathasnoname"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	_ "github.com/lib/pq"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	internalnexus "github.com/nmxmxh/master-ovasabi/internal/nexus"
	bridge "github.com/nmxmxh/master-ovasabi/internal/nexus/service/bridge"
	"github.com/nmxmxh/master-ovasabi/pkg/auth"
	"github.com/nmxmxh/master-ovasabi/pkg/contextx"
	"github.com/nmxmxh/master-ovasabi/pkg/tester"
	"github.com/stretchr/testify/require"
)

// TestSuite holds common test dependencies and setup.
type TestSuite struct {
	tester    *tester.Tester
	service   *Service
	cat       *thecathasnoname.TheCatHasNoName
	ctx       context.Context
	zapLogger *zap.Logger
}

// setupTestSuite initializes the test environment.
func setupTestSuite(tb testing.TB) *TestSuite {
	tb.Helper()
	ctx := context.Background()
	zapLogger := zap.NewNop()
	stdLogger := log.New(os.Stdout, "[thecathasnoname] ", log.LstdFlags)
	cat := thecathasnoname.New(stdLogger)

	testerObj := tester.NewTester("NexusIntegration", nil, cat, nil, nil)
	if err := testerObj.SetupAll(ctx, zapLogger, nil); err != nil {
		tb.Fatalf("failed to setup containers: %v", err)
	}

	repo := NewRepository(testerObj.DB, nil)
	eventRepo := internalnexus.NewSQLEventRepository(testerObj.DB, zapLogger)
	eventBus := bridge.NewEventBusWithRedis(zapLogger, testerObj.Redis)
	cache := testerObj.Redis
	logger := zapLogger

	// Create service with all dependencies
	svc := NewService(repo, eventRepo, cache, logger, eventBus, true, nil)
	service, ok := svc.(*Service)
	if !ok {
		tb.Fatal("failed to type assert service")
	}

	// Test service
	ts := &TestSuite{
		tester:    testerObj,
		service:   service,
		cat:       cat,
		ctx:       ctx,
		zapLogger: zapLogger,
	}

	return ts
}

// cleanupTestSuite performs cleanup after tests.
func (ts *TestSuite) cleanupTestSuite() {
	ts.tester.Cleanup(ts.ctx)
}

// WithGhostAuth is a helper to inject ghost auth context.
func WithGhostAuth(ctx context.Context) context.Context {
	authCtx := &auth.Context{
		UserID: "ghost",
		Roles:  []string{"system", "admin"},
	}
	return contextx.WithAuth(ctx, authCtx)
}

// generateSecureID generates a secure random ID using crypto/rand.
func generateSecureID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("entity_%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("entity_%x", b)
}

// generateSecurePatternID generates a secure random pattern ID using crypto/rand.
func generateSecurePatternID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("pattern_%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("pattern_%x", b)
}

// TestEventEmission tests the event emission functionality.
func TestEventEmission(t *testing.T) {
	ts := setupTestSuite(t)
	defer ts.cleanupTestSuite()

	scenarios := []*metadata.TestScenario{
		ts.tester.NewScenario("EmitEchoEvent", "Emit a single echo event and verify persistence", 1, func(ctx context.Context, meta *commonpb.Metadata, _ int) error {
			entityID := generateSecureID()
			req := &nexusv1.EventRequest{
				EventType: "nexus.echo",
				EntityId:  entityID,
				Metadata:  meta,
			}

			ctxWithAuth := WithGhostAuth(ctx)
			_, err := ts.service.EmitEvent(ctxWithAuth, req)
			if err != nil {
				return fmt.Errorf("failed to emit event: %w", err)
			}

			// Verify event was persisted by checking pattern events
			events, err := ts.service.eventRepo.ListEventsByPattern(ctx, entityID)
			if err != nil {
				return fmt.Errorf("failed to retrieve events: %w", err)
			}
			if len(events) == 0 {
				return fmt.Errorf("no events found for entity %s", entityID)
			}

			return nil
		}),
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			if err := scenario.RunScenario(ts.ctx, nil, nil); err != nil {
				t.Errorf("Scenario failed: %v", err)
			}
		})
	}

	t.Run("TestEventProcessing", func(t *testing.T) {
		entityID := generateSecureID()
		req := &nexusv1.EventRequest{
			EventType: "nexus.echo",
			EntityId:  entityID,
			Metadata: &commonpb.Metadata{
				Audit: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"message": structpb.NewStringValue("Test echo message"),
					},
				},
			},
		}

		_, err := ts.service.EmitEvent(ts.ctx, req)
		if err != nil {
			t.Errorf("Failed to emit event: %v", err)
		}
	})

	t.Run("TestPatternRegistration", func(t *testing.T) {
		patternID := generateSecurePatternID()
		req := &nexusv1.RegisterPatternRequest{
			PatternId:   patternID,
			PatternType: "test_pattern",
			Version:     "1.0.0",
			Metadata: &commonpb.Metadata{
				Audit: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"description": structpb.NewStringValue("Test pattern"),
					},
				},
			},
		}

		_, err := ts.service.RegisterPattern(ts.ctx, req)
		if err != nil {
			t.Errorf("Failed to register pattern: %v", err)
		}
	})

	t.Run("TestEventBenchmark", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			eventID := generateSecureID()
			req := &nexusv1.EventRequest{
				EventType: "nexus.echo",
				EntityId:  eventID,
				Metadata: &commonpb.Metadata{
					Audit: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"message": structpb.NewStringValue("Benchmark message"),
						},
					},
				},
			}

			_, err := ts.service.EmitEvent(ts.ctx, req)
			if err != nil {
				t.Errorf("Failed to emit benchmark event: %v", err)
			}
		}
	})
}

// TestPatternRegistration tests pattern registration functionality.
func TestPatternRegistration(t *testing.T) {
	ts := setupTestSuite(t)
	defer ts.cleanupTestSuite()

	scenarios := []*metadata.TestScenario{
		ts.tester.NewScenario("RegisterPattern", "Register a new pattern", 1, func(ctx context.Context, meta *commonpb.Metadata, _ int) error {
			patternID := generateSecurePatternID()
			req := &nexusv1.RegisterPatternRequest{
				PatternId:   patternID,
				PatternType: "test_pattern",
				Version:     "1.0.0",
				Origin:      "test",
				CampaignId:  1,
				Metadata:    meta,
			}

			ctxWithAuth := WithGhostAuth(ctx)
			resp, err := ts.service.RegisterPattern(ctxWithAuth, req)
			if err != nil {
				return fmt.Errorf("failed to register pattern: %w", err)
			}
			if !resp.Success {
				return fmt.Errorf("pattern registration failed: %s", resp.Error)
			}

			// Verify pattern was registered
			patterns, err := ts.service.ListPatterns(ctx, &nexusv1.ListPatternsRequest{
				PatternType: "test_pattern",
				CampaignId:  1,
			})
			if err != nil {
				return fmt.Errorf("failed to list patterns: %w", err)
			}
			if len(patterns.Patterns) == 0 {
				return fmt.Errorf("no patterns found")
			}

			return nil
		}),
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			if err := scenario.RunScenario(ts.ctx, nil, nil); err != nil {
				t.Errorf("Scenario failed: %v", err)
			}
		})
	}
}

// TestOrchestration tests orchestration functionality.
func TestOrchestration(t *testing.T) {
	ts := setupTestSuite(t)
	defer ts.cleanupTestSuite()

	scenarios := []*metadata.TestScenario{
		ts.tester.NewScenario("OrchestratePattern", "Orchestrate a pattern", 1, func(ctx context.Context, meta *commonpb.Metadata, _ int) error {
			req := &nexusv1.OrchestrateRequest{
				PatternId:  generateSecurePatternID(),
				Input:      &structpb.Struct{Fields: map[string]*structpb.Value{}},
				Metadata:   meta,
				CampaignId: 1,
			}

			ctxWithAuth := WithGhostAuth(ctx)
			resp, err := ts.service.Orchestrate(ctxWithAuth, req)
			if err != nil {
				return fmt.Errorf("failed to orchestrate pattern: %w", err)
			}
			if resp.OrchestrationId == "" {
				return fmt.Errorf("orchestration ID is empty")
			}

			// Verify orchestration trace
			trace, err := ts.service.TracePattern(ctx, &nexusv1.TracePatternRequest{
				OrchestrationId: resp.OrchestrationId,
			})
			if err != nil {
				return fmt.Errorf("failed to trace pattern: %w", err)
			}
			if len(trace.Steps) == 0 {
				return fmt.Errorf("no trace steps found")
			}

			return nil
		}),
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			if err := scenario.RunScenario(ts.ctx, nil, nil); err != nil {
				t.Errorf("Scenario failed: %v", err)
			}
		})
	}
}

// TestFeedback tests feedback functionality.
func TestFeedback(t *testing.T) {
	ts := setupTestSuite(t)
	defer ts.cleanupTestSuite()

	scenarios := []*metadata.TestScenario{
		ts.tester.NewScenario("SubmitFeedback", "Submit feedback for a pattern", 1, func(ctx context.Context, meta *commonpb.Metadata, _ int) error {
			req := &nexusv1.FeedbackRequest{
				PatternId:  generateSecurePatternID(),
				Score:      5.0,
				Comments:   "Test feedback",
				Metadata:   meta,
				CampaignId: 1,
			}

			ctxWithAuth := WithGhostAuth(ctx)
			resp, err := ts.service.Feedback(ctxWithAuth, req)
			if err != nil {
				return fmt.Errorf("failed to submit feedback: %w", err)
			}
			if !resp.Success {
				return fmt.Errorf("feedback submission failed: %s", resp.Error)
			}

			return nil
		}),
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			if err := scenario.RunScenario(ts.ctx, nil, nil); err != nil {
				t.Errorf("Scenario failed: %v", err)
			}
		})
	}
}

// TestOperations tests operations handling.
func TestOperations(t *testing.T) {
	ts := setupTestSuite(t)
	defer ts.cleanupTestSuite()

	scenarios := []*metadata.TestScenario{
		ts.tester.NewScenario("HandleOps", "Handle operations", 1, func(ctx context.Context, meta *commonpb.Metadata, _ int) error {
			req := &nexusv1.HandleOpsRequest{
				Op: "register_pattern",
				Params: map[string]string{
					"pattern_id":   generateSecurePatternID(),
					"pattern_type": "test_pattern",
					"version":      "1.0.0",
					"origin":       "test",
				},
				Metadata:   meta,
				CampaignId: 1,
			}

			ctxWithAuth := WithGhostAuth(ctx)
			resp, err := ts.service.HandleOps(ctxWithAuth, req)
			if err != nil {
				return fmt.Errorf("failed to handle operation: %w", err)
			}
			if !resp.Success {
				return fmt.Errorf("operation failed: %s", resp.Message)
			}

			return nil
		}),
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			if err := scenario.RunScenario(ts.ctx, nil, nil); err != nil {
				t.Errorf("Scenario failed: %v", err)
			}
		})
	}
}

// BenchmarkNexusScenarios benchmarks various nexus operations.
func BenchmarkNexusScenarios(b *testing.B) {
	ts := setupTestSuite(b)
	defer ts.cleanupTestSuite()

	scenarios := []*metadata.TestScenario{
		ts.tester.NewScenario("EmitEchoEvents_Batch1000", "Emit 1000 echo events", 1000, func(ctx context.Context, meta *commonpb.Metadata, i int) error {
			eventID := fmt.Sprintf("echo_bench_%d_%s", i, generateSecureID())
			req := &nexusv1.EventRequest{
				EventType: "nexus.echo",
				EntityId:  eventID,
				Metadata:  meta,
				Payload: &commonpb.Payload{
					Data: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"message": structpb.NewStringValue("Benchmark message"),
						},
					},
				},
			}
			_, err := ts.service.EmitEvent(ctx, req)
			return err
		}),
	}

	for _, scenario := range scenarios {
		b.Run(scenario.Name, func(b *testing.B) {
			if err := scenario.RunScenario(ts.ctx, nil, nil); err != nil {
				b.Errorf("Scenario failed: %v", err)
			}
		})
	}
}

// TestEmitEvent tests event emission.
func (ts *TestSuite) TestEmitEvent(t *testing.T) {
	// Create test event
	req := &nexusv1.EventRequest{
		EventType: "test.event",
		Payload: &commonpb.Payload{
			Data: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"test": structpb.NewStringValue("value"),
				},
			},
		},
		Metadata: &commonpb.Metadata{},
	}

	// Emit event
	resp, err := ts.service.EmitEvent(ts.ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotEmpty(t, resp.EventId)

	// Verify event was stored
	events, err := ts.service.eventRepo.ListEventsByPattern(ts.ctx, "test.event")
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, "test.event", events[0].EventType)
}
