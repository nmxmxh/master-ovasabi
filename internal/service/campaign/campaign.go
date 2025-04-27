package campaign

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"

	campaignpb "github.com/nmxmxh/master-ovasabi/api/protos/campaign/v0"
	"github.com/nmxmxh/master-ovasabi/internal/repository"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// DB interface for database operations (for mocking/testing).
type DB interface {
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

type Service struct {
	campaignpb.UnimplementedCampaignServiceServer
	log  *zap.Logger
	db   DB
	repo *repository.CampaignRepository
}

func NewCampaignService(log *zap.Logger, db DB) campaignpb.CampaignServiceServer {
	repo := repository.NewCampaignRepository(db)
	return &Service{
		log:  log,
		db:   db,
		repo: repo,
	}
}

// SafeInt32 converts an int to int32 with overflow checking.
func SafeInt32(i int) (int32, error) {
	if i > math.MaxInt32 || i < math.MinInt32 {
		return 0, fmt.Errorf("integer overflow: value %d out of int32 range", i)
	}
	return int32(i), nil
}

func (s *Service) CreateCampaign(ctx context.Context, req *campaignpb.CreateCampaignRequest) (*campaignpb.CreateCampaignResponse, error) {
	s.log.Info("Creating campaign", zap.String("slug", req.Slug))

	// For demo, use master_id = 1 (should be set properly in production)
	c := &repository.Campaign{
		MasterID:       1,
		Slug:           req.Slug,
		Title:          req.Title,
		Description:    req.Description,
		RankingFormula: req.RankingFormula,
		Metadata:       req.Metadata,
	}
	if req.StartDate != nil {
		t := req.StartDate.AsTime()
		c.StartDate = &t
	}
	if req.EndDate != nil {
		t := req.EndDate.AsTime()
		c.EndDate = &t
	}

	created, err := s.repo.Create(ctx, c)
	if err != nil {
		if err.Error() == "pq: duplicate key value violates unique constraint \"service_campaign_slug_key\"" {
			return nil, status.Error(codes.AlreadyExists, "campaign already exists")
		}
		return nil, status.Errorf(codes.Internal, "failed to create campaign: %v", err)
	}

	id32, err := SafeInt32(created.ID)
	if err != nil {
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
	if created.StartDate != nil {
		resp.StartDate = timestamppb.New(*created.StartDate)
	}
	if created.EndDate != nil {
		resp.EndDate = timestamppb.New(*created.EndDate)
	}

	return &campaignpb.CreateCampaignResponse{Campaign: resp}, nil
}

func (s *Service) GetCampaign(ctx context.Context, req *campaignpb.GetCampaignRequest) (*campaignpb.GetCampaignResponse, error) {
	s.log.Info("Getting campaign", zap.String("slug", req.Slug))
	c, err := s.repo.GetBySlug(ctx, req.Slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "campaign not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get campaign: %v", err)
	}

	id32, err := SafeInt32(c.ID)
	if err != nil {
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
	if c.StartDate != nil {
		resp.StartDate = timestamppb.New(*c.StartDate)
	}
	if c.EndDate != nil {
		resp.EndDate = timestamppb.New(*c.EndDate)
	}
	return &campaignpb.GetCampaignResponse{Campaign: resp}, nil
}
