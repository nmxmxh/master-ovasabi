package auth

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/nmxmxh/master-ovasabi/api/protos/auth"
	userpb "github.com/nmxmxh/master-ovasabi/api/protos/user"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token expired")
	ErrInvalidEmail       = errors.New("invalid email format")
	ErrWeakPassword       = errors.New("password too weak")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserExists         = errors.New("user already exists")
)

type Claims struct {
	UserID string   `json:"sub"`
	Roles  []string `json:"roles,omitempty"`
	jwt.RegisteredClaims
}

// ServiceImpl implements the AuthService interface
type ServiceImpl struct {
	auth.UnimplementedAuthServiceServer
	log        *zap.Logger
	userSvc    userpb.UserServiceServer
	jwtSecret  []byte
	expiration time.Duration

	// Password storage
	mu           sync.RWMutex
	passwordHash map[string]string // userID -> hashed password
}

// NewService creates a new instance of AuthService
func NewService(log *zap.Logger, userSvc userpb.UserServiceServer) *ServiceImpl {
	return &ServiceImpl{
		log:          log,
		userSvc:      userSvc,
		jwtSecret:    []byte("your-secret-key"), // TODO: Load from config
		expiration:   24 * time.Hour,            // TODO: Load from config
		passwordHash: make(map[string]string),
	}
}

func validateEmail(email string) error {
	if _, err := mail.ParseAddress(email); err != nil {
		return ErrInvalidEmail
	}
	return nil
}

func validatePassword(password string) error {
	if len(password) < 8 {
		return ErrWeakPassword
	}
	// TODO: Add more password strength checks
	return nil
}

// Register handles user registration
func (s *ServiceImpl) Register(ctx context.Context, req *auth.RegisterRequest) (*auth.RegisterResponse, error) {
	if err := validateEmail(req.Email); err != nil {
		return nil, err
	}

	if err := validatePassword(req.Password); err != nil {
		return nil, err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user using UserService
	resp, err := s.userSvc.CreateUser(ctx, &userpb.CreateUserRequest{
		Email:    req.Email,
		Username: req.Email, // Use email as username for now
	})
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return nil, fmt.Errorf("user already exists")
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Store password hash
	s.mu.Lock()
	s.passwordHash[resp.User.Id] = string(hashedPassword)
	s.mu.Unlock()

	return &auth.RegisterResponse{
		UserId:  resp.User.Id,
		Message: "Registration successful",
	}, nil
}

// Login handles user authentication
func (s *ServiceImpl) Login(ctx context.Context, req *auth.LoginRequest) (*auth.LoginResponse, error) {
	// List users to find by email
	resp, err := s.userSvc.ListUsers(ctx, &userpb.ListUsersRequest{
		Filters: map[string]string{"email": req.Email},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if len(resp.Users) == 0 {
		return nil, ErrInvalidCredentials
	}
	user := resp.Users[0]

	// Get password hash
	s.mu.RLock()
	hashedPassword, ok := s.passwordHash[user.Id]
	s.mu.RUnlock()

	if !ok {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	claims := &Claims{
		UserID: user.Id,
		Roles:  user.Roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.expiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to sign token: %w", err)
	}

	return &auth.LoginResponse{
		AccessToken: tokenString,
		ExpiresIn:   int64(s.expiration.Seconds()),
	}, nil
}

// ValidateToken verifies and parses a JWT token
func (s *ServiceImpl) ValidateToken(ctx context.Context, req *auth.ValidateTokenRequest) (*auth.ValidateTokenResponse, error) {
	claims, err := s.parseToken(req.Token)
	if err != nil {
		return nil, err
	}

	// Get user to verify it still exists and is active
	userResp, err := s.userSvc.GetUser(ctx, &userpb.GetUserRequest{UserId: claims.UserID})
	if err != nil {
		return nil, ErrInvalidToken
	}

	return &auth.ValidateTokenResponse{
		Valid:  true,
		UserId: userResp.User.Id,
		Roles:  userResp.User.Roles,
	}, nil
}

func (s *ServiceImpl) parseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
