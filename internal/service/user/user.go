package user

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	userpb "github.com/nmxmxh/master-ovasabi/api/protos/user"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Service implements the UserService gRPC interface
type Service struct {
	userpb.UnimplementedUserServiceServer
	log   *zap.Logger
	mu    sync.RWMutex
	users map[string]*userpb.User
}

// NewUserService creates a new instance of UserService
func NewUserService(log *zap.Logger) userpb.UserServiceServer {
	return &Service{
		log:   log,
		users: make(map[string]*userpb.User),
	}
}

// CreateUser implements the CreateUser RPC method
func (s *Service) CreateUser(ctx context.Context, req *userpb.CreateUserRequest) (*userpb.CreateUserResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if user already exists
	for _, user := range s.users {
		if user.Email == req.Email || user.Username == req.Username {
			return nil, status.Error(codes.AlreadyExists, "user already exists")
		}
	}

	user := &userpb.User{
		Id:        uuid.New().String(),
		Email:     req.Email,
		Username:  req.Username,
		Roles:     req.Roles,
		Profile:   req.Profile,
		Status:    userpb.UserStatus_USER_STATUS_ACTIVE,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
		Metadata:  req.Metadata,
	}

	s.users[user.Id] = user

	return &userpb.CreateUserResponse{
		User: user,
	}, nil
}

// GetUser implements the GetUser RPC method
func (s *Service) GetUser(ctx context.Context, req *userpb.GetUserRequest) (*userpb.GetUserResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[req.UserId]
	if !ok {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	return &userpb.GetUserResponse{
		User: user,
	}, nil
}

// UpdateUser implements the UpdateUser RPC method
func (s *Service) UpdateUser(ctx context.Context, req *userpb.UpdateUserRequest) (*userpb.UpdateUserResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, ok := s.users[req.UserId]
	if !ok {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	// Update only specified fields
	if len(req.FieldsToUpdate) > 0 {
		for _, field := range req.FieldsToUpdate {
			switch field {
			case "email":
				user.Email = req.User.Email
			case "username":
				user.Username = req.User.Username
			case "roles":
				user.Roles = req.User.Roles
			case "profile":
				user.Profile = req.User.Profile
			case "status":
				user.Status = req.User.Status
			case "metadata":
				user.Metadata = req.User.Metadata
			}
		}
	} else {
		// Update all fields if no specific fields are specified
		user.Email = req.User.Email
		user.Username = req.User.Username
		user.Roles = req.User.Roles
		user.Profile = req.User.Profile
		user.Status = req.User.Status
		user.Metadata = req.User.Metadata
	}

	user.UpdatedAt = time.Now().Unix()
	s.users[user.Id] = user

	return &userpb.UpdateUserResponse{
		User: user,
	}, nil
}

// DeleteUser implements the DeleteUser RPC method
func (s *Service) DeleteUser(ctx context.Context, req *userpb.DeleteUserRequest) (*userpb.DeleteUserResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.users[req.UserId]; !ok {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	delete(s.users, req.UserId)

	return &userpb.DeleteUserResponse{
		Success: true,
	}, nil
}

// ListUsers implements the ListUsers RPC method
func (s *Service) ListUsers(ctx context.Context, req *userpb.ListUsersRequest) (*userpb.ListUsersResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var users []*userpb.User
	for _, user := range s.users {
		// Apply filters if any
		if len(req.Filters) > 0 {
			match := true
			for key, value := range req.Filters {
				switch key {
				case "status":
					if user.Status.String() != value {
						match = false
					}
					// Add more filter cases as needed
				}
			}
			if !match {
				continue
			}
		}
		users = append(users, user)
	}

	// Calculate pagination
	totalCount := len(users)
	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 10
	}
	totalPages := (totalCount + pageSize - 1) / pageSize

	start := int(req.Page) * pageSize
	if start >= totalCount {
		return &userpb.ListUsersResponse{
			Users:      []*userpb.User{},
			TotalCount: int32(totalCount),
			Page:       req.Page,
			TotalPages: int32(totalPages),
		}, nil
	}

	end := start + pageSize
	if end > totalCount {
		end = totalCount
	}

	return &userpb.ListUsersResponse{
		Users:      users[start:end],
		TotalCount: int32(totalCount),
		Page:       req.Page,
		TotalPages: int32(totalPages),
	}, nil
}

// UpdatePassword implements the UpdatePassword RPC method
func (s *Service) UpdatePassword(ctx context.Context, req *userpb.UpdatePasswordRequest) (*userpb.UpdatePasswordResponse, error) {
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

// UpdateProfile implements the UpdateProfile RPC method
func (s *Service) UpdateProfile(ctx context.Context, req *userpb.UpdateProfileRequest) (*userpb.UpdateProfileResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, ok := s.users[req.UserId]
	if !ok {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	if len(req.FieldsToUpdate) > 0 {
		for _, field := range req.FieldsToUpdate {
			switch field {
			case "first_name":
				user.Profile.FirstName = req.Profile.FirstName
			case "last_name":
				user.Profile.LastName = req.Profile.LastName
			case "phone_number":
				user.Profile.PhoneNumber = req.Profile.PhoneNumber
			case "avatar_url":
				user.Profile.AvatarUrl = req.Profile.AvatarUrl
			case "bio":
				user.Profile.Bio = req.Profile.Bio
			case "location":
				user.Profile.Location = req.Profile.Location
			case "timezone":
				user.Profile.Timezone = req.Profile.Timezone
			case "language":
				user.Profile.Language = req.Profile.Language
			case "custom_fields":
				user.Profile.CustomFields = req.Profile.CustomFields
			}
		}
	} else {
		user.Profile = req.Profile
	}

	user.UpdatedAt = time.Now().Unix()
	s.users[user.Id] = user

	return &userpb.UpdateProfileResponse{
		User: user,
	}, nil
}
