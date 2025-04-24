package benchmarks

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/nmxmxh/master-ovasabi/api/protos/auth"
	userpb "github.com/nmxmxh/master-ovasabi/api/protos/user"
	authsvc "github.com/nmxmxh/master-ovasabi/internal/service/auth"
	"github.com/nmxmxh/master-ovasabi/pkg/models"
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
			return nil, authsvc.ErrUserExists
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
		return nil, authsvc.ErrUserNotFound
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
		return nil, authsvc.ErrUserNotFound
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
		return nil, authsvc.ErrUserNotFound
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
		return nil, authsvc.ErrUserNotFound
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
		return nil, authsvc.ErrUserNotFound
	}

	if req.Profile != nil {
		user.Username = req.Profile.FirstName // Use FirstName as username for simplicity
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

func BenchmarkAuthService_Register(b *testing.B) {
	log := zap.NewNop()
	userSvc := newMockUserService()
	svc := authsvc.NewService(log, userSvc)

	req := &auth.RegisterRequest{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.Register(context.Background(), req)
	}
}

func BenchmarkAuthService_Login(b *testing.B) {
	log := zap.NewNop()
	userSvc := newMockUserService()
	svc := authsvc.NewService(log, userSvc)

	// Register a user first
	regReq := &auth.RegisterRequest{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
	}
	_, _ = svc.Register(context.Background(), regReq)

	loginReq := &auth.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.Login(context.Background(), loginReq)
	}
}

func BenchmarkAuthService_LoginParallel(b *testing.B) {
	log := zap.NewNop()
	userSvc := newMockUserService()
	svc := authsvc.NewService(log, userSvc)

	// Register a user first
	regReq := &auth.RegisterRequest{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
	}
	_, _ = svc.Register(context.Background(), regReq)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			loginReq := &auth.LoginRequest{
				Email:    "test@example.com",
				Password: "password123",
			}
			_, _ = svc.Login(context.Background(), loginReq)
		}
	})
}

func BenchmarkAuthService_GetUser(b *testing.B) {
	log := zap.NewNop()
	userSvc := newMockUserService()
	svc := authsvc.NewService(log, userSvc)

	// Register a user first
	regReq := &auth.RegisterRequest{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
	}
	_, _ = svc.Register(context.Background(), regReq)

	req := &auth.GetUserRequest{
		UserId: "test@example.com",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.GetUser(context.Background(), req)
	}
}

func BenchmarkAuthService_GetUserParallel(b *testing.B) {
	log := zap.NewNop()
	userSvc := newMockUserService()
	svc := authsvc.NewService(log, userSvc)

	// Register a user first
	regReq := &auth.RegisterRequest{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
	}
	_, _ = svc.Register(context.Background(), regReq)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := &auth.GetUserRequest{
				UserId: "test@example.com",
			}
			_, _ = svc.GetUser(context.Background(), req)
		}
	})
}

func BenchmarkAuthService_ValidateToken(b *testing.B) {
	log := zap.NewNop()
	userSvc := newMockUserService()
	svc := authsvc.NewService(log, userSvc)

	// Register a user and get a token
	regReq := &auth.RegisterRequest{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
	}
	_, _ = svc.Register(context.Background(), regReq)

	loginReq := &auth.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}
	loginResp, _ := svc.Login(context.Background(), loginReq)

	req := &auth.ValidateTokenRequest{
		Token: loginResp.AccessToken,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.ValidateToken(context.Background(), req)
	}
}

func BenchmarkAuthService_ValidateTokenParallel(b *testing.B) {
	log := zap.NewNop()
	userSvc := newMockUserService()
	svc := authsvc.NewService(log, userSvc)

	// Register a user and get a token
	regReq := &auth.RegisterRequest{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
	}
	_, _ = svc.Register(context.Background(), regReq)

	loginReq := &auth.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}
	loginResp, _ := svc.Login(context.Background(), loginReq)

	req := &auth.ValidateTokenRequest{
		Token: loginResp.AccessToken,
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = svc.ValidateToken(context.Background(), req)
		}
	})
}
