package campaign

import (
	"context"
	"sync"
	"time"

	ws "github.com/nmxmxh/master-ovasabi/internal/server/ws"
	events "github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

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
		return err
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
			meta, err := FromStruct(c.Metadata.ServiceSpecific.Fields["campaign"].GetStructValue())
			if err != nil {
				s.log.Warn("Invalid campaign metadata, skipping", zap.String("slug", c.Slug), zap.Error(err))
				return
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
func (s *Service) prepareBroadcastData(c *Campaign, userID string) ws.WebSocketEvent {
	// TODO: Customize this for your use case. Example:
	return ws.WebSocketEvent{
		Type: "campaign_update",
		Payload: map[string]interface{}{
			"campaign":  c.Slug,
			"user":      userID,
			"timestamp": time.Now().UTC(),
			// Add more fields as needed (leaderboard, personalized stats, etc.)
		},
	}
}

// startBroadcast starts or updates a broadcast for a campaign using metadata and streams to all users.
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
				if s.eventEnabled && s.eventEmitter != nil {
					events.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "campaign.broadcast_stopped", c.Slug, c.Metadata)
				}
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
				if s.eventEnabled && s.eventEmitter != nil {
					events.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "campaign.broadcast_tick", c.Slug, c.Metadata)
				}
			}
		}
	}()
	if s.eventEnabled && s.eventEmitter != nil {
		events.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "campaign.broadcast_started", c.Slug, c.Metadata)
	}
}

// stopBroadcast stops a broadcast for a campaign and emits an event.
func (s *Service) stopBroadcast(ctx context.Context, slug string, c *Campaign) {
	s.broadcastMu.Lock()
	defer s.broadcastMu.Unlock()
	if cancel, exists := s.activeBroadcasts[slug]; exists {
		cancel()
		delete(s.activeBroadcasts, slug)
		s.log.Info("Stopped broadcast for campaign", zap.String("slug", slug))
		if s.eventEnabled && s.eventEmitter != nil && c != nil {
			events.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "campaign.broadcast_stopped", slug, c.Metadata)
		}
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
	if j.s.eventEnabled && j.s.eventEmitter != nil {
		events.EmitEventWithLogging(j.ctx, j.s.eventEmitter, j.s.log, "campaign.job_executed", j.c.Slug, j.c.Metadata)
	}
	// TODO: Implement actual job logic here
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
	if s.eventEnabled && s.eventEmitter != nil {
		events.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "campaign.job_scheduled", c.Slug, c.Metadata)
	}
}

// stopJobs stops all jobs for a campaign and emits an event.
func (s *Service) stopJobs(ctx context.Context, slug string, c *Campaign) {
	s.InitScheduler()
	if ids, ok := s.scheduledJobs[slug]; ok {
		for _, id := range ids {
			s.cronScheduler.Remove(id)
		}
		delete(s.scheduledJobs, slug)
		s.log.Info("Stopped all jobs for campaign", zap.String("slug", slug))
		if s.eventEnabled && s.eventEmitter != nil && c != nil {
			events.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "campaign.jobs_stopped", slug, c.Metadata)
		}
	}
}
