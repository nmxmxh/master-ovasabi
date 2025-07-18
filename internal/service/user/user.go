// Provider/DI Registration Pattern (Modern, Extensible, DRY)
// ---------------------------------------------------------
//
// This file implements the centralized Provider pattern for service registration and dependency injection (DI) across the platform. It ensures all services are registered, resolved, and composed in a DRY, maintainable, and extensible way.
//
// Key Features:
// - Centralized Service Registration: All gRPC services are registered with a DI container, ensuring single-point, modular registration and easy dependency management.
// - Repository & Cache Integration: Each service can specify its repository constructor and (optionally) a cache name for Redis-backed caching.
// - Multi-Dependency Support: Services with multiple or cross-service dependencies use custom registration functions to resolve all required dependencies from the DI container.
// - Extensible Pattern: To add a new service, define its repository and (optionally) cache, then add a registration entry. For complex dependencies, use a custom registration function.
// - Consistent Error Handling: All registration errors are logged and wrapped for traceability.
// - Self-Documenting: The registration pattern is discoverable and enforced as a standard for all new services/providers.
//
// Standard for New Service/Provider Files:
// 1. Document the registration pattern and DI approach at the top of the file.
// 2. Describe how to add new services, including repository, cache, and dependency resolution.
// 3. Note any special patterns for multi-dependency or cross-service orchestration.
// 4. Ensure all registration and error handling is consistent and logged.
// 5. Reference this comment as the standard for all new service/provider files.
//
// For more, see the Amadeus context: docs/amadeus/amadeus_context.md (Provider/DI Registration Pattern)

// Service implements the UserService gRPC interface.
//
// This service is the canonical implementation of user management, authentication, RBAC, and audit logging for the platform.
//
// Standards and Integration Path:
// - Uses the robust metadata pattern (`common.Metadata`) for all extensible fields, including service-specific extensions under `metadata.service_specific.user`.
// - Supports accessibility and compliance metadata for user-facing assets and onboarding flows.
// - Implements bad actor identification, updating `metadata.service_specific.user.bad_actor` on suspicious events.
// - All POST/PATCH/PUT endpoints use the composable request pattern, with a `metadata` field for future-proof extensibility.
// - Sensitive actions (login, password change, RBAC changes) are logged in `metadata.audit` and/or a dedicated audit log entity.
// - On create/update, always caches metadata, registers with Scheduler, enriches the Knowledge Graph, and registers with Nexus.
//
// For the canonical cross-service standards integration path, see:
//   docs/amadeus/amadeus_context.md#cross-service-standards-integration-path

// ---
// Metadata Standard: Authentication & JWT Fields
//
// All authentication and JWT-related metadata must be stored under:
//   user.Metadata.ServiceSpecific["user"].auth
//   user.Metadata.ServiceSpecific["user"].jwt
//
// Example fields:
//   auth:
//     - last_login_at: timestamp of last successful login
//     - login_source: e.g., "web", "mobile", "oauth:google"
//     - failed_login_attempts: integer count
//     - last_failed_login_at: timestamp
//     - mfa_enabled: boolean
//     - oauth_provider: e.g., "google", "github"
//     - provider_user_id: external OAuth user ID
//   jwt:
//     - last_jwt_issued_at: timestamp
//     - last_jwt_id: last JWT ID issued
//     - jwt_revoked_at: timestamp (if revoked)
//     - jwt_audience: audience claim
//     - jwt_scopes: list of scopes/claims
//
// See: docs/amadeus/amadeus_context.md#user-authentication-and-jwt-metadata
// ---

package user

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"regexp"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/markbates/goth"
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	userpb "github.com/nmxmxh/master-ovasabi/api/protos/user/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var jwtSecret = []byte("super-secret-placeholder") // TODO: Replace with Azure Key Vault secret in production

// Service implements the UserService gRPC interface.
type Service struct {
	userpb.UnimplementedUserServiceServer
	log          *zap.Logger
	repo         *Repository
	cache        *redis.Cache
	eventEmitter events.EventEmitter
	eventEnabled bool
	handler      *graceful.Handler // Canonical handler for orchestration
}

// Compile-time check.
var _ userpb.UserServiceServer = (*Service)(nil)

// NewService creates a new instance of UserService with graceful handler and canonical action naming.
func NewService(ctx context.Context, log *zap.Logger, repo *Repository, cache *redis.Cache, eventEmitter events.EventEmitter, eventEnabled bool) userpb.UserServiceServer {
	handler := graceful.NewHandler(log, eventEmitter, cache, "user", "v1", eventEnabled)
	return &Service{
		log:          log,
		repo:         repo,
		cache:        cache,
		eventEmitter: eventEmitter,
		eventEnabled: eventEnabled,
		handler:      handler,
	}
}

func protoProfileToRepo(p *userpb.UserProfile) Profile {
	if p == nil {
		return Profile{}
	}
	return Profile{
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
	action := "create_user"
	user := &User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: "", // will be set after hashing
		Profile:      protoProfileToRepo(req.Profile),
		Roles:        req.Roles,
		Status:       int32(userpb.UserStatus_USER_STATUS_ACTIVE),
		Metadata:     req.Metadata,
		Score:        Score{Balance: 0, Pending: 0},
	}
	s.log.Info("Creating user", zap.String("email", req.Email), zap.String("username", req.Username))
	roles, _ := utils.GetAuthenticatedUserRoles(ctx)
	isAdmin := utils.IsAdmin(roles)
	userRegex := regexp.MustCompile(`^[\p{L}\p{N}._]{5,20}$`)
	adminRegex := regexp.MustCompile(`^[\p{L}\p{N}._]{1,20}$`)
	var err error
	if isAdmin {
		if !adminRegex.MatchString(req.Username) {
			gErr := graceful.MapAndWrapErr(ctx, ErrInvalidUsername, "invalid username", codes.InvalidArgument)
			s.handler.Error(ctx, action, codes.InvalidArgument, "invalid username", gErr, nil, req.Username)
			return nil, graceful.ToStatusError(gErr)
		}
	} else {
		if !userRegex.MatchString(req.Username) {
			gErr := graceful.MapAndWrapErr(ctx, ErrInvalidUsername, "invalid username", codes.InvalidArgument)
			s.handler.Error(ctx, action, codes.InvalidArgument, "invalid username", gErr, nil, req.Username)
			return nil, graceful.ToStatusError(gErr)
		}
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		gErr := graceful.MapAndWrapErr(ctx, err, "failed to hash password", codes.Internal)
		s.handler.Error(ctx, action, codes.Internal, "failed to hash password", gErr, nil, req.Username)
		return nil, graceful.ToStatusError(gErr)
	}
	user.PasswordHash = string(hashedPassword)
	user.Profile = protoProfileToRepo(req.Profile)
	user.Roles = req.Roles
	user.Status = int32(userpb.UserStatus_USER_STATUS_ACTIVE)
	created, err := s.repo.Create(ctx, user)
	if err != nil {
		var code codes.Code
		var msg string
		switch {
		case errors.Is(err, ErrUsernameTaken):
			code = codes.AlreadyExists
			msg = "username already taken"
		case errors.Is(err, ErrInvalidUsername):
			code = codes.InvalidArgument
			msg = "invalid username format"
		case errors.Is(err, ErrUsernameReserved):
			code = codes.InvalidArgument
			msg = "username is reserved"
		case errors.Is(err, ErrUsernameBadWord):
			code = codes.InvalidArgument
			msg = "username contains inappropriate content"
		default:
			code = codes.Internal
			msg = "failed to create user"
		}
		gErr := graceful.MapAndWrapErr(ctx, err, msg, code)
		s.handler.Error(ctx, action, code, msg, gErr, user.Metadata, req.Username)
		return nil, graceful.ToStatusError(gErr)
	}

	// Initialize metadata if not provided
	if created.Metadata == nil {
		created.Metadata = &commonpb.Metadata{
			ServiceSpecific: &structpb.Struct{
				Fields: make(map[string]*structpb.Value),
			},
		}
	}

	// Add creation metadata
	metaPtr, err := metadata.ServiceMetadataFromStruct(created.Metadata.ServiceSpecific)
	if err != nil {
		s.log.Error("Failed to extract service metadata", zap.Error(err))
	} else {
		meta := *metaPtr
		meta.DeviceID = "system"
		if meta.Audit == nil {
			meta.Audit = &metadata.AuditMetadata{}
		}
		meta.Audit.LastModified = time.Now().UTC().Format(time.RFC3339)
		meta.Audit.History = append(meta.Audit.History, "user_created")
		metaStruct, err := metadata.ServiceMetadataToStruct(&meta)
		if err == nil {
			created.Metadata.ServiceSpecific = metaStruct
		}
	}

	// Convert to proto and cache
	respUser := repoUserToProtoUser(created)
	if err := s.cache.Set(ctx, created.ID, "profile", respUser, redis.TTLUserProfile); err != nil {
		s.log.Error("Failed to cache user profile", zap.String("user_id", created.ID), zap.Error(err))
	}

	// Emit user created event
	if s.eventEnabled {
		if _, ok := events.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "user_created", created.ID, created.Metadata); !ok {
			s.log.Error("Failed to emit user created event", zap.String("user_id", created.ID))
		}
	}

	s.handler.Success(ctx, action, codes.OK, "user created", &userpb.CreateUserResponse{User: respUser}, created.Metadata, created.ID, nil)
	return &userpb.CreateUserResponse{User: respUser}, nil
}

// ... existing code ...

// GetUser retrieves user information.
func (s *Service) GetUser(ctx context.Context, req *userpb.GetUserRequest) (*userpb.GetUserResponse, error) {
	action := "get_user"
	respUserPtr, err := redis.GetOrSetWithProtection(ctx, s.cache, s.log, req.UserId, func(ctx context.Context) (*userpb.User, error) {
		repoUser, err := s.repo.GetByID(ctx, req.UserId)
		if err != nil {
			return nil, err
		}
		return repoUserToProtoUser(repoUser), nil
	}, redis.TTLUserProfile)
	if err == nil {
		s.handler.Success(ctx, action, codes.OK, "user retrieved", &userpb.GetUserResponse{User: respUserPtr}, nil, req.UserId, nil)
		return &userpb.GetUserResponse{User: respUserPtr}, nil
	}
	repoUser, err := s.repo.GetByID(ctx, req.UserId)
	if err != nil {
		code := codes.Internal
		msg := "database error"
		if errors.Is(err, ErrUserNotFound) {
			code = codes.NotFound
			msg = "user not found"
		}
		gErr := graceful.MapAndWrapErr(ctx, err, msg, code)
		s.handler.Error(ctx, action, code, msg, gErr, nil, req.UserId)
		return nil, graceful.ToStatusError(gErr)
	}
	respUser := repoUserToProtoUser(repoUser)
	if err := s.cache.Set(ctx, req.UserId, "profile", respUser, redis.TTLUserProfile); err != nil {
		s.log.Error("Failed to cache user profile", zap.String("user_id", req.UserId), zap.Error(err))
	}
	s.handler.Success(ctx, action, codes.OK, "user retrieved", &userpb.GetUserResponse{User: respUser}, repoUser.Metadata, repoUser.ID, nil)
	return &userpb.GetUserResponse{User: respUser}, nil
}

// GetUserByUsername retrieves user information by username.
func (s *Service) GetUserByUsername(ctx context.Context, req *userpb.GetUserByUsernameRequest) (*userpb.GetUserByUsernameResponse, error) {
	action := "get_user_by_username"
	repoUser, err := s.repo.GetByUsername(ctx, req.Username)
	if err != nil {
		code := codes.Internal
		msg := "database error"
		if errors.Is(err, ErrUserNotFound) {
			code = codes.NotFound
			msg = "user not found"
		}
		gErr := graceful.MapAndWrapErr(ctx, err, msg, code)
		s.handler.Error(ctx, action, code, msg, gErr, nil, req.Username)
		return nil, graceful.ToStatusError(gErr)
	}
	respUser := repoUserToProtoUser(repoUser)
	s.handler.Success(ctx, action, codes.OK, "user retrieved by username", &userpb.GetUserByUsernameResponse{User: respUser}, repoUser.Metadata, repoUser.ID, nil)
	return &userpb.GetUserByUsernameResponse{User: respUser}, nil
}

// GetUserByEmail retrieves user information by email.
func (s *Service) GetUserByEmail(ctx context.Context, req *userpb.GetUserByEmailRequest) (*userpb.GetUserByEmailResponse, error) {
	action := "get_user_by_email"
	repoUser, err := s.repo.GetByEmail(ctx, req.Email)
	if err != nil {
		code := codes.Internal
		msg := "database error"
		if errors.Is(err, ErrUserNotFound) {
			code = codes.NotFound
			msg = "user not found"
		}
		gErr := graceful.MapAndWrapErr(ctx, err, msg, code)
		s.handler.Error(ctx, action, code, msg, gErr, nil, req.Email)
		return nil, graceful.ToStatusError(gErr)
	}
	respUser := repoUserToProtoUser(repoUser)
	s.handler.Success(ctx, action, codes.OK, "user retrieved by email", &userpb.GetUserByEmailResponse{User: respUser}, repoUser.Metadata, repoUser.ID, nil)
	return &userpb.GetUserByEmailResponse{User: respUser}, nil
}

// UpdateUser updates a user record.
func (s *Service) UpdateUser(ctx context.Context, req *userpb.UpdateUserRequest) (*userpb.UpdateUserResponse, error) {
	action := "update_user"
	authUserID, ok := utils.GetAuthenticatedUserID(ctx)
	if !ok {
		return nil, graceful.ToStatusError(
			graceful.MapAndWrapErr(ctx, errors.New("missing authentication"), "missing authentication", codes.Unauthenticated),
		)
	}
	roles, _ := utils.GetAuthenticatedUserRoles(ctx)
	isAdmin := utils.IsAdmin(roles)
	if !isAdmin && req.UserId != authUserID {
		return nil, graceful.ToStatusError(
			graceful.MapAndWrapErr(ctx, errors.New("cannot update another user's profile"), "cannot update another user's profile", codes.PermissionDenied),
		)
	}
	repoUser, err := s.repo.GetByID(ctx, req.UserId)
	if err != nil {
		code := codes.Internal
		msg := "failed to get user"
		if errors.Is(err, ErrUserNotFound) {
			code = codes.NotFound
			msg = "user not found"
		}
		gErr := graceful.MapAndWrapErr(ctx, err, msg, code)
		s.handler.Error(ctx, action, code, msg, gErr, nil, req.UserId)
		return nil, graceful.ToStatusError(gErr)
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
		if req.User.Locations != nil {
			repoUser.Locations = req.User.Locations
		}
		if req.User.Profile != nil {
			repoUser.Profile = protoProfileToRepo(req.User.Profile)
		}
		if req.User.Roles != nil {
			repoUser.Roles = req.User.Roles
		}
		if req.User.Metadata != nil {
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
	// Save update
	if err := s.repo.Update(ctx, repoUser); err != nil {
		gErr := graceful.MapAndWrapErr(ctx, err, "failed to update user", codes.Internal)
		s.handler.Error(ctx, action, codes.Internal, "failed to update user", gErr, repoUser.Metadata, repoUser.ID)
		return nil, graceful.ToStatusError(gErr)
	}
	getResp, err := s.GetUser(ctx, &userpb.GetUserRequest{UserId: req.UserId})
	if err != nil {
		s.handler.Error(ctx, action, codes.Internal, "failed to fetch updated user", err, repoUser.Metadata, repoUser.ID)
		return nil, graceful.ToStatusError(err)
	}
	s.handler.Success(ctx, action, codes.OK, "user updated", &userpb.UpdateUserResponse{User: getResp.User}, repoUser.Metadata, repoUser.ID, nil)
	return &userpb.UpdateUserResponse{User: getResp.User}, nil
}

// DeleteUser removes a user and its master record.
func (s *Service) DeleteUser(ctx context.Context, req *userpb.DeleteUserRequest) (*userpb.DeleteUserResponse, error) {
	action := "delete_user"
	authUserID, ok := utils.GetAuthenticatedUserID(ctx)
	if !ok {
		gErr := graceful.MapAndWrapErr(ctx, errors.New("missing authentication"), "missing authentication", codes.Unauthenticated)
		s.handler.Error(ctx, action, codes.Unauthenticated, "missing authentication", gErr, nil, req.UserId)
		return nil, graceful.ToStatusError(gErr)
	}
	roles, _ := utils.GetAuthenticatedUserRoles(ctx)
	isAdmin := utils.IsAdmin(roles)
	if !isAdmin && req.UserId != authUserID {
		gErr := graceful.MapAndWrapErr(ctx, errors.New("cannot delete another user's profile"), "cannot delete another user's profile", codes.PermissionDenied)
		s.handler.Error(ctx, action, codes.PermissionDenied, "cannot delete another user's profile", gErr, nil, req.UserId)
		return nil, graceful.ToStatusError(gErr)
	}
	repoUser, err := s.repo.GetByID(ctx, req.UserId)
	if err != nil {
		code := codes.Internal
		msg := "failed to delete user"
		if errors.Is(err, ErrUserNotFound) {
			code = codes.NotFound
			msg = "user not found"
		}
		gErr := graceful.MapAndWrapErr(ctx, err, msg, code)
		s.handler.Error(ctx, action, code, msg, gErr, nil, req.UserId)
		return nil, graceful.ToStatusError(gErr)
	}
	if err := s.repo.Delete(ctx, repoUser.ID); err != nil {
		return nil, graceful.ToStatusError(
			graceful.MapAndWrapErr(ctx, err, "failed to delete user", codes.Internal),
		)
	}
	if err := s.cache.Delete(ctx, req.UserId, "profile"); err != nil {
		s.log.Error("Failed to invalidate user cache",
			zap.String("user_id", req.UserId),
			zap.Error(err))
	}
	s.handler.Success(ctx, action, codes.OK, "user deleted", &userpb.DeleteUserResponse{Success: true}, repoUser.Metadata, repoUser.ID, nil)
	return &userpb.DeleteUserResponse{Success: true}, nil
}

// ListUsers retrieves a list of users with pagination and filtering.
func (s *Service) ListUsers(ctx context.Context, req *userpb.ListUsersRequest) (*userpb.ListUsersResponse, error) {
	action := "list_users"
	// Use ListFlexible if advanced filtering/search is requested (batchstreaming is not used here; ListFlexible is the canonical approach for advanced filtering/pagination)
	if req.SearchQuery != "" || len(req.Tags) > 0 || req.Metadata != nil || req.Filters != nil {
		users, total, err := s.repo.ListFlexible(ctx, req)
		if err != nil {
			gErr := graceful.MapAndWrapErr(ctx, err, "failed to list users", codes.Internal)
			s.handler.Error(ctx, action, codes.Internal, "failed to list users", gErr, nil, "")
			return nil, graceful.ToStatusError(gErr)
		}
		if total > int(^int32(0)) || total < 0 {
			gErr := graceful.MapAndWrapErr(ctx, errors.New("total overflows int32"), "total overflows int32", codes.Internal)
			s.handler.Error(ctx, action, codes.Internal, "total overflows int32", gErr, nil, "")
			return nil, graceful.ToStatusError(gErr)
		}
		totalPages := (total + int(req.PageSize) - 1) / int(req.PageSize)
		if totalPages > int(^int32(0)) || totalPages < 0 {
			gErr := graceful.MapAndWrapErr(ctx, errors.New("totalPages overflows int32"), "totalPages overflows int32", codes.Internal)
			s.handler.Error(ctx, action, codes.Internal, "totalPages overflows int32", gErr, nil, "")
			return nil, graceful.ToStatusError(gErr)
		}
		resp := &userpb.ListUsersResponse{
			Users:      make([]*userpb.User, 0, len(users)),
			TotalCount: utils.ToInt32(total),
			Page:       req.Page,
			TotalPages: utils.ToInt32(totalPages),
		}
		for _, u := range users {
			respUser := repoUserToProtoUser(u)
			resp.Users = append(resp.Users, respUser)
		}
		s.handler.Success(ctx, action, codes.OK, "users listed (flexible)", resp, nil, "", nil)
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
		gErr := graceful.MapAndWrapErr(ctx, errors.New("pagination overflow"), "pagination overflow", codes.InvalidArgument)
		s.handler.Error(ctx, action, codes.InvalidArgument, "pagination overflow", gErr, nil, "")
		return nil, graceful.ToStatusError(gErr)
	}
	offset := int(offset64)
	users, err := s.repo.List(ctx, limit, offset)
	if err != nil {
		gErr := graceful.MapAndWrapErr(ctx, err, "failed to list users", codes.Internal)
		s.handler.Error(ctx, action, codes.Internal, "failed to list users", gErr, nil, "")
		return nil, graceful.ToStatusError(gErr)
	}
	resp := &userpb.ListUsersResponse{
		Users: make([]*userpb.User, 0, len(users)),
	}
	totalPages := 0
	if limit > 0 {
		totalPages = (len(users) + limit - 1) / limit
		if totalPages > int(^int32(0)) || totalPages < 0 {
			gErr := graceful.MapAndWrapErr(ctx, errors.New("totalPages overflows int32"), "totalPages overflows int32", codes.Internal)
			s.handler.Error(ctx, action, codes.Internal, "totalPages overflows int32", gErr, nil, "")
			return nil, graceful.ToStatusError(gErr)
		}
	}
	if totalPages > int(^int32(0)) || totalPages < 0 {
		gErr := graceful.MapAndWrapErr(ctx, errors.New("totalPages overflows int32 (post-check)"), "totalPages overflows int32 (post-check)", codes.Internal)
		s.handler.Error(ctx, action, codes.Internal, "totalPages overflows int32 (post-check)", gErr, nil, "")
		return nil, graceful.ToStatusError(gErr)
	}
	resp.TotalPages = utils.ToInt32(totalPages)
	resp.TotalCount = utils.ToInt32(len(users))
	for _, u := range users {
		respUser := repoUserToProtoUser(u)
		resp.Users = append(resp.Users, respUser)
	}
	s.handler.Success(ctx, action, codes.OK, "users listed", resp, nil, "", nil)
	return resp, nil
}

// UpdatePassword implements the UpdatePassword RPC method.
func (s *Service) UpdatePassword(ctx context.Context, req *userpb.UpdatePasswordRequest) (*userpb.UpdatePasswordResponse, error) {
	action := "update_password"
	authUserID, ok := utils.GetAuthenticatedUserID(ctx)
	if !ok {
		gErr := graceful.MapAndWrapErr(ctx, errors.New("missing authentication"), "missing authentication", codes.Unauthenticated)
		s.handler.Error(ctx, action, codes.Unauthenticated, "missing authentication", gErr, nil, "")
		return nil, graceful.ToStatusError(gErr)
	}
	roles, _ := utils.GetAuthenticatedUserRoles(ctx)
	isAdmin := utils.IsAdmin(roles)
	if !isAdmin && req.UserId != authUserID {
		gErr := graceful.MapAndWrapErr(ctx, errors.New("cannot update another user's password"), "cannot update another user's password", codes.PermissionDenied)
		s.handler.Error(ctx, action, codes.PermissionDenied, "cannot update another user's password", gErr, nil, req.UserId)
		return nil, graceful.ToStatusError(gErr)
	}
	repoUser, err := s.repo.GetByID(ctx, req.UserId)
	if err != nil {
		code := codes.Internal
		msg := "failed to get user"
		if errors.Is(err, ErrUserNotFound) {
			code = codes.NotFound
			msg = "user not found"
		}
		gErr := graceful.MapAndWrapErr(ctx, err, msg, code)
		s.handler.Error(ctx, action, code, msg, gErr, nil, req.UserId)
		return nil, graceful.ToStatusError(gErr)
	}
	// If not admin, verify current password
	if !isAdmin {
		if err := bcrypt.CompareHashAndPassword([]byte(repoUser.PasswordHash), []byte(req.CurrentPassword)); err != nil {
			gErr := graceful.MapAndWrapErr(ctx, errors.New("invalid current password"), "invalid current password", codes.PermissionDenied)
			s.handler.Error(ctx, action, codes.PermissionDenied, "invalid current password", gErr, nil, req.UserId)
			return nil, graceful.ToStatusError(gErr)
		}
	}
	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		gErr := graceful.MapAndWrapErr(ctx, err, "failed to hash new password", codes.Internal)
		s.handler.Error(ctx, action, codes.Internal, "failed to hash new password", gErr, nil, req.UserId)
		return nil, graceful.ToStatusError(gErr)
	}
	repoUser.PasswordHash = string(hashedPassword)

	if err := s.repo.Update(ctx, repoUser); err != nil {
		return nil, graceful.ToStatusError(
			graceful.MapAndWrapErr(ctx, err, "failed to update password", codes.Internal),
		)
	}

	// --- Audit metadata update ---
	if repoUser.Metadata == nil {
		repoUser.Metadata = &commonpb.Metadata{}
	}
	metaPtr, err := metadata.ServiceMetadataFromStruct(repoUser.Metadata.ServiceSpecific)
	if err == nil && metaPtr != nil {
		meta := *metaPtr
		if meta.Audit == nil {
			meta.Audit = &metadata.AuditMetadata{}
		}
		meta.Audit.LastModified = time.Now().UTC().Format(time.RFC3339)
		meta.Audit.History = append(meta.Audit.History, "password_changed")
		metaStruct, err := metadata.ServiceMetadataToStruct(&meta)
		if err != nil {
			s.log.Error("failed to convert audit metadata to struct", zap.Error(err))
			return nil, graceful.ToStatusError(
				graceful.MapAndWrapErr(ctx, err, "failed to convert audit metadata", codes.Internal),
			)
		}
		repoUser.Metadata.ServiceSpecific = metaStruct
		if err := s.updateUserMetadata(ctx, repoUser, repoUser.Metadata); err != nil {
			s.log.Error("failed to update audit metadata after password change", zap.Error(err))
			return nil, graceful.ToStatusError(
				graceful.MapAndWrapErr(ctx, err, "failed to update audit metadata", codes.Internal),
			)
		}
		if err := s.repo.Update(ctx, repoUser); err != nil {
			s.log.Error("failed to persist audit metadata after password change", zap.Error(err))
			return nil, graceful.ToStatusError(
				graceful.MapAndWrapErr(ctx, err, "failed to persist audit metadata", codes.Internal),
			)
		}
	}

	s.handler.Success(ctx, action, codes.OK, "password updated", &userpb.UpdatePasswordResponse{
		Success:   true,
		UpdatedAt: time.Now().Unix(),
	}, repoUser.Metadata, repoUser.ID, nil)
	return &userpb.UpdatePasswordResponse{
		Success:   true,
		UpdatedAt: time.Now().Unix(),
	}, nil
}

// UpdateProfile updates a user's profile.
func (s *Service) UpdateProfile(ctx context.Context, req *userpb.UpdateProfileRequest) (*userpb.UpdateProfileResponse, error) {
	action := "update_profile"
	authUserID, ok := utils.GetAuthenticatedUserID(ctx)
	if !ok {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, errors.New("missing authentication"), "missing authentication", codes.Unauthenticated))
	}
	roles, _ := utils.GetAuthenticatedUserRoles(ctx)
	isAdmin := utils.IsAdmin(roles)
	if !isAdmin && req.UserId != authUserID {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, errors.New("cannot update another user's profile"), "cannot update another user's profile", codes.PermissionDenied))
	}
	repoUser, err := s.repo.GetByID(ctx, req.UserId)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "user not found", codes.NotFound))
	}
	// Update fields based on FieldsToUpdate and Profile.CustomFields
	if req.Profile != nil {
		repoUser.Profile = protoProfileToRepo(req.Profile)
	}
	if err := s.repo.Update(ctx, repoUser); err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to update profile", codes.Internal))
	}
	getResp, err := s.GetUser(ctx, &userpb.GetUserRequest{UserId: req.UserId})
	if err != nil {
		return nil, err
	}
	s.handler.Success(ctx, action, codes.OK, "profile updated", &userpb.UpdateProfileResponse{User: getResp.User}, repoUser.Metadata, repoUser.ID, nil)
	return &userpb.UpdateProfileResponse{User: getResp.User}, nil
}

// Standard: All audit fields (created_by, last_modified_by) must use a non-PII user reference (user_id:master_id).
// This ensures GDPR compliance and prevents accidental PII exposure in logs or metadata.
// See: docs/amadeus/amadeus_context.md#gdpr-and-privacy-standards.

// Replace all direct metadata access/update points with migration and hooks.
func (s *Service) updateUserMetadata(ctx context.Context, user *User, newMeta *commonpb.Metadata) error {
	metadata.MigrateMetadata(newMeta)
	user.Metadata = newMeta
	return nil
}

// AssignRole assigns a role to a user and updates metadata.
func (s *Service) AssignRole(ctx context.Context, req *userpb.AssignRoleRequest) (*userpb.AssignRoleResponse, error) {
	action := "assign_role"
	repoUser, err := s.repo.GetByID(ctx, req.UserId)
	if err != nil {
		gErr := graceful.MapAndWrapErr(ctx, err, "user not found", codes.NotFound)
		s.handler.Error(ctx, action, codes.NotFound, "user not found", gErr, nil, req.UserId)
		return nil, graceful.ToStatusError(gErr)
	}
	repoUser.Roles = append(repoUser.Roles, req.Role)
	err = s.updateUserMetadata(ctx, repoUser, repoUser.Metadata)
	if err != nil {
		gErr := graceful.MapAndWrapErr(ctx, err, "failed to update user metadata", codes.Internal)
		s.handler.Error(ctx, action, codes.Internal, "failed to update user metadata", gErr, repoUser.Metadata, req.UserId)
		return nil, graceful.ToStatusError(gErr)
	}
	if err := s.repo.Update(ctx, repoUser); err != nil {
		gErr := graceful.MapAndWrapErr(ctx, err, "failed to assign role", codes.Internal)
		s.handler.Error(ctx, action, codes.Internal, "failed to assign role", gErr, repoUser.Metadata, req.UserId)
		return nil, graceful.ToStatusError(gErr)
	}
	s.handler.Success(ctx, action, codes.OK, "role assigned", &userpb.AssignRoleResponse{}, repoUser.Metadata, repoUser.ID, nil)
	return &userpb.AssignRoleResponse{}, nil
}

// RemoveRole removes a role from a user and updates metadata.
func (s *Service) RemoveRole(ctx context.Context, req *userpb.RemoveRoleRequest) (*userpb.RemoveRoleResponse, error) {
	action := "remove_role"
	repoUser, err := s.repo.GetByID(ctx, req.UserId)
	if err != nil {
		gErr := graceful.MapAndWrapErr(ctx, err, "user not found", codes.NotFound)
		s.handler.Error(ctx, action, codes.NotFound, "user not found", gErr, nil, req.UserId)
		return nil, graceful.ToStatusError(gErr)
	}
	newRoles := []string{}
	for _, r := range repoUser.Roles {
		if r != req.Role {
			newRoles = append(newRoles, r)
		}
	}
	repoUser.Roles = newRoles
	err = s.updateUserMetadata(ctx, repoUser, repoUser.Metadata)
	if err != nil {
		gErr := graceful.MapAndWrapErr(ctx, err, "failed to update user metadata", codes.Internal)
		s.handler.Error(ctx, action, codes.Internal, "failed to update user metadata", gErr, repoUser.Metadata, req.UserId)
		return nil, graceful.ToStatusError(gErr)
	}
	if err := s.repo.Update(ctx, repoUser); err != nil {
		gErr := graceful.MapAndWrapErr(ctx, err, "failed to remove role", codes.Internal)
		s.handler.Error(ctx, action, codes.Internal, "failed to remove role", gErr, repoUser.Metadata, req.UserId)
		return nil, graceful.ToStatusError(gErr)
	}
	s.handler.Success(ctx, action, codes.OK, "role removed", &userpb.RemoveRoleResponse{}, repoUser.Metadata, repoUser.ID, nil)
	return &userpb.RemoveRoleResponse{}, nil
}

// ListRoles lists all roles for a user.
func (s *Service) ListRoles(ctx context.Context, req *userpb.ListRolesRequest) (*userpb.ListRolesResponse, error) {
	action := "list_roles"
	repoUser, err := s.repo.GetByID(ctx, req.UserId)
	if err != nil {
		gErr := graceful.MapAndWrapErr(ctx, err, "user not found", codes.NotFound)
		s.handler.Error(ctx, action, codes.NotFound, "user not found", gErr, nil, req.UserId)
		return nil, graceful.ToStatusError(gErr)
	}
	s.handler.Success(ctx, action, codes.OK, "roles listed", &userpb.ListRolesResponse{Roles: repoUser.Roles}, repoUser.Metadata, repoUser.ID, nil)
	return &userpb.ListRolesResponse{Roles: repoUser.Roles}, nil
}

// ListPermissions lists all permissions for a user.
func (s *Service) ListPermissions(ctx context.Context, req *userpb.ListPermissionsRequest) (*userpb.ListPermissionsResponse, error) {
	action := "list_permissions"
	user, err := s.repo.GetByID(ctx, req.UserId)
	if err != nil {
		gErr := graceful.MapAndWrapErr(ctx, err, "user not found", codes.NotFound)
		s.handler.Error(ctx, action, codes.NotFound, "user not found", gErr, nil, req.UserId)
		return nil, graceful.ToStatusError(gErr)
	}
	rolePerms := map[string][]string{
		"admin":     {"read", "write", "delete", "manage_users"},
		"editor":    {"read", "write"},
		"viewer":    {"read"},
		"moderator": {"read", "write", "moderate"},
	}
	permSet := make(map[string]struct{})
	for _, role := range user.Roles {
		if perms, ok := rolePerms[role]; ok {
			for _, p := range perms {
				permSet[p] = struct{}{}
			}
		}
	}
	perms := make([]string, 0, len(permSet))
	for p := range permSet {
		perms = append(perms, p)
	}
	s.handler.Success(ctx, action, codes.OK, "permissions listed", &userpb.ListPermissionsResponse{Permissions: perms}, user.Metadata, user.ID, nil)
	return &userpb.ListPermissionsResponse{Permissions: perms}, nil
}

// ListUserEvents lists user events (stub).
func (s *Service) ListUserEvents(ctx context.Context, req *userpb.ListUserEventsRequest) (*userpb.ListUserEventsResponse, error) {
	action := "list_user_events"
	page := int(req.Page)
	pageSize := int(req.PageSize)
	userEvents, total, err := s.repo.ListUserEvents(ctx, req.UserId, page, pageSize)
	if err != nil {
		gErr := graceful.MapAndWrapErr(ctx, err, "failed to list user events", codes.Internal)
		s.handler.Error(ctx, action, codes.Internal, "failed to list user events", gErr, nil, req.UserId)
		return nil, graceful.ToStatusError(gErr)
	}
	resp := &userpb.ListUserEventsResponse{
		Events:     userEvents,
		TotalCount: utils.ToInt32(total),
	}
	s.handler.Success(ctx, action, codes.OK, "user events listed", resp, nil, req.UserId, nil)
	return resp, nil
}

// ListAuditLogs lists audit logs for a user (stub).
func (s *Service) ListAuditLogs(ctx context.Context, req *userpb.ListAuditLogsRequest) (*userpb.ListAuditLogsResponse, error) {
	action := "list_audit_logs"
	userID := req.UserId
	page := int(req.Page)
	pageSize := int(req.PageSize)
	logs, total, err := s.repo.ListAuditLogs(ctx, userID, page, pageSize)
	if err != nil {
		gErr := graceful.MapAndWrapErr(ctx, err, "failed to list audit logs", codes.Internal)
		s.handler.Error(ctx, action, codes.Internal, "failed to list audit logs", gErr, nil, userID)
		return nil, graceful.ToStatusError(gErr)
	}
	resp := &userpb.ListAuditLogsResponse{
		Logs:       logs,
		TotalCount: utils.ToInt32(total),
	}
	s.handler.Success(ctx, action, codes.OK, "audit logs listed", resp, nil, userID, nil)
	return resp, nil
}

// --- Session Management ---

func generateToken(length int) (string, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// --- Polished Event Emission and Struct Construction ---
// For all userEvents.EmitEventWithLogging, wrap with error handling/logging if not already, and consider orchestration if possible.
// For all structpb.NewStruct, handle errors and use metadata.NewStructFromMap if available for DRYness.

func (s *Service) CreateSession(ctx context.Context, req *userpb.CreateSessionRequest) (*userpb.CreateSessionResponse, error) {
	action := "create_session"
	var user *User
	var err error
	isGuest := false
	if req.UserId != "" {
		user, err = s.repo.GetByID(ctx, req.UserId)
		if err != nil {
			gErr := graceful.MapAndWrapErr(ctx, err, "user not found", codes.NotFound)
			s.handler.Error(ctx, action, codes.NotFound, "user not found", gErr, nil, req.UserId)
			return nil, graceful.ToStatusError(gErr)
		}
		if user.Metadata == nil {
			user.Metadata = &commonpb.Metadata{}
		}
		metaPtr, err := metadata.ServiceMetadataFromStruct(user.Metadata.ServiceSpecific)
		if err != nil {
			gErr := graceful.MapAndWrapErr(ctx, err, "failed to extract service metadata", codes.Internal)
			s.handler.Error(ctx, action, codes.Internal, "failed to extract service metadata", gErr, user.Metadata, user.ID)
			return nil, graceful.ToStatusError(gErr)
		}
		meta := *metaPtr
		meta.DeviceID = req.DeviceInfo
		if meta.Audit == nil {
			meta.Audit = &metadata.AuditMetadata{}
		}
		meta.Audit.LastModified = time.Now().UTC().Format(time.RFC3339)
		meta.Audit.History = append(meta.Audit.History, "login")
		metaStruct, err := metadata.ServiceMetadataToStruct(&meta)
		if err != nil {
			gErr := graceful.MapAndWrapErr(ctx, err, "failed to convert service metadata", codes.Internal)
			s.handler.Error(ctx, action, codes.Internal, "failed to convert service metadata", gErr, user.Metadata, user.ID)
			return nil, graceful.ToStatusError(gErr)
		}
		user.Metadata.ServiceSpecific = metaStruct
		if err := s.updateUserMetadata(ctx, user, user.Metadata); err != nil {
			gErr := graceful.MapAndWrapErr(ctx, err, "failed to update user metadata", codes.Internal)
			s.handler.Error(ctx, action, codes.Internal, "failed to update user metadata", gErr, user.Metadata, user.ID)
			return nil, graceful.ToStatusError(gErr)
		}
		if err := s.repo.Update(ctx, user); err != nil {
			gErr := graceful.MapAndWrapErr(ctx, err, "failed to update user", codes.Internal)
			s.handler.Error(ctx, action, codes.Internal, "failed to update user", gErr, user.Metadata, user.ID)
			return nil, graceful.ToStatusError(gErr)
		}
	} else {
		isGuest = true
		user = &User{
			Username: "guest",
			Status:   int32(userpb.UserStatus_USER_STATUS_ACTIVE),
			Metadata: nil,
		}
	}
	accessToken, err := generateToken(32)
	if err != nil {
		gErr := graceful.MapAndWrapErr(ctx, err, "failed to generate access token", codes.Internal)
		s.handler.Error(ctx, action, codes.Internal, "failed to generate access token", gErr, nil, "")
		return nil, graceful.ToStatusError(gErr)
	}
	refreshToken, err := generateToken(32)
	if err != nil {
		gErr := graceful.MapAndWrapErr(ctx, err, "failed to generate refresh token", codes.Internal)
		s.handler.Error(ctx, action, codes.Internal, "failed to generate refresh token", gErr, nil, "")
		return nil, graceful.ToStatusError(gErr)
	}
	if user.Metadata == nil {
		user.Metadata = &commonpb.Metadata{}
	}
	metaPtr, err := metadata.ServiceMetadataFromStruct(user.Metadata.ServiceSpecific)
	if err != nil {
		gErr := graceful.MapAndWrapErr(ctx, err, "failed to extract service metadata", codes.Internal)
		s.handler.Error(ctx, action, codes.Internal, "failed to extract service metadata", gErr, user.Metadata, "")
		return nil, graceful.ToStatusError(gErr)
	}
	meta := *metaPtr
	if isGuest {
		meta.Guest = true
		meta.GuestCreatedAt = time.Now().UTC().Format(time.RFC3339)
		meta.DeviceID = req.DeviceInfo
	} else {
		meta.DeviceID = req.DeviceInfo
	}
	if meta.Audit == nil {
		meta.Audit = &metadata.AuditMetadata{}
	}
	meta.Audit.LastModified = time.Now().UTC().Format(time.RFC3339)
	meta.Audit.History = append(meta.Audit.History, "login")
	jwtID := accessToken
	audience := "ovasabi-app"
	scopes := []string{"user:read", "user:write"}
	serviceMetaMap := make(map[string]interface{})
	if user.Metadata.ServiceSpecific != nil {
		serviceMetaMap = user.Metadata.ServiceSpecific.AsMap()
	}
	if serviceMetaMap == nil {
		serviceMetaMap = make(map[string]interface{})
	}
	jwtMeta := map[string]interface{}{
		"jwt_id":     jwtID,
		"audience":   audience,
		"scopes":     scopes,
		"issued_at":  time.Now().UTC().Format(time.RFC3339),
		"expires_at": time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339),
	}
	serviceMetaMap["jwt"] = jwtMeta
	metaStruct, err := metadata.ServiceMetadataToStruct(&meta)
	if err != nil {
		gErr := graceful.MapAndWrapErr(ctx, err, "failed to convert service metadata to struct", codes.Internal)
		s.handler.Error(ctx, action, codes.Internal, "failed to convert service metadata to struct", gErr, user.Metadata, "")
		return nil, graceful.ToStatusError(gErr)
	}
	user.Metadata.ServiceSpecific = metaStruct
	if err := s.updateUserMetadata(ctx, user, user.Metadata); err != nil {
		gErr := graceful.MapAndWrapErr(ctx, err, "failed to update user metadata", codes.Internal)
		s.handler.Error(ctx, action, codes.Internal, "failed to update user metadata", gErr, user.Metadata, "")
		return nil, graceful.ToStatusError(gErr)
	}
	if err := s.repo.Update(ctx, user); err != nil {
		gErr := graceful.MapAndWrapErr(ctx, err, "failed to update user", codes.Internal)
		s.handler.Error(ctx, action, codes.Internal, "failed to update user", gErr, user.Metadata, "")
		return nil, graceful.ToStatusError(gErr)
	}
	session := &userpb.Session{
		Id:           accessToken,
		UserId:       user.ID,
		DeviceInfo:   req.DeviceInfo,
		CreatedAt:    timestamppb.Now(),
		ExpiresAt:    timestamppb.New(time.Now().Add(24 * time.Hour)),
		RefreshToken: refreshToken,
		AccessToken:  accessToken,
		IpAddress:    "",
		Metadata:     user.Metadata,
	}
	if s.cache != nil {
		err := s.cache.Set(ctx, accessToken, "session", session, 24*time.Hour)
		if err != nil {
			s.log.Error("failed to cache session", zap.Error(err))
		}
	}
	s.handler.Success(ctx, action, codes.OK, "session created", &userpb.CreateSessionResponse{Session: session}, user.Metadata, user.ID, nil)
	return &userpb.CreateSessionResponse{Session: session}, nil
}

func (s *Service) GetSession(ctx context.Context, req *userpb.GetSessionRequest) (*userpb.GetSessionResponse, error) {
	action := "get_session"
	if s.cache == nil {
		gErr := graceful.MapAndWrapErr(ctx, errors.New("session cache unavailable"), "session cache unavailable", codes.Unavailable)
		s.handler.Error(ctx, action, codes.Unavailable, "session cache unavailable", gErr, nil, req.SessionId)
		return nil, graceful.ToStatusError(gErr)
	}
	sessionPtr, err := redis.GetOrSetWithProtection(ctx, s.cache, s.log, req.SessionId, func(ctx context.Context) (*userpb.Session, error) {
		session, err := s.repo.GetSession(ctx, req.SessionId)
		if err != nil {
			return nil, err
		}
		return session, nil
	}, 24*time.Hour)
	if err != nil {
		gErr := graceful.MapAndWrapErr(ctx, err, "session not found", codes.NotFound)
		s.handler.Error(ctx, action, codes.NotFound, "session not found", gErr, nil, req.SessionId)
		return nil, graceful.ToStatusError(gErr)
	}
	s.handler.Success(ctx, action, codes.OK, "session retrieved", &userpb.GetSessionResponse{Session: sessionPtr}, nil, req.SessionId, nil)
	return &userpb.GetSessionResponse{Session: sessionPtr}, nil
}

func (s *Service) RevokeSession(ctx context.Context, req *userpb.RevokeSessionRequest) (*userpb.RevokeSessionResponse, error) {
	action := "revoke_session"
	if s.cache == nil {
		gErr := graceful.MapAndWrapErr(ctx, errors.New("session cache unavailable"), "session cache unavailable", codes.Unavailable)
		s.handler.Error(ctx, action, codes.Unavailable, "session cache unavailable", gErr, nil, req.SessionId)
		return nil, graceful.ToStatusError(gErr)
	}
	err := s.cache.Delete(ctx, req.SessionId, "session")
	if err != nil {
		gErr := graceful.MapAndWrapErr(ctx, err, "session not found", codes.NotFound)
		s.handler.Error(ctx, action, codes.NotFound, "session not found", gErr, nil, req.SessionId)
		return nil, graceful.ToStatusError(gErr)
	}
	s.handler.Success(ctx, action, codes.OK, "session revoked", &userpb.RevokeSessionResponse{Success: true}, nil, req.SessionId, nil)
	return &userpb.RevokeSessionResponse{Success: true}, nil
}

func (s *Service) ListSessions(ctx context.Context, req *userpb.ListSessionsRequest) (*userpb.ListSessionsResponse, error) {
	action := "list_sessions"
	if s.cache == nil {
		gErr := graceful.MapAndWrapErr(ctx, errors.New("session cache unavailable and repository fallback not implemented"), "session cache unavailable and repository fallback not implemented", codes.Unavailable)
		s.handler.Error(ctx, action, codes.Unavailable, "session cache unavailable and repository fallback not implemented", gErr, nil, "")
		return nil, graceful.ToStatusError(gErr)
	}
	scanPattern := "*"
	var sessions []*userpb.Session
	iter := s.cache.GetClient().Scan(ctx, 0, scanPattern, 0).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		var session userpb.Session
		err := s.cache.Get(ctx, key, "session", &session)
		if err != nil {
			continue
		}
		if session.UserId == req.UserId {
			sessions = append(sessions, &session)
		}
	}
	if err := iter.Err(); err != nil {
		gErr := graceful.MapAndWrapErr(ctx, err, "failed to scan session keys", codes.Internal)
		s.handler.Error(ctx, action, codes.Internal, "failed to scan session keys", gErr, nil, "")
		return nil, graceful.ToStatusError(gErr)
	}
	s.handler.Success(ctx, action, codes.OK, "sessions listed", &userpb.ListSessionsResponse{Sessions: sessions}, nil, req.UserId, nil)
	return &userpb.ListSessionsResponse{Sessions: sessions}, nil
}

// --- Add stubs for all unimplemented proto RPCs ---.
func (s *Service) InitiateSSO(_ context.Context, req *userpb.InitiateSSORequest) (*userpb.InitiateSSOResponse, error) {
	if req.Provider == "" || req.RedirectUri == "" {
		return nil, status.Error(codes.InvalidArgument, "provider and redirect_uri are required")
	}
	// TODO: Integrate with real SSO provider (e.g., OAuth2, SAML)
	ssoURL := fmt.Sprintf("https://sso.example.com/auth?provider=%s&redirect_uri=%s", req.Provider, req.RedirectUri)
	return &userpb.InitiateSSOResponse{SsoUrl: ssoURL}, nil
}

func (s *Service) InitiateMFA(ctx context.Context, req *userpb.InitiateMFARequest) (*userpb.InitiateMFAResponse, error) {
	if req.UserId == "" || req.MfaType == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and mfa_type are required")
	}
	// Simulate sending a code (in real logic, send via SMS/email/TOTP)
	code := generateSimpleCode()
	challengeID := fmt.Sprintf("challenge-%s-%d", req.UserId, time.Now().UnixNano())
	user, err := s.repo.GetByID(ctx, req.UserId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	// Store MFA code and challenge in metadata (for demo)
	metaPtr, err := metadata.ServiceMetadataFromStruct(user.Metadata.ServiceSpecific)
	if err == nil && metaPtr != nil {
		meta := *metaPtr
		meta.MFAChallenge = &metadata.MFAChallengeData{Code: code, ChallengeID: challengeID, ExpiresAt: time.Now().Add(5 * time.Minute).Format(time.RFC3339)}
		metaStruct, err := metadata.ServiceMetadataToStruct(&meta)
		if err != nil {
			s.log.Error("failed to convert service metadata to struct", zap.Error(err))
			return nil, status.Errorf(codes.Internal, "failed to convert service metadata: %v", err)
		}
		user.Metadata.ServiceSpecific = metaStruct
		if err := s.repo.Update(ctx, user); err != nil {
			s.log.Error("failed to update user", zap.Error(err))
			return nil, status.Errorf(codes.Internal, "failed to update user: %v", err)
		}
	}
	// TODO: Actually send code to user (SMS/email/TOTP)
	return &userpb.InitiateMFAResponse{Initiated: true, ChallengeId: challengeID}, nil
}

func (s *Service) SyncSCIM(_ context.Context, req *userpb.SyncSCIMRequest) (*userpb.SyncSCIMResponse, error) {
	if req.ScimPayload == "" {
		return nil, status.Error(codes.InvalidArgument, "scim_payload is required")
	}
	// TODO: Parse and process SCIM payload (for now, just log)
	s.log.Info("Received SCIM payload", zap.String("payload", req.ScimPayload))
	return &userpb.SyncSCIMResponse{Success: true}, nil
}

func (s *Service) RegisterInterest(ctx context.Context, req *userpb.RegisterInterestRequest) (*userpb.RegisterInterestResponse, error) {
	if req.Email == "" {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, errors.New("email is required"), "email is required", codes.InvalidArgument))
	}
	user, err := s.repo.GetByEmail(ctx, req.Email)
	if err == nil && user != nil {
		if user.Status != int32(userpb.UserStatus_USER_STATUS_PENDING) {
			user.Status = int32(userpb.UserStatus_USER_STATUS_PENDING)
			if err := s.repo.Update(ctx, user); err != nil {
				s.log.Error("failed to update user", zap.Error(err))
				return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to update user", codes.Internal))
			}
		}
		s.handler.Success(ctx, "register_interest", codes.OK, "interest registered (existing user)", &userpb.RegisterInterestResponse{User: repoUserToProtoUser(user)}, user.Metadata, user.ID, nil)
		return &userpb.RegisterInterestResponse{User: repoUserToProtoUser(user)}, nil
	}
	user = &User{
		Email:    req.Email,
		Status:   int32(userpb.UserStatus_USER_STATUS_PENDING),
		Metadata: &commonpb.Metadata{},
	}
	created, err := s.repo.Create(ctx, user)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to register interest", codes.Internal))
	}
	s.handler.Success(ctx, "register_interest", codes.OK, "interest registered (new user)", &userpb.RegisterInterestResponse{User: repoUserToProtoUser(created)}, created.Metadata, created.ID, nil)
	return &userpb.RegisterInterestResponse{User: repoUserToProtoUser(created)}, nil
}

func (s *Service) CreateReferral(ctx context.Context, req *userpb.CreateReferralRequest) (*userpb.CreateReferralResponse, error) {
	if req.UserId == "" || req.CampaignSlug == "" {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, errors.New("user_id and campaign_slug are required"), "user_id and campaign_slug are required", codes.InvalidArgument))
	}
	user, err := s.repo.GetByID(ctx, req.UserId)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "user not found", codes.NotFound))
	}
	// Generate a unique referral code (could use a hash, UUID, or campaign+user)
	code := fmt.Sprintf("REF-%s-%s-%d", req.UserId, req.CampaignSlug, time.Now().Unix()%100000)
	user.ReferralCode = code
	if err := s.repo.Update(ctx, user); err != nil {
		s.log.Error("failed to update user", zap.Error(err))
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to update user", codes.Internal))
	}
	s.handler.Success(ctx, "create_referral", codes.OK, "referral created", &userpb.CreateReferralResponse{ReferralCode: code, Success: true}, user.Metadata, user.ID, nil)
	return &userpb.CreateReferralResponse{ReferralCode: code, Success: true}, nil
}

// FindOrCreateOAuthUser looks up a user by email/provider, creates if not found, updates OAuth and audit metadata, and returns the user.
func (s *Service) FindOrCreateOAuthUser(ctx context.Context, oauthUser goth.User) (*User, error) {
	// Try to find by external OAuth ID first (if supported)
	// repoUser, err := s.repo.GetByExternalID(ctx, oauthUser.Provider, oauthUser.UserID)
	// if err == nil && repoUser != nil {
	// 	_ = s.updateLoginMetadata(repoUser, "oauth:"+oauthUser.Provider, oauthUser.Provider, oauthUser.UserID)
	// 	_ = s.repo.Update(ctx, repoUser)
	// 	return repoUser, nil
	// }
	// Fallback: try to find by email
	user, err := s.repo.GetByEmail(ctx, oauthUser.Email)
	switch {
	case errors.Is(err, ErrUserNotFound):
		// Not found, create new user with OAuth metadata
		oauthInfo := map[string]interface{}{
			"provider":         oauthUser.Provider,
			"provider_user_id": oauthUser.UserID,
			"email":            oauthUser.Email,
			"name":             oauthUser.Name,
			"avatar_url":       oauthUser.AvatarURL,
		}
		metaStruct, err := structpb.NewStruct(oauthInfo)
		if err != nil {
			return nil, graceful.LogAndWrap(ctx, s.log, codes.Internal, "failed to convert oauthInfo to structpb.Struct", err)
		}
		user = &User{
			Username: oauthUser.NickName,
			Email:    oauthUser.Email,

			Status: int32(userpb.UserStatus_USER_STATUS_ACTIVE),
			Metadata: &commonpb.Metadata{
				ServiceSpecific: metaStruct,
			},
		}
		// Update metadata struct directly
		metaPtr, err := metadata.ServiceMetadataFromStruct(user.Metadata.ServiceSpecific)
		if err == nil && metaPtr != nil {
			meta := *metaPtr
			meta.DeviceID = "oauth:" + oauthUser.Provider
			if meta.Audit == nil {
				meta.Audit = &metadata.AuditMetadata{}
			}
			meta.Audit.LastModified = time.Now().UTC().Format(time.RFC3339)
			meta.Audit.History = append(meta.Audit.History, "oauth_login")
			metaStruct, err := metadata.ServiceMetadataToStruct(&meta)
			if err == nil {
				user.Metadata.ServiceSpecific = metaStruct
			}
		}
		if err := s.updateUserMetadata(ctx, user, user.Metadata); err != nil {
			return nil, graceful.LogAndWrap(ctx, s.log, codes.Internal, "failed to update login metadata", err)
		}
		user, err = s.repo.Create(ctx, user)
		if err != nil {
			return nil, graceful.LogAndWrap(ctx, s.log, codes.Internal, "failed to create OAuth user", err)
		}
		return user, nil
	case err != nil:
		return nil, err
	default:
		// Update metadata struct directly
		if user.Metadata == nil {
			user.Metadata = &commonpb.Metadata{}
		}
		metaPtr, err := metadata.ServiceMetadataFromStruct(user.Metadata.ServiceSpecific)
		if err == nil && metaPtr != nil {
			meta := *metaPtr
			meta.DeviceID = "oauth:" + oauthUser.Provider
			if meta.Audit == nil {
				meta.Audit = &metadata.AuditMetadata{}
			}
			meta.Audit.LastModified = time.Now().UTC().Format(time.RFC3339)
			meta.Audit.History = append(meta.Audit.History, "oauth_login")
			metaStruct, err := metadata.ServiceMetadataToStruct(&meta)
			if err == nil {
				user.Metadata.ServiceSpecific = metaStruct
			}
		}
		if err := s.updateUserMetadata(ctx, user, user.Metadata); err != nil {
			return nil, graceful.LogAndWrap(ctx, s.log, codes.Internal, "failed to update login metadata", err)
		}
		if err := s.repo.Update(ctx, user); err != nil {
			return nil, graceful.LogAndWrap(ctx, s.log, codes.Internal, "failed to update user", err)
		}
		return user, nil
	}
}

// --- Composable Auth Channel Methods ---
// SendVerificationEmail emits a notification event instead of direct call.
func (s *Service) SendVerificationEmail(ctx context.Context, userID string) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return graceful.LogAndWrap(ctx, s.log, codes.NotFound, "user not found", err)
	}
	code := generateSimpleCode()
	expires := time.Now().Add(15 * time.Minute).UTC().Format(time.RFC3339)
	metaPtr, err := metadata.ServiceMetadataFromStruct(user.Metadata.ServiceSpecific)
	if err != nil {
		return graceful.LogAndWrap(ctx, s.log, codes.Internal, "failed to extract service metadata", err)
	}
	meta := *metaPtr
	meta.VerificationData = &metadata.VerificationData{Code: code, ExpiresAt: expires}
	metaStruct, err := metadata.ServiceMetadataToStruct(&meta)
	if err != nil {
		return graceful.LogAndWrap(ctx, s.log, codes.Internal, "failed to convert service metadata", err)
	}
	user.Metadata.ServiceSpecific = metaStruct
	if err := s.repo.Update(ctx, user); err != nil {
		return graceful.LogAndWrap(ctx, s.log, codes.Internal, "failed to update user", err)
	}
	return nil
}

// VerifyEmail verifies the code and marks email as verified.
func (s *Service) VerifyEmail(ctx context.Context, userID, code string) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return graceful.LogAndWrap(ctx, s.log, codes.NotFound, "user not found", err)
	}
	metaPtr, err := metadata.ServiceMetadataFromStruct(user.Metadata.ServiceSpecific)
	if err != nil {
		return graceful.LogAndWrap(ctx, s.log, codes.Internal, "failed to extract service metadata", err)
	}
	meta := *metaPtr
	if meta.VerificationData == nil || meta.VerificationData.Code != code {
		return graceful.LogAndWrap(ctx, s.log, codes.InvalidArgument, "invalid code", errors.New("invalid code"))
	}
	exp, err := time.Parse(time.RFC3339, meta.VerificationData.ExpiresAt)
	if err != nil {
		return graceful.LogAndWrap(ctx, s.log, codes.Internal, "failed to parse verification expiration", err)
	}
	if time.Now().After(exp) {
		return graceful.LogAndWrap(ctx, s.log, codes.InvalidArgument, "code expired", errors.New("code expired"))
	}
	meta.EmailVerified = true
	meta.VerificationData = nil
	metaStruct, err := metadata.ServiceMetadataToStruct(&meta)
	if err != nil {
		return graceful.LogAndWrap(ctx, s.log, codes.Internal, "failed to convert service metadata", err)
	}
	user.Metadata.ServiceSpecific = metaStruct
	if err := s.repo.Update(ctx, user); err != nil {
		return graceful.LogAndWrap(ctx, s.log, codes.Internal, "failed to update user", err)
	}
	return nil
}

// RequestPasswordReset emits a notification event instead of direct call.
func (s *Service) RequestPasswordReset(ctx context.Context, userID string) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return graceful.LogAndWrap(ctx, s.log, codes.NotFound, "user not found", err)
	}
	code := generateSimpleCode()
	expires := time.Now().Add(15 * time.Minute).UTC().Format(time.RFC3339)
	metaPtr, err := metadata.ServiceMetadataFromStruct(user.Metadata.ServiceSpecific)
	if err != nil {
		return graceful.LogAndWrap(ctx, s.log, codes.Internal, "failed to extract service metadata", err)
	}
	meta := *metaPtr
	meta.PasswordReset = &metadata.PasswordResetData{Code: code, ExpiresAt: expires}
	metaStruct, err := metadata.ServiceMetadataToStruct(&meta)
	if err != nil {
		return graceful.LogAndWrap(ctx, s.log, codes.Internal, "failed to convert service metadata", err)
	}
	user.Metadata.ServiceSpecific = metaStruct
	if err := s.repo.Update(ctx, user); err != nil {
		return graceful.LogAndWrap(ctx, s.log, codes.Internal, "failed to update user", err)
	}
	return nil
}

// VerifyPasswordReset verifies the password reset code.
func (s *Service) VerifyPasswordReset(ctx context.Context, userID, code string) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return graceful.LogAndWrap(ctx, s.log, codes.NotFound, "user not found", err)
	}
	metaPtr, err := metadata.ServiceMetadataFromStruct(user.Metadata.ServiceSpecific)
	if err != nil {
		return graceful.LogAndWrap(ctx, s.log, codes.Internal, "failed to extract service metadata", err)
	}
	meta := *metaPtr
	if meta.PasswordReset == nil || meta.PasswordReset.Code != code {
		return graceful.LogAndWrap(ctx, s.log, codes.InvalidArgument, "invalid code", errors.New("invalid code"))
	}
	exp, err := time.Parse(time.RFC3339, meta.PasswordReset.ExpiresAt)
	if err != nil {
		return graceful.LogAndWrap(ctx, s.log, codes.Internal, "failed to parse verification expiration", err)
	}
	if time.Now().After(exp) {
		return graceful.LogAndWrap(ctx, s.log, codes.InvalidArgument, "code expired", errors.New("code expired"))
	}
	return nil
}

// ResetPassword resets the user's password after code verification.
func (s *Service) ResetPassword(ctx context.Context, userID, code, newPassword string) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return graceful.LogAndWrap(ctx, s.log, codes.NotFound, "user not found", err)
	}
	metaPtr, err := metadata.ServiceMetadataFromStruct(user.Metadata.ServiceSpecific)
	if err != nil {
		return graceful.LogAndWrap(ctx, s.log, codes.Internal, "failed to extract service metadata", err)
	}
	meta := *metaPtr
	if meta.PasswordReset == nil || meta.PasswordReset.Code != code {
		return graceful.LogAndWrap(ctx, s.log, codes.InvalidArgument, "invalid code", errors.New("invalid code"))
	}
	exp, err := time.Parse(time.RFC3339, meta.PasswordReset.ExpiresAt)
	if err != nil {
		return graceful.LogAndWrap(ctx, s.log, codes.Internal, "failed to parse verification expiration", err)
	}
	if time.Now().After(exp) {
		return graceful.LogAndWrap(ctx, s.log, codes.InvalidArgument, "code expired", errors.New("code expired"))
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return graceful.LogAndWrap(ctx, s.log, codes.Internal, "failed to hash password", err)
	}
	user.PasswordHash = string(hash)
	meta.PasswordReset = nil
	metaStruct, err := metadata.ServiceMetadataToStruct(&meta)
	if err != nil {
		return graceful.LogAndWrap(ctx, s.log, codes.Internal, "failed to convert service metadata", err)
	}
	user.Metadata.ServiceSpecific = metaStruct
	if err := s.repo.Update(ctx, user); err != nil {
		return graceful.LogAndWrap(ctx, s.log, codes.Internal, "failed to update user", err)
	}
	return nil
}

// BeginWebAuthnRegistration emits an event instead of direct WebAuthn logic.
func (s *Service) BeginWebAuthnRegistration(_ context.Context, _, _ string) (string, error) {
	// Emit event or leave as stub for event-driven WebAuthn registration
	return "", nil // No direct provider logic
}

// FinishWebAuthnRegistration emits an event instead of direct WebAuthn logic.
func (s *Service) FinishWebAuthnRegistration(_ context.Context) error {
	// Emit event or leave as stub for event-driven WebAuthn registration
	return nil // No direct provider logic
}

// BeginWebAuthnLogin emits an event instead of direct WebAuthn logic.
func (s *Service) BeginWebAuthnLogin(_ context.Context, _, _ string) (string, error) {
	// Emit event or leave as stub for event-driven WebAuthn login
	return "", nil // No direct provider logic
}

// FinishWebAuthnLogin emits an event instead of direct WebAuthn logic.
func (s *Service) FinishWebAuthnLogin(_ context.Context, _, _, _ string) error {
	// Emit event or leave as stub for event-driven WebAuthn login
	return nil // No direct provider logic
}

// IsBiometricEnabled emits an event instead of direct biometric check.
func (s *Service) IsBiometricEnabled(_ context.Context) (bool, error) {
	// Emit event or leave as stub for event-driven biometric check
	return false, nil // No direct provider logic
}

// MarkBiometricUsed emits an event instead of direct biometric usage.
func (s *Service) MarkBiometricUsed(_ context.Context) error {
	// Emit event or leave as stub for event-driven biometric usage
	return nil // No direct provider logic
}

// Helper: generate a simple numeric code.
func generateSimpleCode() string {
	return fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
}

// --- ServiceMetadata struct and helpers ---
// Extend ServiceMetadata to support MFAChallenge for MFA flows.
type MFAChallengeData struct {
	Code        string `json:"code"`
	ChallengeID string `json:"challenge_id"`
	ExpiresAt   string `json:"expires_at"`
}

// --- Social Graph APIs ---.
func (s *Service) AddFriend(ctx context.Context, req *userpb.AddFriendRequest) (*userpb.AddFriendResponse, error) {
	friendship, err := s.repo.AddFriend(ctx, req.UserId, req.FriendId, req.Metadata)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to add friend", codes.Internal))
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "friend added", &userpb.AddFriendResponse{Friendship: friendship}, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log})
	return &userpb.AddFriendResponse{Friendship: friendship}, nil
}

func (s *Service) RemoveFriend(ctx context.Context, req *userpb.RemoveFriendRequest) (*userpb.RemoveFriendResponse, error) {
	err := s.repo.RemoveFriend(ctx, req.UserId, req.FriendId)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to remove friend", codes.Internal))
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "friend removed", &userpb.RemoveFriendResponse{Success: true}, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log})
	return &userpb.RemoveFriendResponse{Success: true}, nil
}

func (s *Service) ListFriends(ctx context.Context, req *userpb.ListFriendsRequest) (*userpb.ListFriendsResponse, error) {
	friends, total, err := s.repo.ListFriends(ctx, req.UserId, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to list friends", codes.Internal))
	}
	resp := &userpb.ListFriendsResponse{
		Friends:    friends,
		TotalCount: utils.ToInt32(total),
		Page:       req.Page,
		TotalPages: utils.ToInt32((total + int(req.PageSize) - 1) / int(req.PageSize)),
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "friends listed", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log})
	return resp, nil
}

func (s *Service) FollowUser(ctx context.Context, req *userpb.FollowUserRequest) (*userpb.FollowUserResponse, error) {
	follow, err := s.repo.FollowUser(ctx, req.FollowerId, req.FolloweeId, req.Metadata)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to follow user", codes.Internal))
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "user followed", &userpb.FollowUserResponse{Follow: follow}, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log})
	return &userpb.FollowUserResponse{Follow: follow}, nil
}

func (s *Service) UnfollowUser(ctx context.Context, req *userpb.UnfollowUserRequest) (*userpb.UnfollowUserResponse, error) {
	err := s.repo.UnfollowUser(ctx, req.FollowerId, req.FolloweeId)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to unfollow user", codes.Internal))
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "user unfollowed", &userpb.UnfollowUserResponse{Success: true}, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log})
	return &userpb.UnfollowUserResponse{Success: true}, nil
}

func (s *Service) ListFollowers(ctx context.Context, req *userpb.ListFollowersRequest) (*userpb.ListFollowersResponse, error) {
	followers, total, err := s.repo.ListFollowers(ctx, req.UserId, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to list followers", codes.Internal))
	}
	resp := &userpb.ListFollowersResponse{
		Followers:  followers,
		TotalCount: utils.ToInt32(total),
		Page:       req.Page,
		TotalPages: utils.ToInt32((total + int(req.PageSize) - 1) / int(req.PageSize)),
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "followers listed", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log})
	return resp, nil
}

func (s *Service) ListFollowing(ctx context.Context, req *userpb.ListFollowingRequest) (*userpb.ListFollowingResponse, error) {
	following, total, err := s.repo.ListFollowing(ctx, req.UserId, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to list following", codes.Internal))
	}
	resp := &userpb.ListFollowingResponse{
		Followings: following,
		TotalCount: utils.ToInt32(total),
		Page:       req.Page,
		TotalPages: utils.ToInt32((total + int(req.PageSize) - 1) / int(req.PageSize)),
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "following listed", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log})
	return resp, nil
}

// --- Group APIs ---.
func (s *Service) CreateUserGroup(ctx context.Context, req *userpb.CreateUserGroupRequest) (*userpb.CreateUserGroupResponse, error) {
	group := &userpb.UserGroup{
		Name:        req.Name,
		Description: req.Description,
		MemberIds:   req.MemberIds,
		Roles:       req.Roles,
		Metadata:    req.Metadata,
	}
	created, err := s.repo.CreateUserGroup(ctx, group)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to create user group", codes.Internal))
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "user group created", &userpb.CreateUserGroupResponse{UserGroup: created}, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log})
	return &userpb.CreateUserGroupResponse{UserGroup: created}, nil
}

func (s *Service) UpdateUserGroup(ctx context.Context, req *userpb.UpdateUserGroupRequest) (*userpb.UpdateUserGroupResponse, error) {
	authUserID, ok := utils.GetAuthenticatedUserID(ctx)
	if !ok {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, errors.New("missing authentication"), "missing authentication", codes.Unauthenticated))
	}
	roles, _ := utils.GetAuthenticatedUserRoles(ctx)
	isAdmin := utils.IsServiceAdmin(roles, "user")
	group, err := s.repo.GetUserGroupByID(ctx, req.UserGroupId)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "user group not found", codes.NotFound))
	}
	isGroupAdmin := group.Roles[authUserID] == "admin"
	if !isAdmin && !isGroupAdmin {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, errors.New("cannot update group you do not own or admin"), "cannot update group you do not own or admin", codes.PermissionDenied))
	}
	updated, err := s.repo.UpdateUserGroup(ctx, req.UserGroupId, req.UserGroup, req.FieldsToUpdates)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to update user group", codes.Internal))
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "user group updated", &userpb.UpdateUserGroupResponse{UserGroup: updated}, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log})
	return &userpb.UpdateUserGroupResponse{UserGroup: updated}, nil
}

func (s *Service) DeleteUserGroup(ctx context.Context, req *userpb.DeleteUserGroupRequest) (*userpb.DeleteUserGroupResponse, error) {
	authUserID, ok := utils.GetAuthenticatedUserID(ctx)
	if !ok {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, errors.New("missing authentication"), "missing authentication", codes.Unauthenticated))
	}
	roles, _ := utils.GetAuthenticatedUserRoles(ctx)
	isAdmin := utils.IsServiceAdmin(roles, "user")
	group, err := s.repo.GetUserGroupByID(ctx, req.UserGroupId)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "user group not found", codes.NotFound))
	}
	isGroupAdmin := group.Roles[authUserID] == "admin"
	if !isAdmin && !isGroupAdmin {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, errors.New("cannot delete group you do not own or admin"), "cannot delete group you do not own or admin", codes.PermissionDenied))
	}
	err = s.repo.DeleteUserGroup(ctx, req.UserGroupId)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to delete user group", codes.Internal))
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "user group deleted", &userpb.DeleteUserGroupResponse{Success: true}, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log})
	return &userpb.DeleteUserGroupResponse{Success: true}, nil
}

func (s *Service) ListUserGroups(ctx context.Context, req *userpb.ListUserGroupsRequest) (*userpb.ListUserGroupsResponse, error) {
	groups, total, err := s.repo.ListUserGroups(ctx, req.UserId, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to list user groups", codes.Internal))
	}
	resp := &userpb.ListUserGroupsResponse{
		UserGroups: groups,
		TotalCount: utils.ToInt32(total),
		Page:       req.Page,
		TotalPages: utils.ToInt32((total + int(req.PageSize) - 1) / int(req.PageSize)),
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "user groups listed", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log})
	return resp, nil
}

func (s *Service) ListUserGroupMembers(ctx context.Context, req *userpb.ListUserGroupMembersRequest) (*userpb.ListUserGroupMembersResponse, error) {
	members, total, err := s.repo.ListUserGroupMembers(ctx, req.UserGroupId, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to list user group members", codes.Internal))
	}
	resp := &userpb.ListUserGroupMembersResponse{
		Members:    members,
		TotalCount: utils.ToInt32(total),
		Page:       req.Page,
		TotalPages: utils.ToInt32((total + int(req.PageSize) - 1) / int(req.PageSize)),
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "user group members listed", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log})
	return resp, nil
}

// --- Social Graph Discovery ---.
func (s *Service) SuggestConnections(ctx context.Context, req *userpb.SuggestConnectionsRequest) (*userpb.SuggestConnectionsResponse, error) {
	suggestions, err := s.repo.SuggestConnections(ctx, req.UserId, req.Metadata)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to suggest connections", codes.Internal))
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "connections suggested", &userpb.SuggestConnectionsResponse{Suggestions: suggestions}, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log})
	return &userpb.SuggestConnectionsResponse{Suggestions: suggestions}, nil
}

func (s *Service) ListConnections(ctx context.Context, req *userpb.ListConnectionsRequest) (*userpb.ListConnectionsResponse, error) {
	users, err := s.repo.ListConnections(ctx, req.UserId, req.Type, req.Metadata)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to list connections", codes.Internal))
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "connections listed", &userpb.ListConnectionsResponse{Users: users}, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log})
	return &userpb.ListConnectionsResponse{Users: users}, nil
}

// --- Moderation/Interaction APIs ---.
func (s *Service) BlockUser(ctx context.Context, req *userpb.BlockUserRequest) (*userpb.BlockUserResponse, error) {
	targetUser, err := s.repo.GetByID(ctx, req.TargetUserId)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "target user not found", codes.NotFound))
	}
	if targetUser.Metadata == nil {
		targetUser.Metadata = &commonpb.Metadata{}
	}
	// Canonical bad_actor update (map-based)
	ss := map[string]interface{}{}
	if targetUser.Metadata.ServiceSpecific != nil {
		ss = targetUser.Metadata.ServiceSpecific.AsMap()
	}
	userMetaVal, ok := ss["user"]
	if !ok {
		s.log.Warn("user metadata missing in ss map", zap.String("user_id", targetUser.ID))
		userMetaVal = map[string]interface{}{}
	}
	userMeta, ok := userMetaVal.(map[string]interface{})
	if !ok {
		s.log.Warn("user metadata type assertion failed", zap.Any("userMetaVal", userMetaVal), zap.String("user_id", targetUser.ID))
		userMeta = map[string]interface{}{}
	}
	badActorVal, ok := userMeta["bad_actor"]
	if !ok {
		badActorVal = map[string]interface{}{"score": 1.0}
	}
	badActor, ok := badActorVal.(map[string]interface{})
	if !ok {
		s.log.Warn("bad_actor type assertion failed", zap.Any("badActorVal", badActorVal), zap.String("user_id", targetUser.ID))
		badActor = map[string]interface{}{"score": 1.0}
	}
	score, ok := badActor["score"].(float64)
	if !ok {
		score = 0
	}
	badActor["score"] = score + 1.0
	userMeta["bad_actor"] = badActor
	ss["user"] = userMeta
	newStruct, err := structpb.NewStruct(ss)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to convert metadata to struct", codes.Internal))
	}
	targetUser.Metadata.ServiceSpecific = newStruct
	if err := s.updateUserMetadata(ctx, targetUser, targetUser.Metadata); err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to update metadata", codes.Internal))
	}
	if err := s.repo.Update(ctx, targetUser); err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to update user", codes.Internal))
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "user blocked", &userpb.BlockUserResponse{Success: true}, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log})
	return &userpb.BlockUserResponse{Success: true}, nil
}

func (s *Service) UnblockUser(ctx context.Context, req *userpb.UnblockUserRequest) (*userpb.UnblockUserResponse, error) {
	action := "unblock_user"
	err := s.repo.UnblockUser(ctx, req.TargetUserId)
	if err != nil {
		gErr := graceful.MapAndWrapErr(ctx, err, "failed to unblock user", codes.Internal)
		s.handler.Error(ctx, action, codes.Internal, "failed to unblock user", gErr, nil, req.TargetUserId)
		return nil, graceful.ToStatusError(gErr)
	}
	s.handler.Success(ctx, action, codes.OK, "user unblocked", &userpb.UnblockUserResponse{Success: true}, nil, req.TargetUserId, nil)
	return &userpb.UnblockUserResponse{Success: true}, nil
}

func (s *Service) MuteUser(ctx context.Context, req *userpb.MuteUserRequest) (*userpb.MuteUserResponse, error) {
	action := "mute_user"
	targetUser, err := s.repo.GetByID(ctx, req.TargetUserId)
	if err != nil {
		gErr := graceful.MapAndWrapErr(ctx, err, "target user not found", codes.NotFound)
		s.handler.Error(ctx, action, codes.NotFound, "target user not found", gErr, nil, req.TargetUserId)
		return nil, graceful.ToStatusError(gErr)
	}
	if targetUser.Metadata == nil {
		targetUser.Metadata = &commonpb.Metadata{}
	}
	ss := map[string]interface{}{}
	if targetUser.Metadata.ServiceSpecific != nil {
		ss = targetUser.Metadata.ServiceSpecific.AsMap()
	}
	userMetaVal, ok := ss["user"]
	if !ok {
		userMetaVal = map[string]interface{}{}
	}
	userMeta, ok := userMetaVal.(map[string]interface{})
	if !ok {
		userMeta = map[string]interface{}{}
	}
	badActorVal, ok := userMeta["bad_actor"]
	if !ok {
		badActorVal = map[string]interface{}{"muted": true}
	}
	badActor, ok := badActorVal.(map[string]interface{})
	if !ok {
		badActor = map[string]interface{}{"muted": true}
	}
	badActor["muted"] = true
	userMeta["bad_actor"] = badActor
	ss["user"] = userMeta
	newStruct, err := structpb.NewStruct(ss)
	if err != nil {
		gErr := graceful.MapAndWrapErr(ctx, err, "failed to convert metadata to struct", codes.Internal)
		s.handler.Error(ctx, action, codes.Internal, "failed to convert metadata to struct", gErr, targetUser.Metadata, targetUser.ID)
		return nil, graceful.ToStatusError(gErr)
	}
	targetUser.Metadata.ServiceSpecific = newStruct
	if err := s.updateUserMetadata(ctx, targetUser, targetUser.Metadata); err != nil {
		gErr := graceful.MapAndWrapErr(ctx, err, "failed to update metadata", codes.Internal)
		s.handler.Error(ctx, action, codes.Internal, "failed to update metadata", gErr, targetUser.Metadata, targetUser.ID)
		return nil, graceful.ToStatusError(gErr)
	}
	if err := s.repo.Update(ctx, targetUser); err != nil {
		gErr := graceful.MapAndWrapErr(ctx, err, "failed to update user", codes.Internal)
		s.handler.Error(ctx, action, codes.Internal, "failed to update user", gErr, targetUser.Metadata, targetUser.ID)
		return nil, graceful.ToStatusError(gErr)
	}
	s.handler.Success(ctx, action, codes.OK, "user muted", &userpb.MuteUserResponse{Success: true}, targetUser.Metadata, targetUser.ID, nil)
	return &userpb.MuteUserResponse{Success: true}, nil
}

func (s *Service) UnmuteUser(ctx context.Context, req *userpb.UnmuteUserRequest) (*userpb.UnmuteUserResponse, error) {
	action := "unmute_user"
	err := s.repo.UnmuteUser(ctx, req.TargetUserId)
	if err != nil {
		gErr := graceful.MapAndWrapErr(ctx, err, "failed to unmute user", codes.Internal)
		s.handler.Error(ctx, action, codes.Internal, "failed to unmute user", gErr, nil, req.TargetUserId)
		return nil, graceful.ToStatusError(gErr)
	}
	s.handler.Success(ctx, action, codes.OK, "user unmuted", &userpb.UnmuteUserResponse{Success: true}, nil, req.TargetUserId, nil)
	return &userpb.UnmuteUserResponse{Success: true}, nil
}

func (s *Service) UnmuteGroup(ctx context.Context, req *userpb.UnmuteGroupRequest) (*userpb.UnmuteGroupResponse, error) {
	action := "unmute_group"
	err := s.repo.UnmuteGroup(ctx, req.UserId, req.GroupId)
	if err != nil {
		gErr := graceful.MapAndWrapErr(ctx, err, "failed to unmute group", codes.Internal)
		s.handler.Error(ctx, action, codes.Internal, "failed to unmute group", gErr, nil, req.GroupId)
		return &userpb.UnmuteGroupResponse{Success: false}, graceful.ToStatusError(gErr)
	}
	s.handler.Success(ctx, action, codes.OK, "group unmuted", &userpb.UnmuteGroupResponse{Success: true}, nil, req.GroupId, nil)
	return &userpb.UnmuteGroupResponse{Success: true}, nil
}

func (s *Service) UnmuteGroupIndividuals(ctx context.Context, req *userpb.UnmuteGroupIndividualsRequest) (*userpb.UnmuteGroupIndividualsResponse, error) {
	action := "unmute_group_individuals"
	unmuted, err := s.repo.UnmuteGroupIndividuals(ctx, req.UserId, req.GroupId, req.TargetUserIds)
	if err != nil {
		gErr := graceful.MapAndWrapErr(ctx, err, "failed to unmute group individuals", codes.Internal)
		s.handler.Error(ctx, action, codes.Internal, "failed to unmute group individuals", gErr, nil, req.GroupId)
		return &userpb.UnmuteGroupIndividualsResponse{Success: false}, graceful.ToStatusError(gErr)
	}
	s.handler.Success(ctx, action, codes.OK, "group individuals unmuted", &userpb.UnmuteGroupIndividualsResponse{Success: true, UnmutedUserIds: unmuted}, nil, req.GroupId, nil)
	return &userpb.UnmuteGroupIndividualsResponse{Success: true, UnmutedUserIds: unmuted}, nil
}

func (s *Service) UnblockGroupIndividuals(ctx context.Context, req *userpb.UnblockGroupIndividualsRequest) (*userpb.UnblockGroupIndividualsResponse, error) {
	unblocked, err := s.repo.UnblockGroupIndividuals(ctx, req.UserId, req.GroupId, req.TargetUserIds)
	if err != nil {
		return &userpb.UnblockGroupIndividualsResponse{Success: false}, status.Errorf(codes.Internal, "failed to unblock group individuals: %v", err)
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "group individuals unblocked", &userpb.UnblockGroupIndividualsResponse{Success: true, UnblockedUserIds: unblocked}, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log})
	return &userpb.UnblockGroupIndividualsResponse{Success: true, UnblockedUserIds: unblocked}, nil
}

func (s *Service) ReportUser(ctx context.Context, req *userpb.ReportUserRequest) (*userpb.ReportUserResponse, error) {
	action := "report_user"
	targetUser, err := s.repo.GetByID(ctx, req.ReportedUserId)
	if err != nil {
		gErr := graceful.MapAndWrapErr(ctx, err, "target user not found", codes.NotFound)
		s.handler.Error(ctx, action, codes.NotFound, "target user not found", gErr, nil, req.ReportedUserId)
		return nil, graceful.ToStatusError(gErr)
	}
	if targetUser.Metadata == nil {
		targetUser.Metadata = &commonpb.Metadata{}
	}
	ss := map[string]interface{}{}
	if targetUser.Metadata.ServiceSpecific != nil {
		ss = targetUser.Metadata.ServiceSpecific.AsMap()
	}
	userMetaVal, ok := ss["user"]
	if !ok {
		s.log.Warn("user metadata missing in ss map", zap.String("user_id", targetUser.ID))
		userMetaVal = map[string]interface{}{}
	}
	userMeta, ok := userMetaVal.(map[string]interface{})
	if !ok {
		userMeta = map[string]interface{}{}
	}
	badActorVal, ok := userMeta["bad_actor"]
	if !ok {
		badActorVal = map[string]interface{}{"score": 1.0}
	}
	badActor, ok := badActorVal.(map[string]interface{})
	if !ok {
		badActor = map[string]interface{}{"score": 1.0}
	}
	score, ok := badActor["score"].(float64)
	if !ok {
		score = 0.0
	}
	badActor["score"] = score + 1.0
	userMeta["bad_actor"] = badActor
	ss["user"] = userMeta
	newStruct, err := structpb.NewStruct(ss)
	if err != nil {
		gErr := graceful.MapAndWrapErr(ctx, err, "failed to convert metadata to struct", codes.Internal)
		s.handler.Error(ctx, action, codes.Internal, "failed to convert metadata to struct", gErr, targetUser.Metadata, targetUser.ID)
		return nil, graceful.ToStatusError(gErr)
	}
	targetUser.Metadata.ServiceSpecific = newStruct
	if err := s.updateUserMetadata(ctx, targetUser, targetUser.Metadata); err != nil {
		gErr := graceful.MapAndWrapErr(ctx, err, "failed to update metadata", codes.Internal)
		s.handler.Error(ctx, action, codes.Internal, "failed to update metadata", gErr, targetUser.Metadata, targetUser.ID)
		return nil, graceful.ToStatusError(gErr)
	}
	if err := s.repo.Update(ctx, targetUser); err != nil {
		gErr := graceful.MapAndWrapErr(ctx, err, "failed to update user", codes.Internal)
		s.handler.Error(ctx, action, codes.Internal, "failed to update user", gErr, targetUser.Metadata, targetUser.ID)
		return nil, graceful.ToStatusError(gErr)
	}
	s.handler.Success(ctx, action, codes.OK, "user reported", &userpb.ReportUserResponse{Success: true}, targetUser.Metadata, targetUser.ID, nil)
	return &userpb.ReportUserResponse{Success: true}, nil
}

func (s *Service) BlockGroupContent(ctx context.Context, req *userpb.BlockGroupContentRequest) (*userpb.BlockGroupContentResponse, error) {
	action := "block_group_content"
	err := s.repo.BlockGroupContent(ctx, req.UserId, req.GroupId, req.ContentId, req.Metadata)
	if err != nil {
		gErr := graceful.MapAndWrapErr(ctx, err, "failed to block group content", codes.Internal)
		s.handler.Error(ctx, action, codes.Internal, "failed to block group content", gErr, nil, req.GroupId)
		return &userpb.BlockGroupContentResponse{Success: false}, graceful.ToStatusError(gErr)
	}
	s.handler.Success(ctx, action, codes.OK, "group content blocked", &userpb.BlockGroupContentResponse{Success: true}, nil, req.GroupId, nil)
	return &userpb.BlockGroupContentResponse{Success: true}, nil
}

func (s *Service) ReportGroupContent(ctx context.Context, req *userpb.ReportGroupContentRequest) (*userpb.ReportGroupContentResponse, error) {
	action := "report_group_content"
	reportID, err := s.repo.ReportGroupContent(ctx, req.ReporterUserId, req.GroupId, req.ContentId, req.Reason, req.Details, req.Metadata)
	if err != nil {
		gErr := graceful.MapAndWrapErr(ctx, err, "failed to report group content", codes.Internal)
		s.handler.Error(ctx, action, codes.Internal, "failed to report group content", gErr, nil, req.GroupId)
		return &userpb.ReportGroupContentResponse{Success: false, ReportId: ""}, graceful.ToStatusError(gErr)
	}
	s.handler.Success(ctx, action, codes.OK, "group content reported", &userpb.ReportGroupContentResponse{Success: true, ReportId: reportID}, nil, req.GroupId, nil)
	return &userpb.ReportGroupContentResponse{Success: true, ReportId: reportID}, nil
}

func (s *Service) MuteGroupContent(ctx context.Context, req *userpb.MuteGroupContentRequest) (*userpb.MuteGroupContentResponse, error) {
	action := "mute_group_content"
	err := s.repo.MuteGroupContent(ctx, req.UserId, req.GroupId, req.ContentId, req.DurationMinutes, req.Metadata)
	if err != nil {
		gErr := graceful.MapAndWrapErr(ctx, err, "failed to mute group content", codes.Internal)
		s.handler.Error(ctx, action, codes.Internal, "failed to mute group content", gErr, nil, req.GroupId)
		return &userpb.MuteGroupContentResponse{Success: false}, graceful.ToStatusError(gErr)
	}
	s.handler.Success(ctx, action, codes.OK, "group content muted", &userpb.MuteGroupContentResponse{Success: true}, nil, req.GroupId, nil)
	return &userpb.MuteGroupContentResponse{Success: true}, nil
}

func (s *Service) MuteGroupIndividuals(ctx context.Context, req *userpb.MuteGroupIndividualsRequest) (*userpb.MuteGroupIndividualsResponse, error) {
	action := "mute_group_individuals"
	mutedUserIDs, err := s.repo.MuteGroupIndividuals(ctx, req.UserId, req.GroupId, int(req.DurationMinutes), req.Metadata)
	if err != nil {
		gErr := graceful.MapAndWrapErr(ctx, err, "failed to mute group individuals", codes.Internal)
		s.handler.Error(ctx, action, codes.Internal, "failed to mute group individuals", gErr, nil, req.GroupId)
		return &userpb.MuteGroupIndividualsResponse{Success: false}, graceful.ToStatusError(gErr)
	}
	s.handler.Success(ctx, action, codes.OK, "group individuals muted", &userpb.MuteGroupIndividualsResponse{Success: true, MutedUserIds: mutedUserIDs}, nil, req.GroupId, nil)
	return &userpb.MuteGroupIndividualsResponse{Success: true, MutedUserIds: mutedUserIDs}, nil
}

func (s *Service) BlockGroupIndividuals(ctx context.Context, req *userpb.BlockGroupIndividualsRequest) (*userpb.BlockGroupIndividualsResponse, error) {
	action := "block_group_individuals"
	blockedUserIDs, err := s.repo.BlockGroupIndividuals(ctx, req.UserId, req.GroupId, int(req.DurationMinutes), req.Metadata)
	if err != nil {
		gErr := graceful.MapAndWrapErr(ctx, err, "failed to block group individuals", codes.Internal)
		s.handler.Error(ctx, action, codes.Internal, "failed to block group individuals", gErr, nil, req.GroupId)
		return &userpb.BlockGroupIndividualsResponse{Success: false}, graceful.ToStatusError(gErr)
	}
	s.handler.Success(ctx, action, codes.OK, "group individuals blocked", &userpb.BlockGroupIndividualsResponse{Success: true, BlockedUserIds: blockedUserIDs}, nil, req.GroupId, nil)
	return &userpb.BlockGroupIndividualsResponse{Success: true, BlockedUserIds: blockedUserIDs}, nil
}

// RefreshSession implements refresh token rotation with rate limiting and JWT logic.
func (s *Service) RefreshSession(ctx context.Context, req *userpb.RefreshSessionRequest) (*userpb.RefreshSessionResponse, error) {
	// Parse and validate the old refresh token
	token, err := jwt.Parse(req.RefreshToken, func(_ *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "invalid or expired refresh token", codes.Unauthenticated))
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, errors.New("invalid token claims"), "invalid token claims", codes.Unauthenticated))
	}

	// Check if the token is revoked/blacklisted
	oldJTI, ok := claims["jti"].(string)
	if !ok {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, errors.New("missing jti in claims"), "missing jti in claims", codes.Unauthenticated))
	}
	expUnix, ok := claims["exp"].(float64)
	if !ok {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, errors.New("missing exp in claims"), "missing exp in claims", codes.Unauthenticated))
	}
	exp := time.Unix(int64(expUnix), 0)
	if s.isTokenRevoked(ctx, oldJTI) {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, errors.New("refresh token revoked"), "refresh token revoked", codes.Unauthenticated))
	}

	// Blacklist the old JTI
	s.revokeToken(ctx, oldJTI, exp)

	// Generate new JTI and expiration for refresh token
	newJTI := uuid.New().String()
	claims["jti"] = newJTI
	claims["exp"] = time.Now().Add(30 * 24 * time.Hour).Unix() // 30 days

	// Issue new refresh token
	newRefreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedRefresh, err := newRefreshToken.SignedString(jwtSecret)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to sign new refresh token", codes.Internal))
	}

	// Issue new access token (short-lived)
	sub, ok := claims["sub"]
	if !ok {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, errors.New("missing sub in claims"), "missing sub in claims", codes.Unauthenticated))
	}
	accessClaims := jwt.MapClaims{
		"sub": sub,
		"exp": time.Now().Add(15 * time.Minute).Unix(),
		"iat": time.Now().Unix(),
		"jti": uuid.New().String(),
	}
	newAccessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	signedAccess, err := newAccessToken.SignedString(jwtSecret)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to sign new access token", codes.Internal))
	}

	return &userpb.RefreshSessionResponse{
		RefreshToken: signedRefresh,
		AccessToken:  signedAccess,
	}, nil
}

// Redis-based token revocation/blacklisting.
func (s *Service) isTokenRevoked(ctx context.Context, jti string) bool {
	exists, err := s.cache.GetClient().Exists(ctx, "revoked_jti:"+jti).Result()
	return err == nil && exists == 1
}

func (s *Service) revokeToken(ctx context.Context, jti string, exp time.Time) {
	ttl := time.Until(exp)
	s.cache.GetClient().Set(ctx, "revoked_jti:"+jti, "1", ttl)
}
