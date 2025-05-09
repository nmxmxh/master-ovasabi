package pattern

import (
	"context"
	"fmt"

	"github.com/nmxmxh/master-ovasabi/amadeus/pkg/kg"
	assetpb "github.com/nmxmxh/master-ovasabi/api/protos/asset/v1"
	notificationpb "github.com/nmxmxh/master-ovasabi/api/protos/notification/v1"
	userpb "github.com/nmxmxh/master-ovasabi/api/protos/user/v1"
)

// UserCreationPattern handles the complete user creation flow
type UserCreationPattern struct {
	knowledgeGraph *kg.KnowledgeGraph
	userService    userpb.UserServiceServer
	assetService   assetpb.AssetServiceServer
	notifyService  notificationpb.NotificationServiceServer
}

// NewUserCreationPattern creates a new UserCreationPattern
func NewUserCreationPattern(
	userSvc userpb.UserServiceServer,
	assetSvc assetpb.AssetServiceServer,
	notifySvc notificationpb.NotificationServiceServer,
) *UserCreationPattern {
	return &UserCreationPattern{
		knowledgeGraph: kg.DefaultKnowledgeGraph(),
		userService:    userSvc,
		assetService:   assetSvc,
		notifyService:  notifySvc,
	}
}

// Execute runs the user creation pattern
func (p *UserCreationPattern) Execute(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	// Step 1: Create user
	userReq := &userpb.CreateUserRequest{
		Username: params["username"].(string),
		Email:    params["email"].(string),
		Password: params["password"].(string),
	}
	userResp, err := p.userService.CreateUser(ctx, userReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Step 2: Upload user avatar if provided
	var assetID string
	// TODO: Implement avatar upload when UploadLightAssetRequest fields are defined in v1 proto

	// Step 3: Send welcome notifications
	// Email notification
	emailReq := &notificationpb.SendEmailRequest{
		To:      userResp.User.Email,
		Subject: "Welcome to OVASABI",
		Body:    fmt.Sprintf("Welcome %s! Thank you for joining OVASABI.", userResp.User.Username),
		Html:    true,
		Metadata: map[string]string{
			"user_id": fmt.Sprintf("%d", userResp.User.Id),
			"type":    "welcome_email",
		},
	}
	if _, err := p.notifyService.SendEmail(ctx, emailReq); err != nil {
		return nil, fmt.Errorf("failed to send welcome email: %w", err)
	}

	// Push notification
	pushReq := &notificationpb.SendPushNotificationRequest{
		UserId:  fmt.Sprintf("%d", userResp.User.Id),
		Title:   "Welcome to OVASABI",
		Message: fmt.Sprintf("Welcome %s! Your account has been created successfully.", userResp.User.Username),
		Metadata: map[string]string{
			"type": "welcome_push",
		},
	}
	if _, err := p.notifyService.SendPushNotification(ctx, pushReq); err != nil {
		return nil, fmt.Errorf("failed to send welcome push: %w", err)
	}

	// Step 4: Track pattern execution in knowledge graph
	patternInfo := map[string]interface{}{
		"user_id":    userResp.User.Id,
		"username":   userResp.User.Username,
		"email":      userResp.User.Email,
		"has_avatar": assetID != "",
		"avatar_id":  assetID,
		"status":     "completed",
		"operations": []string{
			"user_creation",
			"avatar_upload",
			"welcome_email",
			"welcome_push",
		},
	}

	if err := p.knowledgeGraph.AddPattern("user_creation", fmt.Sprintf("%d", userResp.User.Id), patternInfo); err != nil {
		return nil, fmt.Errorf("failed to track pattern: %w", err)
	}

	return map[string]interface{}{
		"status":    "success",
		"user_id":   userResp.User.Id,
		"username":  userResp.User.Username,
		"email":     userResp.User.Email,
		"avatar_id": assetID,
		"message":   "User created successfully with notifications sent",
	}, nil
}
