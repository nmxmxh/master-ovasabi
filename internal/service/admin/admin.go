package adminservice

import (
	"context"
	"errors"

	adminpb "github.com/nmxmxh/master-ovasabi/api/protos/admin/v1"
	userpb "github.com/nmxmxh/master-ovasabi/api/protos/user/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Repository interface {
	// User management
	CreateUser(ctx context.Context, user *adminpb.User) (*adminpb.User, error)
	UpdateUser(ctx context.Context, user *adminpb.User) (*adminpb.User, error)
	DeleteUser(ctx context.Context, userID string) error
	ListUsers(ctx context.Context, page, pageSize int) ([]*adminpb.User, int, error)
	GetUser(ctx context.Context, userID string) (*adminpb.User, error)
	// Role management
	CreateRole(ctx context.Context, role *adminpb.Role) (*adminpb.Role, error)
	UpdateRole(ctx context.Context, role *adminpb.Role) (*adminpb.Role, error)
	DeleteRole(ctx context.Context, roleID string) error
	ListRoles(ctx context.Context, page, pageSize int) ([]*adminpb.Role, int, error)
	// Role assignment
	AssignRole(ctx context.Context, userID, roleID string) error
	RevokeRole(ctx context.Context, userID, roleID string) error
	// Audit logs
	GetAuditLogs(ctx context.Context, page, pageSize int, userID, action string) ([]*adminpb.AuditLog, int, error)
	// Settings
	GetSettings(ctx context.Context) (*adminpb.Settings, error)
	UpdateSettings(ctx context.Context, settings *adminpb.Settings) (*adminpb.Settings, error)
}

type Service struct {
	adminpb.UnimplementedAdminServiceServer
	log        *zap.Logger
	repo       Repository
	userClient userpb.UserServiceClient
}

func NewAdminService(log *zap.Logger, repo Repository, userClient userpb.UserServiceClient) adminpb.AdminServiceServer {
	return &Service{
		log:        log,
		repo:       repo,
		userClient: userClient,
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
				return nil, status.Errorf(codes.Internal, "failed to create main user: %v", err)
			}
			mainUser = createResp.User
		} else {
			return nil, status.Errorf(codes.Internal, "failed to lookup main user: %v", err)
		}
	} else {
		mainUser = userResp.User
	}
	// Use mainUser.MasterId or generate a new master_id if needed
	adminUser, err := s.repo.CreateUser(ctx, &adminpb.User{
		Id:       mainUser.Id,
		MasterId: mainUser.MasterId, // propagate master_id
		Email:    email,
		Name:     req.User.Name,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create admin user: %v", err)
	}
	return &adminpb.CreateUserResponse{
		User: adminUser,
	}, nil
}

func (s *Service) UpdateUser(_ context.Context, _ *adminpb.UpdateUserRequest) (*adminpb.UpdateUserResponse, error) {
	// TODO: Validate input, update user in DB, return response
	// Pseudocode for canonical metadata pattern:
	// 1. Validate metadata (if present) using metadatautil.ValidateMetadata(req.User.Metadata)
	// 2. Store metadata as *common.Metadata in Postgres (jsonb)
	// 3. After successful DB write:
	//    - pattern.CacheMetadata(ctx, s.cache, "admin_user", user.Id, user.Metadata, 10*time.Minute)
	//    - pattern.RegisterSchedule(ctx, "admin_user", user.Id, user.Metadata)
	//    - pattern.EnrichKnowledgeGraph(ctx, "admin_user", user.Id, user.Metadata)
	//    - pattern.RegisterWithNexus(ctx, "admin_user", user.Metadata)
	return nil, errors.New("not implemented")
}

func (s *Service) DeleteUser(_ context.Context, _ *adminpb.DeleteUserRequest) (*adminpb.DeleteUserResponse, error) {
	// TODO: Delete user from DB, handle dependencies
	// Pseudocode:
	// 1. Fetch user by ID
	// 2. Delete user from DB
	// 3. Return success or error
	return nil, errors.New("not implemented")
}

func (s *Service) ListUsers(_ context.Context, _ *adminpb.ListUsersRequest) (*adminpb.ListUsersResponse, error) {
	// TODO: List users with pagination/filtering (each user includes metadata if present)
	return nil, errors.New("not implemented")
}

func (s *Service) GetUser(_ context.Context, _ *adminpb.GetUserRequest) (*adminpb.GetUserResponse, error) {
	// TODO: Fetch user by ID
	// Pseudocode:
	// 1. Fetch user from DB by ID
	// 2. Return user or error
	return nil, errors.New("not implemented")
}

// Role management.
func (s *Service) CreateRole(_ context.Context, _ *adminpb.CreateRoleRequest) (*adminpb.CreateRoleResponse, error) {
	// TODO: Create new role
	// Pseudocode for canonical metadata pattern:
	// 1. Validate metadata (if present) using metadatautil.ValidateMetadata(req.Role.Metadata)
	// 2. Store metadata as *common.Metadata in Postgres (jsonb)
	// 3. After successful DB write:
	//    - pattern.CacheMetadata(ctx, s.cache, "admin_role", role.Id, role.Metadata, 10*time.Minute)
	//    - pattern.RegisterSchedule(ctx, "admin_role", role.Id, role.Metadata)
	//    - pattern.EnrichKnowledgeGraph(ctx, "admin_role", role.Id, role.Metadata)
	//    - pattern.RegisterWithNexus(ctx, "admin_role", role.Metadata)
	return nil, errors.New("not implemented")
}

func (s *Service) UpdateRole(_ context.Context, _ *adminpb.UpdateRoleRequest) (*adminpb.UpdateRoleResponse, error) {
	// TODO: Update role data
	// Pseudocode for canonical metadata pattern:
	// 1. Validate metadata (if present) using metadatautil.ValidateMetadata(req.Role.Metadata)
	// 2. Store metadata as *common.Metadata in Postgres (jsonb)
	// 3. After successful DB write:
	//    - pattern.CacheMetadata(ctx, s.cache, "admin_role", role.Id, role.Metadata, 10*time.Minute)
	//    - pattern.RegisterSchedule(ctx, "admin_role", role.Id, role.Metadata)
	//    - pattern.EnrichKnowledgeGraph(ctx, "admin_role", role.Id, role.Metadata)
	//    - pattern.RegisterWithNexus(ctx, "admin_role", role.Metadata)
	return nil, errors.New("not implemented")
}

func (s *Service) DeleteRole(_ context.Context, _ *adminpb.DeleteRoleRequest) (*adminpb.DeleteRoleResponse, error) {
	// TODO: Delete role
	// Pseudocode:
	// 1. Fetch role by ID
	// 2. Delete from DB
	// 3. Return success or error
	return nil, errors.New("not implemented")
}

func (s *Service) ListRoles(_ context.Context, _ *adminpb.ListRolesRequest) (*adminpb.ListRolesResponse, error) {
	// TODO: List all roles (each role includes metadata if present)
	return nil, errors.New("not implemented")
}

// Role assignment.
func (s *Service) AssignRole(_ context.Context, _ *adminpb.AssignRoleRequest) (*adminpb.AssignRoleResponse, error) {
	// TODO: Assign role to user
	// Pseudocode:
	// 1. Validate user and role
	// 2. Update user-role mapping
	// 3. Return success or error
	return nil, errors.New("not implemented")
}

func (s *Service) RevokeRole(_ context.Context, _ *adminpb.RevokeRoleRequest) (*adminpb.RevokeRoleResponse, error) {
	// TODO: Revoke role from user
	// Pseudocode:
	// 1. Validate user and role
	// 2. Remove user-role mapping
	// 3. Return success or error
	return nil, errors.New("not implemented")
}

// Audit logs.
func (s *Service) GetAuditLogs(_ context.Context, _ *adminpb.GetAuditLogsRequest) (*adminpb.GetAuditLogsResponse, error) {
	// TODO: Fetch audit logs
	// Pseudocode:
	// 1. Query logs from DB
	// 2. Return logs
	return nil, errors.New("not implemented")
}

// Settings.
func (s *Service) GetSettings(_ context.Context, _ *adminpb.GetSettingsRequest) (*adminpb.GetSettingsResponse, error) {
	// TODO: Get system settings
	// Pseudocode:
	// 1. Fetch settings from DB/config
	// 2. Return settings
	return nil, errors.New("not implemented")
}

func (s *Service) UpdateSettings(_ context.Context, _ *adminpb.UpdateSettingsRequest) (*adminpb.UpdateSettingsResponse, error) {
	// TODO: Update system settings
	// Pseudocode for canonical metadata pattern:
	// 1. Validate metadata (if present) using metadatautil.ValidateMetadata(req.Settings.Metadata)
	// 2. Store metadata as *common.Metadata in Postgres (jsonb)
	// 3. After successful DB write:
	//    - pattern.CacheMetadata(ctx, s.cache, "admin_settings", settings.Id, settings.Metadata, 10*time.Minute)
	//    - pattern.RegisterSchedule(ctx, "admin_settings", settings.Id, settings.Metadata)
	//    - pattern.EnrichKnowledgeGraph(ctx, "admin_settings", settings.Id, settings.Metadata)
	//    - pattern.RegisterWithNexus(ctx, "admin_settings", settings.Metadata)
	return nil, errors.New("not implemented")
}
