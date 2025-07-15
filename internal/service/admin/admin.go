package admin

import (
	"context"
	"time"

	adminpb "github.com/nmxmxh/master-ovasabi/api/protos/admin/v1"
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	userpb "github.com/nmxmxh/master-ovasabi/api/protos/user/v1"
	"github.com/nmxmxh/master-ovasabi/internal/repository"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

func handleUserAction(ctx context.Context, svc *Service, event *nexusv1.EventResponse) {
	action, state := parseActionAndState(event.GetEventType())
	switch action {
	case "user":
		switch state {
		case "create":
			var req adminpb.CreateUserRequest
			if event.Payload != nil && event.Payload.Data != nil {
				b, err := protojson.Marshal(event.Payload.Data)
				if err == nil {
					err = protojson.Unmarshal(b, &req)
				}
				if err != nil {
					svc.log.Error("Failed to unmarshal user create event payload", zap.Error(err))
					return
				}
			}
			svc.CreateUser(ctx, &req)
		case "update":
			var req adminpb.UpdateUserRequest
			if event.Payload != nil && event.Payload.Data != nil {
				b, err := protojson.Marshal(event.Payload.Data)
				if err == nil {
					err = protojson.Unmarshal(b, &req)
				}
				if err != nil {
					svc.log.Error("Failed to unmarshal user update event payload", zap.Error(err))
					return
				}
			}
			svc.UpdateUser(ctx, &req)
		case "delete":
			var req adminpb.DeleteUserRequest
			if event.Payload != nil && event.Payload.Data != nil {
				b, err := protojson.Marshal(event.Payload.Data)
				if err == nil {
					err = protojson.Unmarshal(b, &req)
				}
				if err != nil {
					svc.log.Error("Failed to unmarshal user delete event payload", zap.Error(err))
					return
				}
			}
			svc.DeleteUser(ctx, &req)
		}
	}
}

func handleRoleAction(ctx context.Context, svc *Service, event *nexusv1.EventResponse) {
	action, state := parseActionAndState(event.GetEventType())
	switch action {
	case "role":
		switch state {
		case "create":
			var req adminpb.CreateRoleRequest
			if event.Payload != nil && event.Payload.Data != nil {
				b, err := protojson.Marshal(event.Payload.Data)
				if err == nil {
					err = protojson.Unmarshal(b, &req)
				}
				if err != nil {
					svc.log.Error("Failed to unmarshal role create event payload", zap.Error(err))
					return
				}
			}
			svc.CreateRole(ctx, &req)
		case "update":
			var req adminpb.UpdateRoleRequest
			if event.Payload != nil && event.Payload.Data != nil {
				b, err := protojson.Marshal(event.Payload.Data)
				if err == nil {
					err = protojson.Unmarshal(b, &req)
				}
				if err != nil {
					svc.log.Error("Failed to unmarshal role update event payload", zap.Error(err))
					return
				}
			}
			svc.UpdateRole(ctx, &req)
		case "delete":
			var req adminpb.DeleteRoleRequest
			if event.Payload != nil && event.Payload.Data != nil {
				b, err := protojson.Marshal(event.Payload.Data)
				if err == nil {
					err = protojson.Unmarshal(b, &req)
				}
				if err != nil {
					svc.log.Error("Failed to unmarshal role delete event payload", zap.Error(err))
					return
				}
			}
			svc.DeleteRole(ctx, &req)
		}
	}
}

func handleSettingsAction(ctx context.Context, svc *Service, event *nexusv1.EventResponse) {
	action, state := parseActionAndState(event.GetEventType())
	switch action {
	case "settings":
		switch state {
		case "update":
			var req adminpb.UpdateSettingsRequest
			if event.Payload != nil && event.Payload.Data != nil {
				b, err := protojson.Marshal(event.Payload.Data)
				if err == nil {
					err = protojson.Unmarshal(b, &req)
				}
				if err != nil {
					svc.log.Error("Failed to unmarshal settings update event payload", zap.Error(err))
					return
				}
			}
			svc.UpdateSettings(ctx, &req)
		}
	}
}

type Service struct {
	adminpb.UnimplementedAdminServiceServer
	log          *zap.Logger
	repo         *Repository
	masterRepo   repository.MasterRepository
	userClient   userpb.UserServiceClient
	Cache        *redis.Cache
	eventEmitter events.EventEmitter
	eventEnabled bool
	handler      *graceful.Handler
}

func NewService(log *zap.Logger, repo *Repository, userClient userpb.UserServiceClient, cache *redis.Cache, eventEmitter events.EventEmitter, eventEnabled bool) adminpb.AdminServiceServer {
	return &Service{
		log:          log,
		repo:         repo,
		masterRepo:   repo.masterRepo, // Get masterRepo from the admin repository
		userClient:   userClient,
		Cache:        cache,
		eventEmitter: eventEmitter,
		eventEnabled: eventEnabled,
		handler:      graceful.NewHandler(log, eventEmitter, cache, "admin", "v1", eventEnabled),
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
				errObj := graceful.MapAndWrapErr(ctx, err, "failed to create main user", codes.Internal)
				s.handler.Error(ctx, "create_user", codes.Internal, "failed to create main user", err, req.User.Metadata, email)
				return nil, errObj
			}
			mainUser = createResp.User
		} else {
			errObj := graceful.MapAndWrapErr(ctx, err, "failed to lookup main user", codes.Internal)
			s.handler.Error(ctx, "lookup_user", codes.Internal, "failed to lookup main user", err, req.User.Metadata, email)
			return nil, errObj
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
		"system_version":         CurrentVersion,
		AdminFieldServiceVersion: CurrentVersion,
		"environment":            "prod",
		"last_migrated_at":       time.Now().Format(time.RFC3339),
	}
	if err := metadata.SetServiceSpecificField(req.User.Metadata, "admin", "versioning", versioning); err != nil {
		errObj := graceful.MapAndWrapErr(ctx, err, "failed to set admin versioning", codes.Internal)
		s.handler.Error(ctx, "create_user", codes.Internal, "failed to set admin versioning", err, req.User.Metadata, email)
		return nil, errObj
	}

	adminUser, err := s.repo.CreateUser(ctx, &adminpb.User{
		Id:         mainUser.Id,
		MasterId:   mainUser.MasterId,
		MasterUuid: mainUser.MasterUuid, // Propagate master_uuid
		Email:      email,
		UserId:     mainUser.Id, // Ensure UserId is propagated
		Name:       req.User.Name,
		Metadata:   req.User.Metadata,
	})
	if err != nil {
		s.handler.Error(ctx, "create_user", codes.Internal, "failed to create admin user", err, req.User.Metadata, email)
		return nil, graceful.MapAndWrapErr(ctx, err, "failed to create admin user", codes.Internal)
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
		s.handler.Error(ctx, "create_user", codes.Internal, "failed to build user metadata struct", err, adminUser.Metadata, email)
		return nil, graceful.MapAndWrapErr(ctx, err, "failed to build user metadata struct", codes.Internal)
	}
	adminUser.Metadata.ServiceSpecific.Fields["user"] = structpb.NewStructValue(userStruct)

	resp := &adminpb.CreateUserResponse{
		User: adminUser,
	}
	// Orchestration event emission (success)
	s.handler.Success(ctx, "create_user", codes.OK, "admin user created", resp, adminUser.Metadata, adminUser.Id, nil)
	return resp, nil
}

func (s *Service) UpdateUser(ctx context.Context, req *adminpb.UpdateUserRequest) (*adminpb.UpdateUserResponse, error) {
	if req.User.Metadata == nil {
		req.User.Metadata = &commonpb.Metadata{}
	}
	// Set versioning using canonical helper
	versioning := map[string]interface{}{
		"system_version":         CurrentVersion,
		AdminFieldServiceVersion: CurrentVersion,
		"environment":            "prod",
		"last_migrated_at":       time.Now().Format(time.RFC3339),
	}
	if err := metadata.SetServiceSpecificField(req.User.Metadata, AdminNamespace, "versioning", versioning); err != nil {
		errObj := graceful.MapAndWrapErr(ctx, err, "failed to set admin versioning", codes.Internal)
		s.handler.Error(ctx, "update_user", codes.Internal, "failed to set admin versioning", err, req.User.Metadata, req.User.Email)
		return nil, errObj
	}

	user, err := s.repo.UpdateUser(ctx, req.User)
	if err != nil {
		return nil, graceful.MapAndWrapErr(ctx, err, "failed to update admin user", codes.Internal)
	}
	resp := &adminpb.UpdateUserResponse{User: user}
	s.handler.Success(ctx, "update_user", codes.OK, "admin user updated", resp, user.Metadata, user.Id, nil)
	return resp, nil
}

func (s *Service) DeleteUser(ctx context.Context, req *adminpb.DeleteUserRequest) (*adminpb.DeleteUserResponse, error) {
	err := s.repo.DeleteUser(ctx, req.UserId)
	if err != nil {
		return nil, graceful.MapAndWrapErr(ctx, err, "failed to delete admin user", codes.Internal)
	}
	resp := &adminpb.DeleteUserResponse{Success: true}
	s.handler.Success(ctx, "delete_user", codes.OK, "admin user deleted", resp, nil, req.UserId, nil)
	return resp, nil
}

func (s *Service) ListUsers(ctx context.Context, req *adminpb.ListUsersRequest) (*adminpb.ListUsersResponse, error) {
	users, total, err := s.repo.ListUsers(ctx, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, graceful.MapAndWrapErr(ctx, err, "failed to list admin users", codes.Internal)
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
		return nil, graceful.MapAndWrapErr(ctx, err, "failed to get admin user", codes.Internal)
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
		"system_version":         CurrentVersion,
		AdminFieldServiceVersion: CurrentVersion,
		"environment":            "prod",
		"last_migrated_at":       time.Now().Format(time.RFC3339),
	}
	if err := metadata.SetServiceSpecificField(req.Role.Metadata, AdminNamespace, "versioning", versioning); err != nil {
		errObj := graceful.MapAndWrapErr(ctx, err, "failed to set admin versioning", codes.Internal)
		s.handler.Error(ctx, "create_role", codes.Internal, "failed to set admin versioning", err, req.Role.Metadata, req.Role.Id)
		return nil, errObj
	}

	role, err := s.repo.CreateRole(ctx, req.Role)
	if err != nil {
		return nil, graceful.MapAndWrapErr(ctx, err, "failed to create admin role", codes.Internal)
	}
	resp := &adminpb.CreateRoleResponse{Role: role}
	s.handler.Success(ctx, "create_role", codes.OK, "admin role created", resp, role.Metadata, role.Id, nil)
	return resp, nil
}

func (s *Service) UpdateRole(ctx context.Context, req *adminpb.UpdateRoleRequest) (*adminpb.UpdateRoleResponse, error) {
	if req.Role.Metadata == nil {
		req.Role.Metadata = &commonpb.Metadata{}
	}
	// Set versioning using canonical helper
	versioning := map[string]interface{}{
		"system_version":         CurrentVersion,
		AdminFieldServiceVersion: CurrentVersion,
		"environment":            "prod",
		"last_migrated_at":       time.Now().Format(time.RFC3339),
	}
	if err := metadata.SetServiceSpecificField(req.Role.Metadata, AdminNamespace, "versioning", versioning); err != nil {
		errObj := graceful.MapAndWrapErr(ctx, err, "failed to set admin versioning", codes.Internal)
		s.handler.Error(ctx, "update_role", codes.Internal, "failed to set admin versioning", err, req.Role.Metadata, req.Role.Id)
		return nil, errObj
	}

	role, err := s.repo.UpdateRole(ctx, req.Role)
	if err != nil {
		return nil, graceful.MapAndWrapErr(ctx, err, "failed to update admin role", codes.Internal)
	}
	resp := &adminpb.UpdateRoleResponse{Role: role}
	s.handler.Success(ctx, "update_role", codes.OK, "admin role updated", resp, role.Metadata, role.Id, nil)
	return resp, nil
}

func (s *Service) DeleteRole(ctx context.Context, req *adminpb.DeleteRoleRequest) (*adminpb.DeleteRoleResponse, error) {
	err := s.repo.DeleteRole(ctx, req.RoleId)
	if err != nil {
		return nil, graceful.MapAndWrapErr(ctx, err, "failed to delete admin role", codes.Internal)
	}
	resp := &adminpb.DeleteRoleResponse{Success: true}
	s.handler.Success(ctx, "delete_role", codes.OK, "admin role deleted", resp, nil, req.RoleId, nil)
	return resp, nil
}

func (s *Service) ListRoles(ctx context.Context, req *adminpb.ListRolesRequest) (*adminpb.ListRolesResponse, error) {
	roles, total, err := s.repo.ListRoles(ctx, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, graceful.MapAndWrapErr(ctx, err, "failed to list admin roles", codes.Internal)
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
		return nil, graceful.MapAndWrapErr(ctx, err, "failed to assign role", codes.Internal)
	}
	resp := &adminpb.AssignRoleResponse{Success: true}
	s.handler.Success(ctx, "assign_role", codes.OK, "admin role assigned", resp, nil, req.UserId+":"+req.RoleId, nil)
	return resp, nil
}

func (s *Service) RevokeRole(ctx context.Context, req *adminpb.RevokeRoleRequest) (*adminpb.RevokeRoleResponse, error) {
	err := s.repo.RevokeRole(ctx, req.UserId, req.RoleId)
	if err != nil {
		return nil, graceful.MapAndWrapErr(ctx, err, "failed to revoke role", codes.Internal)
	}
	resp := &adminpb.RevokeRoleResponse{Success: true}
	s.handler.Success(ctx, "revoke_role", codes.OK, "admin role revoked", resp, nil, req.UserId+":"+req.RoleId, nil)
	return resp, nil
}

// Audit logs.
func (s *Service) GetAuditLogs(ctx context.Context, req *adminpb.GetAuditLogsRequest) (*adminpb.GetAuditLogsResponse, error) {
	logs, total, err := s.repo.GetAuditLogs(ctx, int(req.Page), int(req.PageSize), req.UserId, req.Action)
	if err != nil {
		return nil, graceful.MapAndWrapErr(ctx, err, "failed to get audit logs", codes.Internal)
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
		return nil, graceful.MapAndWrapErr(ctx, err, "failed to get settings", codes.Internal)
	}
	return &adminpb.GetSettingsResponse{Settings: settings}, nil
}

func (s *Service) UpdateSettings(ctx context.Context, req *adminpb.UpdateSettingsRequest) (*adminpb.UpdateSettingsResponse, error) {
	if req.Settings.Metadata == nil {
		req.Settings.Metadata = &commonpb.Metadata{}
	}
	// Set versioning using canonical helper
	versioning := map[string]interface{}{
		"system_version":         CurrentVersion,
		AdminFieldServiceVersion: CurrentVersion,
		"environment":            "prod",
		"last_migrated_at":       time.Now().Format(time.RFC3339),
	}
	if err := metadata.SetServiceSpecificField(req.Settings.Metadata, AdminNamespace, "versioning", versioning); err != nil {
		errObj := graceful.MapAndWrapErr(ctx, err, "failed to set admin versioning", codes.Internal)
		s.handler.Error(ctx, "update_settings", codes.Internal, "failed to set admin versioning", err, req.Settings.Metadata, "settings")
		return nil, errObj
	}

	settings, err := s.repo.UpdateSettings(ctx, req.Settings)
	if err != nil {
		return nil, graceful.MapAndWrapErr(ctx, err, "failed to update settings", codes.Internal)
	}
	resp := &adminpb.UpdateSettingsResponse{Settings: settings}
	s.handler.Success(ctx, "update_settings", codes.OK, "admin settings updated", resp, settings.Metadata, "settings", nil)
	return resp, nil
}
