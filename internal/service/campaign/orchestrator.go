package campaign

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	mediapb "github.com/nmxmxh/master-ovasabi/api/protos/media/v1"
	userpb "github.com/nmxmxh/master-ovasabi/api/protos/user/v1"
	"github.com/nmxmxh/master-ovasabi/internal/server/ws"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
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
	wsClients     *ws.ClientMap // WebSocket client registry (set at startup)
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
	wsClients  *ws.ClientMap // WebSocket client registry for this campaign
	// Add campaign-specific state here (users, scores, etc.)
	container *di.Container
}

// NewOrchestratorManager creates a new orchestrator manager.
func NewOrchestratorManager(log *zap.Logger, cache *redis.Cache, wsClients *ws.ClientMap, container *di.Container) *OrchestratorManager {
	return &OrchestratorManager{
		log:        log,
		cache:      cache,
		dispatcher: make(chan OrchestratorEvent, 1024),
		stopCh:     make(chan struct{}),
		keyBuilder: redis.NewKeyBuilder(redis.NamespaceCache, redis.ContextCampaign),
		wsClients:  wsClients,
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
		wsClients:  m.wsClients,
		container:  m.container,
	}
	go torch.run(ctx)
	return torch
}

// Domain orchestrator event loop.
func (o *DomainOrchestrator) run(ctx context.Context) {
	var (
		defaultFPS   = 1.0
		maxBatchSize = 100 // Tune as needed
		interval     = time.Second / time.Duration(defaultFPS)
		fps          = defaultFPS
		intervalCh   = make(chan float64, 1)  // For dynamic FPS updates
		immediateCh  = make(chan struct{}, 8) // For event-driven triggers
		loadMu       sync.Mutex
		loadPercent  float64
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
			o.broadcastBatch(ctx, maxBatchSize)
			loadMu.Lock()
			loadPercent = o.estimateLoad()
			loadMu.Unlock()
			if loadPercent > 0.95 {
				o.throttleOrScale()
			}
		case <-immediateCh:
			// Event-driven broadcast
			o.broadcastBatch(ctx, maxBatchSize)
		case newFPS := <-intervalCh:
			ticker.Stop()
			updateInterval(newFPS)
		case <-o.stopCh:
			ticker.Stop()
			return
		}
	}
}

// broadcastBatch sends updates to users in batches, using async workers for large campaigns.
func (o *DomainOrchestrator) broadcastBatch(ctx context.Context, batchSize int) {
	users := o.getActiveUsers()
	n := len(users)
	if n == 0 {
		return
	}
	batches := (n + batchSize - 1) / batchSize
	var wg sync.WaitGroup
	for i := 0; i < batches; i++ {
		start := i * batchSize
		end := start + batchSize
		if end > n {
			end = n
		}
		wg.Add(1)
		go func(batch []string) {
			defer wg.Done()
			for _, userID := range batch {
				client, ok := o.getClient(userID)
				if !ok {
					continue
				}
				select {
				case client.Send() <- o.prepareBroadcastData(userID):
					// sent
				default:
					// drop frame for slow client
				}
			}
		}(users[start:end])
	}
	wg.Wait()
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

// getActiveUsers returns a list of active user IDs for this campaign from ws.ClientMap.
func (o *DomainOrchestrator) getActiveUsers() []string {
	var users []string
	if o.wsClients == nil {
		return users
	}
	o.wsClients.Range(func(campaignID, userID string, _ *ws.Client) bool {
		if campaignID == o.campaignID {
			users = append(users, userID)
		}
		return true
	})
	return users
}

// getClient returns the WebSocket client for a user in this campaign.
func (o *DomainOrchestrator) getClient(userID string) (*ws.Client, bool) {
	if o.wsClients == nil {
		return nil, false
	}
	return o.wsClients.Load(o.campaignID, userID)
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
func BuildCampaignUserState(campaign *Campaign, user *userpb.User, _ []LeaderboardEntry, mediaState *mediapb.Media, opts ...StateBuilderOption) map[string]interface{} {
	// Canonical extraction from metadata
	var campaignVars, userVars map[string]interface{}
	if campaign != nil && campaign.Metadata != nil {
		campaignVars = metadata.ExtractServiceVariables(campaign.Metadata, "campaign")
	}
	if user != nil && user.Metadata != nil {
		userVars = metadata.ExtractServiceVariables(user.Metadata, "user")
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
			"id":           user.Id,
			"username":     user.Username,
			"email":        user.Email,
			"roles":        user.Roles,
			"status":       user.Status,
			"profile":      user.Profile,
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

// Refactor DomainOrchestrator.prepareBroadcastData to use BuildCampaignUserState.
func (o *DomainOrchestrator) prepareBroadcastData(_ string) ws.WebSocketEvent {
	payload := BuildCampaignUserState(nil, nil, nil, nil)
	return ws.WebSocketEvent{
		Type:    "campaign_update",
		Payload: payload,
	}
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
// wsClients *ws.WsClientMap

// InitBroadcasts initializes the broadcast map.
func (s *Service) InitBroadcasts() {
	s.broadcastMu.Lock()
	defer s.broadcastMu.Unlock()
	if s.activeBroadcasts == nil {
		s.activeBroadcasts = make(map[string]context.CancelFunc)
	}
}

// InitScheduler initializes the cron scheduler and job map.
func (s *Service) InitScheduler() {
	if s.cronScheduler == nil {
		s.cronScheduler = cron.New()
		s.scheduledJobs = make(map[string][]cron.EntryID)
		s.cronScheduler.Start()
	}
}

// SetWSClients sets the WebSocket client map for orchestrator integration.
func (s *Service) SetWSClients(clients *ws.ClientMap) {
	s.clients = clients
}

// prepareBroadcastData prepares the data to send to a user in a campaign (global or user-specific).
func (s *Service) prepareBroadcastData(c *Campaign, _ string) ws.WebSocketEvent {
	payload := BuildCampaignUserState(c, nil, nil, nil)
	return ws.WebSocketEvent{
		Type:    "campaign_update",
		Payload: payload,
	}
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
				if s.clients == nil {
					s.log.Warn("No wsClients registry set; skipping broadcast", zap.String("slug", c.Slug))
					continue
				}
				s.clients.Range(func(campaignID, userID string, client *ws.Client) bool {
					if campaignID == c.Slug {
						data := s.prepareBroadcastData(c, userID)
						select {
						case client.Send() <- data:
							// sent
						default:
							s.log.Warn("Dropping frame for slow client", zap.String("campaign", campaignID), zap.String("user", userID))
						}
					}
					return true
				})
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
