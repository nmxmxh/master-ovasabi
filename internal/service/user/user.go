package user

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	userpb "github.com/nmxmxh/master-ovasabi/api/protos/user/v1"
	userrepo "github.com/nmxmxh/master-ovasabi/internal/repository/user"
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

// NewUserService creates a new instance of UserService.
func NewUserService(log *zap.Logger, repo *userrepo.UserRepository, cache *redis.Cache) userpb.UserServiceServer {
	return &Service{
		log:   log,
		repo:  repo,
		cache: cache,
	}
}

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
		Username: req.Username,
		Email:    req.Email,
		Password: string(hashedPassword),
	}
	if req.Metadata != nil {
		metadata, _ := json.Marshal(req.Metadata)
		user.Metadata = metadata
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

	respUser := &userpb.User{
		Id:           int32(created.ID),
		Username:     created.Username,
		Email:        created.Email,
		CreatedAt:    timestamppb.New(created.CreatedAt),
		UpdatedAt:    timestamppb.New(created.UpdatedAt),
		PasswordHash: created.Password,
	}

	// Cache the new user
	if err := s.cache.Set(ctx, fmt.Sprint(created.ID), "profile", respUser, redis.TTLUserProfile); err != nil {
		s.log.Error("Failed to cache user profile",
			zap.Int64("user_id", created.ID),
			zap.Error(err))
		// Don't fail creation if caching fails
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

	id, err := strconv.ParseInt(req.UserId, 10, 64)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}
	repoUser, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, userrepo.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "database error: %v", err)
	}
	respUser := &userpb.User{
		Id:           int32(repoUser.ID),
		Username:     repoUser.Username,
		Email:        repoUser.Email,
		CreatedAt:    timestamppb.New(repoUser.CreatedAt),
		UpdatedAt:    timestamppb.New(repoUser.UpdatedAt),
		PasswordHash: repoUser.Password,
	}

	if err := s.cache.Set(ctx, req.UserId, "profile", respUser, redis.TTLUserProfile); err != nil {
		s.log.Error("Failed to cache user profile",
			zap.String("user_id", req.UserId),
			zap.Error(err))
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

	respUser := &userpb.User{
		Id:           int32(repoUser.ID),
		Username:     repoUser.Username,
		Email:        repoUser.Email,
		CreatedAt:    timestamppb.New(repoUser.CreatedAt),
		UpdatedAt:    timestamppb.New(repoUser.UpdatedAt),
		PasswordHash: repoUser.Password,
	}

	return &userpb.GetUserByUsernameResponse{User: respUser}, nil
}

// UpdateUser updates a user record.
func (s *Service) UpdateUser(ctx context.Context, req *userpb.UpdateUserRequest) (*userpb.UpdateUserResponse, error) {
	id, err := strconv.ParseInt(req.UserId, 10, 64)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}
	repoUser, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, userrepo.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, "failed to get user")
	}

	// Update fields
	if req.User.Username != "" {
		repoUser.Username = req.User.Username
	}
	if req.User.Email != "" {
		repoUser.Email = req.User.Email
	}
	if req.User.PasswordHash != "" {
		repoUser.Password = req.User.PasswordHash
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
	return &userpb.UpdateUserResponse{User: getResp.User}, nil
}

// DeleteUser removes a user and its master record.
func (s *Service) DeleteUser(ctx context.Context, req *userpb.DeleteUserRequest) (*userpb.DeleteUserResponse, error) {
	id, err := strconv.ParseInt(req.UserId, 10, 64)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
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
	limit := 10
	if req.PageSize > 0 {
		limit = int(req.PageSize)
	}
	offset := int(req.Page * int32(limit))
	users, err := s.repo.List(ctx, limit, offset)
	if err != nil {
		s.log.Error("failed to list users", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to list users: %v", err)
	}
	resp := &userpb.ListUsersResponse{
		Users: make([]*userpb.User, 0, len(users)),
	}
	for _, u := range users {
		respUser := &userpb.User{
			Id:           int32(u.ID),
			Email:        u.Email,
			CreatedAt:    timestamppb.New(u.CreatedAt),
			UpdatedAt:    timestamppb.New(u.UpdatedAt),
			PasswordHash: u.Password,
		}
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
	id, err := strconv.ParseInt(req.UserId, 10, 64)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}
	repoUser, err := s.repo.GetByID(ctx, id)
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
	if err := s.repo.Update(ctx, repoUser); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update profile: %v", err)
	}
	getResp, err := s.GetUser(ctx, &userpb.GetUserRequest{UserId: req.UserId})
	if err != nil {
		return nil, err
	}
	return &userpb.UpdateProfileResponse{User: getResp.User}, nil
}
