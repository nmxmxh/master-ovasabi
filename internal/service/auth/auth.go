package auth

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"os"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	auth "github.com/nmxmxh/master-ovasabi/api/protos/auth/v0"
	userpb "github.com/nmxmxh/master-ovasabi/api/protos/user/v0"
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

// ServiceImpl implements the AuthService interface.
type ServiceImpl struct {
	auth.UnimplementedAuthServiceServer
	log          *zap.Logger
	userSvc      userpb.UserServiceServer
	jwtSecret    []byte
	expiration   time.Duration
	mu           sync.RWMutex
	passwordHash map[string]string
}

// NewService creates a new instance of AuthService with proper logging.
func NewService(log *zap.Logger, userSvc userpb.UserServiceServer) *ServiceImpl {
	return &ServiceImpl{
		log:          log.With(zap.String("service", "auth")),
		userSvc:      userSvc,
		jwtSecret:    []byte(os.Getenv("JWT_SECRET")),
		expiration:   24 * time.Hour,
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

// Register handles user registration.
func (s *ServiceImpl) Register(ctx context.Context, req *auth.RegisterRequest) (*auth.RegisterResponse, error) {
	log := s.log.With(
		zap.String("operation", "register"),
		zap.String("username", req.Username))

	log.Info("Starting user registration")

	if req.Username == "" {
		log.Warn("Registration failed: empty username")
		return nil, status.Error(codes.InvalidArgument, "username required")
	}

	if err := validateEmail(req.Username); err != nil {
		log.Warn("Registration failed: invalid email format",
			zap.Error(err))
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if err := validatePassword(req.Password); err != nil {
		log.Warn("Registration failed: Password validation failed",
			zap.Error(err))
		return nil, err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Error("Registration failed: password hashing failed",
			zap.Error(err))
		return nil, status.Error(codes.Internal, "internal error")
	}

	// Create user
	resp, err := s.userSvc.CreateUser(ctx, &userpb.CreateUserRequest{
		Email:    req.Username,
		Username: req.Username,
		Password: req.Password,
		Metadata: req.Metadata,
	})
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			log.Warn("Registration failed: User already exists",
				zap.Error(err))
			return nil, fmt.Errorf("user already exists")
		}
		log.Error("Registration failed: user creation failed",
			zap.Error(err))
		return nil, err
	}

	s.mu.Lock()
	s.passwordHash[fmt.Sprint(resp.User.Id)] = string(hashedPassword)
	s.mu.Unlock()

	log.Info("User registered successfully",
		zap.Int32("user_id", resp.User.Id))

	return &auth.RegisterResponse{
		UserId:  resp.User.Id,
		Success: true,
		Message: "Registration successful",
	}, nil
}

// Login handles user authentication.
func (s *ServiceImpl) Login(ctx context.Context, req *auth.LoginRequest) (*auth.LoginResponse, error) {
	log := s.log.With(
		zap.String("operation", "login"),
		zap.String("username", req.Username))

	log.Info("Processing login request")

	resp, err := s.userSvc.ListUsers(ctx, &userpb.ListUsersRequest{
		Filters: map[string]string{"username": req.Username},
	})
	if err != nil {
		log.Error("Login failed: user retrieval failed",
			zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to retrieve user")
	}

	if len(resp.Users) == 0 {
		log.Warn("Login failed: user not found")
		return nil, ErrInvalidCredentials
	}

	user := resp.Users[0]
	s.mu.RLock()
	hashedPassword, ok := s.passwordHash[fmt.Sprint(user.Id)]
	s.mu.RUnlock()

	if !ok {
		log.Warn("Login failed: no password hash found",
			zap.Int32("user_id", user.Id))
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(req.Password)); err != nil {
		log.Warn("Login failed: invalid password",
			zap.Int32("user_id", user.Id))
		return nil, ErrInvalidCredentials
	}

	log.Info("Password validated successfully, generating token",
		zap.Int32("user_id", user.Id))

	claims := &Claims{
		UserID: fmt.Sprint(user.Id),
		Roles:  nil,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.expiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		log.Error("Login failed: token generation failed",
			zap.Int32("user_id", user.Id),
			zap.Error(err))
		return nil, fmt.Errorf("failed to sign token: %w", err)
	}

	log.Info("Login successful",
		zap.Int32("user_id", user.Id))

	return &auth.LoginResponse{
		Token:   tokenString,
		UserId:  user.Id,
		Success: true,
		Message: "Login successful",
	}, nil
}

// ValidateToken verifies and parses a JWT token.
func (s *ServiceImpl) ValidateToken(ctx context.Context, req *auth.ValidateTokenRequest) (*auth.ValidateTokenResponse, error) {
	log := s.log.With(
		zap.String("operation", "validate_token"))

	log.Info("Starting token validation")

	claims, err := s.parseToken(req.Token)
	if err != nil {
		log.Warn("Token validation failed",
			zap.Error(err))
		return nil, err
	}

	log.Info("Token parsed successfully, verifying user",
		zap.String("user_id", claims.UserID))

	userResp, err := s.userSvc.GetUser(ctx, &userpb.GetUserRequest{UserId: claims.UserID})
	if err != nil {
		log.Error("Token validation failed: User verification failed",
			zap.String("user_id", claims.UserID),
			zap.Error(err))
		return nil, ErrInvalidToken
	}

	log.Info("Token validation completed successfully",
		zap.String("user_id", claims.UserID))

	return &auth.ValidateTokenResponse{
		Valid:  true,
		UserId: fmt.Sprint(userResp.User.Id),
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
