package admin

import (
	"context"
	"time"

	adminpb "github.com/nmxmxh/master-ovasabi/api/protos/admin/v1"
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	userpb "github.com/nmxmxh/master-ovasabi/api/protos/user/v1"
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
				return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to create main user", codes.Internal))
			}
			mainUser = createResp.User
		} else {
			return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to lookup main user", codes.Internal))
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
		s.log.Warn("Failed to build user metadata struct", zap.Error(err))
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to build user metadata struct", codes.Internal))
	}
	adminUser.Metadata.ServiceSpecific.Fields["user"] = structpb.NewStructValue(userStruct)

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
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to update admin user", codes.Internal))
	}
	return &adminpb.UpdateUserResponse{User: user}, nil
}

func (s *Service) DeleteUser(ctx context.Context, req *adminpb.DeleteUserRequest) (*adminpb.DeleteUserResponse, error) {
	err := s.repo.DeleteUser(ctx, req.UserId)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to delete admin user", codes.Internal))
	}
	return &adminpb.DeleteUserResponse{Success: true}, nil
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
	SetAdminVersioning(req.Role.Metadata, map[string]interface{}{
		"system_version":   "1.0.0",
		"service_version":  "1.0.0",
		"environment":      "prod",
		"last_migrated_at": time.Now().Format(time.RFC3339),
	})
	role, err := s.repo.CreateRole(ctx, req.Role)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to create admin role", codes.Internal))
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
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to update admin role", codes.Internal))
	}
	return &adminpb.UpdateRoleResponse{Role: role}, nil
}

func (s *Service) DeleteRole(ctx context.Context, req *adminpb.DeleteRoleRequest) (*adminpb.DeleteRoleResponse, error) {
	err := s.repo.DeleteRole(ctx, req.RoleId)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to delete admin role", codes.Internal))
	}
	return &adminpb.DeleteRoleResponse{Success: true}, nil
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
	return &adminpb.AssignRoleResponse{Success: true}, nil
}

func (s *Service) RevokeRole(ctx context.Context, req *adminpb.RevokeRoleRequest) (*adminpb.RevokeRoleResponse, error) {
	err := s.repo.RevokeRole(ctx, req.UserId, req.RoleId)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to revoke role", codes.Internal))
	}
	return &adminpb.RevokeRoleResponse{Success: true}, nil
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
	SetAdminVersioning(req.Settings.Metadata, map[string]interface{}{
		"system_version":   "1.0.0",
		"service_version":  "1.0.0",
		"environment":      "prod",
		"last_migrated_at": time.Now().Format(time.RFC3339),
	})
	settings, err := s.repo.UpdateSettings(ctx, req.Settings)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to update settings", codes.Internal))
	}
	return &adminpb.UpdateSettingsResponse{Settings: settings}, nil
}
