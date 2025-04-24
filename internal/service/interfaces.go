package service

import (
	"context"
	"time"

	authpb "github.com/ovasabi/master-ovasabi/api/protos/auth"
	broadcastpb "github.com/ovasabi/master-ovasabi/api/protos/broadcast"
	i18npb "github.com/ovasabi/master-ovasabi/api/protos/i18n"
	quotespb "github.com/ovasabi/master-ovasabi/api/protos/quotes"
	referralpb "github.com/ovasabi/master-ovasabi/api/protos/referral"
	"github.com/ovasabi/master-ovasabi/pkg/models"
)

// User is an alias for the models.User type
type User = models.User

// AuthService is an alias for the gRPC server interface
type AuthService = authpb.AuthServiceServer

// BroadcastService is an alias for the gRPC server interface
type BroadcastService = broadcastpb.BroadcastServiceServer

// I18nService is an alias for the gRPC server interface
type I18nService = i18npb.I18NServiceServer

// UserService handles user management
type UserService interface {
	GetUser(ctx context.Context, id string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id string) (*User, error)
	CreateUser(ctx context.Context, email, password string) error
	UpdateUser(ctx context.Context, user *User) error
	DeleteUser(ctx context.Context, id string) error
	ListUsers(ctx context.Context, offset, limit int) ([]*User, error)
}

// NotificationService handles sending notifications
type NotificationService interface {
	SendEmail(ctx context.Context, to, subject, body string) error
	SendSMS(ctx context.Context, to, message string) error
	SendPushNotification(ctx context.Context, userID, title, message string) error
}

// Quote represents a financial quote
type Quote struct {
	Symbol    string
	Price     float64
	Volume    int64
	Timestamp time.Time
}

// ReferralStats represents referral statistics
type ReferralStats struct {
	TotalReferrals  int
	ActiveReferrals int
	TotalRewards    float64
}

// ServiceProvider defines the interface for accessing services
type ServiceProvider interface {
	Auth() authpb.AuthServiceServer
	Users() authpb.AuthServiceServer
	Notifications() NotificationService
	Broadcast() broadcastpb.BroadcastServiceServer
	I18n() i18npb.I18NServiceServer
	Quotes() quotespb.QuotesServiceServer
	Referrals() referralpb.ReferralServiceServer
}
