package campaign

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	mediapb "github.com/nmxmxh/master-ovasabi/api/protos/media/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/structpb"
)

// Constants for Redis channels, must match ws-gateway.
const (
	redisEgressSystem   = "ws:egress:system"
	redisEgressCampaign = "ws:egress:campaign:" // + {campaign_id}
	redisEgressUser     = "ws:egress:user:"     // + {user_id}
)

// OrchestratorEvent represents an event for campaign orchestration.
type OrchestratorEvent struct {
	CampaignID string
	Type       string
	Payload    interface{}
	Metadata   *commonpb.Metadata // canonical metadata for extensibility
}

// OrchestratorManager manages orchestrator goroutines per campaign (domain).
type OrchestratorManager struct {
	log           *zap.Logger
	cache         *redis.Cache
	orchestrators sync.Map // map[string]*DomainOrchestrator
	dispatcher    chan OrchestratorEvent
	stopCh        chan struct{}
	keyBuilder    *redis.KeyBuilder
	container     *di.Container
}

// DomainOrchestrator is a goroutine managing orchestration for a single campaign.
type DomainOrchestrator struct {
	campaignID string
	eventCh    chan OrchestratorEvent
	stopCh     chan struct{}
	log        *zap.Logger
	cache      *redis.Cache
	keyBuilder *redis.KeyBuilder
	// Add campaign-specific state here (users, scores, etc.)
	container *di.Container
	// Metrics for this orchestrator
	broadcastCount int64
}

// NewOrchestratorManager creates a new orchestrator manager.
func NewOrchestratorManager(log *zap.Logger, cache *redis.Cache, container *di.Container) *OrchestratorManager {
	return &OrchestratorManager{
		log:        log,
		cache:      cache,
		dispatcher: make(chan OrchestratorEvent, 1024),
		stopCh:     make(chan struct{}),
		keyBuilder: redis.NewKeyBuilder(redis.NamespaceCache, redis.ContextCampaign),
		container:  container,
	}
}

// Start launches the orchestrator nervous system.
func (m *OrchestratorManager) Start(ctx context.Context) {
	go m.redisSubscriber(ctx)
	go m.dispatchLoop(ctx)
}

// redisSubscriber subscribes to campaign events and feeds them into the dispatcher.
func (m *OrchestratorManager) redisSubscriber(ctx context.Context) {
	pubsub := m.cache.GetClient().Subscribe(ctx, "campaign:events")
	ch := pubsub.Channel()
	for {
		select {
		case msg := <-ch:
			// Parse event (assume JSON for production)
			var evt OrchestratorEvent
			if err := json.Unmarshal([]byte(msg.Payload), &evt); err == nil {
				m.dispatcher <- evt
			} else {
				m.log.Warn("Failed to parse orchestrator event", zap.Error(err), zap.String("payload", msg.Payload))
			}
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		}
	}
}

// dispatchLoop routes events to the correct orchestrator goroutine.
func (m *OrchestratorManager) dispatchLoop(ctx context.Context) {
	for {
		select {
		case event := <-m.dispatcher:
			value, loaded := m.orchestrators.LoadOrStore(event.CampaignID, m.newDomainOrchestrator(ctx, event.CampaignID))
			if !loaded {
				m.log.Info("Created new domain orchestrator", zap.String("campaign_id", event.CampaignID))
			}
			torch, ok := value.(*DomainOrchestrator)
			if !ok {
				m.log.Error("Failed type assertion to *DomainOrchestrator", zap.Any("value", value))
				continue
			}
			select {
			case torch.eventCh <- event:
				// delivered
			default:
				m.log.Warn("Orchestrator event channel full", zap.String("campaign", event.CampaignID))
			}
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		}
	}
}

// newDomainOrchestrator creates and starts a new orchestrator goroutine for a campaign.
func (m *OrchestratorManager) newDomainOrchestrator(ctx context.Context, campaignID string) *DomainOrchestrator {
	torch := &DomainOrchestrator{
		campaignID: campaignID,
		eventCh:    make(chan OrchestratorEvent, 256),
		stopCh:     make(chan struct{}),
		log:        m.log.With(zap.String("campaign", campaignID)),
		cache:      m.cache,
		keyBuilder: m.keyBuilder,
		container:  m.container,
	}
	go torch.run(ctx)
	return torch
}

// Domain orchestrator event loop.
func (o *DomainOrchestrator) run(ctx context.Context) {
	var (
		defaultFPS  = 1.0
		interval    = time.Second / time.Duration(defaultFPS)
		fps         = defaultFPS
		intervalCh  = make(chan float64, 1) // For dynamic FPS updates
		loadMu      sync.Mutex
		loadPercent float64
	)
	// Helper to update interval
	updateInterval := func(newFPS float64) {
		if newFPS <= 0 {
			newFPS = defaultFPS
		}
		fps = newFPS
		interval = time.Second / time.Duration(fps)
	}
	// Add a real-time metadata watcher using Redis pub/sub
	go o.watchMetadataChanges(ctx, intervalCh)
	// Main broadcast loop
	for {
		ticker := time.NewTicker(interval)
		select {
		case <-ticker.C:
			// Interval-based broadcast
			o.broadcastCampaignUpdate(ctx)
			loadMu.Lock()
			loadPercent = o.estimateLoad()
			loadMu.Unlock()
			if loadPercent > 0.95 {
				o.throttleOrScale()
			}
		case newFPS := <-intervalCh:
			ticker.Stop()
			updateInterval(newFPS)
		case <-o.stopCh:
			ticker.Stop()
			return
		}
	}
}

// broadcastCampaignUpdate prepares and sends a full campaign state update via Redis.
func (o *DomainOrchestrator) broadcastCampaignUpdate(ctx context.Context) {
	// In a real scenario, you would fetch the full, live campaign state here.
	// For this example, we'll construct a mock state.
	// This logic should be enriched to fetch real data from services.
	campaignState := map[string]interface{}{
		"id":                     o.campaignID,
		"live_participant_count": 123, // Example data
		"leaderboard": []map[string]interface{}{
			{"user": "alice", "score": 150},
			{"user": "bob", "score": 140},
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
	}

	event := map[string]interface{}{
		"type":    "campaign_update",
		"payload": campaignState,
	}

	payloadBytes, err := json.Marshal(event)
	if err != nil {
		o.log.Error("Failed to marshal campaign update payload", zap.Error(err))
		return
	}

	// Publish to the campaign-specific channel for the ws-gateway to pick up
	channel := redisEgressCampaign + o.campaignID
	if err := o.cache.GetClient().Publish(ctx, channel, payloadBytes).Err(); err != nil {
		o.log.Error("Failed to publish campaign update to Redis", zap.Error(err), zap.String("channel", channel))
	}
	atomic.AddInt64(&o.broadcastCount, 1)

	// After broadcasting, orchestrate with graceful
	success := graceful.WrapSuccess(ctx, codes.OK, "campaign broadcast", nil, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:         o.log,
		EventType:   "campaign.broadcast",
		EventID:     o.campaignID,
		PatternType: "campaign",
		PatternID:   o.campaignID,
		PatternMeta: o.getLatestMetadata(ctx),
		// Add EventEmitter, Cache, etc. if needed
	})
}

// estimateLoad estimates orchestrator load using queue length and batch wait time.
func (o *DomainOrchestrator) estimateLoad() float64 {
	// Use eventCh length and batch wait time as a proxy for load
	queueLen := len(o.eventCh)
	maxQueue := cap(o.eventCh)
	load := float64(queueLen) / float64(maxQueue)
	// Optionally, add batch wait time or CPU metrics here
	return load
}

// throttleOrScale throttles broadcast or triggers horizontal scaling if load is too high.
func (o *DomainOrchestrator) throttleOrScale() {
	load := o.estimateLoad()
	if load > 0.95 {
		o.log.Warn("Orchestrator load > 95%, consider horizontal scaling and distributed orchestration", zap.String("campaign", o.campaignID))
		// Future: implement worker/shard handoff for distributed orchestration
	} else if load > 0.85 {
		o.log.Info("Orchestrator load > 85%, reducing FPS", zap.String("campaign", o.campaignID))
		// Reduce FPS by 20% (min 1Hz)
		// This is a placeholder; in production, use a channel or atomic to update interval
	}
}

// getLatestMetadata fetches the latest campaign metadata from Redis cache.
func (o *DomainOrchestrator) getLatestMetadata(ctx context.Context) *commonpb.Metadata {
	metaKey := o.keyBuilder.Build("campaign", o.campaignID)
	var meta commonpb.Metadata
	if err := o.cache.Get(ctx, metaKey, "campaign", &meta); err == nil {
		return &meta
	}
	return nil
}

// Option type for partial update logic.
type (
	StateBuilderOption func(*stateBuilderConfig)
	stateBuilderConfig struct {
		Fields           []string
		Changed          map[string]bool // map of changed fields for partial update
		PercentThreshold float64         // e.g., 0.6 for 60%
	}
)

// WithFields specifies which fields to include in the state output.
func WithFields(fields []string) StateBuilderOption {
	return func(cfg *stateBuilderConfig) { cfg.Fields = fields }
}

// WithChangedFields specifies which fields have changed.
func WithChangedFields(changed map[string]bool) StateBuilderOption {
	return func(cfg *stateBuilderConfig) { cfg.Changed = changed }
}

// WithPercentThreshold sets the percent threshold for sending full state.
func WithPercentThreshold(threshold float64) StateBuilderOption {
	return func(cfg *stateBuilderConfig) { cfg.PercentThreshold = threshold }
}

// BuildCampaignUserState builds the minimal, gamified, partial-update-ready state for a campaign/user.
// Now uses canonical metadata.ExtractServiceVariables for all variable extraction.
func BuildCampaignUserState(campaign *Campaign, user *structpb.Struct, _ []LeaderboardEntry, mediaState *mediapb.Media, opts ...StateBuilderOption) map[string]interface{} {
	// Canonical extraction from metadata
	var campaignVars, userVars map[string]interface{}
	if campaign != nil && campaign.Metadata != nil {
		campaignVars = metadata.ExtractServiceVariables(campaign.Metadata, "campaign")
	}
	var userMap map[string]interface{}
	if user != nil {
		userMap = user.AsMap()
		if meta, ok := userMap["metadata"].(map[string]interface{}); ok {
			// Assuming metadata is a structpb.Struct within the user struct
			b, _ := json.Marshal(meta)
			var commonMeta commonpb.Metadata
			if json.Unmarshal(b, &commonMeta) == nil {
				userVars = metadata.ExtractServiceVariables(&commonMeta, "user")
			}
		}
	}

	// Compose all fields
	state := map[string]interface{}{
		"campaign": map[string]interface{}{
			"id":          campaign.Slug,
			"title":       campaign.Title,
			"status":      campaignVars["status"],
			"features":    campaignVars["features"],
			"trending":    campaignVars["trending"],
			"leaderboard": campaignVars["leaderboard"],
			"focus": map[string]interface{}{
				"description": campaign.Description,
				"rules":       campaignVars["rules"],
			},
		},
		"user": map[string]interface{}{
			"details":      userMap,
			"gamification": userVars,
		},
		"media": mediaState,
		"catalog": map[string]interface{}{
			"components": []map[string]interface{}{{"id": "Leaderboard", "props": map[string]interface{}{"top": 3}}},
			"search":     []map[string]interface{}{{"id": "Leaderboard", "title": "Leaderboard"}},
		},
		"effect_state": campaignVars["effect_state"],
		"timestamp":    time.Now().UTC(),
	}
	// Partial update logic
	cfg := &stateBuilderConfig{PercentThreshold: 0.6}
	for _, opt := range opts {
		opt(cfg)
	}
	allFields := []string{"campaign", "user", "media", "catalog", "effect_state", "timestamp"}
	if len(cfg.Fields) == 0 && len(cfg.Changed) == 0 {
		return state
	}
	if len(cfg.Changed) > 0 && float64(len(cfg.Changed))/float64(len(allFields)) >= cfg.PercentThreshold {
		return state
	}
	partial := make(map[string]interface{})
	if len(cfg.Fields) > 0 {
		for _, f := range cfg.Fields {
			if v, ok := state[f]; ok {
				partial[f] = v
			}
		}
		return partial
	}
	if len(cfg.Changed) > 0 {
		for f := range cfg.Changed {
			if v, ok := state[f]; ok {
				partial[f] = v
			}
		}
		return partial
	}
	return state
}

// Add a real-time metadata watcher using Redis pub/sub.
func (o *DomainOrchestrator) watchMetadataChanges(ctx context.Context, intervalCh chan<- float64) {
	channel := "campaign:metadata:" + o.campaignID
	pubsub := o.cache.GetClient().Subscribe(ctx, channel)
	defer func() {
		if err := pubsub.Close(); err != nil {
			o.log.Warn("Failed to close pubsub in watchMetadataChanges", zap.Error(err))
		}
	}()
	ch := pubsub.Channel()
	for {
		select {
		case <-ch:
			// Metadata changed, fetch latest
			meta := o.getLatestMetadata(ctx)
			if meta != nil && meta.GetScheduling() != nil {
				fields := meta.GetScheduling().GetFields()
				if freqVal, ok := fields["frequency"]; ok {
					f := freqVal.GetNumberValue()
					if f > 0 {
						intervalCh <- f
					}
				}
			}
		case <-o.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// Stop gracefully shuts down the orchestrator manager and all orchestrators.
func (m *OrchestratorManager) Stop() {
	close(m.stopCh)
	m.orchestrators.Range(func(_, value interface{}) bool {
		if orchestrator, ok := value.(*DomainOrchestrator); ok {
			close(orchestrator.stopCh)
		}
		return true
	})
}

// Extensibility: Add hooks for Nexus, metadata enrichment, real-time leaderboards, etc.
// This pattern is ready for gaming, real-time campaigns, and cross-service orchestration.

// OrchestrateActiveCampaignsAdvanced scans and orchestrates all active campaigns efficiently.
// - Uses SQL filtering for active campaigns.
// - Runs orchestration concurrently (worker pool).
// - Integrates with the event bus for orchestration events.
func (s *Service) OrchestrateActiveCampaignsAdvanced(ctx context.Context, maxWorkers int) error {
	s.log.Info("Starting advanced campaign orchestration scan")
	// 1. SQL filter: only fetch campaigns that are active and within their scheduling window
	now := time.Now()
	campaigns, err := s.repo.ListActiveWithinWindow(ctx, now)
	if err != nil {
		s.log.Error("Failed to list active campaigns for orchestration", zap.Error(err))
		return graceful.WrapErr(ctx, codes.Internal, "Failed to list active campaigns for orchestration", err)
	}
	if len(campaigns) == 0 {
		s.log.Info("No active campaigns to orchestrate at this time")
		return nil
	}

	// 2. Concurrency: worker pool
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxWorkers)
	for _, c := range campaigns {
		c := c // capture loop var
		wg.Add(1)
		sem <- struct{}{} // acquire
		go func() {
			defer wg.Done()
			defer func() { <-sem }() // release
			// Use canonical handler to convert structpb.Struct to map
			metaStruct := c.Metadata.ServiceSpecific.Fields["campaign"].GetStructValue()
			var meta *Metadata
			if metaStruct != nil {
				metaMap := metadata.StructToMap(metaStruct)
				meta = &Metadata{}
				if f, ok := metaMap["features"].([]interface{}); ok {
					for _, feat := range f {
						if fs, ok := feat.(string); ok {
							meta.Features = append(meta.Features, fs)
						}
					}
				}
				if s, ok := metaMap["scheduling"].(map[string]interface{}); ok {
					meta.Scheduling = mapToSchedulingInfo(s)
				}
				if c, ok := metaMap["custom"].(map[string]interface{}); ok {
					meta.Custom = c
				}
				// Add more fields as needed per canonical handler
			}
			s.advancedOrchestrateCampaign(ctx, c, meta)
		}()
	}
	wg.Wait()
	return nil
}

// advancedOrchestrateCampaign orchestrates a single campaign and emits events.
func (s *Service) advancedOrchestrateCampaign(ctx context.Context, c *Campaign, meta *Metadata) {
	s.log.Info("Orchestrating campaign (advanced)", zap.String("slug", c.Slug))
	// Example: Start broadcast if enabled via feature toggle
	if contains(meta.Features, "broadcast") {
		s.startBroadcast(ctx, c, meta)
	}
	// Example: Schedule jobs
	if meta.Scheduling != nil && len(meta.Scheduling.Jobs) > 0 {
		for _, job := range meta.Scheduling.Jobs {
			s.scheduleJob(ctx, c, meta, job)
		}
	}
	// Add more orchestration logic as needed (WebSocket, notifications, etc.)
}

// Add to Service struct:
// broadcastMu sync.Mutex
// activeBroadcasts map[string]context.CancelFunc
// cronScheduler *cron.Cron
// scheduledJobs map[string][]cron.EntryID

// InitBroadcasts initializes the broadcast map.
func (s *Service) InitBroadcasts() {
	s.broadcastMu.Lock()
	if s.activeBroadcasts == nil {
		s.activeBroadcasts = make(map[string]context.CancelFunc)
	}
	s.broadcastMu.Unlock()
}

// InitScheduler initializes the cron scheduler and job map.
func (s *Service) InitScheduler() {
	s.broadcastMu.Lock() // Using broadcastMu for scheduler as well for simplicity
	defer s.broadcastMu.Unlock()
	if s.cronScheduler == nil {
		s.cronScheduler = cron.New()
		s.scheduledJobs = make(map[string][]cron.EntryID)
		s.cronScheduler.Start()
	}
}

// prepareBroadcastData prepares the data to send to a user in a campaign (global or user-specific).
func (s *Service) prepareBroadcastData(c *Campaign, _ string) ([]byte, error) {
	payload := BuildCampaignUserState(c, nil, nil, nil, nil)
	event := map[string]interface{}{
		"type":    "campaign_update",
		"payload": payload,
	}
	bytes, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

// orchestrateBroadcastEvent is a helper to DRY up graceful orchestration for broadcast events.
// It ensures all broadcast lifecycle events are handled consistently, with event emission, audit, and extensibility.
func (s *Service) orchestrateBroadcastEvent(ctx context.Context, _, eventID string, meta *commonpb.Metadata, msg string) {
	success := graceful.WrapSuccess(ctx, codes.OK, msg, nil, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:         s.log,
		Metadata:    meta,
		PatternType: "campaign",
		PatternID:   eventID,
		PatternMeta: meta,
	})
}

// startBroadcast starts or updates a broadcast for a campaign using metadata and streams to all users.
// In the new context, this function ensures that all broadcast start and tick events are orchestrated via graceful,
// enabling audit, event bus emission, and future extensibility.
func (s *Service) startBroadcast(ctx context.Context, c *Campaign, meta *Metadata) {
	s.broadcastMu.Lock()
	defer s.broadcastMu.Unlock()
	if s.activeBroadcasts == nil {
		s.activeBroadcasts = make(map[string]context.CancelFunc)
	}
	if cancel, exists := s.activeBroadcasts[c.Slug]; exists {
		cancel()
		s.log.Info("Canceled existing broadcast for campaign before starting new one", zap.String("slug", c.Slug))
		delete(s.activeBroadcasts, c.Slug)
	}
	bctx, cancel := context.WithCancel(context.Background())
	s.activeBroadcasts[c.Slug] = cancel
	// Use a default frequency or allow override via custom rules in meta.Custom
	freq := 1
	if meta.Custom != nil {
		if f, ok := meta.Custom["broadcast_frequency"].(int); ok && f > 0 {
			freq = f
		}
	}
	interval := time.Second / time.Duration(freq)
	s.log.Info("Starting broadcast for campaign", zap.String("slug", c.Slug), zap.Int("frequency", freq))
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-bctx.Done():
				s.log.Info("Broadcast stopped for campaign", zap.String("slug", c.Slug))
				s.orchestrateBroadcastEvent(ctx, "campaign.broadcast_stopped", c.Slug, c.Metadata, "campaign broadcast stopped")
				return
			case <-ticker.C:
				data, err := s.prepareBroadcastData(c, "")
				if err != nil {
					s.log.Error("Failed to prepare broadcast data", zap.String("slug", c.Slug), zap.Error(err))
					continue
				}
				channel := redisEgressCampaign + c.Slug
				if err := s.cache.GetClient().Publish(ctx, channel, data).Err(); err != nil {
					s.log.Error("Failed to publish broadcast to Redis", zap.String("slug", c.Slug), zap.Error(err))
				}

				s.orchestrateBroadcastEvent(ctx, "campaign.broadcast_tick", c.Slug, c.Metadata, "campaign broadcast tick")
			}
		}
	}()
	s.orchestrateBroadcastEvent(ctx, "campaign.broadcast_started", c.Slug, c.Metadata, "campaign broadcast started")
}

// stopBroadcast stops a broadcast for a campaign and emits an event using graceful orchestration.
// In the new context, this function ensures that all broadcast stop events are orchestrated via graceful,
// enabling audit, event bus emission, and future extensibility.
func (s *Service) stopBroadcast(ctx context.Context, slug string, c *Campaign) {
	s.broadcastMu.Lock()
	defer s.broadcastMu.Unlock()
	if cancel, exists := s.activeBroadcasts[slug]; exists {
		cancel()
		delete(s.activeBroadcasts, slug)
		s.log.Info("Stopped broadcast for campaign", zap.String("slug", slug))
		s.orchestrateBroadcastEvent(ctx, "campaign.broadcast_stopped", slug, c.Metadata, "campaign broadcast stopped")
	}
}

// campaignJob is a cron job for campaign scheduling that holds context.
type campaignJob struct {
	ctx     context.Context
	s       *Service
	c       *Campaign
	meta    *Metadata
	jobType string
}

func (j *campaignJob) Run() {
	j.s.log.Info("Running job", zap.String("slug", j.c.Slug), zap.String("type", j.jobType))
	// Use graceful to emit a job execution event for relevant services to consume
	success := graceful.WrapSuccess(j.ctx, codes.OK, "campaign job executed", nil, nil)
	success.StandardOrchestrate(j.ctx, graceful.SuccessOrchestrationConfig{
		Log:         j.s.log,
		EventType:   "campaign.job_executed",
		EventID:     j.c.Slug,
		PatternType: "campaign",
		PatternID:   j.c.Slug,
		PatternMeta: j.c.Metadata,
	})
}

// scheduleJob schedules or triggers a job for a campaign using metadata.
func (s *Service) scheduleJob(ctx context.Context, c *Campaign, meta *Metadata, job map[string]interface{}) {
	s.InitScheduler()
	if s.scheduledJobs == nil {
		s.scheduledJobs = make(map[string][]cron.EntryID)
	}
	jobType, ok := job["type"].(string)
	if !ok {
		s.log.Warn("Job type is not a string; skipping job scheduling", zap.Any("job", job))
		return
	}
	cronExpr := "0 0 * * *" // default: daily
	if expr, ok := job["cron"].(string); ok {
		cronExpr = expr
	}
	jobInstance := &campaignJob{
		ctx:     ctx,
		s:       s,
		c:       c,
		meta:    meta,
		jobType: jobType,
	}
	id, err := s.cronScheduler.AddJob(cronExpr, jobInstance)
	if err != nil {
		s.log.Warn("Failed to schedule cron job", zap.String("slug", c.Slug), zap.Error(err))
		return
	}
	s.scheduledJobs[c.Slug] = append(s.scheduledJobs[c.Slug], id)
	s.log.Info("Scheduled cron job", zap.String("slug", c.Slug), zap.String("cron", cronExpr))
}

// stopJobs stops all jobs for a campaign and emits an event.
func (s *Service) stopJobs(_ context.Context, slug string, _ *Campaign) {
	s.InitScheduler()
	if ids, ok := s.scheduledJobs[slug]; ok {
		for _, id := range ids {
			s.cronScheduler.Remove(id)
		}
		delete(s.scheduledJobs, slug)
		s.log.Info("Stopped all jobs for campaign", zap.String("slug", slug))
	}
}

// Helper to convert map[string]interface{} to *SchedulingInfo.
func mapToSchedulingInfo(m map[string]interface{}) *SchedulingInfo {
	if m == nil {
		return nil
	}
	si := &SchedulingInfo{}
	if start, ok := m["start"].(string); ok {
		if t, err := time.Parse(time.RFC3339, start); err == nil {
			si.Start = t
		}
	}
	if end, ok := m["end"].(string); ok {
		if t, err := time.Parse(time.RFC3339, end); err == nil {
			si.End = t
		}
	}
	if rec, ok := m["recurrence"].(string); ok {
		si.Recurrence = rec
	}
	if jobs, ok := m["jobs"].([]interface{}); ok {
		for _, job := range jobs {
			if jm, ok := job.(map[string]interface{}); ok {
				si.Jobs = append(si.Jobs, jm)
			}
		}
	}
	return si
}
