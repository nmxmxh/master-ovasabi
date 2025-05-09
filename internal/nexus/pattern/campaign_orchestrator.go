// Package pattern provides the CampaignOrchestrator, a generic orchestrator for campaign workflows.
//
// The CampaignOrchestrator enables dynamic campaign logic by reading campaign metadata (JSON) to determine
// which features to enable (waitlist, referral, leaderboard, i18n, broadcast, etc). This allows the platform
// to support many campaigns with different behaviors using a single proto/service interface.
//
// To add new campaign features (e.g., more translation fields, new gamification, analytics, etc):
//   1. Extend the campaignMeta struct and update parseCampaignMetadata.
//   2. Update orchestrator methods to handle new fields/logic as needed.
//   3. Document new metadata fields and their expected behavior here.
//
// Example campaign metadata (as JSON):
// {
//   "waitlist": true,
//   "referral": true,
//   "leaderboard": true,
//   "i18n_keys": ["welcome_banner", "signup_cta", "referral_message", "new_field1", "new_field2"],
//   "broadcast_enabled": true,
//   "custom_field": "value"
// }
//
// This pattern ensures scalability and maintainability as campaign complexity grows.

package pattern

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

type campaignMeta struct {
	Waitlist         bool     `json:"waitlist"`
	Referral         bool     `json:"referral"`
	Leaderboard      bool     `json:"leaderboard"`
	I18nKeys         []string `json:"i18n_keys"`
	BroadcastEnabled bool     `json:"broadcast_enabled"`
	// Add new fields here as campaign complexity increases
}

func parseCampaignMetadata(metaStr string) (campaignMeta, error) {
	var meta campaignMeta
	if err := json.Unmarshal([]byte(metaStr), &meta); err != nil {
		return meta, err
	}
	return meta, nil
}

type CampaignOrchestrator struct {
	UserService interface {
		Register(ctx context.Context, email, username string) (interface{}, error)
	}
	CampaignService interface {
		AddToWaitlist(ctx context.Context, userID int64, campaignSlug string) error
		UpdateLeaderboard(ctx context.Context, campaignSlug string) error
		GetLeaderboard(ctx context.Context, campaignSlug string, limit int) ([]interface{}, error)
		GetBySlug(ctx context.Context, slug string) (interface{ Metadata() string }, error)
	}
	ReferralService interface {
		RecordReferral(ctx context.Context, referrerUsername string, newUserID int64, campaignSlug string) error
	}
	I18nService interface {
		EnsureCampaignTranslations(ctx context.Context, campaignSlug, locale string, keys []string) error
	}
	BroadcastService interface {
		Broadcast(ctx context.Context, campaignSlug, message string) error
	}
}

func (o *CampaignOrchestrator) Signup(ctx context.Context, slug, email, username, referral, locale string) error {
	campaignIface, err := o.CampaignService.GetBySlug(ctx, slug)
	if err != nil {
		return fmt.Errorf("campaign not found: %w", err)
	}
	meta, err := parseCampaignMetadata(campaignIface.Metadata())
	if err != nil {
		return fmt.Errorf("invalid campaign metadata: %w", err)
	}
	userIface, err := o.UserService.Register(ctx, email, username)
	if err != nil {
		return fmt.Errorf("user registration failed: %w", err)
	}
	userIDField := userIface.(interface{ GetID() int64 })
	if meta.Waitlist {
		_ = o.CampaignService.AddToWaitlist(ctx, userIDField.GetID(), slug)
	}
	if meta.Referral && referral != "" {
		_ = o.ReferralService.RecordReferral(ctx, referral, userIDField.GetID(), slug)
		_ = o.CampaignService.UpdateLeaderboard(ctx, slug)
	}
	if len(meta.I18nKeys) > 0 {
		_ = o.I18nService.EnsureCampaignTranslations(ctx, slug, locale, meta.I18nKeys)
	}
	if meta.BroadcastEnabled {
		_ = o.BroadcastService.Broadcast(ctx, slug, fmt.Sprintf("New user joined: %s", username))
	}
	return nil
}

func (o *CampaignOrchestrator) SendBroadcast(ctx context.Context, slug, message string) error {
	campaignIface, err := o.CampaignService.GetBySlug(ctx, slug)
	if err != nil {
		return fmt.Errorf("campaign not found: %w", err)
	}
	meta, err := parseCampaignMetadata(campaignIface.Metadata())
	if err != nil {
		return fmt.Errorf("invalid campaign metadata: %w", err)
	}
	if !meta.BroadcastEnabled {
		return errors.New("broadcast not enabled for this campaign")
	}
	return o.BroadcastService.Broadcast(ctx, slug, message)
}

func (o *CampaignOrchestrator) GetReferralLeaderboard(ctx context.Context, slug string, limit int) ([]interface{}, error) {
	campaignIface, err := o.CampaignService.GetBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("campaign not found: %w", err)
	}
	meta, err := parseCampaignMetadata(campaignIface.Metadata())
	if err != nil {
		return nil, fmt.Errorf("invalid campaign metadata: %w", err)
	}
	if !meta.Leaderboard {
		return nil, errors.New("leaderboard not enabled for this campaign")
	}
	return o.CampaignService.GetLeaderboard(ctx, slug, limit)
}
