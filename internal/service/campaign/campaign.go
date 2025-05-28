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
	"github.com/nmxmxh/master-ovasabi/internal/service/pattern"
	events "github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Compile-time check.
var _ campaignpb.CampaignServiceServer = (*Service)(nil)

// Service implements the CampaignService gRPC interface.
type Service struct {
	campaignpb.UnimplementedCampaignServiceServer
	log          *zap.Logger
	repo         *Repository
	cache        *redis.Cache
	eventEmitter EventEmitter
	eventEnabled bool

	// Broadcast and job scheduling fields
	broadcastMu      sync.Mutex
	activeBroadcasts map[string]context.CancelFunc
	cronScheduler    *cron.Cron
	scheduledJobs    map[string][]cron.EntryID

	// WebSocket client registry for campaign/user streaming
	clients *ws.ClientMap
}

func NewService(log *zap.Logger, repo *Repository, cache *redis.Cache, eventEmitter EventEmitter, eventEnabled bool) *Service {
	return &Service{
		log:          log,
		repo:         repo,
		cache:        cache,
		eventEmitter: eventEmitter,
		eventEnabled: eventEnabled,
	}
}

// SafeInt32 converts an int64 to int32 with overflow checking.
func SafeInt32(i int64) (int32, error) {
	if i > math.MaxInt32 || i < math.MinInt32 {
		return 0, fmt.Errorf("integer overflow: value %d out of int32 range", i)
	}
	return int32(i), nil
}

// Remove all TODO comments and any outdated pseudocode or comments about unimplemented features.

func (s *Service) CreateCampaign(ctx context.Context, req *campaignpb.CreateCampaignRequest) (*campaignpb.CreateCampaignResponse, error) {
	log := s.log.With(
		zap.String("operation", "create_campaign"),
		zap.String("slug", req.Slug))

	log.Info("Creating campaign")

	// Input validation
	if req.Slug == "" {
		if s.eventEnabled && s.eventEmitter != nil {
			errMeta := &commonpb.Metadata{
				ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{"error": "slug is required"}, nil),
				Tags:            []string{},
				Features:        []string{},
			}
			errEmit := s.eventEmitter.EmitEvent(ctx, "campaign.create_failed", "", errMeta)
			if errEmit != nil {
				log.Warn("Failed to emit campaign.create_failed event", zap.Error(errEmit))
			}
		}
		return nil, status.Error(codes.InvalidArgument, "slug is required")
	}
	if req.Title == "" {
		if s.eventEnabled && s.eventEmitter != nil {
			errMeta := &commonpb.Metadata{
				ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{"error": "title is required"}, nil),
				Tags:            []string{},
				Features:        []string{},
			}
			errEmit := s.eventEmitter.EmitEvent(ctx, "campaign.create_failed", "", errMeta)
			if errEmit != nil {
				log.Warn("Failed to emit campaign.create_failed event", zap.Error(errEmit))
			}
		}
		return nil, status.Error(codes.InvalidArgument, "title is required")
	}
	// Parse and validate campaign metadata
	var campaignMeta *Metadata
	if req.Metadata != nil && req.Metadata.ServiceSpecific != nil {
		if campaignField, ok := req.Metadata.ServiceSpecific.Fields["campaign"]; ok {
			metaStruct := campaignField.GetStructValue()
			var err error
			campaignMeta, err = FromStruct(metaStruct)
			if err != nil {
				log.Error("Invalid campaign metadata", zap.Error(err))
				if s.eventEnabled && s.eventEmitter != nil {
					errMeta := &commonpb.Metadata{
						ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{"error": err.Error()}, nil),
						Tags:            []string{},
						Features:        []string{},
					}
					errEmit := s.eventEmitter.EmitEvent(ctx, "campaign.create_failed", "", errMeta)
					if errEmit != nil {
						log.Warn("Failed to emit campaign.create_failed event", zap.Error(errEmit))
					}
				}
				return nil, status.Errorf(codes.InvalidArgument, "invalid campaign metadata: %v", err)
			}
			if err := campaignMeta.Validate(); err != nil {
				log.Error("Missing required campaign metadata", zap.Error(err))
				if s.eventEnabled && s.eventEmitter != nil {
					errMeta := &commonpb.Metadata{
						ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{"error": err.Error()}, nil),
						Tags:            []string{},
						Features:        []string{},
					}
					errEmit := s.eventEmitter.EmitEvent(ctx, "campaign.create_failed", "", errMeta)
					if errEmit != nil {
						log.Warn("Failed to emit campaign.create_failed event", zap.Error(errEmit))
					}
				}
				return nil, status.Errorf(codes.InvalidArgument, "invalid campaign metadata: %v", err)
			}
		}
	}
	// Start transaction
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		log.Error("Failed to begin transaction", zap.Error(err))
		if s.eventEnabled && s.eventEmitter != nil {
			errMeta := &commonpb.Metadata{
				ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{"error": err.Error()}, nil),
				Tags:            []string{},
				Features:        []string{},
			}
			errEmit := s.eventEmitter.EmitEvent(ctx, "campaign.create_failed", "", errMeta)
			if errEmit != nil {
				log.Warn("Failed to emit campaign.create_failed event", zap.Error(errEmit))
			}
		}
		return nil, status.Error(codes.Internal, "failed to begin transaction")
	}
	defer func() {
		if rerr := tx.Rollback(); rerr != nil {
			s.log.Error("error rolling back transaction", zap.Error(rerr))
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
		if errors.Is(err, ErrCampaignExists) {
			log.Warn("Campaign already exists", zap.Error(err))
			if s.eventEnabled && s.eventEmitter != nil {
				errMeta := &commonpb.Metadata{
					ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{"error": err.Error()}, nil),
					Tags:            []string{},
					Features:        []string{},
				}
				errEmit := s.eventEmitter.EmitEvent(ctx, "campaign.create_failed", "", errMeta)
				if errEmit != nil {
					log.Warn("Failed to emit campaign.create_failed event", zap.Error(errEmit))
				}
			}
			return nil, status.Error(codes.AlreadyExists, "campaign already exists")
		}
		log.Error("Failed to create campaign", zap.Error(err))
		if s.eventEnabled && s.eventEmitter != nil {
			errMeta := &commonpb.Metadata{
				ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{"error": err.Error()}, nil),
				Tags:            []string{},
				Features:        []string{},
			}
			errEmit := s.eventEmitter.EmitEvent(ctx, "campaign.create_failed", "", errMeta)
			if errEmit != nil {
				log.Warn("Failed to emit campaign.create_failed event", zap.Error(errEmit))
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to create campaign: %v", err)
	}
	// Commit transaction
	if err := tx.Commit(); err != nil {
		log.Error("Failed to commit transaction", zap.Error(err))
		if s.eventEnabled && s.eventEmitter != nil {
			errMeta := &commonpb.Metadata{
				ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{"error": err.Error()}, nil),
				Tags:            []string{},
				Features:        []string{},
			}
			errEmit := s.eventEmitter.EmitEvent(ctx, "campaign.create_failed", "", errMeta)
			if errEmit != nil {
				log.Warn("Failed to emit campaign.create_failed event", zap.Error(errEmit))
			}
		}
		return nil, status.Error(codes.Internal, "failed to commit transaction")
	}
	id32, err := SafeInt32(created.ID)
	if err != nil {
		log.Error("Campaign ID overflow", zap.Error(err))
		if s.eventEnabled && s.eventEmitter != nil {
			errMeta := &commonpb.Metadata{
				ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{"error": err.Error()}, nil),
				Tags:            []string{},
				Features:        []string{},
			}
			errEmit := s.eventEmitter.EmitEvent(ctx, "campaign.create_failed", "", errMeta)
			if errEmit != nil {
				log.Warn("Failed to emit campaign.create_failed event", zap.Error(errEmit))
			}
		}
		return nil, status.Errorf(codes.Internal, "campaign ID overflow: %v", err)
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
	// Cache the new campaign
	if s.cache != nil && created.Metadata != nil {
		err := pattern.CacheMetadata(ctx, s.log, s.cache, "campaign", created.Slug, created.Metadata, 10*time.Minute)
		if err != nil {
			s.log.Error("failed to cache metadata", zap.Error(err))
		}
	}
	err = pattern.RegisterSchedule(ctx, s.log, "campaign", created.Slug, created.Metadata)
	if err != nil {
		s.log.Error("failed to register schedule", zap.Error(err))
	}
	err = pattern.EnrichKnowledgeGraph(ctx, s.log, "campaign", created.Slug, created.Metadata)
	if err != nil {
		s.log.Error("failed to enrich knowledge graph", zap.Error(err))
	}
	err = pattern.RegisterWithNexus(ctx, s.log, "campaign", created.Metadata)
	if err != nil {
		s.log.Error("failed to register with nexus", zap.Error(err))
	}
	log.Info("Campaign created successfully",
		zap.Int32("id", id32),
		zap.String("slug", created.Slug))
	if s.eventEnabled && s.eventEmitter != nil {
		successMeta := &commonpb.Metadata{ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{"campaign_id": id32}, nil)}
		_, ok := events.EmitCallbackEvent(ctx, s.eventEmitter, s.log, s.cache, "campaign", "created", fmt.Sprint(id32), successMeta, zap.Int32("campaign_id", id32))
		if !ok {
			s.log.Warn("Failed to emit workflow step event")
		}
	}
	created.Metadata, _ = events.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "campaign.created", created.Slug, created.Metadata)
	return &campaignpb.CreateCampaignResponse{Campaign: resp}, nil
}

func (s *Service) GetCampaign(ctx context.Context, req *campaignpb.GetCampaignRequest) (*campaignpb.GetCampaignResponse, error) {
	log := s.log.With(
		zap.String("operation", "get_campaign"),
		zap.String("slug", req.Slug))

	log.Info("Getting campaign")

	// Try to get from cache first
	var campaign campaignpb.Campaign
	if err := s.cache.Get(ctx, req.Slug, "campaign", &campaign); err == nil {
		return &campaignpb.GetCampaignResponse{Campaign: &campaign}, nil
	}

	// If not in cache, get from database
	c, err := s.repo.GetBySlug(ctx, req.Slug)
	if err != nil {
		if errors.Is(err, ErrCampaignNotFound) {
			log.Warn("Campaign not found", zap.Error(err))
			return nil, status.Error(codes.NotFound, "campaign not found")
		}
		log.Error("Failed to get campaign", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to get campaign: %v", err)
	}

	id32, err := SafeInt32(c.ID)
	if err != nil {
		log.Error("Campaign ID overflow", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "campaign ID overflow: %v", err)
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

	// Cache the campaign
	if s.cache != nil && c.Metadata != nil {
		err := pattern.CacheMetadata(ctx, s.log, s.cache, "campaign", c.Slug, c.Metadata, 10*time.Minute)
		if err != nil {
			s.log.Error("failed to cache metadata", zap.Error(err))
		}
	}
	err = pattern.RegisterSchedule(ctx, s.log, "campaign", c.Slug, c.Metadata)
	if err != nil {
		s.log.Error("failed to register schedule", zap.Error(err))
	}
	err = pattern.EnrichKnowledgeGraph(ctx, s.log, "campaign", c.Slug, c.Metadata)
	if err != nil {
		s.log.Error("failed to enrich knowledge graph", zap.Error(err))
	}
	err = pattern.RegisterWithNexus(ctx, s.log, "campaign", c.Metadata)
	if err != nil {
		s.log.Error("failed to register with nexus", zap.Error(err))
	}

	return &campaignpb.GetCampaignResponse{Campaign: resp}, nil
}

func (s *Service) UpdateCampaign(ctx context.Context, req *campaignpb.UpdateCampaignRequest) (*campaignpb.UpdateCampaignResponse, error) {
	authUserID, ok := utils.GetAuthenticatedUserID(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing authentication")
	}
	roles, _ := utils.GetAuthenticatedUserRoles(ctx)
	isAdmin := utils.IsServiceAdmin(roles, "campaign")
	existing, err := s.repo.GetBySlug(ctx, req.Campaign.Slug)
	if err != nil {
		if errors.Is(err, ErrCampaignNotFound) {
			s.log.Warn("Campaign not found", zap.Error(err))
			return nil, status.Error(codes.NotFound, "campaign not found")
		}
		s.log.Error("Failed to get existing campaign", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to get existing campaign: %v", err)
	}
	if !isAdmin && existing.OwnerID != authUserID {
		return nil, status.Error(codes.PermissionDenied, "cannot update campaign you do not own")
	}
	log := s.log.With(
		zap.String("operation", "update_campaign"),
		zap.String("slug", req.Campaign.Slug))

	log.Info("Updating campaign")

	// Check if campaign is being deactivated or window is ending
	wasActive := false
	if existing.Metadata != nil && existing.Metadata.ServiceSpecific != nil {
		if campaignField, ok := existing.Metadata.ServiceSpecific.Fields["campaign"]; ok {
			if campaignStruct := campaignField.GetStructValue(); campaignStruct != nil {
				if statusVal, ok := campaignStruct.Fields["status"]; ok {
					if statusVal.GetStringValue() == "active" {
						wasActive = true
					}
				}
			}
		}
	}

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
			return nil, status.Error(codes.NotFound, "campaign not found")
		}
		log.Error("Failed to update campaign", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to update campaign: %v", err)
	}

	// Invalidate cache
	if s.cache != nil && existing.Metadata != nil {
		err := pattern.CacheMetadata(ctx, s.log, s.cache, "campaign", existing.Slug, existing.Metadata, 10*time.Minute)
		if err != nil {
			s.log.Error("failed to cache metadata", zap.Error(err))
		}
	}
	err = pattern.RegisterSchedule(ctx, s.log, "campaign", existing.Slug, existing.Metadata)
	if err != nil {
		s.log.Error("failed to register schedule", zap.Error(err))
	}
	err = pattern.EnrichKnowledgeGraph(ctx, s.log, "campaign", existing.Slug, existing.Metadata)
	if err != nil {
		s.log.Error("failed to enrich knowledge graph", zap.Error(err))
	}
	err = pattern.RegisterWithNexus(ctx, s.log, "campaign", existing.Metadata)
	if err != nil {
		s.log.Error("failed to register with nexus", zap.Error(err))
	}

	// If campaign is now inactive or window ended, stop jobs and broadcasts
	isActive := false
	if existing.Metadata != nil && existing.Metadata.ServiceSpecific != nil {
		if campaignField, ok := existing.Metadata.ServiceSpecific.Fields["campaign"]; ok {
			if campaignStruct := campaignField.GetStructValue(); campaignStruct != nil {
				if statusVal, ok := campaignStruct.Fields["status"]; ok {
					if statusVal.GetStringValue() == "active" {
						isActive = true
					}
				}
			}
		}
	}
	if (!isActive && wasActive) || (!existing.EndDate.IsZero() && existing.EndDate.Before(time.Now())) {
		s.stopJobs(ctx, existing.Slug, existing)
		s.stopBroadcast(ctx, existing.Slug, existing)
	}

	// Get updated campaign
	getResp, err := s.GetCampaign(ctx, &campaignpb.GetCampaignRequest{Slug: existing.Slug})
	if err != nil {
		return nil, err
	}
	return &campaignpb.UpdateCampaignResponse{Campaign: getResp.Campaign}, nil
}

func (s *Service) DeleteCampaign(ctx context.Context, req *campaignpb.DeleteCampaignRequest) (*campaignpb.DeleteCampaignResponse, error) {
	authUserID, ok := utils.GetAuthenticatedUserID(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing authentication")
	}
	roles, _ := utils.GetAuthenticatedUserRoles(ctx)
	isAdmin := utils.IsServiceAdmin(roles, "campaign")
	campaign, err := s.repo.GetBySlug(ctx, req.Slug)
	if err != nil {
		if errors.Is(err, ErrCampaignNotFound) {
			s.log.Warn("Campaign not found", zap.Error(err))
			return nil, status.Error(codes.NotFound, "campaign not found")
		}
		s.log.Error("Failed to get campaign", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to get campaign: %v", err)
	}
	if !isAdmin && campaign.OwnerID != authUserID {
		return nil, status.Error(codes.PermissionDenied, "cannot delete campaign you do not own")
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
			return nil, status.Error(codes.NotFound, "campaign not found")
		}
		log.Error("Failed to delete campaign", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to delete campaign: %v", err)
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

	return &campaignpb.DeleteCampaignResponse{Success: true}, nil
}

func (s *Service) ListCampaigns(ctx context.Context, req *campaignpb.ListCampaignsRequest) (*campaignpb.ListCampaignsResponse, error) {
	log := s.log.With(
		zap.String("operation", "list_campaigns"),
		zap.Int32("page", req.Page),
		zap.Int32("page_size", req.PageSize))

	log.Info("Listing campaigns")

	// Input validation
	if req.Page < 0 {
		return nil, status.Error(codes.InvalidArgument, "page number cannot be negative")
	}
	if req.PageSize < 0 || req.PageSize > 100 {
		return nil, status.Error(codes.InvalidArgument, "page size must be between 0 and 100")
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
		return nil, status.Errorf(codes.Internal, "failed to list campaigns: %v", err)
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

	return resp, nil
}

// GetLeaderboard returns the leaderboard for a campaign, applying the dynamic ranking formula.
func (s *Service) GetLeaderboard(ctx context.Context, campaignSlug string, limit int) ([]LeaderboardEntry, error) {
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
	// Compile the formula once
	program, err := expr.Compile(formula, expr.Env(map[string]interface{}{}))
	if err != nil {
		return nil, fmt.Errorf("invalid ranking formula: %w", err)
	}
	for i := range entries {
		vars := entries[i].Variables // map[string]interface{} with user metrics
		output, err := expr.Run(program, vars)
		if err != nil {
			entries[i].Score = 0 // or handle error as needed
			continue
		}
		if score, ok := output.(float64); ok {
			entries[i].Score = score
		} else {
			entries[i].Score = 0
		}
	}
	// Sort entries by score descending
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Score > entries[j].Score
	})
	if limit > 0 && len(entries) > limit {
		entries = entries[:limit]
	}
	return entries, nil
}
