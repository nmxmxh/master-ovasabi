package admin

import (
	"context"
	"time"

	adminpb "github.com/nmxmxh/master-ovasabi/api/protos/admin/v1"
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	userpb "github.com/nmxmxh/master-ovasabi/api/protos/user/v1"
	"github.com/nmxmxh/master-ovasabi/internal/service/pattern"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	events "github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"google.golang.org/protobuf/types/known/structpb"
)

// EventEmitter defines the interface for emitting events.
type EventEmitter interface {
	EmitEvent(ctx context.Context, eventType, entityID string, metadata *commonpb.Metadata) error
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
				// Emit failure event
				if s.eventEnabled && s.eventEmitter != nil {
					errStruct := metadata.NewStructFromMap(map[string]interface{}{
						"error": err.Error(),
						"email": email,
					})
					errMeta := &commonpb.Metadata{}
					errMeta.ServiceSpecific = errStruct
					errEmit := s.eventEmitter.EmitEvent(ctx, "admin.user_create_failed", "", errMeta)
					if errEmit != nil {
						s.log.Warn("Failed to emit admin.user_create_failed event", zap.Error(errEmit))
					}
				}
				return nil, status.Errorf(codes.Internal, "failed to create main user: %v", err)
			}
			mainUser = createResp.User
		} else {
			// Emit failure event
			if s.eventEnabled && s.eventEmitter != nil {
				errStruct := metadata.NewStructFromMap(map[string]interface{}{
					"error": err.Error(),
					"email": email,
				})
				errMeta := &commonpb.Metadata{}
				errMeta.ServiceSpecific = errStruct
				errEmit := s.eventEmitter.EmitEvent(ctx, "admin.user_create_failed", "", errMeta)
				if errEmit != nil {
					s.log.Warn("Failed to emit admin.user_create_failed event", zap.Error(errEmit))
				}
			}
			return nil, status.Errorf(codes.Internal, "failed to lookup main user: %v", err)
		}
	} else {
		mainUser = userResp.User
	}
	// Enrich metadata
	if req.User.Metadata == nil {
		req.User.Metadata = &commonpb.Metadata{}
	}
	SetAdminVersioning(req.User.Metadata, map[string]interface{}{
		"system_version":   "1.0.0",
		"service_version":  "1.0.0",
		"environment":      "prod",
		"last_migrated_at": time.Now().Format(time.RFC3339),
	})
	adminUser, err := s.repo.CreateUser(ctx, &adminpb.User{
		Id:       mainUser.Id,
		MasterId: mainUser.MasterId, // propagate master_id
		Email:    email,
		Name:     req.User.Name,
		Metadata: req.User.Metadata,
	})
	if err != nil {
		// Emit failure event
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct := metadata.NewStructFromMap(map[string]interface{}{
				"error": err.Error(),
				"email": email,
			})
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "admin.user_create_failed", "", errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit admin.user_create_failed event", zap.Error(errEmit))
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to create admin user: %v", err)
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
		s.log.Warn("Failed to build user metadata struct", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to build user metadata struct: %v", err)
	}
	adminUser.Metadata.ServiceSpecific.Fields["user"] = structpb.NewStructValue(userStruct)

	// Emit admin.user_created event after successful creation using EmitCallbackEvent
	if s.eventEnabled && s.eventEmitter != nil {
		_, ok := events.EmitCallbackEvent(ctx, s.eventEmitter, s.log, s.Cache, "admin_user", "admin.user_created", adminUser.Id, adminUser.Metadata, zap.String("user_id", adminUser.Id))
		if !ok {
			s.log.Warn("Failed to emit callback event")
		}
		// Canonical metadata enrichment helpers
		if s.Cache != nil && adminUser.Metadata != nil {
			if err := pattern.CacheMetadata(ctx, s.log, s.Cache, "admin_user", adminUser.Id, adminUser.Metadata, 10*time.Minute); err != nil {
				s.log.Error("failed to cache admin user metadata", zap.Error(err))
			}
		}
		if err := pattern.RegisterSchedule(ctx, s.log, "admin_user", adminUser.Id, adminUser.Metadata); err != nil {
			s.log.Error("failed to register admin user schedule", zap.Error(err))
		}
		if err := pattern.EnrichKnowledgeGraph(ctx, s.log, "admin_user", adminUser.Id, adminUser.Metadata); err != nil {
			s.log.Error("failed to enrich admin user knowledge graph", zap.Error(err))
		}
		if err := pattern.RegisterWithNexus(ctx, s.log, "admin_user", adminUser.Metadata); err != nil {
			s.log.Error("failed to register admin user with nexus", zap.Error(err))
		}
	}

	return &adminpb.CreateUserResponse{
		User: adminUser,
	}, nil
}

func (s *Service) UpdateUser(ctx context.Context, req *adminpb.UpdateUserRequest) (*adminpb.UpdateUserResponse, error) {
	if req.User.Metadata == nil {
		req.User.Metadata = &commonpb.Metadata{}
	}
	SetAdminVersioning(req.User.Metadata, map[string]interface{}{
		"system_version":   "1.0.0",
		"service_version":  "1.0.0",
		"environment":      "prod",
		"last_migrated_at": time.Now().Format(time.RFC3339),
	})
	user, err := s.repo.UpdateUser(ctx, req.User)
	if err != nil {
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct := metadata.NewStructFromMap(map[string]interface{}{"error": err.Error(), "user_id": req.User.Id})
			_, ok := events.EmitCallbackEvent(ctx, s.eventEmitter, s.log, s.Cache, "admin_user", "admin.user_update_failed", req.User.Id, &commonpb.Metadata{ServiceSpecific: errStruct}, zap.Error(err))
			if !ok {
				s.log.Warn("Failed to emit callback event")
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to update admin user: %v", err)
	}
	if s.eventEnabled && s.eventEmitter != nil {
		_, ok := events.EmitCallbackEvent(ctx, s.eventEmitter, s.log, s.Cache, "admin_user", "admin.user_updated", user.Id, user.Metadata, zap.String("user_id", user.Id))
		if !ok {
			s.log.Warn("Failed to emit callback event")
		}
		// Canonical metadata enrichment helpers
		if s.Cache != nil && user.Metadata != nil {
			if err := pattern.CacheMetadata(ctx, s.log, s.Cache, "admin_user", user.Id, user.Metadata, 10*time.Minute); err != nil {
				s.log.Error("failed to cache admin user metadata", zap.Error(err))
			}
		}
		if err := pattern.RegisterSchedule(ctx, s.log, "admin_user", user.Id, user.Metadata); err != nil {
			s.log.Error("failed to register admin user schedule", zap.Error(err))
		}
		if err := pattern.EnrichKnowledgeGraph(ctx, s.log, "admin_user", user.Id, user.Metadata); err != nil {
			s.log.Error("failed to enrich admin user knowledge graph", zap.Error(err))
		}
		if err := pattern.RegisterWithNexus(ctx, s.log, "admin_user", user.Metadata); err != nil {
			s.log.Error("failed to register admin user with nexus", zap.Error(err))
		}
	}
	return &adminpb.UpdateUserResponse{User: user}, nil
}

func (s *Service) DeleteUser(ctx context.Context, req *adminpb.DeleteUserRequest) (*adminpb.DeleteUserResponse, error) {
	err := s.repo.DeleteUser(ctx, req.UserId)
	if err != nil {
		// Emit failure event
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct := metadata.NewStructFromMap(map[string]interface{}{
				"error":   err.Error(),
				"user_id": req.UserId,
			})
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "admin.user_update_failed", req.UserId, errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit admin.user_update_failed event", zap.Error(errEmit))
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to delete admin user: %v", err)
	}
	// Emit admin.user_deleted event after successful deletion
	_, _ = events.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "admin.user_deleted", req.UserId, nil)
	return &adminpb.DeleteUserResponse{Success: true}, nil
}

func (s *Service) ListUsers(ctx context.Context, req *adminpb.ListUsersRequest) (*adminpb.ListUsersResponse, error) {
	users, total, err := s.repo.ListUsers(ctx, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list admin users: %v", err)
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
		return nil, status.Errorf(codes.Internal, "failed to get admin user: %v", err)
	}
	return &adminpb.GetUserResponse{User: user}, nil
}

// Role management.
func (s *Service) CreateRole(ctx context.Context, req *adminpb.CreateRoleRequest) (*adminpb.CreateRoleResponse, error) {
	if req.Role.Metadata == nil {
		req.Role.Metadata = &commonpb.Metadata{}
	}
	SetAdminVersioning(req.Role.Metadata, map[string]interface{}{
		"system_version":   "1.0.0",
		"service_version":  "1.0.0",
		"environment":      "prod",
		"last_migrated_at": time.Now().Format(time.RFC3339),
	})
	role, err := s.repo.CreateRole(ctx, req.Role)
	if err != nil {
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct := metadata.NewStructFromMap(map[string]interface{}{
				"error":   err.Error(),
				"role_id": req.Role.Id,
			})
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "admin.role_create_failed", req.Role.Id, errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit admin.role_create_failed event", zap.Error(errEmit))
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to create admin role: %v", err)
	}
	if s.eventEnabled && s.eventEmitter != nil {
		errEvt := s.eventEmitter.EmitEvent(ctx, "admin.role_created", role.Id, role.Metadata)
		if errEvt != nil {
			s.log.Warn("Failed to emit admin.role_created event", zap.Error(errEvt))
		}
	}
	return &adminpb.CreateRoleResponse{Role: role}, nil
}

func (s *Service) UpdateRole(ctx context.Context, req *adminpb.UpdateRoleRequest) (*adminpb.UpdateRoleResponse, error) {
	if req.Role.Metadata == nil {
		req.Role.Metadata = &commonpb.Metadata{}
	}
	SetAdminVersioning(req.Role.Metadata, map[string]interface{}{
		"system_version":   "1.0.0",
		"service_version":  "1.0.0",
		"environment":      "prod",
		"last_migrated_at": time.Now().Format(time.RFC3339),
	})
	role, err := s.repo.UpdateRole(ctx, req.Role)
	if err != nil {
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct := metadata.NewStructFromMap(map[string]interface{}{
				"error":   err.Error(),
				"role_id": req.Role.Id,
			})
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEvt := s.eventEmitter.EmitEvent(ctx, "admin.role_update_failed", req.Role.Id, errMeta)
			if errEvt != nil {
				s.log.Warn("Failed to emit admin.role_update_failed event", zap.Error(errEvt))
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to update admin role: %v", err)
	}
	if s.eventEnabled && s.eventEmitter != nil {
		errEvt := s.eventEmitter.EmitEvent(ctx, "admin.role_updated", role.Id, role.Metadata)
		if errEvt != nil {
			s.log.Warn("Failed to emit admin.role_updated event", zap.Error(errEvt))
		}
	}
	return &adminpb.UpdateRoleResponse{Role: role}, nil
}

func (s *Service) DeleteRole(ctx context.Context, req *adminpb.DeleteRoleRequest) (*adminpb.DeleteRoleResponse, error) {
	err := s.repo.DeleteRole(ctx, req.RoleId)
	if err != nil {
		errStruct := metadata.NewStructFromMap(map[string]interface{}{
			"error":   err.Error(),
			"role_id": req.RoleId,
		})
		errMeta := &commonpb.Metadata{}
		errMeta.ServiceSpecific = errStruct
		errEvt := s.eventEmitter.EmitEvent(ctx, "admin.role_delete_failed", req.RoleId, errMeta)
		if errEvt != nil {
			s.log.Warn("Failed to emit admin.role_delete_failed event", zap.Error(errEvt))
		}
		return nil, status.Errorf(codes.Internal, "failed to delete admin role: %v", err)
	}
	if s.eventEnabled && s.eventEmitter != nil {
		errEvt := s.eventEmitter.EmitEvent(ctx, "admin.role_deleted", req.RoleId, nil)
		if errEvt != nil {
			s.log.Warn("Failed to emit admin.role_deleted event", zap.Error(errEvt))
		}
	}
	return &adminpb.DeleteRoleResponse{Success: true}, nil
}

func (s *Service) ListRoles(ctx context.Context, req *adminpb.ListRolesRequest) (*adminpb.ListRolesResponse, error) {
	roles, total, err := s.repo.ListRoles(ctx, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list admin roles: %v", err)
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
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct := metadata.NewStructFromMap(map[string]interface{}{"error": err.Error(), "user_id": req.UserId, "role_id": req.RoleId})
			_, ok := events.EmitCallbackEvent(ctx, s.eventEmitter, s.log, s.Cache, "admin_user", "admin.role_assign_failed", req.UserId, &commonpb.Metadata{ServiceSpecific: errStruct}, zap.String("user_id", req.UserId), zap.String("role_id", req.RoleId))
			if !ok {
				s.log.Warn("Failed to emit callback event")
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to assign role: %v", err)
	}
	if s.eventEnabled && s.eventEmitter != nil {
		successStruct := metadata.NewStructFromMap(map[string]interface{}{"user_id": req.UserId, "role_id": req.RoleId})
		successMeta := &commonpb.Metadata{ServiceSpecific: successStruct}
		_, ok := events.EmitCallbackEvent(ctx, s.eventEmitter, s.log, s.Cache, "admin_user", "admin.role_assigned", req.UserId, successMeta, zap.String("user_id", req.UserId), zap.String("role_id", req.RoleId))
		if !ok {
			s.log.Warn("Failed to emit callback event")
		}
	}
	return &adminpb.AssignRoleResponse{Success: true}, nil
}

func (s *Service) RevokeRole(ctx context.Context, req *adminpb.RevokeRoleRequest) (*adminpb.RevokeRoleResponse, error) {
	err := s.repo.RevokeRole(ctx, req.UserId, req.RoleId)
	if err != nil {
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct := metadata.NewStructFromMap(map[string]interface{}{"error": err.Error(), "user_id": req.UserId, "role_id": req.RoleId})
			_, ok := events.EmitCallbackEvent(ctx, s.eventEmitter, s.log, s.Cache, "admin_user", "admin.role_revoke_failed", req.UserId, &commonpb.Metadata{ServiceSpecific: errStruct}, zap.String("user_id", req.UserId), zap.String("role_id", req.RoleId))
			if !ok {
				s.log.Warn("Failed to emit callback event")
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to revoke role: %v", err)
	}
	if s.eventEnabled && s.eventEmitter != nil {
		successStruct := metadata.NewStructFromMap(map[string]interface{}{"user_id": req.UserId, "role_id": req.RoleId})
		successMeta := &commonpb.Metadata{ServiceSpecific: successStruct}
		_, ok := events.EmitCallbackEvent(ctx, s.eventEmitter, s.log, s.Cache, "admin_user", "admin.role_revoked", req.UserId, successMeta, zap.String("user_id", req.UserId), zap.String("role_id", req.RoleId))
		if !ok {
			s.log.Warn("Failed to emit callback event")
		}
	}
	return &adminpb.RevokeRoleResponse{Success: true}, nil
}

// Audit logs.
func (s *Service) GetAuditLogs(ctx context.Context, req *adminpb.GetAuditLogsRequest) (*adminpb.GetAuditLogsResponse, error) {
	logs, total, err := s.repo.GetAuditLogs(ctx, int(req.Page), int(req.PageSize), req.UserId, req.Action)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get audit logs: %v", err)
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
		return nil, status.Errorf(codes.Internal, "failed to get settings: %v", err)
	}
	return &adminpb.GetSettingsResponse{Settings: settings}, nil
}

func (s *Service) UpdateSettings(ctx context.Context, req *adminpb.UpdateSettingsRequest) (*adminpb.UpdateSettingsResponse, error) {
	if req.Settings.Metadata == nil {
		req.Settings.Metadata = &commonpb.Metadata{}
	}
	SetAdminVersioning(req.Settings.Metadata, map[string]interface{}{
		"system_version":   "1.0.0",
		"service_version":  "1.0.0",
		"environment":      "prod",
		"last_migrated_at": time.Now().Format(time.RFC3339),
	})
	settings, err := s.repo.UpdateSettings(ctx, req.Settings)
	if err != nil {
		errStruct := metadata.NewStructFromMap(map[string]interface{}{
			"error": err.Error(),
		})
		errMeta := &commonpb.Metadata{}
		errMeta.ServiceSpecific = errStruct
		errEvt := s.eventEmitter.EmitEvent(ctx, "admin.settings_update_failed", "", errMeta)
		if errEvt != nil {
			s.log.Warn("Failed to emit admin.settings_update_failed event", zap.Error(errEvt))
		}
		return nil, status.Errorf(codes.Internal, "failed to update settings: %v", err)
	}
	if s.eventEnabled && s.eventEmitter != nil {
		errEvt := s.eventEmitter.EmitEvent(ctx, "admin.settings_updated", "", settings.Metadata)
		if errEvt != nil {
			s.log.Warn("Failed to emit admin.settings_updated event", zap.Error(errEvt))
		}
	}
	return &adminpb.UpdateSettingsResponse{Settings: settings}, nil
}
