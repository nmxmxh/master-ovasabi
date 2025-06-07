package tester

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"github.com/nmxmxh/master-ovasabi/pkg/thecathasnoname"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"
)

// Speaker defines the interface for announcement speakers.
type Speaker interface {
	Speak(ctx context.Context, msg string, meta *commonpb.Metadata) error
}

// Tester is a test helper that provides common test functionality.
type Tester struct {
	Name              string
	Nexus             NexusSpeaker // interface for Nexus communication (emit events, etc.)
	Announcer         *thecathasnoname.TheCatHasNoName
	DB                *sql.DB
	Redis             *redis.Cache
	postgresContainer testcontainers.Container
	redisContainer    testcontainers.Container
	PostgresConnStr   string
	RedisConnStr      string
	Log               *zap.Logger
}

// NexusSpeaker is an interface for communicating with Nexus (emit events, call service methods, etc.)
type NexusSpeaker interface {
	Speak(ctx context.Context, msg string, meta *commonpb.Metadata) error
}

// OrchestratingSpeaker is a speaker that orchestrates announcements with timing and status.
type OrchestratingSpeaker struct {
	Announcer Speaker
	Emitter   Speaker
	Log       *zap.Logger
}

// Speak announces before/after, logs duration, and forwards to the real emitter.
func (o *OrchestratingSpeaker) Speak(ctx context.Context, msg string, meta *commonpb.Metadata) error {
	if o.Announcer != nil {
		if err := o.Announcer.Speak(ctx, "before_emit: "+msg, meta); err != nil {
			o.Log.Error("Failed to announce before emit",
				zap.Error(err),
				zap.String("message", msg))
		}
	}
	start := time.Now()
	err := o.Emitter.Speak(ctx, msg, meta)
	dur := time.Since(start)
	if o.Announcer != nil {
		status := "success"
		if err != nil {
			status = "error"
		}
		if err := o.Announcer.Speak(ctx, fmt.Sprintf("after_emit: %s | duration: %v | status: %s", msg, dur, status), meta); err != nil {
			o.Log.Error("Failed to announce after emit",
				zap.Error(err),
				zap.String("message", msg),
				zap.Duration("duration", dur),
				zap.String("status", status))
		}
	}
	return err
}

// NewTester creates a new Tester with the given name, Nexus speaker, announcer, and optional DB/Redis.
func NewTester(name string, nexus NexusSpeaker, announcer *thecathasnoname.TheCatHasNoName, db *sql.DB, cache *redis.Cache) *Tester {
	return &Tester{
		Name:      name,
		Nexus:     nexus,
		Announcer: announcer,
		DB:        db,
		Redis:     cache,
	}
}

// SetupPostgres starts a Postgres testcontainer, assigns DB and connection string, and runs optional migration.
func (t *Tester) SetupPostgres(ctx context.Context, migration func(db *sql.DB) error) error {
	req := testcontainers.ContainerRequest{
		Image:        "postgres:14-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_DB":       "test_db",
			"POSTGRES_USER":     "test_user",
			"POSTGRES_PASSWORD": "test_password",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp"),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return fmt.Errorf("failed to start Postgres container: %w", err)
	}
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			fmt.Printf("Failed to terminate Postgres container: %v\n", err)
		}
	}()

	// Get Postgres connection details
	host, err := container.Host(ctx)
	if err != nil {
		return fmt.Errorf("failed to get Postgres host: %w", err)
	}
	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		return fmt.Errorf("failed to get Postgres port: %w", err)
	}

	// Create Postgres connection string
	connStr := fmt.Sprintf("host=%s port=%s user=test_user password=test_password dbname=test_db sslmode=disable",
		host, port.Port())

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to connect to Postgres: %w", err)
	}
	if err := waitForPostgres(db, 10*time.Second); err != nil {
		return fmt.Errorf("postgres not ready: %w", err)
	}
	if migration != nil {
		if err := migration(db); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}
	t.DB = db
	t.postgresContainer = container
	t.PostgresConnStr = connStr
	return nil
}

// waitForPostgres pings the DB until it is ready or times out.
func waitForPostgres(db *sql.DB, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for Postgres to be ready")
		}
		if err := db.Ping(); err == nil {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// SetupRedis starts a tcredis container and assigns a redis.Cache to the Tester.
// Requires a *zap.Logger for logging.
func (t *Tester) SetupRedis(ctx context.Context, log *zap.Logger) error {
	req := testcontainers.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		Env: map[string]string{
			"REDIS_PASSWORD": "test_password",
		},
		WaitingFor: wait.ForListeningPort("6379/tcp"),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return fmt.Errorf("failed to start Redis container: %w", err)
	}
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			fmt.Printf("Failed to terminate Redis container: %v\n", err)
		}
	}()

	// Get Redis connection details
	host, err := container.Host(ctx)
	if err != nil {
		return fmt.Errorf("failed to get Redis host: %w", err)
	}
	port, err := container.MappedPort(ctx, "6379")
	if err != nil {
		return fmt.Errorf("failed to get Redis port: %w", err)
	}

	// Create Redis connection string
	connStr := fmt.Sprintf("redis://:%s@%s:%s/0", "test_password", host, port.Port())

	redisCfg := redis.Config{
		Host:     host,
		Port:     port.Port(),
		Password: "",
		DB:       0,
	}
	opts := &redis.Options{
		Addr:     connStr,
		Password: redisCfg.Password,
		DB:       redisCfg.DB,
	}
	cache, err := redis.NewCache(ctx, opts, log)
	if err != nil {
		return fmt.Errorf("failed to create Redis cache: %w", err)
	}
	t.Redis = cache
	t.redisContainer = container
	return nil
}

// SetupAll orchestrates both Postgres and Redis setup.
func (t *Tester) SetupAll(ctx context.Context, log *zap.Logger, migration func(db *sql.DB) error) error {
	if err := t.SetupPostgres(ctx, migration); err != nil {
		return err
	}
	if err := t.SetupRedis(ctx, log); err != nil {
		t.Cleanup(ctx)
		return err
	}
	return nil
}

// Cleanup cleans up DB and Redis containers if needed.
func (t *Tester) Cleanup(ctx context.Context) {
	if t.redisContainer != nil {
		if err := t.redisContainer.Terminate(ctx); err != nil {
			fmt.Printf("Failed to terminate Redis container: %v\n", err)
		}
	}
	if t.postgresContainer != nil {
		if err := t.postgresContainer.Terminate(ctx); err != nil {
			fmt.Printf("Failed to terminate Postgres container: %v\n", err)
		}
	}
	if t.DB != nil {
		if err := t.DB.Close(); err != nil {
			fmt.Printf("Failed to close DB connection: %v\n", err)
		}
	}
}

// Speak sends a message to Nexus with metadata.
func (t *Tester) Speak(ctx context.Context, msg string, meta *commonpb.Metadata) error {
	return t.Nexus.Speak(ctx, msg, meta)
}

// Announce uses thecathasnoname to announce a system event.
func (t *Tester) Announce(ctx context.Context, event, scenario string, meta *commonpb.Metadata, extra ...interface{}) {
	if t.Announcer != nil {
		t.Announcer.AnnounceSystemEvent(ctx, event, t.Name, scenario, metadata.StructToMap(meta.ServiceSpecific), extra...)
	}
}

// DefaultTestMeta returns a default test metadata object.
func DefaultTestMeta() *commonpb.Metadata {
	return metadata.NewTestMetadataBuilder().AssignDummyValues().Build()
}

// GetIntParam extracts an int parameter from a scenario, with fallback.
func GetIntParam(s *metadata.TestScenario, key string, fallback int) int {
	if v, ok := s.Params[key]; ok {
		if n, ok := v.(int); ok {
			return n
		}
	}
	return fallback
}

// NewScenario creates a new test scenario.
func (t *Tester) NewScenario(name, desc string, n int, action func(ctx context.Context, meta *commonpb.Metadata, i int) error) *metadata.TestScenario {
	return &metadata.TestScenario{
		Name:        name,
		Description: desc,
		InitialMeta: DefaultTestMeta(),
		Params:      map[string]interface{}{"N": n},
		Action: func(ctx context.Context, meta *commonpb.Metadata) error {
			for i := 0; i < n; i++ {
				if err := action(ctx, meta, i); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

// NewTestSuite creates a new test suite with default hooks and the given scenarios.
func (t *Tester) NewTestSuite(name string, scenarios []*metadata.TestScenario) *metadata.TestSuite {
	return &metadata.TestSuite{
		Name:        name,
		BeforeEach:  DefaultBeforeEach,
		AfterEach:   DefaultAfterEach,
		OnError:     DefaultOnError,
		Announcer:   t.Announce,
		GhostLogger: DefaultGhostLogger,
		Scenarios:   scenarios,
	}
}

// Default announcer/logging hooks for reuse.
var (
	DefaultAnnouncer = func(ctx context.Context, position, name string, meta *commonpb.Metadata, extra ...interface{}) {
		cat, ok := ctx.Value("cat_announcer").(*thecathasnoname.TheCatHasNoName)
		if !ok || cat == nil {
			// fallback: print as JSON
			if meta != nil {
				b, err := json.MarshalIndent(meta, "", "  ")
				if err != nil {
					fmt.Printf("[%s] %s: Failed to marshal metadata: %v\n", position, name, err)
					return
				}
				fmt.Printf("[%s] %s:\n%s\n%v\n", position, name, string(b), extra)
			} else {
				fmt.Printf("[%s] %s: %v\n", position, name, extra)
			}
			return
		}
		cat.AnnounceSystemEvent(ctx, position, name, "TestScenario", meta, extra...)
	}
	DefaultGhostLogger = func(ctx context.Context, meta *commonpb.Metadata) {
		cat, ok := ctx.Value("cat_announcer").(*thecathasnoname.TheCatHasNoName)
		if !ok || cat == nil {
			if meta != nil {
				b, err := json.MarshalIndent(meta, "", "  ")
				if err != nil {
					fmt.Printf("[ghost] Failed to marshal metadata: %v\n", err)
					return
				}
				fmt.Printf("[ghost]\n%s\n", string(b))
			}
			return
		}
		cat.AnnounceSystemEvent(ctx, "ghost", "GhostLogger", "TestScenario", meta)
	}
	DefaultBeforeEach = func(ctx context.Context, s *metadata.TestScenario) {
		DefaultAnnouncer(ctx, "before_each", s.Name, s.InitialMeta)
	}
	DefaultAfterEach = func(ctx context.Context, s *metadata.TestScenario, err error) {
		// Build a benchmark summary map for beautiful announcement
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
		// If you have timing, concurrency, or custom metrics, add them here
		// e.g., summary["Duration"] = ...
		if cat, ok := s.Params["cat_announcer"].(*thecathasnoname.TheCatHasNoName); ok && cat != nil {
			cat.AnnounceSystemEvent(ctx, "after_each", "TestRunner", s.Name, metadata.StructToMap(s.InitialMeta.ServiceSpecific), summary)
		} else {
			fmt.Printf("[BENCH-ANNOUNCE][after_each] %s: %+v %v\n", s.Name, s.InitialMeta, err)
		}
	}
	DefaultOnError = func(ctx context.Context, s *metadata.TestScenario, err error) {
		DefaultAnnouncer(ctx, "error", s.Name, s.InitialMeta, err)
	}
)

// GenerateTestDocs generates Markdown documentation for a test suite and its scenarios.
func GenerateTestDocs(suite *metadata.TestSuite) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Test Suite: %s\n\n", suite.Name))
	if len(suite.Scenarios) == 0 {
		sb.WriteString("_No scenarios defined._\n")
		return sb.String()
	}
	for _, s := range suite.Scenarios {
		sb.WriteString(fmt.Sprintf("## Scenario: %s\n", s.Name))
		if s.Description != "" {
			sb.WriteString(fmt.Sprintf("**Description:** %s\n\n", s.Description))
		}
		if len(s.Params) > 0 {
			sb.WriteString("**Parameters:**\n")
			for k, v := range s.Params {
				sb.WriteString(fmt.Sprintf("- `%s`: `%v`\n", k, v))
			}
			sb.WriteString("\n")
		}
		if s.InitialMeta != nil {
			sb.WriteString("**Initial Metadata:**\n")
			sb.WriteString("```json\n")
			b, err := json.MarshalIndent(s.InitialMeta, "", "  ")
			if err != nil {
				sb.WriteString("Error marshaling metadata: " + err.Error())
			} else {
				sb.WriteString(string(b))
			}
			sb.WriteString("\n```")
		}
		sb.WriteString("\n---\n\n")
	}
	return sb.String()
}
