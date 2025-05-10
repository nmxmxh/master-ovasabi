package auth

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	authpb "github.com/nmxmxh/master-ovasabi/api/protos/auth/v1"
	userpb "github.com/nmxmxh/master-ovasabi/api/protos/user/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
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
	authpb.UnimplementedAuthServiceServer
	log        *zap.Logger
	userSvc    userpb.UserServiceServer
	jwtSecret  []byte
	expiration time.Duration
	cache      *redis.Cache
}

// Compile-time check.
var _ authpb.AuthServiceServer = (*ServiceImpl)(nil)

// NewService creates a new auth service instance.
func NewService(log *zap.Logger, userSvc userpb.UserServiceServer, cache *redis.Cache) *ServiceImpl {
	return &ServiceImpl{
		log:        log,
		userSvc:    userSvc,
		jwtSecret:  []byte(os.Getenv("JWT_SECRET")),
		expiration: 24 * time.Hour,
		cache:      cache,
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
	// Password strength checks not yet implemented
	return nil
}

// Register handles user registration.
func (s *ServiceImpl) Register(ctx context.Context, req *authpb.RegisterRequest) (*authpb.RegisterResponse, error) {
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

	// Create user with password hash
	resp, err := s.userSvc.CreateUser(ctx, &userpb.CreateUserRequest{
		Email:    req.Username,
		Username: req.Username,
		Password: string(hashedPassword), // Pass the hashed password to be stored
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

	log.Info("User registered successfully",
		zap.Int32("user_id", resp.User.Id))

	return &authpb.RegisterResponse{
		UserId:  resp.User.Id,
		Success: true,
		Message: "Registration successful",
	}, nil
}

// Login handles user authentication.
func (s *ServiceImpl) Login(ctx context.Context, req *authpb.LoginRequest) (*authpb.LoginResponse, error) {
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

	// Get user details including password hash
	userDetails, err := s.userSvc.GetUser(ctx, &userpb.GetUserRequest{
		UserId: fmt.Sprint(user.Id),
	})
	if err != nil {
		log.Error("Login failed: could not retrieve user details",
			zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to retrieve user details")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(userDetails.User.PasswordHash), []byte(req.Password)); err != nil {
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

	// Cache the token with user info
	if err := s.cache.Set(ctx, fmt.Sprint(user.Id), "token", tokenString, s.expiration); err != nil {
		log.Error("Failed to cache token",
			zap.Int32("user_id", user.Id),
			zap.Error(err))
		// Don't fail the login if caching fails
	}

	log.Info("Login successful",
		zap.Int32("user_id", user.Id))

	return &authpb.LoginResponse{
		Token:   tokenString,
		UserId:  user.Id,
		Success: true,
		Message: "Login successful",
	}, nil
}

// ValidateToken validates a JWT token.
func (s *ServiceImpl) ValidateToken(ctx context.Context, req *authpb.ValidateTokenRequest) (*authpb.ValidateTokenResponse, error) {
	log := s.log.With(zap.String("operation", "validate_token"))

	claims, err := s.parseToken(req.Token)
	if err != nil {
		log.Warn("Token validation failed", zap.Error(err))
		return nil, err
	}

	// Check token in cache
	var cachedToken string
	if err := s.cache.Get(ctx, claims.UserID, "token", &cachedToken); err != nil {
		log.Warn("Failed to get token from cache",
			zap.String("user_id", claims.UserID),
			zap.Error(err))
	} else if cachedToken == req.Token {
		// Token found in cache and matches
		return &authpb.ValidateTokenResponse{
			Valid:  true,
			UserId: claims.UserID,
		}, nil
	}

	// If not in cache or cache check failed, validate token signature
	if _, err := jwt.ParseWithClaims(req.Token, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	}); err != nil {
		log.Warn("Token signature validation failed", zap.Error(err))
		return nil, ErrInvalidToken
	}

	// Cache the validated token
	if err := s.cache.Set(ctx, claims.UserID, "token", req.Token, s.expiration); err != nil {
		log.Error("Failed to cache validated token",
			zap.String("user_id", claims.UserID),
			zap.Error(err))
		// Don't fail validation if caching fails
	}

	return &authpb.ValidateTokenResponse{
		Valid:  true,
		UserId: claims.UserID,
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
