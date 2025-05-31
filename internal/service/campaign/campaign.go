package campaign

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/expr-lang/expr"
	campaignpb "github.com/nmxmxh/master-ovasabi/api/protos/campaign/v1"
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	ws "github.com/nmxmxh/master-ovasabi/internal/server/ws"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Compile-time check.
var _ campaignpb.CampaignServiceServer = (*Service)(nil)

// Helper to check if a campaign is active based on metadata.
func isCampaignActive(meta *commonpb.Metadata) bool {
	if meta != nil && meta.ServiceSpecific != nil {
		if campaignField, ok := meta.ServiceSpecific.Fields["campaign"]; ok {
			if campaignStruct := campaignField.GetStructValue(); campaignStruct != nil {
				if statusVal, ok := campaignStruct.Fields["status"]; ok {
					return statusVal.GetStringValue() == "active"
				}
			}
		}
	}
	return false
}

// Service implements the CampaignService gRPC interface.
type Service struct {
	campaignpb.UnimplementedCampaignServiceServer
	log          *zap.Logger
	repo         *Repository
	cache        *redis.Cache
	eventEnabled bool

	// Broadcast and job scheduling fields
	broadcastMu      sync.Mutex
	activeBroadcasts map[string]context.CancelFunc
	cronScheduler    *cron.Cron
	scheduledJobs    map[string][]cron.EntryID

	// WebSocket client registry for campaign/user streaming
	clients *ws.ClientMap
}

func NewService(log *zap.Logger, repo *Repository, cache *redis.Cache, eventEnabled bool) *Service {
	return &Service{
		log:          log,
		repo:         repo,
		cache:        cache,
		eventEnabled: eventEnabled,
	}
}

func SafeInt32(i int64) (int32, error) {
	if i > math.MaxInt32 || i < math.MinInt32 {
		return 0, fmt.Errorf("integer overflow: value %d out of int32 range", i)
	}
	return int32(i), nil
}

func (s *Service) CreateCampaign(ctx context.Context, req *campaignpb.CreateCampaignRequest) (*campaignpb.CreateCampaignResponse, error) {
	log := s.log.With(
		zap.String("operation", "create_campaign"),
		zap.String("slug", req.Slug))

	log.Info("Creating campaign")

	// Input validation
	if req.Slug == "" {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "slug is required", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{
			Log: log,
			// Add audit, alert, fallback hooks as needed
		})
		return nil, graceful.ToStatusError(err)
	}
	if req.Title == "" {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "title is required", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{
			Log: log,
		})
		return nil, graceful.ToStatusError(err)
	}
	// Parse and validate campaign metadata
	var campaignMeta *Metadata
	if req.Metadata != nil && req.Metadata.ServiceSpecific != nil {
		if campaignField, ok := req.Metadata.ServiceSpecific.Fields["campaign"]; ok {
			metaStruct := campaignField.GetStructValue()
			var err error
			campaignMeta, err = FromStruct(metaStruct)
			if err != nil {
				err := graceful.WrapErr(ctx, codes.InvalidArgument, "invalid campaign metadata", err)
				err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
				return nil, graceful.ToStatusError(err)
			}
			if err := campaignMeta.Validate(); err != nil {
				err := graceful.WrapErr(ctx, codes.InvalidArgument, "invalid campaign metadata", err)
				err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
				return nil, graceful.ToStatusError(err)
			}
		}
	}
	// Start transaction
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to begin transaction", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	defer func() {
		if rerr := tx.Rollback(); rerr != nil {
			log.Error("error rolling back transaction", zap.Error(rerr))
		}
	}()
	// Create campaign with transaction
	c := &Campaign{
		Slug:           req.Slug,
		Title:          req.Title,
		Description:    req.Description,
		RankingFormula: req.RankingFormula,
		Metadata:       req.Metadata,
	}
	if req.StartDate != nil {
		c.StartDate = req.StartDate.AsTime()
	}
	if req.EndDate != nil {
		c.EndDate = req.EndDate.AsTime()
	}
	created, err := s.repo.CreateWithTransaction(ctx, tx, c)
	if err != nil {
		code := codes.Internal
		msg := "failed to create campaign"
		if errors.Is(err, ErrCampaignExists) {
			code = codes.AlreadyExists
			msg = "campaign already exists"
		}
		err := graceful.WrapErr(ctx, code, msg, err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	// Commit transaction
	if err := tx.Commit(); err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to commit transaction", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	id32, err := SafeInt32(created.ID)
	if err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "campaign ID overflow", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: log})
		return nil, graceful.ToStatusError(err)
	}
	resp := &campaignpb.Campaign{
		Id:             id32,
		Slug:           created.Slug,
		Title:          created.Title,
		Description:    created.Description,
		RankingFormula: created.RankingFormula,
		Metadata:       created.Metadata,
		CreatedAt:      timestamppb.New(created.CreatedAt),
		UpdatedAt:      timestamppb.New(created.UpdatedAt),
	}
	if !created.StartDate.IsZero() {
		resp.StartDate = timestamppb.New(created.StartDate)
	}
	if !created.EndDate.IsZero() {
		resp.EndDate = timestamppb.New(created.EndDate)
	}
	// Orchestrate all post-success actions via graceful
	success := graceful.WrapSuccess(ctx, codes.OK, "campaign created", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          log,
		Cache:        s.cache,
		CacheKey:     created.Slug,
		CacheValue:   resp,
		CacheTTL:     10 * time.Minute,
		Metadata:     created.Metadata,
		EventEnabled: s.eventEnabled,
		EventType:    "campaign.created",
		EventID:      created.Slug,
		PatternType:  "campaign",
		PatternID:    created.Slug,
		PatternMeta:  created.Metadata,
		// Optionally add custom hooks for knowledge graph, scheduler, etc.
	})
	return &campaignpb.CreateCampaignResponse{Campaign: resp}, nil
}

func (s *Service) GetCampaign(ctx context.Context, req *campaignpb.GetCampaignRequest) (*campaignpb.GetCampaignResponse, error) {
	log := s.log.With(
		zap.String("operation", "get_campaign"),
		zap.String("slug", req.Slug))

	log.Info("Getting campaign")

	// Try to get from cache first
	campaign, err := redis.GetOrSetWithProtection(ctx, s.cache, s.log, req.Slug, func(ctx context.Context) (*campaignpb.Campaign, error) {
		c, err := s.repo.GetBySlug(ctx, req.Slug)
		if err != nil {
			return nil, err
		}
		id32, err := SafeInt32(c.ID)
		if err != nil {
			return nil, err
		}
		resp := &campaignpb.Campaign{
			Id:             id32,
			Slug:           c.Slug,
			Title:          c.Title,
			Description:    c.Description,
			RankingFormula: c.RankingFormula,
			Metadata:       c.Metadata,
			CreatedAt:      timestamppb.New(c.CreatedAt),
			UpdatedAt:      timestamppb.New(c.UpdatedAt),
		}
		if !c.StartDate.IsZero() {
			resp.StartDate = timestamppb.New(c.StartDate)
		}
		if !c.EndDate.IsZero() {
			resp.EndDate = timestamppb.New(c.EndDate)
		}
		return resp, nil
	}, 10*time.Minute)
	if err == nil {
		resp := &campaignpb.GetCampaignResponse{Campaign: campaign}
		success := graceful.WrapSuccess(ctx, codes.OK, "campaign fetched", resp, nil)
		success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
			Log:          log,
			Cache:        s.cache,
			CacheKey:     req.Slug,
			CacheValue:   resp,
			CacheTTL:     10 * time.Minute,
			Metadata:     campaign.Metadata,
			EventEnabled: s.eventEnabled,
			EventType:    "campaign.fetched",
			EventID:      req.Slug,
			PatternType:  "campaign",
			PatternID:    req.Slug,
			PatternMeta:  campaign.Metadata,
		})
		return resp, nil
	}

	log.Error("Failed to get campaign", zap.Error(err))
	return nil, graceful.ToStatusError(err)
}

func (s *Service) UpdateCampaign(ctx context.Context, req *campaignpb.UpdateCampaignRequest) (*campaignpb.UpdateCampaignResponse, error) {
	authUserID, ok := utils.GetAuthenticatedUserID(ctx)
	if !ok {
		return nil, graceful.ToStatusError(graceful.WrapErr(ctx, codes.Unauthenticated, "missing authentication", nil))
	}
	roles, _ := utils.GetAuthenticatedUserRoles(ctx)
	isPlatformAdmin := utils.IsServiceAdmin(roles, "campaign")
	existing, err := s.repo.GetBySlug(ctx, req.Campaign.Slug)
	if err != nil {
		if errors.Is(err, ErrCampaignNotFound) {
			s.log.Warn("Campaign not found", zap.Error(err))
			return nil, graceful.ToStatusError(graceful.WrapErr(ctx, codes.NotFound, "campaign not found", err))
		}
		s.log.Error("Failed to get existing campaign", zap.Error(err))
		return nil, graceful.ToStatusError(graceful.WrapErr(ctx, codes.Internal, "failed to get existing campaign", err))
	}
	// --- Permission check: campaign membership/role ---
	role := GetUserRoleInCampaign(existing.Metadata, authUserID, existing.OwnerID)
	isSystem := IsSystemCampaign(existing.Metadata)
	if role != "admin" && role != "user" && (!isSystem || !isPlatformAdmin) {
		return nil, graceful.ToStatusError(graceful.WrapErr(ctx, codes.PermissionDenied, "insufficient campaign role", nil))
	}
	log := s.log.With(
		zap.String("operation", "update_campaign"),
		zap.String("slug", req.Campaign.Slug))

	log.Info("Updating campaign")

	wasActive := isCampaignActive(existing.Metadata)

	// Update fields
	existing.Title = req.Campaign.Title
	existing.Description = req.Campaign.Description
	existing.RankingFormula = req.Campaign.RankingFormula
	existing.Metadata = req.Campaign.Metadata
	if req.Campaign.StartDate != nil {
		existing.StartDate = req.Campaign.StartDate.AsTime()
	}
	if req.Campaign.EndDate != nil {
		existing.EndDate = req.Campaign.EndDate.AsTime()
	}

	// Update in database
	if err := s.repo.Update(ctx, existing); err != nil {
		if errors.Is(err, ErrCampaignNotFound) {
			log.Warn("Campaign not found during update", zap.Error(err))
			return nil, graceful.ToStatusError(graceful.WrapErr(ctx, codes.NotFound, "campaign not found", err))
		}
		log.Error("Failed to update campaign", zap.Error(err))
		return nil, graceful.ToStatusError(graceful.WrapErr(ctx, codes.Internal, "failed to update campaign", err))
	}

	// If campaign is now inactive or window ended, stop jobs and broadcasts
	isActive := isCampaignActive(existing.Metadata)

	if (!isActive && wasActive) || (!existing.EndDate.IsZero() && existing.EndDate.Before(time.Now())) {
		s.stopJobs(ctx, existing.Slug, existing)
		s.stopBroadcast(ctx, existing.Slug, existing)
	}

	// Get updated campaign
	getResp, err := s.GetCampaign(ctx, &campaignpb.GetCampaignRequest{Slug: existing.Slug})
	if err != nil {
		return nil, err
	}
	resp := &campaignpb.UpdateCampaignResponse{Campaign: getResp.Campaign}
	success := graceful.WrapSuccess(ctx, codes.OK, "campaign updated", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          log,
		Cache:        s.cache,
		CacheKey:     existing.Slug,
		CacheValue:   resp,
		CacheTTL:     10 * time.Minute,
		Metadata:     getResp.Campaign.Metadata,
		EventEnabled: s.eventEnabled,
		EventType:    "campaign.updated",
		EventID:      existing.Slug,
		PatternType:  "campaign",
		PatternID:    existing.Slug,
		PatternMeta:  getResp.Campaign.Metadata,
	})
	return resp, nil
}

func (s *Service) DeleteCampaign(ctx context.Context, req *campaignpb.DeleteCampaignRequest) (*campaignpb.DeleteCampaignResponse, error) {
	authUserID, ok := utils.GetAuthenticatedUserID(ctx)
	if !ok {
		return nil, graceful.ToStatusError(graceful.WrapErr(ctx, codes.Unauthenticated, "missing authentication", nil))
	}
	roles, _ := utils.GetAuthenticatedUserRoles(ctx)
	isPlatformAdmin := utils.IsServiceAdmin(roles, "campaign")
	campaign, err := s.repo.GetBySlug(ctx, req.Slug)
	if err != nil {
		if errors.Is(err, ErrCampaignNotFound) {
			s.log.Warn("Campaign not found", zap.Error(err))
			return nil, graceful.ToStatusError(graceful.WrapErr(ctx, codes.NotFound, "campaign not found", err))
		}
		s.log.Error("Failed to get campaign", zap.Error(err))
		return nil, graceful.ToStatusError(graceful.WrapErr(ctx, codes.Internal, "failed to get campaign", err))
	}
	// --- Permission check: campaign membership/role ---
	role := GetUserRoleInCampaign(campaign.Metadata, authUserID, campaign.OwnerID)
	isSystem := IsSystemCampaign(campaign.Metadata)
	if role != "admin" && role != "user" && (!isSystem || !isPlatformAdmin) {
		return nil, graceful.ToStatusError(graceful.WrapErr(ctx, codes.PermissionDenied, "insufficient campaign role", nil))
	}
	log := s.log.With(
		zap.String("operation", "delete_campaign"),
		zap.Int32("id", req.Id))

	log.Info("Deleting campaign")

	// Get campaign first to get the slug for cache invalidation and to stop jobs/broadcasts
	if campaign != nil {
		s.stopJobs(ctx, campaign.Slug, campaign)
		s.stopBroadcast(ctx, campaign.Slug, campaign)
	}

	if err := s.repo.Delete(ctx, int64(req.Id)); err != nil {
		if errors.Is(err, ErrCampaignNotFound) {
			log.Warn("Campaign not found", zap.Error(err))
			return nil, graceful.ToStatusError(graceful.WrapErr(ctx, codes.NotFound, "campaign not found", err))
		}
		log.Error("Failed to delete campaign", zap.Error(err))
		return nil, graceful.ToStatusError(graceful.WrapErr(ctx, codes.Internal, "failed to delete campaign", err))
	}

	// Invalidate cache if we have the slug
	if campaign != nil {
		if err := s.cache.Delete(ctx, campaign.Slug, "campaign"); err != nil {
			log.Error("Failed to invalidate campaign cache",
				zap.String("slug", campaign.Slug),
				zap.Error(err))
			// Don't fail the delete if cache invalidation fails
		}
	}

	resp := &campaignpb.DeleteCampaignResponse{Success: true}
	success := graceful.WrapSuccess(ctx, codes.OK, "campaign deleted", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          log,
		EventEnabled: s.eventEnabled,
		EventType:    "campaign.deleted",
		EventID:      campaign.Slug,
		PatternType:  "campaign",
		PatternID:    campaign.Slug,
		PatternMeta:  campaign.Metadata,
	})
	return resp, nil
}

func (s *Service) ListCampaigns(ctx context.Context, req *campaignpb.ListCampaignsRequest) (*campaignpb.ListCampaignsResponse, error) {
	log := s.log.With(
		zap.String("operation", "list_campaigns"),
		zap.Int32("page", req.Page),
		zap.Int32("page_size", req.PageSize))

	log.Info("Listing campaigns")

	// Input validation
	if req.Page < 0 {
		return nil, graceful.ToStatusError(graceful.WrapErr(ctx, codes.InvalidArgument, "page number cannot be negative", nil))
	}
	if req.PageSize < 0 || req.PageSize > 100 {
		return nil, graceful.ToStatusError(graceful.WrapErr(ctx, codes.InvalidArgument, "page size must be between 0 and 100", nil))
	}

	// Try to get from cache first
	cacheKey := fmt.Sprintf("campaigns:page:%d:size:%d", req.Page, req.PageSize)
	var response campaignpb.ListCampaignsResponse
	if err := s.cache.Get(ctx, cacheKey, "", &response); err == nil {
		return &response, nil
	}

	// If not in cache, get from database
	campaigns, err := s.repo.List(ctx, int(req.PageSize), int(req.Page*req.PageSize))
	if err != nil {
		log.Error("Failed to list campaigns", zap.Error(err))
		return nil, graceful.ToStatusError(graceful.WrapErr(ctx, codes.Internal, "failed to list campaigns", err))
	}

	resp := &campaignpb.ListCampaignsResponse{
		Campaigns: make([]*campaignpb.Campaign, 0, len(campaigns)),
	}

	for _, c := range campaigns {
		id32, err := SafeInt32(c.ID)
		if err != nil {
			log.Error("Campaign ID overflow",
				zap.Int64("id", c.ID),
				zap.Error(err))
			continue
		}

		campaign := &campaignpb.Campaign{
			Id:             id32,
			Slug:           c.Slug,
			Title:          c.Title,
			Description:    c.Description,
			RankingFormula: c.RankingFormula,
			Metadata:       c.Metadata,
			CreatedAt:      timestamppb.New(c.CreatedAt),
			UpdatedAt:      timestamppb.New(c.UpdatedAt),
		}
		if !c.StartDate.IsZero() {
			campaign.StartDate = timestamppb.New(c.StartDate)
		}
		if !c.EndDate.IsZero() {
			campaign.EndDate = timestamppb.New(c.EndDate)
		}
		resp.Campaigns = append(resp.Campaigns, campaign)
	}

	// Cache the response
	if err := s.cache.Set(ctx, cacheKey, "", resp, 5*time.Minute); err != nil {
		log.Error("Failed to cache campaign list",
			zap.String("cache_key", cacheKey),
			zap.Error(err))
		// Don't fail the list if caching fails
	}

	success := graceful.WrapSuccess(ctx, codes.OK, "campaigns listed", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:        log,
		Cache:      s.cache,
		CacheKey:   cacheKey,
		CacheValue: resp,
		CacheTTL:   5 * time.Minute,
		// Optionally add event emission, etc.
	})
	return resp, nil
}

// GetLeaderboard returns the leaderboard for a campaign, applying the dynamic ranking formula.
func (s *Service) GetLeaderboard(ctx context.Context, campaignSlug string, limit int) ([]LeaderboardEntry, error) {
	cacheKey := fmt.Sprintf("leaderboard:%s:%d", campaignSlug, limit)
	entries, err := redis.GetOrSetWithProtection(ctx, s.cache, s.log, cacheKey, func(ctx context.Context) ([]LeaderboardEntry, error) {
		c, err := s.repo.GetBySlug(ctx, campaignSlug)
		if err != nil {
			return nil, err
		}
		formula := c.RankingFormula
		if formula == "" {
			return nil, fmt.Errorf("no ranking formula defined for campaign")
		}
		entries, err := s.repo.GetLeaderboard(ctx, campaignSlug, formula, limit)
		if err != nil {
			return nil, err
		}
		program, err := expr.Compile(formula, expr.Env(map[string]interface{}{}))
		if err != nil {
			return nil, fmt.Errorf("invalid ranking formula: %w", err)
		}
		for i := range entries {
			vars := entries[i].Variables
			output, err := expr.Run(program, vars)
			if err != nil {
				entries[i].Score = 0
				continue
			}
			if score, ok := output.(float64); ok {
				entries[i].Score = score
			} else {
				entries[i].Score = 0
			}
		}
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Score > entries[j].Score
		})
		if limit > 0 && len(entries) > limit {
			entries = entries[:limit]
		}
		return entries, nil
	}, 1*time.Minute)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.WrapErr(ctx, codes.Internal, "failed to get leaderboard", err))
	}
	return entries, nil
}
