package service

import (
	"context"
	"time"

	authpb "github.com/nmxmxh/master-ovasabi/api/protos/auth/v0"
	broadcastpb "github.com/nmxmxh/master-ovasabi/api/protos/broadcast/v0"
	i18npb "github.com/nmxmxh/master-ovasabi/api/protos/i18n/v0"
	quotespb "github.com/nmxmxh/master-ovasabi/api/protos/quotes/v0"
	referralpb "github.com/nmxmxh/master-ovasabi/api/protos/referral/v0"
	"github.com/nmxmxh/master-ovasabi/pkg/models"
)

// User is an alias for the models.User type.
type User = models.User

// AuthService is an alias for the gRPC server interface.
type AuthService = authpb.AuthServiceServer

// BroadcastService is an alias for the gRPC server interface.
type BroadcastService = broadcastpb.BroadcastServiceServer

// I18nService is an alias for the gRPC server interface.
type I18nService = i18npb.I18NServiceServer

// UserService handles user management.
type UserService interface {
	GetUser(ctx context.Context, id int32) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id int32) (*User, error)
	CreateUser(ctx context.Context, email, password string) error
	UpdateUser(ctx context.Context, user *User) error
	DeleteUser(ctx context.Context, id int32) error
	ListUsers(ctx context.Context, offset, limit int) ([]*User, error)
}

// NotificationService handles sending notifications.
type NotificationService interface {
	SendEmail(ctx context.Context, to, subject, body string) error
	SendSMS(ctx context.Context, to, message string) error
	SendPushNotification(ctx context.Context, userID, title, message string) error
}

// Quote represents a financial quote.
type Quote struct {
	Symbol    string
	Price     float64
	Volume    int64
	Timestamp time.Time
}

// ReferralStats represents referral statistics.
type ReferralStats struct {
	TotalReferrals  int
	ActiveReferrals int
	TotalRewards    float64
}

// Registry defines the interface for service registration.
type Registry interface {
	RegisterService(name string, service interface{}) error
	GetService(name string) (interface{}, error)
	ListServices() []string
}

// ServiceContainer defines the interface for accessing all service implementations.
type Container interface {
	// Initialize initializes the service provider
	Initialize() error

	// Auth returns the authentication service
	Auth() authpb.AuthServiceServer

	// User returns the user service
	User() UserService

	// Notification returns the notification service
	Notification() NotificationService

	// Broadcast returns the broadcast service
	Broadcast() broadcastpb.BroadcastServiceServer

	// I18n returns the internationalization service
	I18n() i18npb.I18NServiceServer

	// Quotes returns the quotes service
	Quotes() quotespb.QuotesServiceServer

	// Referrals returns the referral service
	Referrals() referralpb.ReferralServiceServer
}
