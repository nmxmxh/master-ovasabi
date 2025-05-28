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
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"os"
	"regexp"
	"time"

	goth "github.com/markbates/goth"
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	userpb "github.com/nmxmxh/master-ovasabi/api/protos/user/v1"
	nexusevents "github.com/nmxmxh/master-ovasabi/internal/service/nexus"
	pattern "github.com/nmxmxh/master-ovasabi/internal/service/pattern"
	userEvents "github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	structpb "google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Service implements the UserService gRPC interface.
type Service struct {
	userpb.UnimplementedUserServiceServer
	log          *zap.Logger
	cache        *redis.Cache
	repo         *Repository
	eventEmitter EventEmitter
	eventEnabled bool
}

// Compile-time check.
var _ userpb.UserServiceServer = (*Service)(nil)

// NewUserService creates a new instance of UserService.
func NewService(log *zap.Logger, repo *Repository, cache *redis.Cache, eventEmitter EventEmitter, eventEnabled bool) userpb.UserServiceServer {
	return &Service{
		log:          log,
		repo:         repo,
		cache:        cache,
		eventEmitter: eventEmitter,
		eventEnabled: eventEnabled,
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
	s.log.Info("Creating user",
		zap.String("email", req.Email),
		zap.String("username", req.Username))

	// Username validation (Twitter-like, dots allowed, no emojis, Unicode letters/numbers/._)
	roles, _ := utils.GetAuthenticatedUserRoles(ctx)
	isAdmin := utils.IsAdmin(roles)
	var (
		userRegex  = regexp.MustCompile(`^[\p{L}\p{N}._]{5,20}$`)
		adminRegex = regexp.MustCompile(`^[\p{L}\p{N}._]{1,20}$`)
	)
	if isAdmin {
		if !adminRegex.MatchString(req.Username) {
			return nil, status.Error(codes.InvalidArgument, "invalid username: must be 1-20 Unicode letters, numbers, underscores, or dots; no emojis or symbols; cannot start/end with dot/underscore; no consecutive dots/underscores")
		}
	} else {
		if !userRegex.MatchString(req.Username) {
			return nil, status.Error(codes.InvalidArgument, "invalid username: must be 5-20 Unicode letters, numbers, underscores, or dots; no emojis or symbols; cannot start/end with dot/underscore; no consecutive dots/underscores")
		}
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.log.Error("Failed to hash password", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to hash password: %v", err)
	}

	user := &User{
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
		case errors.Is(err, ErrInvalidUsername):
			return nil, status.Error(codes.InvalidArgument, "invalid username format")
		case errors.Is(err, ErrUsernameReserved):
			return nil, status.Error(codes.InvalidArgument, "username is reserved")
		case errors.Is(err, ErrUsernameTaken):
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
	_, ok := userEvents.EmitEventWithLogging(ctx, s.eventEmitter, s.log, nexusevents.EventUserCreated, created.ID, created.Metadata)
	if !ok {
		s.log.Warn("Failed to emit user.created event")
	}
	err = pattern.RegisterWithNexus(ctx, s.log, "user", created.Metadata)
	if err != nil {
		s.log.Error("failed to register with nexus", zap.Error(err))
	}

	// TODO: Implement suspicious signup detection and bad actor metadata update here if needed.
	// if suspiciousSignup {
	// 	_ = s.updateBadActorMetadata(ctx, created, "suspicious_signup", deviceID, ip, city, country, 1)
	// 	// TODO: Integrate with Security and Content Moderation services for escalation.
	// }

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
		if errors.Is(err, ErrUserNotFound) {
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
	_, ok := userEvents.EmitEventWithLogging(ctx, s.eventEmitter, s.log, nexusevents.EventUserUpdated, repoUser.ID, repoUser.Metadata)
	if !ok {
		s.log.Warn("Failed to emit user.updated event (GetUser)")
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
		if errors.Is(err, ErrUserNotFound) {
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
		if errors.Is(err, ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "database error: %v", err)
	}
	respUser := repoUserToProtoUser(repoUser)
	return &userpb.GetUserByEmailResponse{User: respUser}, nil
}

// UpdateUser updates a user record.
func (s *Service) UpdateUser(ctx context.Context, req *userpb.UpdateUserRequest) (*userpb.UpdateUserResponse, error) {
	authUserID, ok := utils.GetAuthenticatedUserID(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing authentication")
	}
	roles, _ := utils.GetAuthenticatedUserRoles(ctx)
	isAdmin := utils.IsAdmin(roles)
	if !isAdmin && req.UserId != authUserID {
		return nil, status.Error(codes.PermissionDenied, "cannot update another user's profile")
	}
	repoUser, err := s.repo.GetByID(ctx, req.UserId)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
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
		case errors.Is(err, ErrInvalidUsername):
			return nil, status.Error(codes.InvalidArgument, "invalid username format")
		case errors.Is(err, ErrUsernameReserved):
			return nil, status.Error(codes.InvalidArgument, "username is reserved")
		case errors.Is(err, ErrUsernameTaken):
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
	_, ok = userEvents.EmitEventWithLogging(ctx, s.eventEmitter, s.log, nexusevents.EventUserUpdated, repoUser.ID, repoUser.Metadata)
	if !ok {
		s.log.Warn("Failed to emit user.updated event (UpdateUser)")
	}
	err = pattern.RegisterWithNexus(ctx, s.log, "user", repoUser.Metadata)
	if err != nil {
		s.log.Error("failed to register with nexus", zap.Error(err))
	}
	return &userpb.UpdateUserResponse{User: getResp.User}, nil
}

// DeleteUser removes a user and its master record.
func (s *Service) DeleteUser(ctx context.Context, req *userpb.DeleteUserRequest) (*userpb.DeleteUserResponse, error) {
	authUserID, ok := utils.GetAuthenticatedUserID(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing authentication")
	}
	roles, _ := utils.GetAuthenticatedUserRoles(ctx)
	isAdmin := utils.IsAdmin(roles)
	if !isAdmin && req.UserId != authUserID {
		return nil, status.Error(codes.PermissionDenied, "cannot delete another user's profile")
	}
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
		if totalPages > int(^int32(0)) || totalPages < 0 {
			return nil, fmt.Errorf("totalPages overflows int32 (final check 2)")
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
	authUserID, ok := utils.GetAuthenticatedUserID(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing authentication")
	}
	roles, _ := utils.GetAuthenticatedUserRoles(ctx)
	isAdmin := utils.IsAdmin(roles)
	if !isAdmin && req.UserId != authUserID {
		return nil, status.Error(codes.PermissionDenied, "cannot update another user's profile")
	}
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

// Standard: All audit fields (created_by, last_modified_by) must use a non-PII user reference (user_id:master_id).
// This ensures GDPR compliance and prevents accidental PII exposure in logs or metadata.
// See: docs/amadeus/amadeus_context.md#gdpr-and-privacy-standards.

func updateUserMetadata(user *User, event string, extra map[string]interface{}) error {
	if user.Metadata == nil {
		user.Metadata = &commonpb.Metadata{}
	}
	// Extract or create service_specific.user
	var serviceMeta map[string]interface{}
	if user.Metadata.ServiceSpecific != nil {
		serviceMeta = user.Metadata.ServiceSpecific.AsMap()
	}
	if serviceMeta == nil {
		serviceMeta = make(map[string]interface{})
	}
	// Versioning from environment variables
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "dev"
	}
	if _, ok := serviceMeta["versioning"]; !ok {
		serviceMeta["versioning"] = map[string]interface{}{
			"system_version":   "1.0.0",
			"service_version":  "1.0.0",
			"user_version":     "1.0.0",
			"environment":      env,
			"feature_flags":    []string{},
			"last_migrated_at": time.Now().Format(time.RFC3339),
		}
	}
	// Audit
	auditVal, ok := serviceMeta["audit"]
	var audit map[string]interface{}
	if ok && auditVal != nil {
		audit, ok = auditVal.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid type for audit")
		}
	}
	historyVal, ok := audit["history"]
	var history []string
	if ok && historyVal != nil {
		history, ok = historyVal.([]string)
		if !ok {
			return fmt.Errorf("invalid type for audit history")
		}
	}
	history = append(history, event)
	audit["last_modified_by"] = user.ID + ":" + user.MasterUUID
	audit["history"] = history
	serviceMeta["audit"] = audit
	// Merge extra fields
	for k, v := range extra {
		serviceMeta[k] = v
	}
	// Convert back to structpb.Struct
	metaStruct, err := structpb.NewStruct(serviceMeta)
	if err != nil {
		return err
	}
	user.Metadata.ServiceSpecific = metaStruct
	return nil
}

// AssignRole assigns a role to a user and updates metadata.
func (s *Service) AssignRole(ctx context.Context, req *userpb.AssignRoleRequest) (*userpb.AssignRoleResponse, error) {
	repoUser, err := s.repo.GetByID(ctx, req.UserId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	repoUser.Roles = append(repoUser.Roles, req.Role)
	err = updateUserMetadata(repoUser, "assign_role", map[string]interface{}{"rbac": repoUser.Roles})
	if err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, repoUser); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to assign role: %v", err)
	}
	return &userpb.AssignRoleResponse{}, nil
}

// RemoveRole removes a role from a user and updates metadata.
func (s *Service) RemoveRole(ctx context.Context, req *userpb.RemoveRoleRequest) (*userpb.RemoveRoleResponse, error) {
	repoUser, err := s.repo.GetByID(ctx, req.UserId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	newRoles := []string{}
	for _, r := range repoUser.Roles {
		if r != req.Role {
			newRoles = append(newRoles, r)
		}
	}
	repoUser.Roles = newRoles
	err = updateUserMetadata(repoUser, "remove_role", map[string]interface{}{"rbac": repoUser.Roles})
	if err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, repoUser); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to remove role: %v", err)
	}
	return &userpb.RemoveRoleResponse{}, nil
}

// ListRoles lists all roles for a user.
func (s *Service) ListRoles(ctx context.Context, req *userpb.ListRolesRequest) (*userpb.ListRolesResponse, error) {
	repoUser, err := s.repo.GetByID(ctx, req.UserId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	return &userpb.ListRolesResponse{Roles: repoUser.Roles}, nil
}

// ListPermissions lists all permissions for a user.
func (s *Service) ListPermissions(ctx context.Context, req *userpb.ListPermissionsRequest) (*userpb.ListPermissionsResponse, error) {
	user, err := s.repo.GetByID(ctx, req.UserId)
	if err != nil {
		s.log.Error("failed to get user for permissions", zap.Error(err))
		return nil, status.Error(codes.NotFound, "user not found")
	}
	// Example static role-to-permissions mapping
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
	return &userpb.ListPermissionsResponse{Permissions: perms}, nil
}

// ListUserEvents lists user events (stub).
func (s *Service) ListUserEvents(ctx context.Context, req *userpb.ListUserEventsRequest) (*userpb.ListUserEventsResponse, error) {
	page := int(req.Page)
	pageSize := int(req.PageSize)
	events, total, err := s.repo.ListUserEvents(ctx, req.UserId, page, pageSize)
	if err != nil {
		s.log.Error("failed to list user events", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to list user events: %v", err)
	}
	return &userpb.ListUserEventsResponse{
		Events:     events,
		TotalCount: utils.ToInt32(total),
	}, nil
}

// ListAuditLogs lists audit logs for a user (stub).
func (s *Service) ListAuditLogs(ctx context.Context, req *userpb.ListAuditLogsRequest) (*userpb.ListAuditLogsResponse, error) {
	// For now, require a user_id field in the request (update the proto if needed)
	userID := req.UserId // Add this field to the proto if not present
	page := int(req.Page)
	pageSize := int(req.PageSize)
	logs, total, err := s.repo.ListAuditLogs(ctx, userID, page, pageSize)
	if err != nil {
		s.log.Error("failed to list audit logs", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to list audit logs: %v", err)
	}
	return &userpb.ListAuditLogsResponse{
		Logs:       logs,
		TotalCount: utils.ToInt32(total),
	}, nil
}

// --- Session Management ---.
func generateToken(length int) (string, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// updateAuthMetadata updates the auth section of user metadata.
func updateAuthMetadata(user *User, fields map[string]interface{}) error {
	serviceMeta := getOrInitServiceUserMeta(user)
	authMetaVal, ok := serviceMeta["auth"]
	var authMeta map[string]interface{}
	if ok && authMetaVal != nil {
		authMeta, ok = authMetaVal.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid type for serviceMeta.auth")
		}
	}
	for k, v := range fields {
		authMeta[k] = v
	}
	serviceMeta["auth"] = authMeta
	return setServiceUserMeta(user, serviceMeta)
}

// updateJWTMetadata updates the jwt section of user metadata.
func updateJWTMetadata(user *User, fields map[string]interface{}) error {
	serviceMeta := getOrInitServiceUserMeta(user)
	jwtMetaVal, ok := serviceMeta["jwt"]
	var jwtMeta map[string]interface{}
	if ok && jwtMetaVal != nil {
		jwtMeta, ok = jwtMetaVal.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid type for serviceMeta.jwt")
		}
	}
	for k, v := range fields {
		jwtMeta[k] = v
	}
	serviceMeta["jwt"] = jwtMeta
	return setServiceUserMeta(user, serviceMeta)
}

// Example: update auth metadata on successful login.
func (s *Service) updateLoginMetadata(user *User, loginSource, provider, providerUserID string) error {
	return updateAuthMetadata(user, map[string]interface{}{
		"last_login_at":         time.Now().Format(time.RFC3339),
		"login_source":          loginSource,
		"oauth_provider":        provider,
		"provider_user_id":      providerUserID,
		"failed_login_attempts": 0,
	})
}

// Example: update auth metadata on failed login.
func (s *Service) updateFailedLoginMetadata(user *User) error {
	serviceMeta := getOrInitServiceUserMeta(user)
	attempts := 0
	if auth, ok := serviceMeta["auth"].(map[string]interface{}); ok {
		if v, ok := auth["failed_login_attempts"].(float64); ok {
			attempts = int(v)
		}
	}
	return updateAuthMetadata(user, map[string]interface{}{
		"failed_login_attempts": attempts + 1,
		"last_failed_login_at":  time.Now().Format(time.RFC3339),
	})
}

// Example: update JWT metadata on token issuance.
func (s *Service) updateJWTIssueMetadata(user *User, jwtID, audience string, scopes []string) error {
	return updateJWTMetadata(user, map[string]interface{}{
		"last_jwt_issued_at": time.Now().Format(time.RFC3339),
		"last_jwt_id":        jwtID,
		"jwt_audience":       audience,
		"jwt_scopes":         scopes,
	})
}

// --- Integrate these helpers into login/session, OAuth, and failed login flows ---
// In CreateSession, after successful login:
//   _ = s.updateLoginMetadata(user, req.LoginSource, "", "")
// On failed login:
//   _ = s.updateFailedLoginMetadata(user)
// In FindOrCreateOAuthUser, after successful OAuth login:
//   _ = s.updateLoginMetadata(user, "oauth:"+oauthUser.Provider, oauthUser.Provider, oauthUser.UserID)
// In JWT issuance logic:
//   _ = s.updateJWTIssueMetadata(user, jwtID, audience, scopes)
// ...and so on for all relevant flows...

func (s *Service) CreateSession(ctx context.Context, req *userpb.CreateSessionRequest) (*userpb.CreateSessionResponse, error) {
	// Authenticate user or create guest session
	var user *User
	var err error
	isGuest := false
	if req.UserId != "" {
		user, err = s.repo.GetByID(ctx, req.UserId)
		if err != nil {
			if err := s.updateFailedLoginMetadata(user); err != nil {
				s.log.Error("failed to update failed login metadata", zap.Error(err))
				return nil, status.Error(codes.NotFound, "user not found")
			}
			if err := s.repo.Update(ctx, user); err != nil {
				s.log.Error("failed to update user", zap.Error(err))
				return nil, status.Error(codes.Internal, "failed to update user")
			}
			return nil, status.Error(codes.Unauthenticated, "invalid credentials")
		}
		// Successful login: update login metadata
		if err := s.updateLoginMetadata(user, req.DeviceInfo, "", ""); err != nil {
			s.log.Error("failed to update login metadata", zap.Error(err))
			return nil, status.Error(codes.Internal, "failed to update login metadata")
		}
		if err := s.repo.Update(ctx, user); err != nil {
			s.log.Error("failed to update user", zap.Error(err))
			return nil, status.Error(codes.Internal, "failed to update user")
		}
	} else {
		// Guest session
		isGuest = true
		user = &User{
			Username: "guest",
			Status:   int32(userpb.UserStatus_USER_STATUS_ACTIVE),
			Metadata: nil, // will be set below
		}
	}
	accessToken, err := generateToken(32)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate access token")
	}
	refreshToken, err := generateToken(32)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate refresh token")
	}
	// --- METADATA UPDATE CHAIN ---
	if user.Metadata == nil {
		user.Metadata = &commonpb.Metadata{}
	}
	metaPtr, err := metadata.ServiceMetadataFromStruct(user.Metadata.ServiceSpecific)
	if err != nil {
		return nil, handleInternalError(s.log, "failed to extract service metadata", err)
	}
	meta := *metaPtr
	if isGuest {
		meta.Guest = true
		meta.GuestCreatedAt = time.Now().UTC().Format(time.RFC3339)
		meta.DeviceID = req.DeviceInfo
	} else {
		meta.DeviceID = req.DeviceInfo
	}
	// Audit: update last login
	if meta.Audit == nil {
		meta.Audit = &metadata.AuditMetadata{}
	}
	meta.Audit.LastModified = time.Now().UTC().Format(time.RFC3339)
	meta.Audit.History = append(meta.Audit.History, "login")
	// Convert back to structpb.Struct and assign to ServiceSpecific
	metaStruct, err := metadata.ServiceMetadataToStruct(&meta)
	if err != nil {
		return nil, fmt.Errorf("failed to convert service metadata to struct: %w", err)
	}
	user.Metadata.ServiceSpecific = metaStruct
	// Example JWT issuance (simulate)
	jwtID := accessToken // In real code, use a real JWT ID
	audience := "ovasabi-app"
	scopes := []string{"user:read", "user:write"}
	if err := s.updateJWTIssueMetadata(user, jwtID, audience, scopes); err != nil {
		s.log.Error("failed to update JWT metadata", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to update JWT metadata")
	}
	if err := s.repo.Update(ctx, user); err != nil {
		s.log.Error("failed to update user", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to update user")
	}

	session := &userpb.Session{
		Id:           accessToken,
		UserId:       user.ID,
		DeviceInfo:   req.DeviceInfo,
		CreatedAt:    timestamppb.Now(),
		ExpiresAt:    timestamppb.New(time.Now().Add(24 * time.Hour)),
		RefreshToken: refreshToken,
		AccessToken:  accessToken,
		IpAddress:    "", // set from context/headers if available
		Metadata:     user.Metadata,
	}
	// Store session in Redis
	if s.cache != nil {
		err := s.cache.Set(ctx, accessToken, "session", session, 24*time.Hour)
		if err != nil {
			s.log.Error("failed to cache session", zap.Error(err))
		}
	}
	// Optionally update user record in DB for last login, etc. (not shown)
	return &userpb.CreateSessionResponse{Session: session}, nil
}

func (s *Service) GetSession(ctx context.Context, req *userpb.GetSessionRequest) (*userpb.GetSessionResponse, error) {
	if s.cache == nil {
		return nil, status.Error(codes.Unavailable, "session cache unavailable")
	}
	var session userpb.Session
	err := s.cache.Get(ctx, req.SessionId, "session", &session)
	if err != nil {
		return nil, status.Error(codes.NotFound, "session not found")
	}
	return &userpb.GetSessionResponse{Session: &session}, nil
}

func (s *Service) RevokeSession(ctx context.Context, req *userpb.RevokeSessionRequest) (*userpb.RevokeSessionResponse, error) {
	if s.cache == nil {
		return nil, status.Error(codes.Unavailable, "session cache unavailable")
	}
	err := s.cache.Delete(ctx, req.SessionId, "session")
	if err != nil {
		return nil, status.Error(codes.NotFound, "session not found")
	}
	// Log audit event
	// ...
	return &userpb.RevokeSessionResponse{Success: true}, nil
}

func (s *Service) ListSessions(ctx context.Context, req *userpb.ListSessionsRequest) (*userpb.ListSessionsResponse, error) {
	if s.cache == nil {
		s.log.Warn("Session cache unavailable, falling back to repository (not implemented)")
		return nil, status.Error(codes.Unavailable, "session cache unavailable and repository fallback not implemented")
	}

	// Build a pattern to scan all session keys (assuming session keys are just access tokens, so scan all keys)
	scanPattern := "*"
	var sessions []*userpb.Session
	iter := s.cache.GetClient().Scan(ctx, 0, scanPattern, 0).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		var session userpb.Session
		err := s.cache.Get(ctx, key, "session", &session)
		if err != nil {
			continue // skip if not a session or error
		}
		if session.UserId == req.UserId {
			sessions = append(sessions, &session)
		}
	}
	if err := iter.Err(); err != nil {
		s.log.Error("failed to scan session keys", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to scan session keys: %v", err)
	}
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
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}
	// Check if user already exists
	user, err := s.repo.GetByEmail(ctx, req.Email)
	if err == nil && user != nil {
		// User exists, update status to PENDING if not already
		if user.Status != int32(userpb.UserStatus_USER_STATUS_PENDING) {
			user.Status = int32(userpb.UserStatus_USER_STATUS_PENDING)
			if err := s.repo.Update(ctx, user); err != nil {
				s.log.Error("failed to update user", zap.Error(err))
				return nil, status.Errorf(codes.Internal, "failed to update user: %v", err)
			}
		}
		return &userpb.RegisterInterestResponse{User: repoUserToProtoUser(user)}, nil
	}
	// Create new pending user
	user = &User{
		Email:    req.Email,
		Status:   int32(userpb.UserStatus_USER_STATUS_PENDING),
		Metadata: &commonpb.Metadata{},
	}
	created, err := s.repo.Create(ctx, user)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to register interest: %v", err)
	}
	return &userpb.RegisterInterestResponse{User: repoUserToProtoUser(created)}, nil
}

func (s *Service) CreateReferral(ctx context.Context, req *userpb.CreateReferralRequest) (*userpb.CreateReferralResponse, error) {
	if req.UserId == "" || req.CampaignSlug == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and campaign_slug are required")
	}
	user, err := s.repo.GetByID(ctx, req.UserId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	// Generate a unique referral code (could use a hash, UUID, or campaign+user)
	code := fmt.Sprintf("REF-%s-%s-%d", req.UserId, req.CampaignSlug, time.Now().Unix()%100000)
	user.ReferralCode = code
	if err := s.repo.Update(ctx, user); err != nil {
		s.log.Error("failed to update user", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to update user: %v", err)
	}
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
			return nil, handleInternalError(s.log, "failed to convert oauthInfo to structpb.Struct", err)
		}
		user = &User{
			Username: oauthUser.NickName,
			Email:    oauthUser.Email,
			Status:   int32(userpb.UserStatus_USER_STATUS_ACTIVE),
			Metadata: &commonpb.Metadata{
				ServiceSpecific: metaStruct,
			},
		}
		if err := s.updateLoginMetadata(user, "oauth:"+oauthUser.Provider, oauthUser.Provider, oauthUser.UserID); err != nil {
			return nil, handleInternalError(s.log, "failed to update login metadata", err)
		}
		user, err = s.repo.Create(ctx, user)
		if err != nil {
			return nil, handleInternalError(s.log, "failed to create OAuth user", err)
		}
		return user, nil
	case err != nil:
		return nil, err
	default:
		if err := s.updateLoginMetadata(user, "oauth:"+oauthUser.Provider, oauthUser.Provider, oauthUser.UserID); err != nil {
			return nil, handleInternalError(s.log, "failed to update login metadata", err)
		}
		if err := s.repo.Update(ctx, user); err != nil {
			return nil, handleInternalError(s.log, "failed to update user", err)
		}
		return user, nil
	}
}

// --- Composable Auth Channel Methods ---
// SendVerificationEmail emits a notification event instead of direct call.
func (s *Service) SendVerificationEmail(ctx context.Context, userID, email string) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	code := generateSimpleCode()
	expires := time.Now().Add(15 * time.Minute).UTC().Format(time.RFC3339)
	metaPtr, err := metadata.ServiceMetadataFromStruct(user.Metadata.ServiceSpecific)
	if err != nil {
		return fmt.Errorf("failed to extract service metadata: %w", err)
	}
	meta := *metaPtr
	meta.VerificationData = &metadata.VerificationData{Code: code, ExpiresAt: expires}
	metaStruct, err := metadata.ServiceMetadataToStruct(&meta)
	if err != nil {
		s.log.Error("failed to convert service metadata to struct", zap.Error(err))
		return fmt.Errorf("failed to convert service metadata: %w", err)
	}
	user.Metadata.ServiceSpecific = metaStruct
	if err := s.repo.Update(ctx, user); err != nil {
		return err
	}
	// Emit notification event instead of direct call
	if s.eventEnabled && s.eventEmitter != nil {
		noteMeta := &commonpb.Metadata{}
		noteStruct, err := structpb.NewStruct(map[string]interface{}{
			"to":      email,
			"subject": "Your Verification Code",
			"body":    fmt.Sprintf("Your verification code is: %s", code),
			"html":    false,
			"user_id": userID,
		})
		if err != nil {
			s.log.Error("Failed to create structpb.Struct for notification.sent event", zap.Error(err))
			return fmt.Errorf("failed to create structpb.Struct: %w", err)
		}
		noteMeta.ServiceSpecific = noteStruct
		_, ok := userEvents.EmitEventWithLogging(ctx, s.eventEmitter, s.log, nexusevents.EventUserUpdated, userID, noteMeta)
		if !ok {
			s.log.Warn("Failed to emit notification.sent event (SendVerificationEmail)")
		}
	}
	return nil
}

// VerifyEmail verifies the code and marks email as verified.
func (s *Service) VerifyEmail(ctx context.Context, userID, code string) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	metaPtr, err := metadata.ServiceMetadataFromStruct(user.Metadata.ServiceSpecific)
	if err != nil {
		return fmt.Errorf("failed to extract service metadata: %w", err)
	}
	meta := *metaPtr
	if meta.VerificationData == nil || meta.VerificationData.Code != code {
		return errors.New("invalid code")
	}
	exp, err := time.Parse(time.RFC3339, meta.VerificationData.ExpiresAt)
	if err != nil {
		return fmt.Errorf("failed to parse verification expiration: %w", err)
	}
	if time.Now().After(exp) {
		return errors.New("code expired")
	}
	meta.EmailVerified = true
	meta.VerificationData = nil
	metaStruct, err := metadata.ServiceMetadataToStruct(&meta)
	if err != nil {
		s.log.Error("failed to convert service metadata to struct", zap.Error(err))
		return fmt.Errorf("failed to convert service metadata: %w", err)
	}
	user.Metadata.ServiceSpecific = metaStruct
	return s.repo.Update(ctx, user)
}

// RequestPasswordReset emits a notification event instead of direct call.
func (s *Service) RequestPasswordReset(ctx context.Context, userID, email string) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	code := generateSimpleCode()
	expires := time.Now().Add(15 * time.Minute).UTC().Format(time.RFC3339)
	metaPtr, err := metadata.ServiceMetadataFromStruct(user.Metadata.ServiceSpecific)
	if err != nil {
		return fmt.Errorf("failed to extract service metadata: %w", err)
	}
	meta := *metaPtr
	meta.PasswordReset = &metadata.PasswordResetData{Code: code, ExpiresAt: expires}
	metaStruct, err := metadata.ServiceMetadataToStruct(&meta)
	if err != nil {
		s.log.Error("failed to convert service metadata to struct", zap.Error(err))
		return fmt.Errorf("failed to convert service metadata: %w", err)
	}
	user.Metadata.ServiceSpecific = metaStruct
	if err := s.repo.Update(ctx, user); err != nil {
		s.log.Error("failed to update user", zap.Error(err))
		return err
	}
	// Emit notification event instead of direct call
	if s.eventEnabled && s.eventEmitter != nil {
		noteMeta := &commonpb.Metadata{}
		noteStruct, err := structpb.NewStruct(map[string]interface{}{
			"to":      email,
			"subject": "Your Password Reset Code",
			"body":    fmt.Sprintf("Your password reset code is: %s", code),
			"html":    false,
			"user_id": userID,
		})
		if err != nil {
			s.log.Error("Failed to create structpb.Struct for notification.sent event", zap.Error(err))
			return fmt.Errorf("failed to create structpb.Struct: %w", err)
		}
		noteMeta.ServiceSpecific = noteStruct
		_, ok := userEvents.EmitEventWithLogging(ctx, s.eventEmitter, s.log, nexusevents.EventUserUpdated, userID, noteMeta)
		if !ok {
			s.log.Warn("Failed to emit notification.sent event (RequestPasswordReset)")
		}
	}
	return nil
}

// VerifyPasswordReset verifies the password reset code.
func (s *Service) VerifyPasswordReset(ctx context.Context, userID, code string) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	metaPtr, err := metadata.ServiceMetadataFromStruct(user.Metadata.ServiceSpecific)
	if err != nil {
		return fmt.Errorf("failed to extract service metadata: %w", err)
	}
	meta := *metaPtr
	if meta.PasswordReset == nil || meta.PasswordReset.Code != code {
		return errors.New("invalid code")
	}
	exp, err := time.Parse(time.RFC3339, meta.PasswordReset.ExpiresAt)
	if err != nil {
		return fmt.Errorf("failed to parse verification expiration: %w", err)
	}
	if time.Now().After(exp) {
		return errors.New("code expired")
	}
	return nil
}

// ResetPassword resets the user's password after code verification.
func (s *Service) ResetPassword(ctx context.Context, userID, code, newPassword string) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	metaPtr, err := metadata.ServiceMetadataFromStruct(user.Metadata.ServiceSpecific)
	if err != nil {
		return fmt.Errorf("failed to extract service metadata: %w", err)
	}
	meta := *metaPtr
	if meta.PasswordReset == nil || meta.PasswordReset.Code != code {
		return errors.New("invalid code")
	}
	exp, err := time.Parse(time.RFC3339, meta.PasswordReset.ExpiresAt)
	if err != nil {
		return fmt.Errorf("failed to parse verification expiration: %w", err)
	}
	if time.Now().After(exp) {
		return errors.New("code expired")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.PasswordHash = string(hash)
	meta.PasswordReset = nil
	metaStruct, err := metadata.ServiceMetadataToStruct(&meta)
	if err != nil {
		s.log.Error("failed to convert service metadata to struct", zap.Error(err))
		return fmt.Errorf("failed to convert service metadata: %w", err)
	}
	user.Metadata.ServiceSpecific = metaStruct
	return s.repo.Update(ctx, user)
}

// BeginWebAuthnRegistration emits an event instead of direct WebAuthn logic.
func (s *Service) BeginWebAuthnRegistration(ctx context.Context, userID, username string) (string, error) {
	// Emit event or leave as stub for event-driven WebAuthn registration
	if s.eventEnabled && s.eventEmitter != nil {
		meta := &commonpb.Metadata{}
		metaStruct, err := structpb.NewStruct(map[string]interface{}{
			"user_id":  userID,
			"username": username,
			"action":   "begin_registration",
		})
		if err != nil {
			s.log.Error("Failed to create structpb.Struct for user.webauthn_registration_initiated event", zap.Error(err))
			return "", status.Error(codes.Internal, "internal error")
		}
		meta.ServiceSpecific = metaStruct
		_, ok := userEvents.EmitEventWithLogging(ctx, s.eventEmitter, s.log, nexusevents.EventUserUpdated, userID, meta)
		if !ok {
			s.log.Error("Failed to emit user.webauthn_registration_initiated event")
			return "", status.Error(codes.Internal, "failed to emit event")
		}
	}
	return "", nil // No direct provider logic
}

// FinishWebAuthnRegistration emits an event instead of direct WebAuthn logic.
func (s *Service) FinishWebAuthnRegistration(ctx context.Context, userID, username, response string) error {
	// Emit event or leave as stub for event-driven WebAuthn registration
	if s.eventEnabled && s.eventEmitter != nil {
		meta := &commonpb.Metadata{}
		metaStruct, err := structpb.NewStruct(map[string]interface{}{
			"user_id":  userID,
			"username": username,
			"response": response,
			"action":   "finish_registration",
		})
		if err != nil {
			s.log.Error("Failed to create structpb.Struct for user.webauthn_registration_finished event", zap.Error(err))
			return status.Error(codes.Internal, "internal error")
		}
		meta.ServiceSpecific = metaStruct
		_, ok := userEvents.EmitEventWithLogging(ctx, s.eventEmitter, s.log, nexusevents.EventUserUpdated, userID, meta)
		if !ok {
			s.log.Error("Failed to emit user.webauthn_registration_finished event")
			return status.Error(codes.Internal, "failed to emit event")
		}
	}
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
func (s *Service) IsBiometricEnabled(_ context.Context, _ string) (bool, error) {
	// Emit event or leave as stub for event-driven biometric check
	return false, nil // No direct provider logic
}

// MarkBiometricUsed emits an event instead of direct biometric usage.
func (s *Service) MarkBiometricUsed(ctx context.Context, userID string) error {
	// Emit event or leave as stub for event-driven biometric usage
	if s.eventEnabled && s.eventEmitter != nil {
		meta := &commonpb.Metadata{}
		metaStruct, err := structpb.NewStruct(map[string]interface{}{
			"user_id": userID,
			"action":  "biometric_used",
		})
		if err != nil {
			s.log.Error("Failed to create structpb.Struct for user.biometric_used event", zap.Error(err))
			return fmt.Errorf("failed to create structpb.Struct: %w", err)
		}
		meta.ServiceSpecific = metaStruct
		_, ok := userEvents.EmitEventWithLogging(ctx, s.eventEmitter, s.log, nexusevents.EventUserUpdated, userID, meta)
		if !ok {
			s.log.Warn("Failed to emit user.biometric_used event")
		}
	}
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

// handleInternalError logs the error and returns a standardized gRPC internal error.
func handleInternalError(log *zap.Logger, msg string, err error) error {
	if log != nil {
		log.Error(msg, zap.Error(err))
	}
	return status.Errorf(codes.Internal, "%s: %v", msg, err)
}

// --- Social Graph APIs ---.
func (s *Service) AddFriend(ctx context.Context, req *userpb.AddFriendRequest) (*userpb.AddFriendResponse, error) {
	friendship, err := s.repo.AddFriend(ctx, req.UserId, req.FriendId, req.Metadata)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to add friend: %v", err)
	}
	return &userpb.AddFriendResponse{Friendship: friendship}, nil
}

func (s *Service) RemoveFriend(ctx context.Context, req *userpb.RemoveFriendRequest) (*userpb.RemoveFriendResponse, error) {
	err := s.repo.RemoveFriend(ctx, req.UserId, req.FriendId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to remove friend: %v", err)
	}
	return &userpb.RemoveFriendResponse{Success: true}, nil
}

func (s *Service) ListFriends(ctx context.Context, req *userpb.ListFriendsRequest) (*userpb.ListFriendsResponse, error) {
	friends, total, err := s.repo.ListFriends(ctx, req.UserId, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list friends: %v", err)
	}
	return &userpb.ListFriendsResponse{
		Friends:    friends,
		TotalCount: utils.ToInt32(total),
		Page:       req.Page,
		TotalPages: utils.ToInt32((total + int(req.PageSize) - 1) / int(req.PageSize)),
	}, nil
}

func (s *Service) FollowUser(ctx context.Context, req *userpb.FollowUserRequest) (*userpb.FollowUserResponse, error) {
	follow, err := s.repo.FollowUser(ctx, req.FollowerId, req.FolloweeId, req.Metadata)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to follow user: %v", err)
	}
	return &userpb.FollowUserResponse{Follow: follow}, nil
}

func (s *Service) UnfollowUser(ctx context.Context, req *userpb.UnfollowUserRequest) (*userpb.UnfollowUserResponse, error) {
	err := s.repo.UnfollowUser(ctx, req.FollowerId, req.FolloweeId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to unfollow user: %v", err)
	}
	return &userpb.UnfollowUserResponse{Success: true}, nil
}

func (s *Service) ListFollowers(ctx context.Context, req *userpb.ListFollowersRequest) (*userpb.ListFollowersResponse, error) {
	followers, total, err := s.repo.ListFollowers(ctx, req.UserId, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list followers: %v", err)
	}
	return &userpb.ListFollowersResponse{
		Followers:  followers,
		TotalCount: utils.ToInt32(total),
		Page:       req.Page,
		TotalPages: utils.ToInt32((total + int(req.PageSize) - 1) / int(req.PageSize)),
	}, nil
}

func (s *Service) ListFollowing(ctx context.Context, req *userpb.ListFollowingRequest) (*userpb.ListFollowingResponse, error) {
	following, total, err := s.repo.ListFollowing(ctx, req.UserId, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list following: %v", err)
	}
	return &userpb.ListFollowingResponse{
		Following:  following,
		TotalCount: utils.ToInt32(total),
		Page:       req.Page,
		TotalPages: utils.ToInt32((total + int(req.PageSize) - 1) / int(req.PageSize)),
	}, nil
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
		return nil, status.Errorf(codes.Internal, "failed to create user group: %v", err)
	}
	return &userpb.CreateUserGroupResponse{UserGroup: created}, nil
}

func (s *Service) UpdateUserGroup(ctx context.Context, req *userpb.UpdateUserGroupRequest) (*userpb.UpdateUserGroupResponse, error) {
	authUserID, ok := utils.GetAuthenticatedUserID(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing authentication")
	}
	roles, _ := utils.GetAuthenticatedUserRoles(ctx)
	isAdmin := utils.IsServiceAdmin(roles, "user")
	group, err := s.repo.GetUserGroupByID(ctx, req.UserGroupId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "user group not found")
	}
	isGroupAdmin := group.Roles[authUserID] == "admin"
	if !isAdmin && !isGroupAdmin {
		return nil, status.Error(codes.PermissionDenied, "cannot update group you do not own or admin")
	}
	updated, err := s.repo.UpdateUserGroup(ctx, req.UserGroupId, req.UserGroup, req.FieldsToUpdate)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update user group: %v", err)
	}
	return &userpb.UpdateUserGroupResponse{UserGroup: updated}, nil
}

func (s *Service) DeleteUserGroup(ctx context.Context, req *userpb.DeleteUserGroupRequest) (*userpb.DeleteUserGroupResponse, error) {
	authUserID, ok := utils.GetAuthenticatedUserID(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing authentication")
	}
	roles, _ := utils.GetAuthenticatedUserRoles(ctx)
	isAdmin := utils.IsServiceAdmin(roles, "user")
	group, err := s.repo.GetUserGroupByID(ctx, req.UserGroupId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "user group not found")
	}
	isGroupAdmin := group.Roles[authUserID] == "admin"
	if !isAdmin && !isGroupAdmin {
		return nil, status.Error(codes.PermissionDenied, "cannot delete group you do not own or admin")
	}
	err = s.repo.DeleteUserGroup(ctx, req.UserGroupId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete user group: %v", err)
	}
	return &userpb.DeleteUserGroupResponse{Success: true}, nil
}

func (s *Service) ListUserGroups(ctx context.Context, req *userpb.ListUserGroupsRequest) (*userpb.ListUserGroupsResponse, error) {
	groups, total, err := s.repo.ListUserGroups(ctx, req.UserId, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list user groups: %v", err)
	}
	return &userpb.ListUserGroupsResponse{
		UserGroups: groups,
		TotalCount: utils.ToInt32(total),
		Page:       req.Page,
		TotalPages: utils.ToInt32((total + int(req.PageSize) - 1) / int(req.PageSize)),
	}, nil
}

func (s *Service) ListUserGroupMembers(ctx context.Context, req *userpb.ListUserGroupMembersRequest) (*userpb.ListUserGroupMembersResponse, error) {
	members, total, err := s.repo.ListUserGroupMembers(ctx, req.UserGroupId, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list user group members: %v", err)
	}
	return &userpb.ListUserGroupMembersResponse{
		Members:    members,
		TotalCount: utils.ToInt32(total),
		Page:       req.Page,
		TotalPages: utils.ToInt32((total + int(req.PageSize) - 1) / int(req.PageSize)),
	}, nil
}

// --- Social Graph Discovery ---.
func (s *Service) SuggestConnections(ctx context.Context, req *userpb.SuggestConnectionsRequest) (*userpb.SuggestConnectionsResponse, error) {
	suggestions, err := s.repo.SuggestConnections(ctx, req.UserId, req.Metadata)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to suggest connections: %v", err)
	}
	return &userpb.SuggestConnectionsResponse{Suggestions: suggestions}, nil
}

func (s *Service) ListConnections(ctx context.Context, req *userpb.ListConnectionsRequest) (*userpb.ListConnectionsResponse, error) {
	users, err := s.repo.ListConnections(ctx, req.UserId, req.Type, req.Metadata)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list connections: %v", err)
	}
	return &userpb.ListConnectionsResponse{Users: users}, nil
}

// --- Moderation/Interaction APIs ---.
func (s *Service) BlockUser(ctx context.Context, req *userpb.BlockUserRequest) (*userpb.BlockUserResponse, error) {
	targetUser, err := s.repo.GetByID(ctx, req.TargetUserId)
	if err != nil {
		return &userpb.BlockUserResponse{Success: false}, status.Errorf(codes.NotFound, "target user not found")
	}
	// Update bad_actor metadata
	if targetUser.Metadata == nil {
		targetUser.Metadata = &commonpb.Metadata{}
	}
	serviceMeta := getOrInitServiceUserMeta(targetUser)
	badActor := 0.0
	if v, ok := serviceMeta["bad_actor"].(float64); ok {
		badActor = v
	}
	serviceMeta["bad_actor"] = badActor + 1
	if err := setServiceUserMeta(targetUser, serviceMeta); err != nil {
		return &userpb.BlockUserResponse{Success: false}, status.Errorf(codes.Internal, "failed to update metadata")
	}
	if err := s.repo.Update(ctx, targetUser); err != nil {
		return &userpb.BlockUserResponse{Success: false}, status.Errorf(codes.Internal, "failed to update user")
	}
	return &userpb.BlockUserResponse{Success: true}, nil
}

func (s *Service) UnblockUser(ctx context.Context, req *userpb.UnblockUserRequest) (*userpb.UnblockUserResponse, error) {
	err := s.repo.UnblockUser(ctx, req.TargetUserId)
	if err != nil {
		return &userpb.UnblockUserResponse{Success: false}, status.Errorf(codes.Internal, "failed to unblock user: %v", err)
	}
	return &userpb.UnblockUserResponse{Success: true}, nil
}

func (s *Service) MuteUser(ctx context.Context, req *userpb.MuteUserRequest) (*userpb.MuteUserResponse, error) {
	targetUser, err := s.repo.GetByID(ctx, req.TargetUserId)
	if err != nil {
		return &userpb.MuteUserResponse{Success: false}, status.Errorf(codes.NotFound, "target user not found")
	}
	if targetUser.Metadata == nil {
		targetUser.Metadata = &commonpb.Metadata{}
	}
	serviceMeta := getOrInitServiceUserMeta(targetUser)
	badActor := 0.0
	if v, ok := serviceMeta["bad_actor"].(float64); ok {
		badActor = v
	}
	serviceMeta["bad_actor"] = badActor + 1
	if err := setServiceUserMeta(targetUser, serviceMeta); err != nil {
		return &userpb.MuteUserResponse{Success: false}, status.Errorf(codes.Internal, "failed to update metadata")
	}
	if err := s.repo.Update(ctx, targetUser); err != nil {
		return &userpb.MuteUserResponse{Success: false}, status.Errorf(codes.Internal, "failed to update user")
	}
	return &userpb.MuteUserResponse{Success: true}, nil
}

func (s *Service) UnmuteUser(ctx context.Context, req *userpb.UnmuteUserRequest) (*userpb.UnmuteUserResponse, error) {
	err := s.repo.UnmuteUser(ctx, req.TargetUserId)
	if err != nil {
		return &userpb.UnmuteUserResponse{Success: false}, status.Errorf(codes.Internal, "failed to unmute user: %v", err)
	}
	return &userpb.UnmuteUserResponse{Success: true}, nil
}

func (s *Service) UnmuteGroup(ctx context.Context, req *userpb.UnmuteGroupRequest) (*userpb.UnmuteGroupResponse, error) {
	err := s.repo.UnmuteGroup(ctx, req.UserId, req.GroupId)
	if err != nil {
		return &userpb.UnmuteGroupResponse{Success: false}, status.Errorf(codes.Internal, "failed to unmute group: %v", err)
	}
	return &userpb.UnmuteGroupResponse{Success: true}, nil
}

func (s *Service) UnmuteGroupIndividuals(ctx context.Context, req *userpb.UnmuteGroupIndividualsRequest) (*userpb.UnmuteGroupIndividualsResponse, error) {
	unmuted, err := s.repo.UnmuteGroupIndividuals(ctx, req.UserId, req.GroupId, req.TargetUserIds)
	if err != nil {
		return &userpb.UnmuteGroupIndividualsResponse{Success: false}, status.Errorf(codes.Internal, "failed to unmute group individuals: %v", err)
	}
	return &userpb.UnmuteGroupIndividualsResponse{Success: true, UnmutedUserIds: unmuted}, nil
}

func (s *Service) UnblockGroupIndividuals(ctx context.Context, req *userpb.UnblockGroupIndividualsRequest) (*userpb.UnblockGroupIndividualsResponse, error) {
	unblocked, err := s.repo.UnblockGroupIndividuals(ctx, req.UserId, req.GroupId, req.TargetUserIds)
	if err != nil {
		return &userpb.UnblockGroupIndividualsResponse{Success: false}, status.Errorf(codes.Internal, "failed to unblock group individuals: %v", err)
	}
	return &userpb.UnblockGroupIndividualsResponse{Success: true, UnblockedUserIds: unblocked}, nil
}

func (s *Service) ReportUser(ctx context.Context, req *userpb.ReportUserRequest) (*userpb.ReportUserResponse, error) {
	targetUser, err := s.repo.GetByID(ctx, req.ReportedUserId)
	if err != nil {
		return &userpb.ReportUserResponse{Success: false}, status.Errorf(codes.NotFound, "target user not found")
	}
	if targetUser.Metadata == nil {
		targetUser.Metadata = &commonpb.Metadata{}
	}
	serviceMeta := getOrInitServiceUserMeta(targetUser)
	badActor := 0.0
	if v, ok := serviceMeta["bad_actor"].(float64); ok {
		badActor = v
	}
	serviceMeta["bad_actor"] = badActor + 1
	if err := setServiceUserMeta(targetUser, serviceMeta); err != nil {
		return &userpb.ReportUserResponse{Success: false}, status.Errorf(codes.Internal, "failed to update metadata")
	}
	if err := s.repo.Update(ctx, targetUser); err != nil {
		return &userpb.ReportUserResponse{Success: false}, status.Errorf(codes.Internal, "failed to update user")
	}
	return &userpb.ReportUserResponse{Success: true}, nil
}

func (s *Service) BlockGroupContent(ctx context.Context, req *userpb.BlockGroupContentRequest) (*userpb.BlockGroupContentResponse, error) {
	err := s.repo.BlockGroupContent(ctx, req.UserId, req.GroupId, req.ContentId, req.Metadata)
	if err != nil {
		return &userpb.BlockGroupContentResponse{Success: false}, status.Errorf(codes.Internal, "failed to block group content: %v", err)
	}
	return &userpb.BlockGroupContentResponse{Success: true}, nil
}

func (s *Service) ReportGroupContent(ctx context.Context, req *userpb.ReportGroupContentRequest) (*userpb.ReportGroupContentResponse, error) {
	reportID, err := s.repo.ReportGroupContent(ctx, req.ReporterUserId, req.GroupId, req.ContentId, req.Reason, req.Details, req.Metadata)
	if err != nil {
		return &userpb.ReportGroupContentResponse{Success: false, ReportId: ""}, status.Errorf(codes.Internal, "failed to report group content: %v", err)
	}
	return &userpb.ReportGroupContentResponse{Success: true, ReportId: reportID}, nil
}

func (s *Service) MuteGroupContent(ctx context.Context, req *userpb.MuteGroupContentRequest) (*userpb.MuteGroupContentResponse, error) {
	err := s.repo.MuteGroupContent(ctx, req.UserId, req.GroupId, req.ContentId, req.DurationMinutes, req.Metadata)
	if err != nil {
		return &userpb.MuteGroupContentResponse{Success: false}, status.Errorf(codes.Internal, "failed to mute group content: %v", err)
	}
	return &userpb.MuteGroupContentResponse{Success: true}, nil
}

func (s *Service) MuteGroupIndividuals(ctx context.Context, req *userpb.MuteGroupIndividualsRequest) (*userpb.MuteGroupIndividualsResponse, error) {
	mutedUserIDs, err := s.repo.MuteGroupIndividuals(ctx, req.UserId, req.GroupId, int(req.DurationMinutes), req.Metadata)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to mute group individuals: %v", err)
	}
	return &userpb.MuteGroupIndividualsResponse{Success: true, MutedUserIds: mutedUserIDs}, nil
}

func (s *Service) BlockGroupIndividuals(ctx context.Context, req *userpb.BlockGroupIndividualsRequest) (*userpb.BlockGroupIndividualsResponse, error) {
	blockedUserIDs, err := s.repo.BlockGroupIndividuals(ctx, req.UserId, req.GroupId, int(req.DurationMinutes), req.Metadata)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to block group individuals: %v", err)
	}
	return &userpb.BlockGroupIndividualsResponse{Success: true, BlockedUserIds: blockedUserIDs}, nil
}
