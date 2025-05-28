package talent

import (
	"context"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	talentpb "github.com/nmxmxh/master-ovasabi/api/protos/talent/v1"
	nexusevents "github.com/nmxmxh/master-ovasabi/internal/service/nexus"
	pattern "github.com/nmxmxh/master-ovasabi/internal/service/pattern"
	events "github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

// EventEmitter defines the interface for emitting events (canonical platform interface).
type EventEmitter interface {
	EmitEvent(ctx context.Context, eventType, entityID string, metadata *commonpb.Metadata) error
}

type Service struct {
	talentpb.UnimplementedTalentServiceServer
	log          *zap.Logger
	repo         *Repository
	Cache        *redis.Cache
	eventEmitter EventEmitter
	eventEnabled bool
}

func NewService(ctx context.Context, log *zap.Logger, repo *Repository, cache *redis.Cache, eventEmitter EventEmitter, eventEnabled bool) talentpb.TalentServiceServer {
	s := &Service{
		log:          log,
		repo:         repo,
		Cache:        cache,
		eventEmitter: eventEmitter,
		eventEnabled: eventEnabled,
	}
	if err := pattern.RegisterWithNexus(ctx, log, "talent", nil); err != nil {
		log.Error("RegisterWithNexus failed in NewTalentService (talent)", zap.Error(err))
	}
	return s
}

var _ talentpb.TalentServiceServer = (*Service)(nil)

func (s *Service) CreateTalentProfile(ctx context.Context, req *talentpb.CreateTalentProfileRequest) (*talentpb.CreateTalentProfileResponse, error) {
	if req == nil || req.Profile == nil {
		return nil, status.Error(codes.InvalidArgument, "missing profile data")
	}
	authUserID, ok := utils.GetAuthenticatedUserID(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing authentication")
	}
	req.Profile.UserId = authUserID
	meta, err := ExtractAndEnrichTalentMetadata(req.Profile.Metadata, authUserID, true)
	if err != nil {
		if s.eventEnabled && s.eventEmitter != nil {
			errMeta := &commonpb.Metadata{
				ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{"error": err.Error(), "talent_id": req.Profile.Id},
					func() *structpb.Struct {
						if req.Profile != nil && req.Profile.Metadata != nil {
							return req.Profile.Metadata.ServiceSpecific
						}
						return nil
					}()),
				Tags:     []string{},
				Features: []string{},
			}
			_, errEmit := events.EmitCallbackEvent(ctx, s.eventEmitter, s.log, s.Cache, "talent", nexusevents.EventTalentProfileCreateFailed, "", errMeta)
			if !errEmit {
				s.log.Warn("Failed to emit talent.profile_create_failed event")
			}
		}
		return nil, status.Errorf(codes.InvalidArgument, "invalid metadata: %v", err)
	}
	req.Profile.Metadata = meta
	created, err := s.repo.CreateTalentProfile(ctx, req.Profile, req.CampaignId)
	if err != nil {
		if s.eventEnabled && s.eventEmitter != nil {
			errMeta := &commonpb.Metadata{
				ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{"error": err.Error(), "talent_id": req.Profile.Id},
					func() *structpb.Struct {
						if req.Profile != nil && req.Profile.Metadata != nil {
							return req.Profile.Metadata.ServiceSpecific
						}
						return nil
					}()),
				Tags:     []string{},
				Features: []string{},
			}
			_, errEmit := events.EmitCallbackEvent(ctx, s.eventEmitter, s.log, s.Cache, "talent", nexusevents.EventTalentProfileCreateFailed, "", errMeta)
			if !errEmit {
				s.log.Warn("Failed to emit talent.profile_create_failed event")
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to create talent profile: %v", err)
	}
	if s.Cache != nil && created.Metadata != nil {
		if err := pattern.CacheMetadata(ctx, s.log, s.Cache, "talent_profile", created.Id, created.Metadata, 10*time.Minute); err != nil {
			s.log.Error("failed to cache talent profile metadata", zap.Error(err))
		}
	}
	if err := pattern.RegisterSchedule(ctx, s.log, "talent_profile", created.Id, created.Metadata); err != nil {
		s.log.Error("failed to register schedule", zap.Error(err))
	}
	if err := pattern.EnrichKnowledgeGraph(ctx, s.log, "talent_profile", created.Id, created.Metadata); err != nil {
		s.log.Error("failed to enrich knowledge graph", zap.Error(err))
	}
	if err := pattern.RegisterWithNexus(ctx, s.log, "talent_profile", created.Metadata); err != nil {
		s.log.Error("failed to register with nexus", zap.Error(err))
	}
	created.Metadata, _ = events.EmitEventWithLogging(ctx, s.eventEmitter, s.log, nexusevents.EventTalentProfileCreated, created.Id, created.Metadata)
	return &talentpb.CreateTalentProfileResponse{Profile: created, CampaignId: req.CampaignId}, nil
}

func (s *Service) UpdateTalentProfile(ctx context.Context, req *talentpb.UpdateTalentProfileRequest) (*talentpb.UpdateTalentProfileResponse, error) {
	authUserID, ok := utils.GetAuthenticatedUserID(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing authentication")
	}
	roles, _ := utils.GetAuthenticatedUserRoles(ctx)
	isAdmin := utils.IsServiceAdmin(roles, "talent")
	profile, err := s.repo.GetTalentProfile(ctx, req.Profile.Id, req.CampaignId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "talent profile not found")
	}
	if !isAdmin && profile.UserId != authUserID {
		return nil, status.Error(codes.PermissionDenied, "cannot update profile you do not own")
	}
	if req == nil || req.Profile == nil {
		return nil, status.Error(codes.InvalidArgument, "profile is required")
	}
	meta, err := ExtractAndEnrichTalentMetadata(req.Profile.Metadata, authUserID, false)
	if err != nil {
		if s.eventEnabled && s.eventEmitter != nil {
			errMeta := &commonpb.Metadata{
				ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{"error": err.Error(), "talent_id": req.Profile.Id},
					func() *structpb.Struct {
						if req.Profile != nil && req.Profile.Metadata != nil {
							return req.Profile.Metadata.ServiceSpecific
						}
						return nil
					}()),
				Tags:     []string{},
				Features: []string{},
			}
			_, errEmit := events.EmitCallbackEvent(ctx, s.eventEmitter, s.log, s.Cache, "talent", nexusevents.EventTalentProfileUpdateFailed, req.Profile.Id, errMeta)
			if !errEmit {
				s.log.Warn("Failed to emit talent.profile_update_failed event")
			}
		}
		return nil, status.Errorf(codes.InvalidArgument, "invalid metadata: %v", err)
	}
	req.Profile.Metadata = meta
	updated, err := s.repo.UpdateTalentProfile(ctx, req.Profile, req.CampaignId)
	if err != nil {
		if s.eventEnabled && s.eventEmitter != nil {
			errMeta := &commonpb.Metadata{
				ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{"error": err.Error(), "talent_id": req.Profile.Id},
					func() *structpb.Struct {
						if req.Profile != nil && req.Profile.Metadata != nil {
							return req.Profile.Metadata.ServiceSpecific
						}
						return nil
					}()),
				Tags:     []string{},
				Features: []string{},
			}
			_, errEmit := events.EmitCallbackEvent(ctx, s.eventEmitter, s.log, s.Cache, "talent", nexusevents.EventTalentProfileUpdateFailed, req.Profile.Id, errMeta)
			if !errEmit {
				s.log.Warn("Failed to emit talent.profile_update_failed event")
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to update talent profile: %v", err)
	}
	if s.Cache != nil && updated.Metadata != nil {
		if err := pattern.CacheMetadata(ctx, s.log, s.Cache, "talent_profile", updated.Id, updated.Metadata, 10*time.Minute); err != nil {
			s.log.Error("failed to cache talent profile metadata", zap.Error(err))
		}
	}
	if err := pattern.RegisterSchedule(ctx, s.log, "talent_profile", updated.Id, updated.Metadata); err != nil {
		s.log.Error("failed to register schedule", zap.Error(err))
	}
	if err := pattern.EnrichKnowledgeGraph(ctx, s.log, "talent_profile", updated.Id, updated.Metadata); err != nil {
		s.log.Error("failed to enrich knowledge graph", zap.Error(err))
	}
	if err := pattern.RegisterWithNexus(ctx, s.log, "talent_profile", updated.Metadata); err != nil {
		s.log.Error("failed to register with nexus", zap.Error(err))
	}
	updated.Metadata, _ = events.EmitEventWithLogging(ctx, s.eventEmitter, s.log, nexusevents.EventTalentProfileUpdated, updated.Id, updated.Metadata)
	return &talentpb.UpdateTalentProfileResponse{Profile: updated, CampaignId: req.CampaignId}, nil
}

func (s *Service) DeleteTalentProfile(ctx context.Context, req *talentpb.DeleteTalentProfileRequest) (*talentpb.DeleteTalentProfileResponse, error) {
	authUserID, ok := utils.GetAuthenticatedUserID(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing authentication")
	}
	roles, _ := utils.GetAuthenticatedUserRoles(ctx)
	isAdmin := utils.IsServiceAdmin(roles, "talent")
	profile, err := s.repo.GetTalentProfile(ctx, req.ProfileId, req.CampaignId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "talent profile not found")
	}
	if !isAdmin && profile.UserId != authUserID {
		return nil, status.Error(codes.PermissionDenied, "cannot delete profile you do not own")
	}
	if req == nil || req.ProfileId == "" {
		return nil, status.Error(codes.InvalidArgument, "profile_id is required")
	}
	err = s.repo.DeleteTalentProfile(ctx, req.ProfileId, req.CampaignId)
	if err != nil {
		if s.eventEnabled && s.eventEmitter != nil {
			errMeta := &commonpb.Metadata{
				ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{"error": err.Error(), "talent_id": req.ProfileId},
					func() *structpb.Struct {
						if profile != nil && profile.Metadata != nil {
							return profile.Metadata.ServiceSpecific
						}
						return nil
					}()),
				Tags:     []string{},
				Features: []string{},
			}
			_, errEmit := events.EmitCallbackEvent(ctx, s.eventEmitter, s.log, s.Cache, "talent", nexusevents.EventTalentProfileDeleteFailed, req.ProfileId, errMeta)
			if !errEmit {
				s.log.Warn("Failed to emit talent.profile_delete_failed event")
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to delete talent profile: %v", err)
	}
	if s.eventEnabled && s.eventEmitter != nil {
		_, errEmit := events.EmitCallbackEvent(ctx, s.eventEmitter, s.log, s.Cache, "talent", nexusevents.EventTalentProfileDeleted, req.ProfileId, nil)
		if !errEmit {
			s.log.Warn("Failed to emit talent.profile_deleted event")
		}
	}
	return &talentpb.DeleteTalentProfileResponse{Success: true, CampaignId: req.CampaignId}, nil
}

func (s *Service) GetTalentProfile(ctx context.Context, req *talentpb.GetTalentProfileRequest) (*talentpb.GetTalentProfileResponse, error) {
	if req == nil || req.ProfileId == "" {
		return nil, status.Error(codes.InvalidArgument, "profile_id is required")
	}
	profile, err := s.repo.GetTalentProfile(ctx, req.ProfileId, req.CampaignId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get talent profile: %v", err)
	}
	if profile == nil {
		return nil, status.Error(codes.NotFound, "talent profile not found")
	}
	return &talentpb.GetTalentProfileResponse{Profile: profile, CampaignId: req.CampaignId}, nil
}

func (s *Service) ListTalentProfiles(ctx context.Context, req *talentpb.ListTalentProfilesRequest) (*talentpb.ListTalentProfilesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	page := int(req.Page)
	if page < 1 {
		page = 1
	}
	pageSize := int(req.PageSize)
	if pageSize < 1 {
		pageSize = 20
	}
	skills := req.Skills
	tags := req.Tags
	location := req.Location
	profiles, total, err := s.repo.ListTalentProfiles(ctx, page, pageSize, skills, tags, location, req.CampaignId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list talent profiles: %v", err)
	}
	totalPages := utils.ToInt32((total + pageSize - 1) / pageSize)
	return &talentpb.ListTalentProfilesResponse{
		Profiles:   profiles,
		TotalCount: utils.ToInt32(total),
		Page:       req.Page,
		TotalPages: totalPages,
		CampaignId: req.CampaignId,
	}, nil
}

func (s *Service) SearchTalentProfiles(ctx context.Context, req *talentpb.SearchTalentProfilesRequest) (*talentpb.SearchTalentProfilesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	page := int(req.Page)
	if page < 1 {
		page = 1
	}
	pageSize := int(req.PageSize)
	if pageSize < 1 {
		pageSize = 20
	}
	skills := req.Skills
	tags := req.Tags
	location := req.Location
	profiles, total, err := s.repo.SearchTalentProfiles(ctx, req.Query, page, pageSize, skills, tags, location, req.CampaignId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to search talent profiles: %v", err)
	}
	totalPages := utils.ToInt32((total + pageSize - 1) / pageSize)
	return &talentpb.SearchTalentProfilesResponse{
		Profiles:   profiles,
		TotalCount: utils.ToInt32(total),
		Page:       req.Page,
		TotalPages: totalPages,
		CampaignId: req.CampaignId,
	}, nil
}

func (s *Service) BookTalent(ctx context.Context, req *talentpb.BookTalentRequest) (*talentpb.BookTalentResponse, error) {
	if req == nil || req.TalentId == "" || req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "talent_id and user_id are required")
	}
	booking, err := s.repo.BookTalent(ctx, req.TalentId, req.UserId, req.StartTime, req.EndTime, req.Notes, req.CampaignId)
	if err != nil {
		if s.eventEnabled && s.eventEmitter != nil {
			errMeta := &commonpb.Metadata{
				ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{"error": err.Error(), "talent_id": req.TalentId},
					func() *structpb.Struct {
						if booking != nil && booking.Metadata != nil {
							return booking.Metadata.ServiceSpecific
						}
						return nil
					}()),
				Tags:     []string{},
				Features: []string{},
			}
			_, errEmit := events.EmitCallbackEvent(ctx, s.eventEmitter, s.log, s.Cache, "talent", nexusevents.EventTalentBookingFailed, req.TalentId, errMeta)
			if !errEmit {
				s.log.Warn("Failed to emit talent.booking_failed event")
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to book talent: %v", err)
	}
	if s.eventEnabled && s.eventEmitter != nil {
		_, errEmit := events.EmitCallbackEvent(ctx, s.eventEmitter, s.log, s.Cache, "talent", nexusevents.EventTalentBooked, req.TalentId, booking.Metadata)
		if !errEmit {
			s.log.Warn("Failed to emit talent.booked event")
		}
	}
	return &talentpb.BookTalentResponse{Booking: booking, CampaignId: req.CampaignId}, nil
}

func (s *Service) ListBookings(ctx context.Context, req *talentpb.ListBookingsRequest) (*talentpb.ListBookingsResponse, error) {
	if req == nil || req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	page := int(req.Page)
	if page < 1 {
		page = 1
	}
	pageSize := int(req.PageSize)
	if pageSize < 1 {
		pageSize = 20
	}
	bookings, total, err := s.repo.ListBookings(ctx, req.UserId, page, pageSize, req.CampaignId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list bookings: %v", err)
	}
	totalPages := utils.ToInt32((total + pageSize - 1) / pageSize)
	return &talentpb.ListBookingsResponse{
		Bookings:   bookings,
		TotalCount: utils.ToInt32(total),
		Page:       req.Page,
		TotalPages: totalPages,
		CampaignId: req.CampaignId,
	}, nil
}
