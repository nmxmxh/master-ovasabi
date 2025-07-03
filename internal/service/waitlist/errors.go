package waitlist

import (
	"errors"
)

// Error definitions for the waitlist service
var (
	// Input validation errors
	ErrInvalidEmail      = errors.New("invalid email address")
	ErrInvalidTier       = errors.New("invalid tier, must be one of: talent, pioneer, hustlers, business")
	ErrInvalidStatus     = errors.New("invalid status, must be one of: pending, approved, rejected, invited")
	ErrEmailRequired     = errors.New("email is required")
	ErrFirstNameRequired = errors.New("first name is required")
	ErrLastNameRequired  = errors.New("last name is required")
	ErrTierRequired      = errors.New("tier is required")
	ErrIntentionRequired = errors.New("intention is required")

	// Business logic errors
	ErrEmailAlreadyExists    = errors.New("email already exists in waitlist")
	ErrUsernameAlreadyTaken  = errors.New("username is already reserved")
	ErrInvalidReferralCode   = errors.New("invalid referral code")
	ErrWaitlistEntryNotFound = errors.New("waitlist entry not found")
	ErrAlreadyInvited        = errors.New("user already invited")
	ErrCannotUpdateInvited   = errors.New("cannot update invited user")

	// System errors
	ErrDatabaseConnection = errors.New("database connection error")
	ErrInternalServer     = errors.New("internal server error")

	// Campaign-specific errors
	ErrReferralUserNotFound    = errors.New("referral username not found")
	ErrSelfReferralNotAllowed  = errors.New("self-referral not allowed")
	ErrReferralAlreadyExists   = errors.New("referral already exists")
	ErrInvalidLocation         = errors.New("invalid location data")
	ErrMissingRequiredFields   = errors.New("missing required fields")
	ErrInvalidEmailFormat      = errors.New("invalid email format")
	ErrInvalidUsernameLength   = errors.New("username must be between 3 and 30 characters")
	ErrInvalidCampaign         = errors.New("invalid campaign name")
	ErrInvalidReferrerUsername = errors.New("invalid referrer username")
	ErrInvalidReferredID       = errors.New("invalid referred user ID")
)

// Valid tiers
var ValidTiers = map[string]bool{
	"talent":   true,
	"pioneer":  true,
	"hustlers": true,
	"business": true,
}

// Valid statuses
var ValidStatuses = map[string]bool{
	"pending":  true,
	"approved": true,
	"rejected": true,
	"invited":  true,
}

// Tier priorities for waitlist ordering (higher = higher priority)
var TierPriorities = map[string]int{
	"business": 400,
	"hustlers": 300,
	"pioneer":  200,
	"talent":   100,
}

// Campaign constants
const (
	DefaultCampaignName   = "ovasabi-website-launch"
	DefaultReferralPoints = 10
	ReferralPriorityBonus = 5 // Half of referral points added to priority
)

// Referral types
var ValidReferralTypes = map[string]bool{
	"username": true,
	"code":     true,
	"link":     true,
}
