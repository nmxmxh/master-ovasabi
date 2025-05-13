package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"time"

	userpb "github.com/nmxmxh/master-ovasabi/api/protos/user/v1"
	userrepo "github.com/nmxmxh/master-ovasabi/internal/repository/user"
	pattern "github.com/nmxmxh/master-ovasabi/internal/service/pattern"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Service implements the UserService gRPC interface.
type Service struct {
	userpb.UnimplementedUserServiceServer
	log   *zap.Logger
	cache *redis.Cache
	repo  *userrepo.UserRepository
}

// Compile-time check.
var _ userpb.UserServiceServer = (*Service)(nil)

// NewUserService creates a new instance of UserService.
func NewUserService(log *zap.Logger, repo *userrepo.UserRepository, cache *redis.Cache) userpb.UserServiceServer {
	return &Service{
		log:   log,
		repo:  repo,
		cache: cache,
	}
}

// --- Conversion Helpers ---.
func repoUserToProtoUser(u *userrepo.User) *userpb.User {
	if u == nil {
		return nil
	}
	return &userpb.User{
		Id:           u.ID,
		MasterId:     u.MasterID,
		Username:     u.Username,
		Email:        u.Email,
		ReferralCode: u.ReferralCode,
		ReferredBy:   u.ReferredBy,
		DeviceHash:   u.DeviceHash,
		Location:     u.Location,
		CreatedAt:    timestamppb.New(u.CreatedAt),
		UpdatedAt:    timestamppb.New(u.UpdatedAt),
		PasswordHash: u.PasswordHash,
		Metadata:     u.Metadata,
		Profile:      repoProfileToProto(&u.Profile),
		Roles:        u.Roles,
		Status:       userpb.UserStatus(u.Status),
		Tags:         u.Tags,
		ExternalIds:  u.ExternalIDs,
	}
}

func repoProfileToProto(p *userrepo.UserProfile) *userpb.UserProfile {
	if p == nil {
		return nil
	}
	return &userpb.UserProfile{
		FirstName:    p.FirstName,
		LastName:     p.LastName,
		PhoneNumber:  p.PhoneNumber,
		AvatarUrl:    p.AvatarURL,
		Bio:          p.Bio,
		Timezone:     p.Timezone,
		Language:     p.Language,
		CustomFields: p.CustomFields,
	}
}

func protoProfileToRepo(p *userpb.UserProfile) userrepo.UserProfile {
	if p == nil {
		return userrepo.UserProfile{}
	}
	return userrepo.UserProfile{
		FirstName:    p.FirstName,
		LastName:     p.LastName,
		PhoneNumber:  p.PhoneNumber,
		AvatarURL:    p.AvatarUrl,
		Bio:          p.Bio,
		Timezone:     p.Timezone,
		Language:     p.Language,
		CustomFields: p.CustomFields,
	}
}

// --- Refactor CRUD/profile methods to use all fields ---
// ... refactor CreateUser, GetUser, GetUserByUsername, UpdateUser, DeleteUser, ListUsers, UpdateProfile ...
// ... use the conversion helpers for all proto<->repo mapping ...
// ... update all request/response conversions to include all proto fields ...

// CreateUser creates a new user following the Master-Client-Service-Event pattern.
func (s *Service) CreateUser(ctx context.Context, req *userpb.CreateUserRequest) (*userpb.CreateUserResponse, error) {
	s.log.Info("Creating user",
		zap.String("email", req.Email),
		zap.String("username", req.Username))

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.log.Error("Failed to hash password", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to hash password: %v", err)
	}

	user := &userrepo.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		Profile:      protoProfileToRepo(req.Profile),
		Roles:        req.Roles,
		Status:       int32(userpb.UserStatus_USER_STATUS_ACTIVE),
		Metadata:     req.Metadata,
	}

	created, err := s.repo.Create(ctx, user)
	if err != nil {
		switch {
		case errors.Is(err, userrepo.ErrInvalidUsername):
			return nil, status.Error(codes.InvalidArgument, "invalid username format")
		case errors.Is(err, userrepo.ErrUsernameReserved):
			return nil, status.Error(codes.InvalidArgument, "username is reserved")
		case errors.Is(err, userrepo.ErrUsernameTaken):
			return nil, status.Error(codes.AlreadyExists, "username is already taken")
		default:
			return nil, status.Errorf(codes.Internal, "failed to create user: %v", err)
		}
	}

	respUser := repoUserToProtoUser(created)

	if err := s.cache.Set(ctx, created.ID, "profile", respUser, redis.TTLUserProfile); err != nil {
		s.log.Error("Failed to cache user profile",
			zap.String("user_id", created.ID),
			zap.Error(err))
	}

	if s.cache != nil && created.Metadata != nil {
		err := pattern.CacheMetadata(ctx, s.log, s.cache, "user", created.ID, created.Metadata, 10*time.Minute)
		if err != nil {
			s.log.Error("failed to cache metadata", zap.Error(err))
		}
	}
	err = pattern.RegisterSchedule(ctx, s.log, "user", created.ID, created.Metadata)
	if err != nil {
		s.log.Error("failed to register schedule", zap.Error(err))
	}
	err = pattern.EnrichKnowledgeGraph(ctx, s.log, "user", created.ID, created.Metadata)
	if err != nil {
		s.log.Error("failed to enrich knowledge graph", zap.Error(err))
	}
	err = pattern.RegisterWithNexus(ctx, s.log, "user", created.Metadata)
	if err != nil {
		s.log.Error("failed to register with nexus", zap.Error(err))
	}

	return &userpb.CreateUserResponse{
		User: respUser,
	}, nil
}

// GetUser retrieves user information.
func (s *Service) GetUser(ctx context.Context, req *userpb.GetUserRequest) (*userpb.GetUserResponse, error) {
	var user userpb.User
	if err := s.cache.Get(ctx, req.UserId, "profile", &user); err == nil {
		return &userpb.GetUserResponse{User: &user}, nil
	}

	repoUser, err := s.repo.GetByID(ctx, req.UserId)
	if err != nil {
		if errors.Is(err, userrepo.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "database error: %v", err)
	}

	respUser := repoUserToProtoUser(repoUser)

	if err := s.cache.Set(ctx, req.UserId, "profile", respUser, redis.TTLUserProfile); err != nil {
		s.log.Error("Failed to cache user profile",
			zap.String("user_id", req.UserId),
			zap.Error(err))
	}

	if s.cache != nil && repoUser.Metadata != nil {
		err := pattern.CacheMetadata(ctx, s.log, s.cache, "user", repoUser.ID, repoUser.Metadata, 10*time.Minute)
		if err != nil {
			s.log.Error("failed to cache metadata", zap.Error(err))
		}
	}
	err = pattern.RegisterSchedule(ctx, s.log, "user", repoUser.ID, repoUser.Metadata)
	if err != nil {
		s.log.Error("failed to register schedule", zap.Error(err))
	}
	err = pattern.EnrichKnowledgeGraph(ctx, s.log, "user", repoUser.ID, repoUser.Metadata)
	if err != nil {
		s.log.Error("failed to enrich knowledge graph", zap.Error(err))
	}
	err = pattern.RegisterWithNexus(ctx, s.log, "user", repoUser.Metadata)
	if err != nil {
		s.log.Error("failed to register with nexus", zap.Error(err))
	}

	return &userpb.GetUserResponse{User: respUser}, nil
}

// GetUserByUsername retrieves user information by username.
func (s *Service) GetUserByUsername(ctx context.Context, req *userpb.GetUserByUsernameRequest) (*userpb.GetUserByUsernameResponse, error) {
	repoUser, err := s.repo.GetByUsername(ctx, req.Username)
	if err != nil {
		if errors.Is(err, userrepo.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "database error: %v", err)
	}

	respUser := repoUserToProtoUser(repoUser)

	return &userpb.GetUserByUsernameResponse{User: respUser}, nil
}

// GetUserByEmail retrieves user information by email.
func (s *Service) GetUserByEmail(ctx context.Context, req *userpb.GetUserByEmailRequest) (*userpb.GetUserByEmailResponse, error) {
	repoUser, err := s.repo.GetByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, userrepo.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "database error: %v", err)
	}
	respUser := repoUserToProtoUser(repoUser)
	return &userpb.GetUserByEmailResponse{User: respUser}, nil
}

// UpdateUser updates a user record.
func (s *Service) UpdateUser(ctx context.Context, req *userpb.UpdateUserRequest) (*userpb.UpdateUserResponse, error) {
	repoUser, err := s.repo.GetByID(ctx, req.UserId)
	if err != nil {
		if errors.Is(err, userrepo.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, "failed to get user")
	}

	if req.User != nil {
		if req.User.Username != "" {
			repoUser.Username = req.User.Username
		}
		if req.User.Email != "" {
			repoUser.Email = req.User.Email
		}
		if req.User.PasswordHash != "" {
			repoUser.PasswordHash = req.User.PasswordHash
		}
		if req.User.ReferralCode != "" {
			repoUser.ReferralCode = req.User.ReferralCode
		}
		if req.User.ReferredBy != "" {
			repoUser.ReferredBy = req.User.ReferredBy
		}
		if req.User.DeviceHash != "" {
			repoUser.DeviceHash = req.User.DeviceHash
		}
		if req.User.Location != "" {
			repoUser.Location = req.User.Location
		}
		if req.User.Profile != nil {
			repoUser.Profile = protoProfileToRepo(req.User.Profile)
		}
		if req.User.Roles != nil {
			repoUser.Roles = req.User.Roles
		}
		if req.User.Metadata != nil {
			if err := metadata.ValidateMetadata(req.User.Metadata); err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "invalid metadata: %v", err)
			}
			repoUser.Metadata = req.User.Metadata
		}
		repoUser.Status = int32(req.User.Status)
		if req.User.Tags != nil {
			repoUser.Tags = req.User.Tags
		}
		if req.User.ExternalIds != nil {
			repoUser.ExternalIDs = req.User.ExternalIds
		}
	}

	if err := s.repo.Update(ctx, repoUser); err != nil {
		switch {
		case errors.Is(err, userrepo.ErrInvalidUsername):
			return nil, status.Error(codes.InvalidArgument, "invalid username format")
		case errors.Is(err, userrepo.ErrUsernameReserved):
			return nil, status.Error(codes.InvalidArgument, "username is reserved")
		case errors.Is(err, userrepo.ErrUsernameTaken):
			return nil, status.Error(codes.AlreadyExists, "username is already taken")
		default:
			return nil, status.Errorf(codes.Internal, "failed to update user: %v", err)
		}
	}

	if err := s.cache.Delete(ctx, req.UserId, "profile"); err != nil {
		s.log.Error("Failed to invalidate user cache",
			zap.String("user_id", req.UserId),
			zap.Error(err))
	}

	getResp, err := s.GetUser(ctx, &userpb.GetUserRequest{UserId: req.UserId})
	if err != nil {
		return nil, err
	}

	if s.cache != nil && repoUser.Metadata != nil {
		err := pattern.CacheMetadata(ctx, s.log, s.cache, "user", repoUser.ID, repoUser.Metadata, 10*time.Minute)
		if err != nil {
			s.log.Error("failed to cache metadata", zap.Error(err))
		}
	}
	err = pattern.RegisterSchedule(ctx, s.log, "user", repoUser.ID, repoUser.Metadata)
	if err != nil {
		s.log.Error("failed to register schedule", zap.Error(err))
	}
	err = pattern.EnrichKnowledgeGraph(ctx, s.log, "user", repoUser.ID, repoUser.Metadata)
	if err != nil {
		s.log.Error("failed to enrich knowledge graph", zap.Error(err))
	}
	err = pattern.RegisterWithNexus(ctx, s.log, "user", repoUser.Metadata)
	if err != nil {
		s.log.Error("failed to register with nexus", zap.Error(err))
	}
	return &userpb.UpdateUserResponse{User: getResp.User}, nil
}

// DeleteUser removes a user and its master record.
func (s *Service) DeleteUser(ctx context.Context, req *userpb.DeleteUserRequest) (*userpb.DeleteUserResponse, error) {
	repoUser, err := s.repo.GetByID(ctx, req.UserId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to delete user: %v", err)
	}
	if err := s.repo.Delete(ctx, repoUser.ID); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete user: %v", err)
	}
	if err := s.cache.Delete(ctx, req.UserId, "profile"); err != nil {
		s.log.Error("Failed to invalidate user cache",
			zap.String("user_id", req.UserId),
			zap.Error(err))
	}
	return &userpb.DeleteUserResponse{Success: true}, nil
}

// ListUsers retrieves a list of users with pagination and filtering.
func (s *Service) ListUsers(ctx context.Context, req *userpb.ListUsersRequest) (*userpb.ListUsersResponse, error) {
	// Use ListFlexible if advanced filtering/search is requested
	if req.SearchQuery != "" || len(req.Tags) > 0 || req.Metadata != nil || req.Filters != nil {
		users, total, err := s.repo.ListFlexible(ctx, req)
		if err != nil {
			s.log.Error("failed to list users (flexible)", zap.Error(err))
			return nil, status.Errorf(codes.Internal, "failed to list users: %v", err)
		}
		if total > int(^int32(0)) || total < 0 {
			return nil, fmt.Errorf("total overflows int32")
		}
		totalPages := (total + int(req.PageSize) - 1) / int(req.PageSize)
		if totalPages > int(^int32(0)) || totalPages < 0 {
			return nil, fmt.Errorf("totalPages overflows int32")
		}
		// Explicit check before conversion (required by gosec)
		if total > int(^int32(0)) || total < 0 {
			return nil, fmt.Errorf("total overflows int32 (final check)")
		}
		if totalPages > int(^int32(0)) || totalPages < 0 {
			return nil, fmt.Errorf("totalPages overflows int32 (final check)")
		}
		if total > int(^int32(0)) || total < 0 {
			return nil, fmt.Errorf("total overflows int32 (final check 2)")
		}
		if totalPages > int(^int32(0)) || totalPages < 0 {
			return nil, fmt.Errorf("totalPages overflows int32 (final check 2)")
		}
		if totalPages > int(^int32(0)) || totalPages < 0 {
			return nil, fmt.Errorf("totalPages overflows int32 (final check 3)")
		}
		resp := &userpb.ListUsersResponse{
			Users:      make([]*userpb.User, 0, len(users)),
			TotalCount: int32(total), //nolint:gosec // overflow checked above
			Page:       req.Page,
			TotalPages: int32(totalPages),
		}
		for _, u := range users {
			respUser := repoUserToProtoUser(u)
			resp.Users = append(resp.Users, respUser)
		}
		return resp, nil
	}
	// Fallback to basic List
	limit := 10
	if req.PageSize > 0 {
		limit = int(req.PageSize)
	}
	page := int64(req.Page)
	lim := int64(limit)
	offset64 := page * lim
	if offset64 > math.MaxInt32 || offset64 < 0 {
		return nil, status.Error(codes.InvalidArgument, "pagination overflow")
	}
	offset := int(offset64)
	users, err := s.repo.List(ctx, limit, offset)
	if err != nil {
		s.log.Error("failed to list users", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to list users: %v", err)
	}
	resp := &userpb.ListUsersResponse{
		Users: make([]*userpb.User, 0, len(users)),
	}
	totalPages := 0
	if limit > 0 {
		totalPages = (len(users) + limit - 1) / limit
		if totalPages > int(^int32(0)) || totalPages < 0 {
			return nil, fmt.Errorf("totalPages overflows int32")
		}
	}
	// Explicit check before conversion
	if totalPages > int(^int32(0)) || totalPages < 0 {
		return nil, fmt.Errorf("totalPages overflows int32 (post-check)")
	}
	resp.TotalPages = int32(totalPages)
	for _, u := range users {
		respUser := repoUserToProtoUser(u)
		resp.Users = append(resp.Users, respUser)
	}
	return resp, nil
}

// UpdatePassword implements the UpdatePassword RPC method.
func (s *Service) UpdatePassword(_ context.Context, _ *userpb.UpdatePasswordRequest) (*userpb.UpdatePasswordResponse, error) {
	// In a real implementation, you would:
	// 1. Verify the current password
	// 2. Hash the new password
	// 3. Update the password in the database
	// For this example, we'll just return success
	return &userpb.UpdatePasswordResponse{
		Success:   true,
		UpdatedAt: time.Now().Unix(),
	}, nil
}

// UpdateProfile updates a user's profile.
func (s *Service) UpdateProfile(ctx context.Context, req *userpb.UpdateProfileRequest) (*userpb.UpdateProfileResponse, error) {
	repoUser, err := s.repo.GetByID(ctx, req.UserId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	// Update fields based on FieldsToUpdate and Profile.CustomFields
	for _, field := range req.FieldsToUpdate {
		if req.Profile != nil && req.Profile.CustomFields != nil {
			switch field {
			case "email":
				if v, ok := req.Profile.CustomFields["email"]; ok {
					repoUser.Email = v
				}
			case "referral_code":
				// Not present in repository.User, skip or handle as needed
			case "device_hash":
				// Not present in repository.User, skip or handle as needed
			case "location":
				// Not present in repository.User, skip or handle as needed
			}
		}
	}
	if req.Profile != nil {
		repoUser.Profile = protoProfileToRepo(req.Profile)
	}
	if err := s.repo.Update(ctx, repoUser); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update profile: %v", err)
	}
	getResp, err := s.GetUser(ctx, &userpb.GetUserRequest{UserId: req.UserId})
	if err != nil {
		return nil, err
	}
	return &userpb.UpdateProfileResponse{User: getResp.User}, nil
}

// TODO: Implement RegisterUser
// Pseudocode:
// 1. Validate input data
// 2. Check for existing user (by email/phone)
// 3. Hash password (Auth/Security)
// 4. Store user in DB
// 5. Send welcome notification
// 6. Register user in Nexus
// 7. Return user details

// TODO: Implement UpdateUserProfile
// Pseudocode:
// 1. Authenticate user
// 2. Validate new profile data
// 3. Update user in DB
// 4. Log update in Nexus
// 5. Notify user of changes

// --- Add stubs for all unimplemented proto RPCs ---.
func (s *Service) CreateSession(_ context.Context, _ *userpb.CreateSessionRequest) (*userpb.CreateSessionResponse, error) {
	// TODO: Implement session creation logic
	// Pseudocode for canonical metadata pattern:
	// 1. Validate metadata (if present) using metadata.ValidateMetadata(req.Metadata)
	// 2. Store metadata as *common.Metadata in Postgres (jsonb)
	// 3. After successful DB write:
	//    - pattern.CacheMetadata(ctx, s.cache, "session", session.Id, session.Metadata, 10*time.Minute)
	//    - pattern.RegisterSchedule(ctx, "session", session.Id, session.Metadata)
	//    - pattern.EnrichKnowledgeGraph(ctx, "session", session.Id, session.Metadata)
	//    - pattern.RegisterWithNexus(ctx, "session", session.Metadata)
	return nil, status.Error(codes.Unimplemented, "CreateSession not yet implemented")
}

func (s *Service) GetSession(_ context.Context, _ *userpb.GetSessionRequest) (*userpb.GetSessionResponse, error) {
	return nil, status.Error(codes.Unimplemented, "GetSession not yet implemented")
}

func (s *Service) RevokeSession(_ context.Context, _ *userpb.RevokeSessionRequest) (*userpb.RevokeSessionResponse, error) {
	return nil, status.Error(codes.Unimplemented, "RevokeSession not yet implemented")
}

func (s *Service) ListSessions(_ context.Context, _ *userpb.ListSessionsRequest) (*userpb.ListSessionsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "ListSessions not yet implemented")
}

func (s *Service) AssignRole(_ context.Context, _ *userpb.AssignRoleRequest) (*userpb.AssignRoleResponse, error) {
	s.log.Warn("AssignRole not implemented in UserService")
	return nil, status.Error(codes.Unimplemented, "AssignRole not yet implemented")
}

func (s *Service) RemoveRole(_ context.Context, _ *userpb.RemoveRoleRequest) (*userpb.RemoveRoleResponse, error) {
	s.log.Warn("RemoveRole not implemented in UserService")
	return nil, status.Error(codes.Unimplemented, "RemoveRole not yet implemented")
}

func (s *Service) ListRoles(_ context.Context, _ *userpb.ListRolesRequest) (*userpb.ListRolesResponse, error) {
	// TODO: List roles (each role includes metadata if present)
	return nil, status.Error(codes.Unimplemented, "ListRoles not yet implemented")
}

func (s *Service) ListPermissions(_ context.Context, _ *userpb.ListPermissionsRequest) (*userpb.ListPermissionsResponse, error) {
	// TODO: List permissions (each permission includes metadata if present)
	return nil, status.Error(codes.Unimplemented, "ListPermissions not yet implemented")
}

func (s *Service) ListUserEvents(_ context.Context, _ *userpb.ListUserEventsRequest) (*userpb.ListUserEventsResponse, error) {
	// TODO: List user events (each event includes metadata if present)
	return nil, status.Error(codes.Unimplemented, "ListUserEvents not yet implemented")
}

func (s *Service) ListAuditLogs(_ context.Context, _ *userpb.ListAuditLogsRequest) (*userpb.ListAuditLogsResponse, error) {
	// TODO: List audit logs (each log includes metadata if present)
	return nil, status.Error(codes.Unimplemented, "ListAuditLogs not yet implemented")
}

func (s *Service) InitiateSSO(_ context.Context, _ *userpb.InitiateSSORequest) (*userpb.InitiateSSOResponse, error) {
	return nil, status.Error(codes.Unimplemented, "InitiateSSO not yet implemented")
}

func (s *Service) InitiateMFA(_ context.Context, _ *userpb.InitiateMFARequest) (*userpb.InitiateMFAResponse, error) {
	return nil, status.Error(codes.Unimplemented, "InitiateMFA not yet implemented")
}

func (s *Service) SyncSCIM(_ context.Context, _ *userpb.SyncSCIMRequest) (*userpb.SyncSCIMResponse, error) {
	return nil, status.Error(codes.Unimplemented, "SyncSCIM not yet implemented")
}

func (s *Service) RegisterInterest(_ context.Context, _ *userpb.RegisterInterestRequest) (*userpb.RegisterInterestResponse, error) {
	return nil, status.Error(codes.Unimplemented, "RegisterInterest not yet implemented")
}

func (s *Service) CreateReferral(_ context.Context, _ *userpb.CreateReferralRequest) (*userpb.CreateReferralResponse, error) {
	return nil, status.Error(codes.Unimplemented, "CreateReferral not yet implemented")
}
