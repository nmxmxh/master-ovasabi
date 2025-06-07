package talent

import (
	"context"
	"errors"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	talentpb "github.com/nmxmxh/master-ovasabi/api/protos/talent/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// EventEmitter defines the interface for emitting events (canonical platform interface).
type EventEmitter interface {
	EmitEventWithLogging(ctx context.Context, emitter interface{}, log *zap.Logger, eventType, eventID string, meta *commonpb.Metadata) (string, bool)
	EmitRawEventWithLogging(ctx context.Context, log *zap.Logger, eventType, eventID string, payload []byte) (string, bool)
}

type Service struct {
	talentpb.UnimplementedTalentServiceServer
	log          *zap.Logger
	repo         *Repository
	Cache        *redis.Cache
	eventEmitter EventEmitter
	eventEnabled bool
}

func NewService(_ context.Context, log *zap.Logger, repo *Repository, cache *redis.Cache, eventEmitter EventEmitter, eventEnabled bool) talentpb.TalentServiceServer {
	s := &Service{
		log:          log,
		repo:         repo,
		Cache:        cache,
		eventEmitter: eventEmitter,
		eventEnabled: eventEnabled,
	}
	return s
}

var _ talentpb.TalentServiceServer = (*Service)(nil)

func (s *Service) CreateTalentProfile(ctx context.Context, req *talentpb.CreateTalentProfileRequest) (*talentpb.CreateTalentProfileResponse, error) {
	if req == nil || req.Profile == nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "missing profile data", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	authUserID, ok := utils.GetAuthenticatedUserID(ctx)
	if !ok {
		err := graceful.WrapErr(ctx, codes.Unauthenticated, "missing authentication", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	req.Profile.UserId = authUserID
	metaStruct := &Metadata{
		Skills:         req.Profile.Skills,
		Languages:      nil,
		Diversity:      nil,
		Certifications: nil,
		Industry:       "",
		Accessibility:  nil,
		Compliance:     nil,
		Audit:          nil,
		Versioning:     nil,
		Custom:         nil,
		Gamified:       nil,
	}
	metaMap := map[string]interface{}{
		"skills":         metaStruct.Skills,
		"languages":      metaStruct.Languages,
		"diversity":      metaStruct.Diversity,
		"certifications": metaStruct.Certifications,
		"industry":       metaStruct.Industry,
		"accessibility":  metaStruct.Accessibility,
		"compliance":     metaStruct.Compliance,
		"audit":          metaStruct.Audit,
		"versioning":     metaStruct.Versioning,
		"custom":         metaStruct.Custom,
		"gamified":       metaStruct.Gamified,
	}
	fullMap := map[string]interface{}{"service_specific": map[string]interface{}{"talent": metaMap}}
	normMeta := metadata.MapToProto(fullMap)
	req.Profile.Metadata = normMeta
	created, err := s.repo.CreateTalentProfile(ctx, req.Profile, req.CampaignId)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to create talent profile", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "talent profile created", created, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Cache: s.Cache, CacheKey: created.Id, CacheValue: created, CacheTTL: 10 * time.Minute, Metadata: created.Metadata, EventEmitter: s.eventEmitter, EventEnabled: s.eventEnabled, EventType: "talent.profile_created", EventID: created.Id, PatternType: "talent_profile", PatternID: created.Id, PatternMeta: created.Metadata})
	return &talentpb.CreateTalentProfileResponse{Profile: created, CampaignId: req.CampaignId}, nil
}

func (s *Service) UpdateTalentProfile(ctx context.Context, req *talentpb.UpdateTalentProfileRequest) (*talentpb.UpdateTalentProfileResponse, error) {
	authUserID, ok := utils.GetAuthenticatedUserID(ctx)
	if !ok {
		err := graceful.WrapErr(ctx, codes.Unauthenticated, "missing authentication", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	roles, _ := utils.GetAuthenticatedUserRoles(ctx)
	isAdmin := utils.IsServiceAdmin(roles, "talent")
	profile, err := s.repo.GetTalentProfile(ctx, req.Profile.Id, req.CampaignId)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.NotFound, "talent profile not found", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	if !isAdmin && profile.UserId != authUserID {
		err := graceful.WrapErr(ctx, codes.PermissionDenied, "cannot update profile you do not own", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	if req == nil || req.Profile == nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "profile is required", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	metaStruct := &Metadata{
		Skills:         req.Profile.Skills,
		Languages:      nil,
		Diversity:      nil,
		Certifications: nil,
		Industry:       "",
		Accessibility:  nil,
		Compliance:     nil,
		Audit:          nil,
		Versioning:     nil,
		Custom:         nil,
		Gamified:       nil,
	}
	metaMap := map[string]interface{}{
		"skills":         metaStruct.Skills,
		"languages":      metaStruct.Languages,
		"diversity":      metaStruct.Diversity,
		"certifications": metaStruct.Certifications,
		"industry":       metaStruct.Industry,
		"accessibility":  metaStruct.Accessibility,
		"compliance":     metaStruct.Compliance,
		"audit":          metaStruct.Audit,
		"versioning":     metaStruct.Versioning,
		"custom":         metaStruct.Custom,
		"gamified":       metaStruct.Gamified,
	}
	fullMap := map[string]interface{}{"service_specific": map[string]interface{}{"talent": metaMap}}
	normMeta := metadata.MapToProto(fullMap)
	req.Profile.Metadata = normMeta
	updated, err := s.repo.UpdateTalentProfile(ctx, req.Profile, req.CampaignId)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to update talent profile", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "talent profile updated", updated, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Cache: s.Cache, CacheKey: updated.Id, CacheValue: updated, CacheTTL: 10 * time.Minute, Metadata: updated.Metadata, EventEmitter: s.eventEmitter, EventEnabled: s.eventEnabled, EventType: "talent.profile_updated", EventID: updated.Id, PatternType: "talent_profile", PatternID: updated.Id, PatternMeta: updated.Metadata})
	return &talentpb.UpdateTalentProfileResponse{Profile: updated, CampaignId: req.CampaignId}, nil
}

func (s *Service) DeleteTalentProfile(ctx context.Context, req *talentpb.DeleteTalentProfileRequest) (*talentpb.DeleteTalentProfileResponse, error) {
	authUserID, ok := utils.GetAuthenticatedUserID(ctx)
	if !ok {
		err := graceful.WrapErr(ctx, codes.Unauthenticated, "missing authentication", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	roles, _ := utils.GetAuthenticatedUserRoles(ctx)
	isAdmin := utils.IsServiceAdmin(roles, "talent")
	profile, err := s.repo.GetTalentProfile(ctx, req.ProfileId, req.CampaignId)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.NotFound, "talent profile not found", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	if !isAdmin && profile.UserId != authUserID {
		err := graceful.WrapErr(ctx, codes.PermissionDenied, "cannot delete profile you do not own", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	if req == nil || req.ProfileId == "" {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "profile_id is required", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	err = s.repo.DeleteTalentProfile(ctx, req.ProfileId, req.CampaignId)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to delete talent profile", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "talent profile deleted", req.ProfileId, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Metadata: profile.Metadata, EventEmitter: s.eventEmitter, EventEnabled: s.eventEnabled, EventType: "talent.profile_deleted", EventID: req.ProfileId, PatternType: "talent_profile", PatternID: req.ProfileId, PatternMeta: profile.Metadata})
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
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "talent_id and user_id are required", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	booking, err := s.repo.BookTalent(ctx, req.TalentId, req.UserId, req.StartTime, req.EndTime, req.Notes, req.CampaignId)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to book talent", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "talent booked", booking, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Metadata: booking.Metadata, EventEmitter: s.eventEmitter, EventEnabled: s.eventEnabled, EventType: "talent.booked", EventID: req.TalentId, PatternType: "talent", PatternID: req.TalentId, PatternMeta: booking.Metadata})
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
