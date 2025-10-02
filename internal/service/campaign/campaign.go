package campaign

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/expr-lang/expr"
	campaignpb "github.com/nmxmxh/master-ovasabi/api/protos/campaign/v1"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
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

// Adapter to bridge event emission to the required orchestration EventEmitter interface.
// EventEmitterAdapter is obsolete. Use graceful event emission and orchestration handlers directly.

// Service implements the CampaignService gRPC interface.
type Service struct {
	campaignpb.UnimplementedCampaignServiceServer
	log              *zap.Logger
	repo             *Repository
	cache            *redis.Cache
	eventEmitter     events.EventEmitter
	eventEnabled     bool
	activeBroadcasts map[string]context.CancelFunc
	scheduledJobs    map[string][]cron.EntryID
	provider         *service.Provider // Add provider field
	// Graceful handler for orchestration, audit, and event emission
	handler *graceful.Handler
}

func NewService(log *zap.Logger, repo *Repository, cache *redis.Cache, eventEmitter events.EventEmitter, eventEnabled bool, provider *service.Provider) *Service {
	return &Service{
		log:              log,
		repo:             repo,
		cache:            cache,
		eventEmitter:     eventEmitter,
		eventEnabled:     eventEnabled,
		activeBroadcasts: make(map[string]context.CancelFunc),
		scheduledJobs:    make(map[string][]cron.EntryID),
		provider:         provider, // Assign provider
		handler:          graceful.NewHandler(log, eventEmitter, cache, "campaign", "v1", eventEnabled),
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

	authUserID, ok := utils.GetAuthenticatedUserID(ctx)
	if !ok {
		authUserID = "system" // Fallback to a system user for internal operations
	}

	log.Info("Creating campaign")

	// Input validation
	if req.Slug == "" {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "slug is required", nil)
		s.handler.Error(ctx, "create_campaign", codes.InvalidArgument, "slug is required", err, nil, "")
		return nil, graceful.ToStatusError(err)
	}
	if req.Title == "" {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "title is required", nil)
		s.handler.Error(ctx, "create_campaign", codes.InvalidArgument, "title is required", err, nil, "")
		return nil, graceful.ToStatusError(err)
	}
	// Start transaction
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		gErr := graceful.MapAndWrapErr(ctx, err, "failed to begin transaction", codes.Internal)
		s.handler.Error(ctx, "create_campaign", codes.Internal, "failed to begin transaction", gErr, nil, "")
		return nil, graceful.ToStatusError(gErr)
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
		Status:         "active", // Default to active status
		Metadata:       req.Metadata,
		OwnerID:        authUserID,
	}
	if req.StartDate != nil {
		c.StartDate = req.StartDate.AsTime()
	}
	if req.EndDate != nil {
		c.EndDate = req.EndDate.AsTime()
	}
	created, err := s.repo.CreateWithTransaction(ctx, tx, c)
	if err != nil {
		gErr := graceful.MapAndWrapErr(ctx, err, "failed to create campaign", codes.Internal)
		s.handler.Error(ctx, "create_campaign", codes.Internal, "failed to create campaign", gErr, nil, "")
		return nil, graceful.ToStatusError(gErr)
	}
	// Commit transaction
	if err := tx.Commit(); err != nil {
		gErr := graceful.MapAndWrapErr(ctx, err, "failed to commit transaction", codes.Internal)
		s.handler.Error(ctx, "create_campaign", codes.Internal, "failed to commit transaction", gErr, nil, "")
		return nil, graceful.ToStatusError(gErr)
	}
	// Convert string ID to int32 for response
	id32, err := strconv.ParseInt(created.ID, 10, 32)
	if err != nil {
		// Fallback: use hash of ID as number
		hash := fnv.New32a()
		hash.Write([]byte(created.ID))
		id32 = int64(hash.Sum32())
	}
	resp := &campaignpb.Campaign{
		Id:             int32(id32),
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
	// Orchestrate all post-success actions via graceful handler
	s.handler.Success(ctx, "create_campaign", codes.OK, "campaign created", resp, created.Metadata, created.Slug, nil)
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
			// Use MapAndWrapErr to ensure consistent error handling even inside the cache function.
			return nil, graceful.MapAndWrapErr(ctx, err, "failed to get campaign by slug", codes.Internal)
		}
		// Convert string ID to int32 for response
		id32, err := strconv.ParseInt(c.ID, 10, 32)
		if err != nil {
			// Fallback: use hash of ID as number
			hash := fnv.New32a()
			hash.Write([]byte(c.ID))
			id32 = int64(hash.Sum32())
		}
		resp := &campaignpb.Campaign{
			Id:             int32(id32),
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
	if err != nil {
		var gErr *graceful.ContextError
		if !errors.As(err, &gErr) {
			// If the error is not already a graceful error, wrap it.
			gErr = graceful.MapAndWrapErr(ctx, err, "failed to get campaign", codes.Internal)
		}
		s.handler.Error(ctx, "get_campaign", codes.Internal, "failed to get campaign", gErr, nil, req.Slug)
		return nil, graceful.ToStatusError(gErr)
	}

	resp := &campaignpb.GetCampaignResponse{Campaign: campaign}
	s.handler.Success(ctx, "get_campaign", codes.OK, "campaign fetched", resp, campaign.Metadata, req.Slug, nil)
	return resp, nil
}

func (s *Service) UpdateCampaign(ctx context.Context, req *campaignpb.UpdateCampaignRequest) (*campaignpb.UpdateCampaignResponse, error) {
	authUserID, ok := utils.GetAuthenticatedUserID(ctx)
	if !ok {
		authUserID = "system" // Fallback to a system user for internal operations
	}
	roles, _ := utils.GetAuthenticatedUserRoles(ctx)
	isPlatformAdmin := utils.IsServiceAdmin(roles, "campaign")
	existing, err := s.repo.GetBySlug(ctx, req.Campaign.Slug)
	if err != nil {
		gErr := graceful.MapAndWrapErr(ctx, err, "failed to get existing campaign", codes.Internal)
		s.handler.Error(ctx, "update_campaign", codes.Internal, "failed to get existing campaign", gErr, nil, req.Campaign.Slug)
		return nil, graceful.ToStatusError(gErr)
	}
	// --- Permission check: campaign membership/role ---
	role := GetUserRoleInCampaign(existing.Metadata, authUserID, existing.OwnerID)
	isSystem := IsSystemCampaign(existing.Metadata)
	if role != "admin" && role != "user" && (!isSystem || !isPlatformAdmin) {
		err := graceful.WrapErr(ctx, codes.PermissionDenied, "insufficient campaign role", nil)
		s.handler.Error(ctx, "update_campaign", codes.PermissionDenied, "insufficient campaign role", err, nil, req.Campaign.Slug)
		return nil, graceful.ToStatusError(err)
	}
	log := s.log.With(
		zap.String("operation", "update_campaign"),
		zap.String("slug", req.Campaign.Slug))

	log.Info("Updating campaign")

	// Update fields
	existing.Title = req.Campaign.Title
	existing.Description = req.Campaign.Description
	existing.RankingFormula = req.Campaign.RankingFormula
	existing.Status = "active" // Keep status as active
	existing.Metadata = req.Campaign.Metadata
	if req.Campaign.StartDate != nil {
		existing.StartDate = req.Campaign.StartDate.AsTime()
	}
	if req.Campaign.EndDate != nil {
		existing.EndDate = req.Campaign.EndDate.AsTime()
	}

	// Update in database
	if err := s.repo.Update(ctx, existing); err != nil {
		gErr := graceful.MapAndWrapErr(ctx, err, "failed to update campaign", codes.Internal)
		s.handler.Error(ctx, "update_campaign", codes.Internal, "failed to update campaign", gErr, nil, req.Campaign.Slug)
		return nil, graceful.ToStatusError(gErr)
	}
	// If campaign is now inactive or window ended, stop jobs and broadcasts

	// Manually construct response to avoid nested orchestration from calling GetCampaign.
	// Convert string ID to int32 for response
	id32, err := strconv.ParseInt(existing.ID, 10, 32)
	if err != nil {
		// Fallback: use hash of ID as number
		hash := fnv.New32a()
		hash.Write([]byte(existing.ID))
		id32 = int64(hash.Sum32())
	}
	campaignResp := &campaignpb.Campaign{
		Id:             int32(id32),
		Slug:           existing.Slug,
		Title:          existing.Title,
		Description:    existing.Description,
		RankingFormula: existing.RankingFormula,
		Metadata:       existing.Metadata,
		CreatedAt:      timestamppb.New(existing.CreatedAt),
		UpdatedAt:      timestamppb.New(existing.UpdatedAt),
	}
	if !existing.StartDate.IsZero() {
		campaignResp.StartDate = timestamppb.New(existing.StartDate)
	}
	if !existing.EndDate.IsZero() {
		campaignResp.EndDate = timestamppb.New(existing.EndDate)
	}

	resp := &campaignpb.UpdateCampaignResponse{Campaign: campaignResp}
	s.handler.Success(ctx, "update_campaign", codes.OK, "campaign updated", resp, campaignResp.Metadata, existing.Slug, nil)
	return resp, nil
}

func (s *Service) DeleteCampaign(ctx context.Context, req *campaignpb.DeleteCampaignRequest) (*campaignpb.DeleteCampaignResponse, error) {
	authUserID, ok := utils.GetAuthenticatedUserID(ctx)
	if !ok {
		authUserID = "system" // Fallback to a system user for internal operations
	}
	roles, _ := utils.GetAuthenticatedUserRoles(ctx)
	isPlatformAdmin := utils.IsServiceAdmin(roles, "campaign")
	campaign, err := s.repo.GetBySlug(ctx, req.Slug)
	if err != nil {
		gErr := graceful.MapAndWrapErr(ctx, err, "failed to get campaign for deletion", codes.Internal)
		s.handler.Error(ctx, "delete_campaign", codes.Internal, "failed to get campaign for deletion", gErr, nil, req.Slug)
		return nil, graceful.ToStatusError(gErr)
	}
	// --- Permission check: campaign membership/role ---
	role := GetUserRoleInCampaign(campaign.Metadata, authUserID, campaign.OwnerID)
	isSystem := IsSystemCampaign(campaign.Metadata)
	if role != "admin" && role != "user" && (!isSystem || !isPlatformAdmin) {
		err := graceful.WrapErr(ctx, codes.PermissionDenied, "insufficient campaign role", nil)
		s.handler.Error(ctx, "delete_campaign", codes.PermissionDenied, "insufficient campaign role", err, nil, req.Slug)
		return nil, graceful.ToStatusError(err)
	}
	log := s.log.With(
		zap.String("operation", "delete_campaign"),
		zap.Int32("id", req.Id))

	log.Info("Deleting campaign")

	if err := s.repo.Delete(ctx, int64(req.Id)); err != nil {
		gErr := graceful.MapAndWrapErr(ctx, err, "failed to delete campaign", codes.Internal)
		s.handler.Error(ctx, "delete_campaign", codes.Internal, "failed to delete campaign", gErr, nil, req.Slug)
		return nil, graceful.ToStatusError(gErr)
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
	s.handler.Success(ctx, "delete_campaign", codes.OK, "campaign deleted", resp, campaign.Metadata, campaign.Slug, nil)
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
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "page number cannot be negative", nil)
		s.handler.Error(ctx, "list_campaigns", codes.InvalidArgument, "page number cannot be negative", err, nil, "")
		return nil, graceful.ToStatusError(err)
	}
	if req.PageSize < 0 || req.PageSize > 100 {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "page size must be between 0 and 100", nil)
		s.handler.Error(ctx, "list_campaigns", codes.InvalidArgument, "page size must be between 0 and 100", err, nil, "")
		return nil, graceful.ToStatusError(err)
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
		gErr := graceful.MapAndWrapErr(ctx, err, "failed to list campaigns", codes.Internal)
		s.handler.Error(ctx, "list_campaigns", codes.Internal, "failed to list campaigns", gErr, nil, "")
		return nil, graceful.ToStatusError(gErr)
	}

	resp := &campaignpb.ListCampaignsResponse{
		Campaigns: make([]*campaignpb.Campaign, 0, len(campaigns)),
	}

	for _, c := range campaigns {
		// Convert string ID to int32 for response
		id32, err := strconv.ParseInt(c.ID, 10, 32)
		if err != nil {
			// Fallback: use hash of ID as number
			hash := fnv.New32a()
			hash.Write([]byte(c.ID))
			id32 = int64(hash.Sum32())
		}

		campaign := &campaignpb.Campaign{
			Id:             int32(id32),
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

	s.handler.Success(ctx, "list_campaigns", codes.OK, "campaigns listed", resp, nil, "", nil)
	return resp, nil
}

// GetLeaderboard returns the leaderboard for a campaign, applying the dynamic ranking formula.
func (s *Service) GetLeaderboard(ctx context.Context, campaignSlug string, limit int) ([]LeaderboardEntry, error) {
	cacheKey := fmt.Sprintf("leaderboard:%s:%d", campaignSlug, limit)
	entries, err := redis.GetOrSetWithProtection(ctx, s.cache, s.log, cacheKey, func(ctx context.Context) ([]LeaderboardEntry, error) {
		_, err := s.repo.GetBySlug(ctx, campaignSlug)
		if err != nil {
			return nil, err
		}
		formula := "" // RankingFormula no longer used, set to empty
		if formula == "" {
			return nil, fmt.Errorf("no ranking formula defined for campaign")
		}
		entries, err := s.repo.GetLeaderboard(ctx, campaignSlug, limit)
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
