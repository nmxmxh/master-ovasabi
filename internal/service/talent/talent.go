package talent

import (
	"context"

	talentpb "github.com/nmxmxh/master-ovasabi/api/protos/talent/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

type Service struct {
	talentpb.UnimplementedTalentServiceServer
	log          *zap.Logger
	repo         *Repository
	Cache        *redis.Cache
	eventEmitter events.EventEmitter
	eventEnabled bool
	handler      *graceful.Handler // Canonical handler for orchestration
}

func NewService(ctx context.Context, log *zap.Logger, repo *Repository, cache *redis.Cache, eventEmitter events.EventEmitter, eventEnabled bool) talentpb.TalentServiceServer {
	// If graceful.NewHandler supports context, pass it; otherwise, keep as is
	handler := graceful.NewHandler(log, eventEmitter, cache, "talent", "v1", eventEnabled)
	s := &Service{
		log:          log,
		repo:         repo,
		Cache:        cache,
		eventEmitter: eventEmitter,
		eventEnabled: eventEnabled,
		handler:      handler,
	}
	return s
}

var _ talentpb.TalentServiceServer = (*Service)(nil)

func (s *Service) CreateTalentProfile(ctx context.Context, req *talentpb.CreateTalentProfileRequest) (*talentpb.CreateTalentProfileResponse, error) {
	if req == nil || req.Profile == nil {
		err := s.handler.Error(ctx, "create_talent_profile", codes.InvalidArgument, "missing profile data", nil, nil, "")
		return nil, graceful.ToStatusError(err)
	}
	authUserID, ok := utils.GetAuthenticatedUserID(ctx)
	if !ok {
		err := s.handler.Error(ctx, "create_talent_profile", codes.Unauthenticated, "missing authentication", nil, nil, "")
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
		err = s.handler.Error(ctx, "create_talent_profile", codes.Internal, "failed to create talent profile", err, nil, authUserID)
		return nil, graceful.ToStatusError(err)
	}
	s.handler.Success(ctx, "create_talent_profile", codes.OK, "talent profile created", created, created.Metadata, authUserID, nil)
	return &talentpb.CreateTalentProfileResponse{Profile: created, CampaignId: req.CampaignId}, nil
}

func (s *Service) UpdateTalentProfile(ctx context.Context, req *talentpb.UpdateTalentProfileRequest) (*talentpb.UpdateTalentProfileResponse, error) {
	authUserID, ok := utils.GetAuthenticatedUserID(ctx)
	if !ok {
		err := s.handler.Error(ctx, "update_talent_profile", codes.Unauthenticated, "missing authentication", nil, nil, "")
		return nil, graceful.ToStatusError(err)
	}
	roles, _ := utils.GetAuthenticatedUserRoles(ctx)
	isAdmin := utils.IsServiceAdmin(roles, "talent")
	profile, err := s.repo.GetTalentProfile(ctx, req.Profile.Id, req.CampaignId)
	if err != nil {
		err = s.handler.Error(ctx, "update_talent_profile", codes.NotFound, "talent profile not found", err, nil, authUserID)
		return nil, graceful.ToStatusError(err)
	}
	if !isAdmin && profile.UserId != authUserID {
		err := s.handler.Error(ctx, "update_talent_profile", codes.PermissionDenied, "cannot update profile you do not own", nil, nil, authUserID)
		return nil, graceful.ToStatusError(err)
	}
	if req == nil || req.Profile == nil {
		err := s.handler.Error(ctx, "update_talent_profile", codes.InvalidArgument, "profile is required", nil, nil, authUserID)
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
		err = s.handler.Error(ctx, "update_talent_profile", codes.Internal, "failed to update talent profile", err, nil, authUserID)
		return nil, graceful.ToStatusError(err)
	}
	s.handler.Success(ctx, "update_talent_profile", codes.OK, "talent profile updated", updated, updated.Metadata, authUserID, nil)
	return &talentpb.UpdateTalentProfileResponse{Profile: updated, CampaignId: req.CampaignId}, nil
}

func (s *Service) DeleteTalentProfile(ctx context.Context, req *talentpb.DeleteTalentProfileRequest) (*talentpb.DeleteTalentProfileResponse, error) {
	authUserID, ok := utils.GetAuthenticatedUserID(ctx)
	if !ok {
		err := s.handler.Error(ctx, "delete_talent_profile", codes.Unauthenticated, "missing authentication", nil, nil, "")
		return nil, graceful.ToStatusError(err)
	}
	roles, _ := utils.GetAuthenticatedUserRoles(ctx)
	isAdmin := utils.IsServiceAdmin(roles, "talent")
	profile, err := s.repo.GetTalentProfile(ctx, req.ProfileId, req.CampaignId)
	if err != nil {
		err = s.handler.Error(ctx, "delete_talent_profile", codes.NotFound, "talent profile not found", err, nil, authUserID)
		return nil, graceful.ToStatusError(err)
	}
	if !isAdmin && profile.UserId != authUserID {
		err := s.handler.Error(ctx, "delete_talent_profile", codes.PermissionDenied, "cannot delete profile you do not own", nil, nil, authUserID)
		return nil, graceful.ToStatusError(err)
	}
	if req == nil || req.ProfileId == "" {
		err := s.handler.Error(ctx, "delete_talent_profile", codes.InvalidArgument, "profile_id is required", nil, nil, authUserID)
		return nil, graceful.ToStatusError(err)
	}
	err = s.repo.DeleteTalentProfile(ctx, req.ProfileId, req.CampaignId)
	if err != nil {
		err = s.handler.Error(ctx, "delete_talent_profile", codes.Internal, "failed to delete talent profile", err, nil, authUserID)
		return nil, graceful.ToStatusError(err)
	}
	s.handler.Success(ctx, "delete_talent_profile", codes.OK, "talent profile deleted", req.ProfileId, profile.Metadata, authUserID, nil)
	return &talentpb.DeleteTalentProfileResponse{Success: true, CampaignId: req.CampaignId}, nil
}

func (s *Service) GetTalentProfile(ctx context.Context, req *talentpb.GetTalentProfileRequest) (*talentpb.GetTalentProfileResponse, error) {
	if req == nil || req.ProfileId == "" {
		err := s.handler.Error(ctx, "get_talent_profile", codes.InvalidArgument, "profile_id is required", nil, nil, "")
		return nil, graceful.ToStatusError(err)
	}
	profile, err := s.repo.GetTalentProfile(ctx, req.ProfileId, req.CampaignId)
	if err != nil {
		err = s.handler.Error(ctx, "get_talent_profile", codes.Internal, "failed to get talent profile", err, nil, "")
		return nil, graceful.ToStatusError(err)
	}
	if profile == nil {
		err := s.handler.Error(ctx, "get_talent_profile", codes.NotFound, "talent profile not found", nil, nil, "")
		return nil, graceful.ToStatusError(err)
	}
	s.handler.Success(ctx, "get_talent_profile", codes.OK, "talent profile fetched", profile, profile.Metadata, profile.UserId, nil)
	return &talentpb.GetTalentProfileResponse{Profile: profile, CampaignId: req.CampaignId}, nil
}

func (s *Service) ListTalentProfiles(ctx context.Context, req *talentpb.ListTalentProfilesRequest) (*talentpb.ListTalentProfilesResponse, error) {
	if req == nil {
		err := s.handler.Error(ctx, "list_talent_profiles", codes.InvalidArgument, "request is required", nil, nil, "")
		return nil, graceful.ToStatusError(err)
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
		err = s.handler.Error(ctx, "list_talent_profiles", codes.Internal, "failed to list talent profiles", err, nil, "")
		return nil, graceful.ToStatusError(err)
	}
	totalPages := utils.ToInt32((total + pageSize - 1) / pageSize)
	s.handler.Success(ctx, "list_talent_profiles", codes.OK, "talent profiles listed", profiles, nil, "", nil)
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
		err := s.handler.Error(ctx, "search_talent_profiles", codes.InvalidArgument, "request is required", nil, nil, "")
		return nil, graceful.ToStatusError(err)
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
		err = s.handler.Error(ctx, "search_talent_profiles", codes.Internal, "failed to search talent profiles", err, nil, "")
		return nil, graceful.ToStatusError(err)
	}
	totalPages := utils.ToInt32((total + pageSize - 1) / pageSize)
	s.handler.Success(ctx, "search_talent_profiles", codes.OK, "talent profiles searched", profiles, nil, "", nil)
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
		err := s.handler.Error(ctx, "book_talent", codes.InvalidArgument, "talent_id and user_id are required", nil, nil, req.UserId)
		return nil, graceful.ToStatusError(err)
	}
	booking, err := s.repo.BookTalent(ctx, req.TalentId, req.UserId, req.StartTime, req.EndTime, req.Notes, req.CampaignId)
	if err != nil {
		err = s.handler.Error(ctx, "book_talent", codes.Internal, "failed to book talent", err, nil, req.UserId)
		return nil, graceful.ToStatusError(err)
	}
	s.handler.Success(ctx, "book_talent", codes.OK, "talent booked", booking, booking.Metadata, req.UserId, nil)
	return &talentpb.BookTalentResponse{Booking: booking, CampaignId: req.CampaignId}, nil
}

func (s *Service) ListBookings(ctx context.Context, req *talentpb.ListBookingsRequest) (*talentpb.ListBookingsResponse, error) {
	if req == nil || req.UserId == "" {
		err := s.handler.Error(ctx, "list_bookings", codes.InvalidArgument, "user_id is required", nil, nil, req.UserId)
		return nil, graceful.ToStatusError(err)
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
		err = s.handler.Error(ctx, "list_bookings", codes.Internal, "failed to list bookings", err, nil, req.UserId)
		return nil, graceful.ToStatusError(err)
	}
	totalPages := utils.ToInt32((total + pageSize - 1) / pageSize)
	s.handler.Success(ctx, "list_bookings", codes.OK, "bookings listed", bookings, nil, req.UserId, nil)
	return &talentpb.ListBookingsResponse{
		Bookings:   bookings,
		TotalCount: utils.ToInt32(total),
		Page:       req.Page,
		TotalPages: totalPages,
		CampaignId: req.CampaignId,
	}, nil
}
