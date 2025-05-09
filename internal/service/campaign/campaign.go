package campaign

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"time"

	campaignpb "github.com/nmxmxh/master-ovasabi/api/protos/campaign/v1"
	"github.com/nmxmxh/master-ovasabi/internal/repository"
	campaignrepo "github.com/nmxmxh/master-ovasabi/internal/repository/campaign"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	ErrCampaignNotFound = errors.New("campaign not found")
	ErrCampaignExists   = errors.New("campaign already exists")
)

// Compile-time check
var _ campaignpb.CampaignServiceServer = (*Service)(nil)

// Service implements the CampaignService gRPC interface
type Service struct {
	campaignpb.UnimplementedCampaignServiceServer
	log        *zap.Logger
	db         *sql.DB
	masterRepo repository.MasterRepository
	repo       *campaignrepo.Repository
	cache      *redis.Cache
}

func NewService(db *sql.DB, log *zap.Logger) *Service {
	return &Service{
		db:         db,
		log:        log,
		masterRepo: repository.NewMasterRepository(db, log),
	}
}

// SafeInt32 converts an int64 to int32 with overflow checking
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
		return nil, status.Error(codes.InvalidArgument, "slug is required")
	}
	if req.Title == "" {
		return nil, status.Error(codes.InvalidArgument, "title is required")
	}

	// Start transaction
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		log.Error("Failed to begin transaction", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to begin transaction")
	}
	defer func() {
		if rerr := tx.Rollback(); rerr != nil {
			s.log.Error("error rolling back transaction", zap.Error(rerr))
		}
	}()

	// Create campaign with transaction
	c := &campaignrepo.Campaign{
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
		if errors.Is(err, campaignrepo.ErrCampaignExists) {
			log.Warn("Campaign already exists", zap.Error(err))
			return nil, status.Error(codes.AlreadyExists, "campaign already exists")
		}
		log.Error("Failed to create campaign", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to create campaign: %v", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		log.Error("Failed to commit transaction", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to commit transaction")
	}

	id32, err := SafeInt32(created.ID)
	if err != nil {
		log.Error("Campaign ID overflow", zap.Error(err))
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
	if err := s.cache.Set(ctx, created.Slug, "campaign", resp, 15*time.Minute); err != nil {
		log.Error("Failed to cache campaign",
			zap.String("slug", created.Slug),
			zap.Error(err))
		// Don't fail creation if caching fails
	}

	log.Info("Campaign created successfully",
		zap.Int32("id", id32),
		zap.String("slug", created.Slug))

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
		if errors.Is(err, campaignrepo.ErrCampaignNotFound) {
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
	if err := s.cache.Set(ctx, c.Slug, "campaign", resp, 15*time.Minute); err != nil {
		log.Error("Failed to cache campaign",
			zap.String("slug", c.Slug),
			zap.Error(err))
		// Don't fail the get if caching fails
	}

	return &campaignpb.GetCampaignResponse{Campaign: resp}, nil
}

func (s *Service) UpdateCampaign(ctx context.Context, req *campaignpb.UpdateCampaignRequest) (*campaignpb.UpdateCampaignResponse, error) {
	log := s.log.With(
		zap.String("operation", "update_campaign"),
		zap.String("slug", req.Campaign.Slug))

	log.Info("Updating campaign")

	// Get existing campaign first
	existing, err := s.repo.GetBySlug(ctx, req.Campaign.Slug)
	if err != nil {
		if errors.Is(err, campaignrepo.ErrCampaignNotFound) {
			log.Warn("Campaign not found", zap.Error(err))
			return nil, status.Error(codes.NotFound, "campaign not found")
		}
		log.Error("Failed to get existing campaign", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to get existing campaign: %v", err)
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
		if errors.Is(err, campaignrepo.ErrCampaignNotFound) {
			log.Warn("Campaign not found during update", zap.Error(err))
			return nil, status.Error(codes.NotFound, "campaign not found")
		}
		log.Error("Failed to update campaign", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to update campaign: %v", err)
	}

	// Invalidate cache
	if err := s.cache.Delete(ctx, existing.Slug, "campaign"); err != nil {
		log.Error("Failed to invalidate campaign cache",
			zap.String("slug", existing.Slug),
			zap.Error(err))
		// Don't fail the update if cache invalidation fails
	}

	// Get updated campaign
	getResp, err := s.GetCampaign(ctx, &campaignpb.GetCampaignRequest{Slug: existing.Slug})
	if err != nil {
		return nil, err
	}
	return &campaignpb.UpdateCampaignResponse{Campaign: getResp.Campaign}, nil
}

func (s *Service) DeleteCampaign(ctx context.Context, req *campaignpb.DeleteCampaignRequest) (*campaignpb.DeleteCampaignResponse, error) {
	log := s.log.With(
		zap.String("operation", "delete_campaign"),
		zap.Int32("id", req.Id))

	log.Info("Deleting campaign")

	// Get campaign first to get the slug for cache invalidation
	campaign, err := s.repo.GetBySlug(ctx, req.Slug)
	if err != nil && !errors.Is(err, campaignrepo.ErrCampaignNotFound) {
		log.Error("Failed to get campaign for cache invalidation", zap.Error(err))
		// Continue with deletion even if we can't get the campaign
	}

	if err := s.repo.Delete(ctx, int64(req.Id)); err != nil {
		if errors.Is(err, campaignrepo.ErrCampaignNotFound) {
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

// GetLeaderboard returns the leaderboard for a campaign, applying the ranking formula.
func (s *Service) GetLeaderboard(ctx context.Context, campaignSlug string, limit int) ([]campaignrepo.LeaderboardEntry, error) {
	campaign, err := s.repo.GetBySlug(ctx, campaignSlug)
	if err != nil {
		return nil, err
	}
	return s.repo.GetLeaderboard(ctx, campaignSlug, campaign.RankingFormula, limit)
}
