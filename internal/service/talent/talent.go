package talentservice

import (
	"context"
	"errors"
	"time"

	talentpb "github.com/nmxmxh/master-ovasabi/api/protos/talent/v1"
	pattern "github.com/nmxmxh/master-ovasabi/internal/service/pattern"
	metadatautil "github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Repository interface {
	CreateTalentProfile(ctx context.Context, profile *talentpb.TalentProfile) (*talentpb.TalentProfile, error)
	UpdateTalentProfile(ctx context.Context, profile *talentpb.TalentProfile) (*talentpb.TalentProfile, error)
	DeleteTalentProfile(ctx context.Context, profileID string) error
	GetTalentProfile(ctx context.Context, profileID string) (*talentpb.TalentProfile, error)
	ListTalentProfiles(ctx context.Context, page, pageSize int, skills, tags []string, location string) ([]*talentpb.TalentProfile, int, error)
	SearchTalentProfiles(ctx context.Context, query string, page, pageSize int, skills, tags []string, location string) ([]*talentpb.TalentProfile, int, error)
	BookTalent(ctx context.Context, talentID, userID string, startTime, endTime int64, notes string) (*talentpb.Booking, error)
	ListBookings(ctx context.Context, userID string, page, pageSize int) ([]*talentpb.Booking, int, error)
}

type Service struct {
	talentpb.UnimplementedTalentServiceServer
	log   *zap.Logger
	repo  Repository
	Cache *redis.Cache
}

func NewTalentService(log *zap.Logger, repo Repository, cache *redis.Cache) talentpb.TalentServiceServer {
	s := &Service{
		log:   log,
		repo:  repo,
		Cache: cache,
	}
	if err := pattern.RegisterWithNexus(context.Background(), log, "talent", nil); err != nil {
		log.Error("RegisterWithNexus failed in NewTalentService (talent)", zap.Error(err))
	}
	return s
}

var _ talentpb.TalentServiceServer = (*Service)(nil)

func (s *Service) CreateTalentProfile(ctx context.Context, req *talentpb.CreateTalentProfileRequest) (*talentpb.CreateTalentProfileResponse, error) {
	s.log.Info("CreateTalentProfile called")
	if err := metadatautil.ValidateMetadata(req.Profile.Metadata); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid metadata: %v", err)
	}
	profile, err := s.repo.CreateTalentProfile(ctx, req.Profile)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create profile: %v", err)
	}
	if s.Cache != nil && profile.Metadata != nil {
		if err := pattern.CacheMetadata(ctx, s.log, s.Cache, "talent_profile", profile.Id, profile.Metadata, 10*time.Minute); err != nil {
			s.log.Error("failed to cache metadata", zap.Error(err))
		}
	}
	if err := pattern.RegisterSchedule(ctx, s.log, "talent_profile", profile.Id, profile.Metadata); err != nil {
		s.log.Error("failed to register schedule", zap.Error(err))
	}
	if err := pattern.EnrichKnowledgeGraph(ctx, s.log, "talent_profile", profile.Id, profile.Metadata); err != nil {
		s.log.Error("failed to enrich knowledge graph", zap.Error(err))
	}
	if err := pattern.RegisterWithNexus(ctx, s.log, "talent_profile", profile.Metadata); err != nil {
		s.log.Error("failed to register with nexus", zap.Error(err))
	}
	return &talentpb.CreateTalentProfileResponse{Profile: profile}, nil
}

func (s *Service) UpdateTalentProfile(ctx context.Context, req *talentpb.UpdateTalentProfileRequest) (*talentpb.UpdateTalentProfileResponse, error) {
	s.log.Info("UpdateTalentProfile called")
	if err := metadatautil.ValidateMetadata(req.Profile.Metadata); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid metadata: %v", err)
	}
	profile, err := s.repo.UpdateTalentProfile(ctx, req.Profile)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update profile: %v", err)
	}
	if s.Cache != nil && profile.Metadata != nil {
		if err := pattern.CacheMetadata(ctx, s.log, s.Cache, "talent_profile", profile.Id, profile.Metadata, 10*time.Minute); err != nil {
			s.log.Error("failed to cache metadata", zap.Error(err))
		}
	}
	if err := pattern.RegisterSchedule(ctx, s.log, "talent_profile", profile.Id, profile.Metadata); err != nil {
		s.log.Error("failed to register schedule", zap.Error(err))
	}
	if err := pattern.EnrichKnowledgeGraph(ctx, s.log, "talent_profile", profile.Id, profile.Metadata); err != nil {
		s.log.Error("failed to enrich knowledge graph", zap.Error(err))
	}
	if err := pattern.RegisterWithNexus(ctx, s.log, "talent_profile", profile.Metadata); err != nil {
		s.log.Error("failed to register with nexus", zap.Error(err))
	}
	return &talentpb.UpdateTalentProfileResponse{Profile: profile}, nil
}

func (s *Service) DeleteTalentProfile(_ context.Context, _ *talentpb.DeleteTalentProfileRequest) (*talentpb.DeleteTalentProfileResponse, error) {
	s.log.Info("DeleteTalentProfile called")
	// TODO (Amadeus Context): Implement DeleteTalentProfile following the canonical metadata pattern.
	// Reference: docs/amadeus/amadeus_context.md, section 'Canonical Metadata Integration Pattern (System-Wide)'.
	// Steps: Delete profile, update orchestration if metadata present, log errors.
	return nil, errors.New("not implemented")
}

func (s *Service) GetTalentProfile(_ context.Context, _ *talentpb.GetTalentProfileRequest) (*talentpb.GetTalentProfileResponse, error) {
	s.log.Info("GetTalentProfile called")
	// TODO (Amadeus Context): Implement GetTalentProfile following the canonical metadata pattern.
	// Reference: docs/amadeus/amadeus_context.md, section 'Canonical Metadata Integration Pattern (System-Wide)'.
	// Steps: Get profile, include metadata, log errors.
	return nil, errors.New("not implemented")
}

func (s *Service) ListTalentProfiles(_ context.Context, _ *talentpb.ListTalentProfilesRequest) (*talentpb.ListTalentProfilesResponse, error) {
	s.log.Info("ListTalentProfiles called")
	// TODO (Amadeus Context): Implement ListTalentProfiles following the canonical metadata pattern.
	// Reference: docs/amadeus/amadeus_context.md, section 'Canonical Metadata Integration Pattern (System-Wide)'.
	// Steps: List profiles, include metadata, log errors.
	return nil, errors.New("not implemented")
}

func (s *Service) SearchTalentProfiles(_ context.Context, _ *talentpb.SearchTalentProfilesRequest) (*talentpb.SearchTalentProfilesResponse, error) {
	s.log.Info("SearchTalentProfiles called")
	// TODO (Amadeus Context): Implement SearchTalentProfiles following the canonical metadata pattern.
	// Reference: docs/amadeus/amadeus_context.md, section 'Canonical Metadata Integration Pattern (System-Wide)'.
	// Steps: Search profiles, include metadata, log errors.
	return nil, errors.New("not implemented")
}

func (s *Service) BookTalent(_ context.Context, _ *talentpb.BookTalentRequest) (*talentpb.BookTalentResponse, error) {
	s.log.Warn("BookTalent not implemented in TalentService")
	return nil, status.Errorf(codes.Unimplemented, "BookTalent integration not yet implemented; see Amadeus context for pattern")
	// If implemented:
	// if s.Cache != nil && booking.Metadata != nil {
	// 	_ = pattern.CacheMetadata(ctx, s.Cache, "talent_booking", booking.Id, booking.Metadata, 10*time.Minute)
	// }
	// _ = pattern.RegisterSchedule(ctx, "talent_booking", booking.Id, booking.Metadata)
	// _ = pattern.EnrichKnowledgeGraph(ctx, "talent_booking", booking.Id, booking.Metadata)
	// _ = pattern.RegisterWithNexus(ctx, "talent_booking", booking.Metadata)
	// return &talentpb.BookTalentResponse{Booking: booking}, nil
}

func (s *Service) ListBookings(_ context.Context, _ *talentpb.ListBookingsRequest) (*talentpb.ListBookingsResponse, error) {
	s.log.Info("ListBookings called")
	// TODO (Amadeus Context): Implement ListBookings following the canonical metadata pattern.
	// Reference: docs/amadeus/amadeus_context.md, section 'Canonical Metadata Integration Pattern (System-Wide)'.
	// Steps: List bookings, include metadata, log errors.
	return nil, errors.New("not implemented")
}
