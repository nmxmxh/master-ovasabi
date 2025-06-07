package admin

import (
	"context"
	"time"

	adminpb "github.com/nmxmxh/master-ovasabi/api/protos/admin/v1"
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	userpb "github.com/nmxmxh/master-ovasabi/api/protos/user/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"google.golang.org/protobuf/types/known/structpb"
)

// EventEmitter defines the interface for emitting events.
type EventEmitter interface {
	EmitEventWithLogging(ctx context.Context, emitter interface{}, log *zap.Logger, eventType, eventID string, meta *commonpb.Metadata) (string, bool)
}

type Service struct {
	adminpb.UnimplementedAdminServiceServer
	log          *zap.Logger
	repo         *Repository
	userClient   userpb.UserServiceClient
	Cache        *redis.Cache
	eventEmitter EventEmitter
	eventEnabled bool
}

func NewService(log *zap.Logger, repo *Repository, userClient userpb.UserServiceClient, cache *redis.Cache, eventEmitter EventEmitter, eventEnabled bool) adminpb.AdminServiceServer {
	return &Service{
		log:          log,
		repo:         repo,
		userClient:   userClient,
		Cache:        cache,
		eventEmitter: eventEmitter,
		eventEnabled: eventEnabled,
	}
}

var _ adminpb.AdminServiceServer = (*Service)(nil)

// Adapter to bridge s.eventEmitter to the required orchestration EventEmitter interface.
type EventEmitterAdapter struct {
	Emitter EventEmitter
}

func (a *EventEmitterAdapter) EmitRawEventWithLogging(ctx context.Context, log *zap.Logger, eventType, eventID string, payload []byte) (string, bool) {
	if emitter, ok := a.Emitter.(interface {
		EmitRawEventWithLogging(context.Context, *zap.Logger, string, string, []byte) (string, bool)
	}); ok {
		return emitter.EmitRawEventWithLogging(ctx, log, eventType, eventID, payload)
	}
	return "", false
}

func (a *EventEmitterAdapter) EmitEventWithLogging(ctx context.Context, event interface{}, log *zap.Logger, eventType, eventID string, meta *commonpb.Metadata) (string, bool) {
	return a.Emitter.EmitEventWithLogging(ctx, event, log, eventType, eventID, meta)
}

// User management.
func (s *Service) CreateUser(ctx context.Context, req *adminpb.CreateUserRequest) (*adminpb.CreateUserResponse, error) {
	email := req.User.Email
	var mainUser *userpb.User
	userResp, err := s.userClient.GetUserByEmail(ctx, &userpb.GetUserByEmailRequest{Email: email})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			createResp, err := s.userClient.CreateUser(ctx, &userpb.CreateUserRequest{
				Email:    email,
				Username: req.User.Name,
			})
			if err != nil {
				errObj := graceful.MapAndWrapErr(ctx, err, "failed to create main user", codes.Internal)
				errObj.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{
					Metadata:     req.User.Metadata,
					EventType:    "admin.user_create_error",
					EventID:      email,
					PatternType:  "admin_user",
					PatternID:    email,
					EventEmitter: &EventEmitterAdapter{Emitter: s.eventEmitter},
					EventEnabled: s.eventEnabled,
				})
				return nil, graceful.ToStatusError(errObj)
			}
			mainUser = createResp.User
		} else {
			errObj := graceful.MapAndWrapErr(ctx, err, "failed to lookup main user", codes.Internal)
			errObj.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{
				Metadata:     req.User.Metadata,
				EventType:    "admin.user_lookup_error",
				EventID:      email,
				PatternType:  "admin_user",
				PatternID:    email,
				EventEmitter: &EventEmitterAdapter{Emitter: s.eventEmitter},
				EventEnabled: s.eventEnabled,
			})
			return nil, graceful.ToStatusError(errObj)
		}
	} else {
		mainUser = userResp.User
	}
	// Enrich metadata
	if req.User.Metadata == nil {
		req.User.Metadata = &commonpb.Metadata{}
	}
	// Set versioning using canonical helper
	versioning := map[string]interface{}{
		"system_version":   "1.0.0",
		"service_version":  "1.0.0",
		"environment":      "prod",
		"last_migrated_at": time.Now().Format(time.RFC3339),
	}
	if err := metadata.SetServiceSpecificField(req.User.Metadata, "admin", "versioning", versioning); err != nil {
		errObj := graceful.MapAndWrapErr(ctx, err, "failed to set admin versioning", codes.Internal)
		errObj.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{
			Metadata:     req.User.Metadata,
			EventType:    "admin.user_metadata_error",
			EventID:      email,
			PatternType:  "admin_user",
			PatternID:    email,
			EventEmitter: &EventEmitterAdapter{Emitter: s.eventEmitter},
			EventEnabled: s.eventEnabled,
		})
		return nil, graceful.ToStatusError(errObj)
	}

	adminUser, err := s.repo.CreateUser(ctx, &adminpb.User{
		Id:       mainUser.Id,
		MasterId: mainUser.MasterId, // propagate master_id
		Email:    email,
		Name:     req.User.Name,
		Metadata: req.User.Metadata,
	})
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to create admin user", codes.Internal))
	}

	// Set initial bad_actor score in metadata
	if adminUser.Metadata == nil {
		adminUser.Metadata = &commonpb.Metadata{}
	}
	if adminUser.Metadata.ServiceSpecific == nil {
		adminUser.Metadata.ServiceSpecific = &structpb.Struct{Fields: map[string]*structpb.Value{}}
	}
	userSS, ok := adminUser.Metadata.ServiceSpecific.Fields["user"]
	var userMap map[string]interface{}
	if ok && userSS != nil && userSS.GetStructValue() != nil {
		userMap = userSS.GetStructValue().AsMap()
	} else {
		userMap = map[string]interface{}{}
	}
	badActor := map[string]interface{}{"score": 0.0}
	userMap["bad_actor"] = badActor
	userStruct, err := structpb.NewStruct(userMap)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to build user metadata struct", codes.Internal))
	}
	adminUser.Metadata.ServiceSpecific.Fields["user"] = structpb.NewStructValue(userStruct)

	resp := &adminpb.CreateUserResponse{
		User: adminUser,
	}
	// Orchestration event emission (success)
	success := graceful.WrapSuccess(ctx, codes.OK, "admin user created", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Metadata:     adminUser.Metadata,
		EventType:    "admin.user_created",
		EventID:      adminUser.Id,
		PatternType:  "admin_user",
		PatternID:    adminUser.Id,
		PatternMeta:  adminUser.Metadata,
		EventEmitter: &EventEmitterAdapter{Emitter: s.eventEmitter},
		EventEnabled: s.eventEnabled,
	})
	return resp, nil
}

func (s *Service) UpdateUser(ctx context.Context, req *adminpb.UpdateUserRequest) (*adminpb.UpdateUserResponse, error) {
	if req.User.Metadata == nil {
		req.User.Metadata = &commonpb.Metadata{}
	}
	// Set versioning using canonical helper
	versioning := map[string]interface{}{
		"system_version":   "1.0.0",
		"service_version":  "1.0.0",
		"environment":      "prod",
		"last_migrated_at": time.Now().Format(time.RFC3339),
	}
	if err := metadata.SetServiceSpecificField(req.User.Metadata, "admin", "versioning", versioning); err != nil {
		errObj := graceful.MapAndWrapErr(ctx, err, "failed to set admin versioning", codes.Internal)
		errObj.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{
			Metadata:     req.User.Metadata,
			EventType:    "admin.user_metadata_error",
			EventID:      req.User.Email,
			PatternType:  "admin_user",
			PatternID:    req.User.Email,
			EventEmitter: &EventEmitterAdapter{Emitter: s.eventEmitter},
			EventEnabled: s.eventEnabled,
		})
		return nil, graceful.ToStatusError(errObj)
	}

	user, err := s.repo.UpdateUser(ctx, req.User)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to update admin user", codes.Internal))
	}
	resp := &adminpb.UpdateUserResponse{User: user}
	success := graceful.WrapSuccess(ctx, codes.OK, "admin user updated", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Metadata:     user.Metadata,
		EventType:    "admin.user_updated",
		EventID:      user.Id,
		PatternType:  "admin_user",
		PatternID:    user.Id,
		PatternMeta:  user.Metadata,
		EventEmitter: &EventEmitterAdapter{Emitter: s.eventEmitter},
		EventEnabled: s.eventEnabled,
	})
	return resp, nil
}

func (s *Service) DeleteUser(ctx context.Context, req *adminpb.DeleteUserRequest) (*adminpb.DeleteUserResponse, error) {
	err := s.repo.DeleteUser(ctx, req.UserId)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to delete admin user", codes.Internal))
	}
	resp := &adminpb.DeleteUserResponse{Success: true}
	success := graceful.WrapSuccess(ctx, codes.OK, "admin user deleted", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		EventType:    "admin.user_deleted",
		EventID:      req.UserId,
		PatternType:  "admin_user",
		PatternID:    req.UserId,
		EventEmitter: &EventEmitterAdapter{Emitter: s.eventEmitter},
		EventEnabled: s.eventEnabled,
	})
	return resp, nil
}

func (s *Service) ListUsers(ctx context.Context, req *adminpb.ListUsersRequest) (*adminpb.ListUsersResponse, error) {
	users, total, err := s.repo.ListUsers(ctx, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to list admin users", codes.Internal))
	}
	return &adminpb.ListUsersResponse{
		Users:      users,
		TotalCount: utils.ToInt32(total),
		Page:       req.Page,
		TotalPages: utils.ToInt32((total + int(req.PageSize) - 1) / int(req.PageSize)),
	}, nil
}

func (s *Service) GetUser(ctx context.Context, req *adminpb.GetUserRequest) (*adminpb.GetUserResponse, error) {
	user, err := s.repo.GetUser(ctx, req.UserId)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to get admin user", codes.Internal))
	}
	return &adminpb.GetUserResponse{User: user}, nil
}

// Role management.
func (s *Service) CreateRole(ctx context.Context, req *adminpb.CreateRoleRequest) (*adminpb.CreateRoleResponse, error) {
	if req.Role.Metadata == nil {
		req.Role.Metadata = &commonpb.Metadata{}
	}
	// Set versioning using canonical helper
	versioning := map[string]interface{}{
		"system_version":   "1.0.0",
		"service_version":  "1.0.0",
		"environment":      "prod",
		"last_migrated_at": time.Now().Format(time.RFC3339),
	}
	if err := metadata.SetServiceSpecificField(req.Role.Metadata, "admin", "versioning", versioning); err != nil {
		errObj := graceful.MapAndWrapErr(ctx, err, "failed to set admin versioning", codes.Internal)
		errObj.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{
			Metadata:     req.Role.Metadata,
			EventType:    "admin.role_metadata_error",
			EventID:      req.Role.Id,
			PatternType:  "admin_role",
			PatternID:    req.Role.Id,
			EventEmitter: &EventEmitterAdapter{Emitter: s.eventEmitter},
			EventEnabled: s.eventEnabled,
		})
		return nil, graceful.ToStatusError(errObj)
	}

	role, err := s.repo.CreateRole(ctx, req.Role)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to create admin role", codes.Internal))
	}
	resp := &adminpb.CreateRoleResponse{Role: role}
	success := graceful.WrapSuccess(ctx, codes.OK, "admin role created", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Metadata:     role.Metadata,
		EventType:    "admin.role_created",
		EventID:      role.Id,
		PatternType:  "admin_role",
		PatternID:    role.Id,
		PatternMeta:  role.Metadata,
		EventEmitter: &EventEmitterAdapter{Emitter: s.eventEmitter},
		EventEnabled: s.eventEnabled,
	})
	return resp, nil
}

func (s *Service) UpdateRole(ctx context.Context, req *adminpb.UpdateRoleRequest) (*adminpb.UpdateRoleResponse, error) {
	if req.Role.Metadata == nil {
		req.Role.Metadata = &commonpb.Metadata{}
	}
	// Set versioning using canonical helper
	versioning := map[string]interface{}{
		"system_version":   "1.0.0",
		"service_version":  "1.0.0",
		"environment":      "prod",
		"last_migrated_at": time.Now().Format(time.RFC3339),
	}
	if err := metadata.SetServiceSpecificField(req.Role.Metadata, "admin", "versioning", versioning); err != nil {
		errObj := graceful.MapAndWrapErr(ctx, err, "failed to set admin versioning", codes.Internal)
		errObj.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{
			Metadata:     req.Role.Metadata,
			EventType:    "admin.role_metadata_error",
			EventID:      req.Role.Id,
			PatternType:  "admin_role",
			PatternID:    req.Role.Id,
			EventEmitter: &EventEmitterAdapter{Emitter: s.eventEmitter},
			EventEnabled: s.eventEnabled,
		})
		return nil, graceful.ToStatusError(errObj)
	}

	role, err := s.repo.UpdateRole(ctx, req.Role)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to update admin role", codes.Internal))
	}
	resp := &adminpb.UpdateRoleResponse{Role: role}
	success := graceful.WrapSuccess(ctx, codes.OK, "admin role updated", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Metadata:     role.Metadata,
		EventType:    "admin.role_updated",
		EventID:      role.Id,
		PatternType:  "admin_role",
		PatternID:    role.Id,
		PatternMeta:  role.Metadata,
		EventEmitter: &EventEmitterAdapter{Emitter: s.eventEmitter},
		EventEnabled: s.eventEnabled,
	})
	return resp, nil
}

func (s *Service) DeleteRole(ctx context.Context, req *adminpb.DeleteRoleRequest) (*adminpb.DeleteRoleResponse, error) {
	err := s.repo.DeleteRole(ctx, req.RoleId)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to delete admin role", codes.Internal))
	}
	resp := &adminpb.DeleteRoleResponse{Success: true}
	success := graceful.WrapSuccess(ctx, codes.OK, "admin role deleted", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		EventType:    "admin.role_deleted",
		EventID:      req.RoleId,
		PatternType:  "admin_role",
		PatternID:    req.RoleId,
		EventEmitter: &EventEmitterAdapter{Emitter: s.eventEmitter},
		EventEnabled: s.eventEnabled,
	})
	return resp, nil
}

func (s *Service) ListRoles(ctx context.Context, req *adminpb.ListRolesRequest) (*adminpb.ListRolesResponse, error) {
	roles, total, err := s.repo.ListRoles(ctx, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to list admin roles", codes.Internal))
	}
	return &adminpb.ListRolesResponse{
		Roles:      roles,
		TotalCount: utils.ToInt32(total),
		Page:       req.Page,
		TotalPages: utils.ToInt32((total + int(req.PageSize) - 1) / int(req.PageSize)),
	}, nil
}

// Role assignment.
func (s *Service) AssignRole(ctx context.Context, req *adminpb.AssignRoleRequest) (*adminpb.AssignRoleResponse, error) {
	err := s.repo.AssignRole(ctx, req.UserId, req.RoleId)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to assign role", codes.Internal))
	}
	resp := &adminpb.AssignRoleResponse{Success: true}
	success := graceful.WrapSuccess(ctx, codes.OK, "admin role assigned", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		EventType:    "admin.role_assigned",
		EventID:      req.UserId + ":" + req.RoleId,
		PatternType:  "admin_role_assignment",
		PatternID:    req.UserId + ":" + req.RoleId,
		EventEmitter: &EventEmitterAdapter{Emitter: s.eventEmitter},
		EventEnabled: s.eventEnabled,
	})
	return resp, nil
}

func (s *Service) RevokeRole(ctx context.Context, req *adminpb.RevokeRoleRequest) (*adminpb.RevokeRoleResponse, error) {
	err := s.repo.RevokeRole(ctx, req.UserId, req.RoleId)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to revoke role", codes.Internal))
	}
	resp := &adminpb.RevokeRoleResponse{Success: true}
	success := graceful.WrapSuccess(ctx, codes.OK, "admin role revoked", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		EventType:    "admin.role_revoked",
		EventID:      req.UserId + ":" + req.RoleId,
		PatternType:  "admin_role_assignment",
		PatternID:    req.UserId + ":" + req.RoleId,
		EventEmitter: &EventEmitterAdapter{Emitter: s.eventEmitter},
		EventEnabled: s.eventEnabled,
	})
	return resp, nil
}

// Audit logs.
func (s *Service) GetAuditLogs(ctx context.Context, req *adminpb.GetAuditLogsRequest) (*adminpb.GetAuditLogsResponse, error) {
	logs, total, err := s.repo.GetAuditLogs(ctx, int(req.Page), int(req.PageSize), req.UserId, req.Action)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to get audit logs", codes.Internal))
	}
	return &adminpb.GetAuditLogsResponse{
		Logs:       logs,
		TotalCount: utils.ToInt32(total),
		Page:       req.Page,
		TotalPages: utils.ToInt32((total + int(req.PageSize) - 1) / int(req.PageSize)),
	}, nil
}

// Settings.
func (s *Service) GetSettings(ctx context.Context, _ *adminpb.GetSettingsRequest) (*adminpb.GetSettingsResponse, error) {
	settings, err := s.repo.GetSettings(ctx)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to get settings", codes.Internal))
	}
	return &adminpb.GetSettingsResponse{Settings: settings}, nil
}

func (s *Service) UpdateSettings(ctx context.Context, req *adminpb.UpdateSettingsRequest) (*adminpb.UpdateSettingsResponse, error) {
	if req.Settings.Metadata == nil {
		req.Settings.Metadata = &commonpb.Metadata{}
	}
	// Set versioning using canonical helper
	versioning := map[string]interface{}{
		"system_version":   "1.0.0",
		"service_version":  "1.0.0",
		"environment":      "prod",
		"last_migrated_at": time.Now().Format(time.RFC3339),
	}
	if err := metadata.SetServiceSpecificField(req.Settings.Metadata, "admin", "versioning", versioning); err != nil {
		errObj := graceful.MapAndWrapErr(ctx, err, "failed to set admin versioning", codes.Internal)
		errObj.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{
			Metadata:     req.Settings.Metadata,
			EventType:    "admin.settings_metadata_error",
			EventID:      "settings",
			PatternType:  "admin_settings",
			PatternID:    "settings",
			EventEmitter: &EventEmitterAdapter{Emitter: s.eventEmitter},
			EventEnabled: s.eventEnabled,
		})
		return nil, graceful.ToStatusError(errObj)
	}

	settings, err := s.repo.UpdateSettings(ctx, req.Settings)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to update settings", codes.Internal))
	}
	return &adminpb.UpdateSettingsResponse{Settings: settings}, nil
}
