package auth

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/nmxmxh/master-ovasabi/api/protos/auth"
	userpb "github.com/nmxmxh/master-ovasabi/api/protos/user"
	"github.com/nmxmxh/master-ovasabi/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// mockUserService implements UserService for testing
type mockUserService struct {
	userpb.UnimplementedUserServiceServer
	mu    sync.RWMutex
	users map[string]*models.User
}

func newMockUserService() *mockUserService {
	return &mockUserService{
		users: make(map[string]*models.User),
	}
}

func (m *mockUserService) CreateUser(ctx context.Context, req *userpb.CreateUserRequest) (*userpb.CreateUserResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for existing user
	for _, user := range m.users {
		if user.Email == req.Email {
			return nil, ErrUserExists
		}
	}

	user := &models.User{
		ID:        req.Email,
		Email:     req.Email,
		Username:  req.Username,
		Roles:     []string{"user"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.users[user.ID] = user

	return &userpb.CreateUserResponse{
		User: &userpb.User{
			Id:        user.ID,
			Email:     user.Email,
			Username:  user.Username,
			Roles:     user.Roles,
			CreatedAt: user.CreatedAt.Unix(),
			UpdatedAt: user.UpdatedAt.Unix(),
		},
	}, nil
}

func (m *mockUserService) GetUser(ctx context.Context, req *userpb.GetUserRequest) (*userpb.GetUserResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, ok := m.users[req.UserId]
	if !ok {
		return nil, ErrUserNotFound
	}

	return &userpb.GetUserResponse{
		User: &userpb.User{
			Id:        user.ID,
			Email:     user.Email,
			Username:  user.Username,
			Roles:     user.Roles,
			CreatedAt: user.CreatedAt.Unix(),
			UpdatedAt: user.UpdatedAt.Unix(),
		},
	}, nil
}

func (m *mockUserService) UpdateUser(ctx context.Context, req *userpb.UpdateUserRequest) (*userpb.UpdateUserResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, ok := m.users[req.User.Id]
	if !ok {
		return nil, ErrUserNotFound
	}

	user.Email = req.User.Email
	user.Username = req.User.Username
	user.Roles = req.User.Roles
	user.UpdatedAt = time.Now()

	return &userpb.UpdateUserResponse{
		User: &userpb.User{
			Id:        user.ID,
			Email:     user.Email,
			Username:  user.Username,
			Roles:     user.Roles,
			CreatedAt: user.CreatedAt.Unix(),
			UpdatedAt: user.UpdatedAt.Unix(),
		},
	}, nil
}

func (m *mockUserService) DeleteUser(ctx context.Context, req *userpb.DeleteUserRequest) (*userpb.DeleteUserResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.users[req.UserId]; !ok {
		return nil, ErrUserNotFound
	}
	delete(m.users, req.UserId)

	return &userpb.DeleteUserResponse{}, nil
}

func (m *mockUserService) ListUsers(ctx context.Context, req *userpb.ListUsersRequest) (*userpb.ListUsersResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	users := make([]*userpb.User, 0, len(m.users))
	for _, user := range m.users {
		users = append(users, &userpb.User{
			Id:        user.ID,
			Email:     user.Email,
			Username:  user.Username,
			Roles:     user.Roles,
			CreatedAt: user.CreatedAt.Unix(),
			UpdatedAt: user.UpdatedAt.Unix(),
		})
	}

	start := int(req.Page * req.PageSize)
	end := start + int(req.PageSize)
	if end > len(users) {
		end = len(users)
	}
	if start >= len(users) {
		return &userpb.ListUsersResponse{Users: []*userpb.User{}}, nil
	}

	return &userpb.ListUsersResponse{Users: users[start:end]}, nil
}

func (m *mockUserService) UpdatePassword(ctx context.Context, req *userpb.UpdatePasswordRequest) (*userpb.UpdatePasswordResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, ok := m.users[req.UserId]
	if !ok {
		return nil, ErrUserNotFound
	}

	// In a real implementation, we would verify the old password and hash the new one
	user.UpdatedAt = time.Now()

	return &userpb.UpdatePasswordResponse{}, nil
}

func (m *mockUserService) UpdateProfile(ctx context.Context, req *userpb.UpdateProfileRequest) (*userpb.UpdateProfileResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, ok := m.users[req.UserId]
	if !ok {
		return nil, ErrUserNotFound
	}

	if req.Profile != nil {
		user.Username = req.Profile.FirstName
	}
	user.UpdatedAt = time.Now()

	return &userpb.UpdateProfileResponse{
		User: &userpb.User{
			Id:        user.ID,
			Email:     user.Email,
			Username:  user.Username,
			Roles:     user.Roles,
			CreatedAt: user.CreatedAt.Unix(),
			UpdatedAt: user.UpdatedAt.Unix(),
		},
	}, nil
}

func TestNewService(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "creates service with dependencies",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zap.NewNop()
			userSvc := newMockUserService()
			svc := NewService(logger, userSvc)

			assert.NotNil(t, svc)
			assert.NotNil(t, svc.log)
			assert.NotNil(t, svc.userSvc)
		})
	}
}

func TestService_Register(t *testing.T) {
	tests := []struct {
		name          string
		request       *auth.RegisterRequest
		setupMock     func(*mockUserService, *testing.T)
		expectedResp  *auth.RegisterResponse
		expectedError error
	}{
		{
			name: "successful registration",
			request: &auth.RegisterRequest{
				Email:    "test@example.com",
				Password: "password123",
				Name:     "Test User",
			},
			setupMock: func(m *mockUserService, t *testing.T) {},
			expectedResp: &auth.RegisterResponse{
				Message: "Registration successful",
			},
			expectedError: nil,
		},
		{
			name: "duplicate email",
			request: &auth.RegisterRequest{
				Email:    "existing@example.com",
				Password: "password123",
				Name:     "Test User",
			},
			setupMock: func(m *mockUserService, t *testing.T) {
				resp, err := m.CreateUser(context.Background(), &userpb.CreateUserRequest{
					Email:    "existing@example.com",
					Username: "existing@example.com",
				})
				if err != nil {
					t.Fatalf("Failed to create test user: %v", err)
				}
				_ = resp
			},
			expectedResp:  nil,
			expectedError: ErrUserExists,
		},
		{
			name: "invalid email",
			request: &auth.RegisterRequest{
				Email:    "invalid-email",
				Password: "password123",
				Name:     "Test User",
			},
			setupMock:     func(m *mockUserService, t *testing.T) {},
			expectedResp:  nil,
			expectedError: ErrInvalidEmail,
		},
		{
			name: "password too short",
			request: &auth.RegisterRequest{
				Email:    "test@example.com",
				Password: "short",
				Name:     "Test User",
			},
			setupMock:     func(m *mockUserService, t *testing.T) {},
			expectedResp:  nil,
			expectedError: ErrWeakPassword,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserSvc := newMockUserService()
			tt.setupMock(mockUserSvc, t)
			svc := NewService(zap.NewNop(), mockUserSvc)

			resp, err := svc.Register(context.Background(), tt.request)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.expectedError))
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedResp.Message, resp.Message)
			}
		})
	}
}

func TestService_Login(t *testing.T) {
	tests := []struct {
		name          string
		request       *auth.LoginRequest
		setupMock     func(*mockUserService, *testing.T)
		expectedResp  *auth.LoginResponse
		expectedError error
	}{
		{
			name: "successful login",
			request: &auth.LoginRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
			setupMock: func(m *mockUserService, t *testing.T) {
				resp, err := m.CreateUser(context.Background(), &userpb.CreateUserRequest{
					Email:    "test@example.com",
					Username: "test@example.com",
				})
				if err != nil {
					t.Fatalf("Failed to create test user: %v", err)
				}
				_ = resp
			},
			expectedResp: &auth.LoginResponse{
				AccessToken: "token",
				ExpiresIn:   3600,
			},
			expectedError: nil,
		},
		{
			name: "invalid credentials",
			request: &auth.LoginRequest{
				Email:    "test@example.com",
				Password: "wrongpassword",
			},
			setupMock: func(m *mockUserService, t *testing.T) {
				resp, err := m.CreateUser(context.Background(), &userpb.CreateUserRequest{
					Email:    "test@example.com",
					Username: "test@example.com",
				})
				if err != nil {
					t.Fatalf("Failed to create test user: %v", err)
				}
				_ = resp
			},
			expectedResp:  nil,
			expectedError: ErrInvalidCredentials,
		},
		{
			name: "user not found",
			request: &auth.LoginRequest{
				Email:    "nonexistent@example.com",
				Password: "password123",
			},
			setupMock:     func(m *mockUserService, t *testing.T) {},
			expectedResp:  nil,
			expectedError: ErrInvalidCredentials,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserSvc := newMockUserService()
			tt.setupMock(mockUserSvc, t)
			svc := NewService(zap.NewNop(), mockUserSvc)

			resp, err := svc.Login(context.Background(), tt.request)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.expectedError))
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, resp.AccessToken)
				assert.Equal(t, tt.expectedResp.ExpiresIn, resp.ExpiresIn)
			}
		})
	}
}

func TestService_GetUser(t *testing.T) {
	tests := []struct {
		name          string
		request       *auth.GetUserRequest
		setupMock     func(*mockUserService, *testing.T)
		expectedResp  *auth.GetUserResponse
		expectedError error
	}{
		{
			name: "successful get user",
			request: &auth.GetUserRequest{
				UserId: "test-user",
			},
			setupMock: func(m *mockUserService, t *testing.T) {
				resp, err := m.CreateUser(context.Background(), &userpb.CreateUserRequest{
					Email:    "test@example.com",
					Username: "test@example.com",
				})
				if err != nil {
					t.Fatalf("Failed to create test user: %v", err)
				}
				_ = resp
			},
			expectedResp: &auth.GetUserResponse{
				UserId: "test-user",
				Email:  "test@example.com",
			},
			expectedError: nil,
		},
		{
			name: "user not found",
			request: &auth.GetUserRequest{
				UserId: "nonexistent-user",
			},
			setupMock:     func(m *mockUserService, t *testing.T) {},
			expectedResp:  nil,
			expectedError: ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserSvc := newMockUserService()
			tt.setupMock(mockUserSvc, t)
			svc := NewService(zap.NewNop(), mockUserSvc)

			resp, err := svc.GetUser(context.Background(), tt.request)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.expectedError))
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedResp.UserId, resp.UserId)
				assert.Equal(t, tt.expectedResp.Email, resp.Email)
			}
		})
	}
}
