package waitlist

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	waitlistpb "github.com/nmxmxh/master-ovasabi/api/protos/waitlist/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Service implements the waitlist gRPC service
type Service struct {
	waitlistpb.UnimplementedWaitlistServiceServer
	log          *zap.Logger
	repo         Repository
	cache        *redis.Cache
	eventEmitter events.EventEmitter
	eventEnabled bool
	handler      *graceful.Handler
}

// NewService creates a new waitlist service instance
func NewService(
	log *zap.Logger,
	repo Repository,
	cache *redis.Cache,
	eventEmitter events.EventEmitter,
	eventEnabled bool,
) *Service {
	return &Service{
		log:          log,
		repo:         repo,
		cache:        cache,
		eventEmitter: eventEmitter,
		eventEnabled: eventEnabled,
		handler:      graceful.NewHandler(log, eventEmitter, cache, "waitlist", "entry", eventEnabled),
	}
}

// CreateWaitlistEntry creates a new waitlist entry
func (s *Service) CreateWaitlistEntry(ctx context.Context, req *waitlistpb.CreateWaitlistEntryRequest) (*waitlistpb.CreateWaitlistEntryResponse, error) {
	// Validate required fields
	if req.Email == "" {
		err := graceful.LogAndWrap(ctx, s.log, codes.InvalidArgument, "email is required", ErrEmailRequired)
		return nil, graceful.ToStatusError(err)
	}
	if req.FirstName == "" {
		err := graceful.LogAndWrap(ctx, s.log, codes.InvalidArgument, "first name is required", ErrFirstNameRequired)
		return nil, graceful.ToStatusError(err)
	}
	if req.LastName == "" {
		err := graceful.LogAndWrap(ctx, s.log, codes.InvalidArgument, "last name is required", ErrLastNameRequired)
		return nil, graceful.ToStatusError(err)
	}
	if req.Tier == "" {
		err := graceful.LogAndWrap(ctx, s.log, codes.InvalidArgument, "tier is required", ErrTierRequired)
		return nil, graceful.ToStatusError(err)
	}
	if req.Intention == "" {
		err := graceful.LogAndWrap(ctx, s.log, codes.InvalidArgument, "intention is required", ErrIntentionRequired)
		return nil, graceful.ToStatusError(err)
	}

	// Validate email format
	if err := ValidateEmail(req.Email); err != nil {
		ctxErr := graceful.LogAndWrap(ctx, s.log, codes.InvalidArgument, "invalid email format", err,
			zap.String("email", req.Email))
		return nil, graceful.ToStatusError(ctxErr)
	}

	// Check if email already exists
	exists, err := s.repo.EmailExists(ctx, req.Email)
	if err != nil {
		ctxErr := graceful.LogAndWrap(ctx, s.log, codes.Internal, "failed to check email existence", err,
			zap.String("email", req.Email))
		return nil, graceful.ToStatusError(ctxErr)
	}
	if exists {
		err := graceful.LogAndWrap(ctx, s.log, codes.AlreadyExists, "email already exists", ErrEmailAlreadyExists,
			zap.String("email", req.Email))
		return nil, graceful.ToStatusError(err)
	}

	// Validate username if provided
	if req.ReservedUsername != nil && *req.ReservedUsername != "" {
		if err := ValidateUsername(*req.ReservedUsername); err != nil {
			ctxErr := graceful.LogAndWrap(ctx, s.log, codes.InvalidArgument, "invalid username format", err,
				zap.String("username", *req.ReservedUsername))
			return nil, graceful.ToStatusError(ctxErr)
		}

		// Check if username is already taken
		exists, err := s.repo.UsernameExists(ctx, *req.ReservedUsername)
		if err != nil {
			ctxErr := graceful.LogAndWrap(ctx, s.log, codes.Internal, "failed to check username existence", err,
				zap.String("username", *req.ReservedUsername))
			return nil, graceful.ToStatusError(ctxErr)
		}
		if exists {
			err := graceful.LogAndWrap(ctx, s.log, codes.AlreadyExists, "username already taken", ErrUsernameAlreadyTaken,
				zap.String("username", *req.ReservedUsername))
			return nil, graceful.ToStatusError(err)
		}
	}

	// Validate referral username if provided
	if req.ReferralUsername != nil && *req.ReferralUsername != "" {
		valid, err := s.repo.ValidateReferralUsername(ctx, *req.ReferralUsername)
		if err != nil {
			ctxErr := graceful.LogAndWrap(ctx, s.log, codes.Internal, "failed to validate referral username", err,
				zap.String("referral_username", *req.ReferralUsername))
			return nil, graceful.ToStatusError(ctxErr)
		}
		if !valid {
			err := graceful.LogAndWrap(ctx, s.log, codes.InvalidArgument, "invalid referral username", ErrReferralUserNotFound,
				zap.String("referral_username", *req.ReferralUsername))
			return nil, graceful.ToStatusError(err)
		}
	}

	// Create waitlist entry
	entry := &waitlistpb.WaitlistEntry{
		Uuid:                 uuid.New().String(),
		Email:                req.Email,
		FirstName:            req.FirstName,
		LastName:             req.LastName,
		Tier:                 req.Tier,
		ReservedUsername:     req.ReservedUsername,
		Intention:            req.Intention,
		Interests:            req.Interests,
		ReferralUsername:     req.ReferralUsername,
		ReferralCode:         req.ReferralCode,
		Feedback:             req.Feedback,
		AdditionalComments:   req.AdditionalComments,
		Status:               "pending",
		PriorityScore:        0,
		CampaignName:         req.CampaignName,
		LocationCountry:      req.LocationCountry,
		LocationRegion:       req.LocationRegion,
		LocationCity:         req.LocationCity,
		LocationLat:          req.LocationLat,
		LocationLng:          req.LocationLng,
		IpAddress:            req.IpAddress,
		UserAgent:            req.UserAgent,
		ReferrerUrl:          req.ReferrerUrl,
		UtmSource:            req.UtmSource,
		UtmMedium:            req.UtmMedium,
		UtmCampaign:          req.UtmCampaign,
		UtmTerm:              req.UtmTerm,
		UtmContent:           req.UtmContent,
		QuestionnaireAnswers: req.QuestionnaireAnswers,
		ContactPreferences:   req.ContactPreferences,
		Metadata:             req.Metadata,
		CreatedAt:            timestamppb.Now(),
		UpdatedAt:            timestamppb.Now(),
	}

	// Calculate priority score based on tier and other factors
	entry.PriorityScore = s.calculatePriorityScore(entry)

	// Save to database
	createdEntry, err := s.repo.Create(ctx, entry)
	if err != nil {
		ctxErr := graceful.LogAndWrap(ctx, s.log, codes.Internal, "failed to create waitlist entry", err,
			zap.String("email", req.Email))
		return nil, graceful.ToStatusError(ctxErr)
	}

	// Create referral record if referral username is provided
	if req.ReferralUsername != nil && *req.ReferralUsername != "" {
		if err := s.repo.CreateReferralRecord(ctx, *req.ReferralUsername, createdEntry.Id); err != nil {
			// Log but don't fail the whole operation for referral record creation failure
			graceful.LogAndWrap(ctx, s.log, codes.Internal, "failed to create referral record", err,
				zap.String("referral_username", *req.ReferralUsername),
				zap.Int64("referred_id", createdEntry.Id))
		}
	}

	// Clear cache
	s.clearCache(ctx, "waitlist_stats", "leaderboard")

	// Success orchestration (graceful handler)
	process := map[string]interface{}{
		"service":  "waitlist",
		"action":   "create",
		"entry_id": createdEntry.Id,
	}
	successData := map[string]interface{}{
		"entry_id":     createdEntry.Id,
		"entry_uuid":   createdEntry.Uuid,
		"email":        createdEntry.Email,
		"tier":         createdEntry.Tier,
		"campaign":     createdEntry.CampaignName,
		"has_referral": req.ReferralUsername != nil && *req.ReferralUsername != "",
	}
	s.handler.Success(
		ctx,
		"create_entry", // action
		codes.OK,
		"Successfully created waitlist entry",
		successData,
		process,
		"waitlist.entry.created",
		nil, // cache info
	)
	return &waitlistpb.CreateWaitlistEntryResponse{
		Entry: createdEntry,
	}, nil
}

// GetWaitlistEntry retrieves a waitlist entry by ID, UUID, or email
func (s *Service) GetWaitlistEntry(ctx context.Context, req *waitlistpb.GetWaitlistEntryRequest) (*waitlistpb.GetWaitlistEntryResponse, error) {
	var entry *waitlistpb.WaitlistEntry
	var err error

	switch identifier := req.Identifier.(type) {
	case *waitlistpb.GetWaitlistEntryRequest_Id:
		entry, err = s.repo.GetByID(ctx, identifier.Id)
	case *waitlistpb.GetWaitlistEntryRequest_Uuid:
		entry, err = s.repo.GetByUUID(ctx, identifier.Uuid)
	case *waitlistpb.GetWaitlistEntryRequest_Email:
		entry, err = s.repo.GetByEmail(ctx, identifier.Email)
	default:
		ctxErr := graceful.LogAndWrap(ctx, s.log, codes.InvalidArgument, "id, uuid, or email is required", nil)
		return nil, graceful.ToStatusError(ctxErr)
	}

	if err != nil {
		if err == ErrWaitlistEntryNotFound {
			ctxErr := graceful.LogAndWrap(ctx, s.log, codes.NotFound, "waitlist entry not found", err)
			return nil, graceful.ToStatusError(ctxErr)
		}
		ctxErr := graceful.LogAndWrap(ctx, s.log, codes.Internal, "failed to get waitlist entry", err)
		return nil, graceful.ToStatusError(ctxErr)
	}

	// Success orchestration
	process := map[string]interface{}{
		"service":  "waitlist",
		"action":   "get_entry",
		"entry_id": entry.Id,
	}

	resultData := map[string]interface{}{
		"entry_id":   entry.Id,
		"entry_uuid": entry.Uuid,
		"email":      entry.Email,
	}

	successCtx := graceful.LogAndWrapSuccess(ctx, s.log, codes.OK,
		"Successfully retrieved waitlist entry", resultData, process,
		zap.Int64("entry_id", entry.Id),
		zap.String("entry_uuid", entry.Uuid))

	// Run success orchestration
	if s.eventEmitter != nil {
		successCtx.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
			Log:          s.log,
			Cache:        s.cache,
			EventEmitter: s.eventEmitter,
			EventEnabled: s.eventEnabled,
			EventType:    "waitlist.entry.retrieved",
			EventID:      entry.Uuid,
			PatternType:  "waitlist",
			PatternID:    fmt.Sprintf("entry-%d", entry.Id),
			Metadata:     entry.Metadata,
			Tags:         []string{"waitlist", "get", "entry"},
		})
	}

	return &waitlistpb.GetWaitlistEntryResponse{
		Entry: entry,
	}, nil
}

// UpdateWaitlistEntry updates an existing waitlist entry
func (s *Service) UpdateWaitlistEntry(ctx context.Context, req *waitlistpb.UpdateWaitlistEntryRequest) (*waitlistpb.UpdateWaitlistEntryResponse, error) {
	if req.Id == 0 {
		ctxErr := graceful.LogAndWrap(ctx, s.log, codes.InvalidArgument, "id is required", nil,
			zap.Int64("id", req.Id))
		return nil, graceful.ToStatusError(ctxErr)
	}

	// Get existing entry
	existing, err := s.repo.GetByID(ctx, req.Id)
	if err != nil {
		if err == ErrWaitlistEntryNotFound {
			ctxErr := graceful.LogAndWrap(ctx, s.log, codes.NotFound, "waitlist entry not found", err,
				zap.Int64("id", req.Id))
			return nil, graceful.ToStatusError(ctxErr)
		}
		ctxErr := graceful.LogAndWrap(ctx, s.log, codes.Internal, "failed to get existing waitlist entry", err,
			zap.Int64("id", req.Id))
		return nil, graceful.ToStatusError(ctxErr)
	}

	// Check if user is already invited (cannot update invited users)
	if existing.Status == "invited" {
		ctxErr := graceful.LogAndWrap(ctx, s.log, codes.PermissionDenied, "cannot update invited user", nil,
			zap.Int64("id", req.Id), zap.String("status", existing.Status))
		return nil, graceful.ToStatusError(ctxErr)
	}

	// Create updated entry from existing entry
	updatedEntry := &waitlistpb.WaitlistEntry{
		Id:                   existing.Id,
		Uuid:                 existing.Uuid,
		MasterId:             existing.MasterId,
		MasterUuid:           existing.MasterUuid,
		Email:                existing.Email,
		FirstName:            existing.FirstName,
		LastName:             existing.LastName,
		Tier:                 existing.Tier,
		ReservedUsername:     existing.ReservedUsername,
		Intention:            existing.Intention,
		QuestionnaireAnswers: existing.QuestionnaireAnswers,
		Interests:            existing.Interests,
		ReferralUsername:     existing.ReferralUsername,
		ReferralCode:         existing.ReferralCode,
		Feedback:             existing.Feedback,
		AdditionalComments:   existing.AdditionalComments,
		Status:               existing.Status,
		PriorityScore:        existing.PriorityScore,
		ContactPreferences:   existing.ContactPreferences,
		Metadata:             existing.Metadata,
		CreatedAt:            existing.CreatedAt,
		UpdatedAt:            timestamppb.Now(),
		InvitedAt:            existing.InvitedAt,
		WaitlistPosition:     existing.WaitlistPosition,
		CampaignName:         existing.CampaignName,
		ReferralCount:        existing.ReferralCount,
		ReferralPoints:       existing.ReferralPoints,
		LocationCountry:      existing.LocationCountry,
		LocationRegion:       existing.LocationRegion,
		LocationCity:         existing.LocationCity,
		LocationLat:          existing.LocationLat,
		LocationLng:          existing.LocationLng,
		IpAddress:            existing.IpAddress,
		UserAgent:            existing.UserAgent,
		ReferrerUrl:          existing.ReferrerUrl,
		UtmSource:            existing.UtmSource,
		UtmMedium:            existing.UtmMedium,
		UtmCampaign:          existing.UtmCampaign,
		UtmTerm:              existing.UtmTerm,
		UtmContent:           existing.UtmContent,
	}

	// Update fields that are provided
	if req.Email != nil {
		// Validate email if being updated
		if *req.Email != existing.Email {
			if err := ValidateEmail(*req.Email); err != nil {
				ctxErr := graceful.LogAndWrap(ctx, s.log, codes.InvalidArgument, "invalid email format", err,
					zap.String("email", *req.Email))
				return nil, graceful.ToStatusError(ctxErr)
			}

			// Check if new email already exists
			exists, err := s.repo.EmailExists(ctx, *req.Email)
			if err != nil {
				ctxErr := graceful.LogAndWrap(ctx, s.log, codes.Internal, "failed to check email existence", err,
					zap.String("email", *req.Email))
				return nil, graceful.ToStatusError(ctxErr)
			}
			if exists {
				ctxErr := graceful.LogAndWrap(ctx, s.log, codes.AlreadyExists, "email already exists", nil,
					zap.String("email", *req.Email))
				return nil, graceful.ToStatusError(ctxErr)
			}
		}
		updatedEntry.Email = *req.Email
	}

	if req.FirstName != nil {
		updatedEntry.FirstName = *req.FirstName
	}

	if req.LastName != nil {
		updatedEntry.LastName = *req.LastName
	}

	if req.Tier != nil {
		updatedEntry.Tier = *req.Tier
	}

	if req.ReservedUsername != nil {
		// Validate username if being updated
		if *req.ReservedUsername != "" {
			if existing.ReservedUsername == nil || *req.ReservedUsername != *existing.ReservedUsername {
				if err := ValidateUsername(*req.ReservedUsername); err != nil {
					ctxErr := graceful.LogAndWrap(ctx, s.log, codes.InvalidArgument, "invalid username format", err,
						zap.String("username", *req.ReservedUsername))
					return nil, graceful.ToStatusError(ctxErr)
				}

				// Check if username is already taken
				exists, err := s.repo.UsernameExists(ctx, *req.ReservedUsername)
				if err != nil {
					ctxErr := graceful.LogAndWrap(ctx, s.log, codes.Internal, "failed to check username existence", err,
						zap.String("username", *req.ReservedUsername))
					return nil, graceful.ToStatusError(ctxErr)
				}
				if exists {
					ctxErr := graceful.LogAndWrap(ctx, s.log, codes.AlreadyExists, "username already taken", nil,
						zap.String("username", *req.ReservedUsername))
					return nil, graceful.ToStatusError(ctxErr)
				}
			}
		}
		updatedEntry.ReservedUsername = req.ReservedUsername
	}

	if req.Intention != nil {
		updatedEntry.Intention = *req.Intention
	}

	if req.QuestionnaireAnswers != nil {
		updatedEntry.QuestionnaireAnswers = req.QuestionnaireAnswers
	}

	if req.Interests != nil {
		updatedEntry.Interests = req.Interests
	}

	if req.ReferralUsername != nil {
		updatedEntry.ReferralUsername = req.ReferralUsername
	}

	if req.ReferralCode != nil {
		updatedEntry.ReferralCode = req.ReferralCode
	}

	if req.Feedback != nil {
		updatedEntry.Feedback = req.Feedback
	}

	if req.AdditionalComments != nil {
		updatedEntry.AdditionalComments = req.AdditionalComments
	}

	if req.Status != nil {
		updatedEntry.Status = *req.Status
	}

	if req.PriorityScore != nil {
		updatedEntry.PriorityScore = *req.PriorityScore
	}

	if req.ContactPreferences != nil {
		updatedEntry.ContactPreferences = req.ContactPreferences
	}

	if req.Metadata != nil {
		updatedEntry.Metadata = req.Metadata
	}

	// Recalculate priority score if tier changed
	if req.Tier != nil && *req.Tier != existing.Tier {
		updatedEntry.PriorityScore = s.calculatePriorityScore(updatedEntry)
	}

	result, err := s.repo.Update(ctx, updatedEntry)
	if err != nil {
		ctxErr := graceful.LogAndWrap(ctx, s.log, codes.Internal, "failed to update waitlist entry", err,
			zap.Int64("id", req.Id))
		return nil, graceful.ToStatusError(ctxErr)
	}

	// Clear cache
	s.clearCache(ctx, "waitlist_stats", "leaderboard")

	// Success orchestration (graceful handler)
	process := map[string]interface{}{
		"service":  "waitlist",
		"action":   "update",
		"entry_id": result.Id,
	}
	resultData := map[string]interface{}{
		"entry_id":   result.Id,
		"entry_uuid": result.Uuid,
		"email":      result.Email,
		"tier":       result.Tier,
		"campaign":   result.CampaignName,
	}
	s.handler.Success(
		ctx,
		"update_entry",
		codes.OK,
		"Successfully updated waitlist entry",
		resultData,
		process,
		"waitlist.entry.updated",
		nil,
	)
	return &waitlistpb.UpdateWaitlistEntryResponse{
		Entry: result,
	}, nil
}

// ListWaitlistEntries lists waitlist entries with pagination and filters
func (s *Service) ListWaitlistEntries(ctx context.Context, req *waitlistpb.ListWaitlistEntriesRequest) (*waitlistpb.ListWaitlistEntriesResponse, error) {
	limit := int(req.Limit)
	offset := int(req.Offset)

	// Apply default limit if not provided
	if limit <= 0 {
		limit = 50
	}

	// Cap limit to prevent large queries
	if limit > 1000 {
		limit = 1000
	}

	var tierFilter, statusFilter, campaignFilter string
	if req.TierFilter != nil {
		tierFilter = *req.TierFilter
	}
	if req.StatusFilter != nil {
		statusFilter = *req.StatusFilter
	}
	if req.CampaignFilter != nil {
		campaignFilter = *req.CampaignFilter
	}

	entries, totalCount, err := s.repo.List(ctx, limit, offset, tierFilter, statusFilter, campaignFilter)
	if err != nil {
		ctxErr := graceful.LogAndWrap(ctx, s.log, codes.Internal, "failed to list waitlist entries", err,
			zap.Int("limit", limit), zap.Int("offset", offset))
		return nil, graceful.ToStatusError(ctxErr)
	}

	// Success orchestration
	process := map[string]interface{}{
		"service": "waitlist",
		"action":  "list_entries",
		"limit":   limit,
		"offset":  offset,
	}

	resultData := map[string]interface{}{
		"entry_count": len(entries),
		"total_count": totalCount,
		"limit":       limit,
		"offset":      offset,
	}

	successCtx := graceful.LogAndWrapSuccess(ctx, s.log, codes.OK,
		"Successfully listed waitlist entries", resultData, process,
		zap.Int("entry_count", len(entries)),
		zap.Int64("total_count", totalCount))

	// Run success orchestration
	if s.eventEmitter != nil {
		successCtx.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
			Log:          s.log,
			Cache:        s.cache,
			EventEmitter: s.eventEmitter,
			EventEnabled: s.eventEnabled,
			EventType:    "waitlist.entries.listed",
			EventID:      fmt.Sprintf("list-%d-%d", offset, limit),
			PatternType:  "waitlist",
			PatternID:    fmt.Sprintf("list-%d-%d", offset, limit),
			Tags:         []string{"waitlist", "list", "entries"},
		})
	}

	return &waitlistpb.ListWaitlistEntriesResponse{
		Entries:    entries,
		TotalCount: totalCount,
	}, nil
}

// InviteUser invites a user (updates status to invited)
func (s *Service) InviteUser(ctx context.Context, req *waitlistpb.InviteUserRequest) (*waitlistpb.InviteUserResponse, error) {
	// Get existing entry
	existing, err := s.repo.GetByID(ctx, req.Id)
	if err != nil {
		if err == ErrWaitlistEntryNotFound {
			ctxErr := graceful.LogAndWrap(ctx, s.log, codes.NotFound, "waitlist entry not found", err,
				zap.Int64("id", req.Id))
			return nil, graceful.ToStatusError(ctxErr)
		}
		ctxErr := graceful.LogAndWrap(ctx, s.log, codes.Internal, "failed to get waitlist entry", err,
			zap.Int64("id", req.Id))
		return nil, graceful.ToStatusError(ctxErr)
	}

	// Check if already invited
	if existing.Status == "invited" {
		return &waitlistpb.InviteUserResponse{
			Success: false,
			Message: "user already invited",
		}, nil
	}

	// Update status to invited
	if err := s.repo.UpdateStatus(ctx, req.Id, "invited"); err != nil {
		ctxErr := graceful.LogAndWrap(ctx, s.log, codes.Internal, "failed to invite user", err,
			zap.Int64("id", req.Id))
		return nil, graceful.ToStatusError(ctxErr)
	}

	// Clear cache
	s.clearCache(ctx, "waitlist_stats", "leaderboard")

	// Success orchestration (graceful handler)
	process := map[string]interface{}{
		"service":  "waitlist",
		"action":   "invite_user",
		"entry_id": req.Id,
	}
	resultData := map[string]interface{}{
		"entry_id":   req.Id,
		"entry_uuid": existing.Uuid,
		"email":      existing.Email,
		"tier":       existing.Tier,
		"campaign":   existing.CampaignName,
	}
	s.handler.Success(
		ctx,
		"invite_user",
		codes.OK,
		"Successfully invited user",
		resultData,
		process,
		"waitlist.user.invited",
		nil,
	)
	return &waitlistpb.InviteUserResponse{
		Success: true,
		Message: "user invited successfully",
	}, nil
}

// CheckUsernameAvailability checks if a username is available
func (s *Service) CheckUsernameAvailability(ctx context.Context, req *waitlistpb.CheckUsernameAvailabilityRequest) (*waitlistpb.CheckUsernameAvailabilityResponse, error) {
	if req.Username == "" {
		ctxErr := graceful.LogAndWrap(ctx, s.log, codes.InvalidArgument, "username is required", nil,
			zap.String("username", req.Username))
		return nil, graceful.ToStatusError(ctxErr)
	}

	// Validate username format
	if err := ValidateUsername(req.Username); err != nil {
		// Username format is invalid, return false but don't treat as error
		return &waitlistpb.CheckUsernameAvailabilityResponse{
			Available: false,
		}, nil
	}

	exists, err := s.repo.UsernameExists(ctx, req.Username)
	if err != nil {
		ctxErr := graceful.LogAndWrap(ctx, s.log, codes.Internal, "failed to check username availability", err,
			zap.String("username", req.Username))
		return nil, graceful.ToStatusError(ctxErr)
	}

	// Success orchestration (graceful handler)
	process := map[string]interface{}{
		"service":  "waitlist",
		"action":   "check_username_availability",
		"username": req.Username,
	}
	resultData := map[string]interface{}{
		"username":  req.Username,
		"available": !exists,
	}
	s.handler.Success(
		ctx,
		"check_username_availability",
		codes.OK,
		"Successfully checked username availability",
		resultData,
		process,
		"waitlist.username.checked",
		nil,
	)
	return &waitlistpb.CheckUsernameAvailabilityResponse{
		Available: !exists,
	}, nil
}

// ValidateReferralUsername validates a referral username
func (s *Service) ValidateReferralUsername(ctx context.Context, req *waitlistpb.ValidateReferralUsernameRequest) (*waitlistpb.ValidateReferralUsernameResponse, error) {
	if req.Username == "" {
		ctxErr := graceful.LogAndWrap(ctx, s.log, codes.InvalidArgument, "username is required", nil,
			zap.String("username", req.Username))
		return nil, graceful.ToStatusError(ctxErr)
	}

	valid, err := s.repo.ValidateReferralUsername(ctx, req.Username)
	if err != nil {
		ctxErr := graceful.LogAndWrap(ctx, s.log, codes.Internal, "failed to validate referral username", err,
			zap.String("username", req.Username))
		return nil, graceful.ToStatusError(ctxErr)
	}

	// Success orchestration (graceful handler)
	process := map[string]interface{}{
		"service":  "waitlist",
		"action":   "validate_referral_username",
		"username": req.Username,
	}
	resultData := map[string]interface{}{
		"username": req.Username,
		"valid":    valid,
	}
	s.handler.Success(
		ctx,
		"validate_referral_username",
		codes.OK,
		"Successfully validated referral username",
		resultData,
		process,
		"waitlist.referral.validated",
		nil,
	)
	return &waitlistpb.ValidateReferralUsernameResponse{
		Valid: valid,
	}, nil
}

// GetLeaderboard gets the referral leaderboard
func (s *Service) GetLeaderboard(ctx context.Context, req *waitlistpb.GetLeaderboardRequest) (*waitlistpb.GetLeaderboardResponse, error) {
	limit := int(req.Limit)
	if limit <= 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000
	}

	var campaign string
	if req.Campaign != nil {
		campaign = *req.Campaign
	}

	// Try to get from cache first
	cacheKey := fmt.Sprintf("leaderboard_%d_%s", limit, campaign)
	if s.cache != nil {
		var cachedEntries []*waitlistpb.LeaderboardEntry
		if err := s.cache.Get(ctx, cacheKey, "entries", &cachedEntries); err == nil {
			// Success orchestration for cache hit
			process := map[string]interface{}{
				"service":  "waitlist",
				"action":   "get_leaderboard",
				"source":   "cache",
				"limit":    limit,
				"campaign": campaign,
			}

			resultData := map[string]interface{}{
				"entry_count": len(cachedEntries),
				"limit":       limit,
				"campaign":    campaign,
				"cached":      true,
			}

			successCtx := graceful.LogAndWrapSuccess(ctx, s.log, codes.OK,
				"Successfully retrieved leaderboard from cache", resultData, process,
				zap.Int("entry_count", len(cachedEntries)),
				zap.Int("limit", limit),
				zap.String("campaign", campaign))

			// Run success orchestration
			if s.eventEmitter != nil {
				successCtx.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
					Log:          s.log,
					Cache:        s.cache,
					EventEmitter: s.eventEmitter,
					EventEnabled: s.eventEnabled,
					EventType:    "waitlist.leaderboard.retrieved",
					EventID:      fmt.Sprintf("leaderboard-%d-%s", limit, campaign),
					PatternType:  "waitlist",
					PatternID:    fmt.Sprintf("leaderboard-%d-%s", limit, campaign),
					Tags:         []string{"waitlist", "leaderboard", "cached"},
				})
			}

			return &waitlistpb.GetLeaderboardResponse{
				Entries: cachedEntries,
			}, nil
		}
	}

	entries, err := s.repo.GetLeaderboard(ctx, limit, campaign)
	if err != nil {
		ctxErr := graceful.LogAndWrap(ctx, s.log, codes.Internal, "failed to get leaderboard", err,
			zap.Int("limit", limit), zap.String("campaign", campaign))
		return nil, graceful.ToStatusError(ctxErr)
	}

	// Cache the result
	if s.cache != nil {
		s.cache.Set(ctx, cacheKey, "entries", entries, 5*time.Minute)
	}

	// Success orchestration (graceful handler)
	process := map[string]interface{}{
		"service":  "waitlist",
		"action":   "get_leaderboard",
		"source":   "database",
		"limit":    limit,
		"campaign": campaign,
	}
	resultData := map[string]interface{}{
		"entry_count": len(entries),
		"limit":       limit,
		"campaign":    campaign,
		"cached":      false,
	}
	s.handler.Success(
		ctx,
		"get_leaderboard",
		codes.OK,
		"Successfully retrieved leaderboard from database",
		resultData,
		process,
		"waitlist.leaderboard.retrieved",
		nil,
	)
	return &waitlistpb.GetLeaderboardResponse{
		Entries: entries,
	}, nil
}

// GetReferralsByUser gets referrals made by a user
func (s *Service) GetReferralsByUser(ctx context.Context, req *waitlistpb.GetReferralsByUserRequest) (*waitlistpb.GetReferralsByUserResponse, error) {
	referrals, err := s.repo.GetReferralsByUser(ctx, req.UserId)
	if err != nil {
		ctxErr := graceful.LogAndWrap(ctx, s.log, codes.Internal, "failed to get referrals by user", err,
			zap.Int64("user_id", req.UserId))
		return nil, graceful.ToStatusError(ctxErr)
	}

	// Success orchestration (graceful handler)
	process := map[string]interface{}{
		"service": "waitlist",
		"action":  "get_referrals_by_user",
		"user_id": req.UserId,
	}
	resultData := map[string]interface{}{
		"user_id":        req.UserId,
		"referral_count": len(referrals),
	}
	s.handler.Success(
		ctx,
		"get_referrals_by_user",
		codes.OK,
		"Successfully retrieved referrals by user",
		resultData,
		process,
		"waitlist.referrals.retrieved",
		nil,
	)
	return &waitlistpb.GetReferralsByUserResponse{
		Referrals: referrals,
	}, nil
}

// GetLocationStats gets location-based statistics
func (s *Service) GetLocationStats(ctx context.Context, req *waitlistpb.GetLocationStatsRequest) (*waitlistpb.GetLocationStatsResponse, error) {
	var campaign string
	if req.Campaign != nil {
		campaign = *req.Campaign
	}

	// Try to get from cache first
	cacheKey := fmt.Sprintf("location_stats_%s", campaign)
	if s.cache != nil {
		var cachedStats []*waitlistpb.LocationStat
		if err := s.cache.Get(ctx, cacheKey, "stats", &cachedStats); err == nil {
			// Success orchestration for cache hit
			process := map[string]interface{}{
				"service":  "waitlist",
				"action":   "get_location_stats",
				"source":   "cache",
				"campaign": campaign,
			}

			resultData := map[string]interface{}{
				"stats_count": len(cachedStats),
				"campaign":    campaign,
				"cached":      true,
			}

			successCtx := graceful.LogAndWrapSuccess(ctx, s.log, codes.OK,
				"Successfully retrieved location stats from cache", resultData, process,
				zap.Int("stats_count", len(cachedStats)),
				zap.String("campaign", campaign))

			// Run success orchestration
			if s.eventEmitter != nil {
				successCtx.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
					Log:          s.log,
					Cache:        s.cache,
					EventEmitter: s.eventEmitter,
					EventEnabled: s.eventEnabled,
					EventType:    "waitlist.location_stats.retrieved",
					EventID:      fmt.Sprintf("location-stats-%s", campaign),
					PatternType:  "waitlist",
					PatternID:    fmt.Sprintf("location-stats-%s", campaign),
					Tags:         []string{"waitlist", "location", "stats", "cached"},
				})
			}

			return &waitlistpb.GetLocationStatsResponse{
				Stats: cachedStats,
			}, nil
		}
	}

	stats, err := s.repo.GetLocationStats(ctx, campaign)
	if err != nil {
		ctxErr := graceful.LogAndWrap(ctx, s.log, codes.Internal, "failed to get location stats", err,
			zap.String("campaign", campaign))
		return nil, graceful.ToStatusError(ctxErr)
	}

	// Cache the result
	if s.cache != nil {
		s.cache.Set(ctx, cacheKey, "stats", stats, 10*time.Minute)
	}

	// Success orchestration (graceful handler)
	process := map[string]interface{}{
		"service":  "waitlist",
		"action":   "get_location_stats",
		"source":   "database",
		"campaign": campaign,
	}
	resultData := map[string]interface{}{
		"stats_count": len(stats),
		"campaign":    campaign,
		"cached":      false,
	}
	s.handler.Success(
		ctx,
		"get_location_stats",
		codes.OK,
		"Successfully retrieved location stats from database",
		resultData,
		process,
		"waitlist.location_stats.retrieved",
		nil,
	)
	return &waitlistpb.GetLocationStatsResponse{
		Stats: stats,
	}, nil
}

// GetWaitlistStats gets waitlist statistics
func (s *Service) GetWaitlistStats(ctx context.Context, req *waitlistpb.GetWaitlistStatsRequest) (*waitlistpb.GetWaitlistStatsResponse, error) {
	var campaign string
	if req.Campaign != nil {
		campaign = *req.Campaign
	}

	// Try to get from cache first
	cacheKey := fmt.Sprintf("waitlist_stats_%s", campaign)
	if s.cache != nil {
		var cachedStats *waitlistpb.WaitlistStats
		if err := s.cache.Get(ctx, cacheKey, "stats", &cachedStats); err == nil {
			// Success orchestration for cache hit
			process := map[string]interface{}{
				"service":  "waitlist",
				"action":   "get_stats",
				"source":   "cache",
				"campaign": campaign,
			}

			resultData := map[string]interface{}{
				"campaign": campaign,
				"cached":   true,
			}

			s.handler.Success(
				ctx,
				"get_stats", // canonical action name
				codes.OK,
				"Successfully retrieved waitlist stats from cache",
				resultData,
				process,
				"waitlist.stats.retrieved",
				nil, // cache info
			)

			return &waitlistpb.GetWaitlistStatsResponse{
				Stats: cachedStats,
			}, nil
		}
	}

	stats, err := s.repo.GetStats(ctx, campaign)
	if err != nil {
		ctxErr := graceful.LogAndWrap(ctx, s.log, codes.Internal, "failed to get waitlist stats", err,
			zap.String("campaign", campaign))
		return nil, graceful.ToStatusError(ctxErr)
	}

	// Cache the result
	if s.cache != nil {
		s.cache.Set(ctx, cacheKey, "stats", stats, 5*time.Minute)
	}

	// Success orchestration (graceful handler)
	process := map[string]interface{}{
		"service":  "waitlist",
		"action":   "get_waitlist_stats",
		"source":   "database",
		"campaign": campaign,
	}
	resultData := map[string]interface{}{
		"campaign": campaign,
		"cached":   false,
	}
	s.handler.Success(
		ctx,
		"get_waitlist_stats",
		codes.OK,
		"Successfully retrieved waitlist stats from database",
		resultData,
		process,
		"waitlist.stats.retrieved",
		nil,
	)
	return &waitlistpb.GetWaitlistStatsResponse{
		Stats: stats,
	}, nil
}

// GetWaitlistPosition gets a user's position in the waitlist
func (s *Service) GetWaitlistPosition(ctx context.Context, req *waitlistpb.GetWaitlistPositionRequest) (*waitlistpb.GetWaitlistPositionResponse, error) {
	position, err := s.repo.GetWaitlistPosition(ctx, req.Id)
	if err != nil {
		ctxErr := graceful.LogAndWrap(ctx, s.log, codes.Internal, "failed to get waitlist position", err,
			zap.Int64("id", req.Id))
		return nil, graceful.ToStatusError(ctxErr)
	}

	// Success orchestration (graceful handler)
	process := map[string]interface{}{
		"service":  "waitlist",
		"action":   "get_waitlist_position",
		"entry_id": req.Id,
	}
	resultData := map[string]interface{}{
		"entry_id": req.Id,
		"position": position,
	}
	s.handler.Success(
		ctx,
		"get_waitlist_position",
		codes.OK,
		"Successfully retrieved waitlist position",
		resultData,
		process,
		"waitlist.position.retrieved",
		nil,
	)
	return &waitlistpb.GetWaitlistPositionResponse{
		Position: int32(position),
	}, nil
}

// Helper methods

func (s *Service) calculatePriorityScore(entry *waitlistpb.WaitlistEntry) int32 {
	score := int32(0)

	// Base score by tier
	switch strings.ToLower(entry.Tier) {
	case "pioneer":
		score += 100
	case "talent":
		score += 80
	case "hustler":
		score += 60
	case "business":
		score += 40
	case "user":
		score += 20
	}

	// Bonus for having referrals
	if entry.ReferralCount > 0 {
		score += entry.ReferralCount * 10
	}

	// Bonus for having reserved username
	if entry.ReservedUsername != nil && *entry.ReservedUsername != "" {
		score += 5
	}

	// Bonus for completing questionnaire
	if entry.QuestionnaireAnswers != nil && len(entry.QuestionnaireAnswers.Fields) > 0 {
		score += 10
	}

	return score
}

func (s *Service) clearCache(ctx context.Context, patterns ...string) {
	if s.cache == nil {
		return
	}

	for _, pattern := range patterns {
		if err := s.cache.DeletePattern(ctx, pattern+"*"); err != nil {
			s.log.Warn("Failed to clear cache pattern", zap.Error(err), zap.String("pattern", pattern))
		}
	}
}
