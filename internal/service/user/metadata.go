// Metadata Standard Reference
// --------------------------
// All service-specific metadata must include the `versioning` field as described in:
//   - docs/services/versioning.md
//   - docs/amadeus/amadeus_context.md
// For all available metadata actions, patterns, and service-specific extensions, see:
//   - docs/services/metadata.md (general metadata documentation)
//   - docs/services/versioning.md (versioning/environment standard)
//
// This file implements user service-specific metadata patterns. See above for required fields and integration points.
//
// Service-Specific Metadata Pattern for User Service
// -------------------------------------------------
//
// This file defines the canonical Go struct for all user service-specific metadata fields,
// covering all platform standards (bad actor, accessibility, compliance, audit, etc.).
//
// Usage:
// - Use ServiceMetadata to read/update all service-specific metadata fields in Go.
// - Use the provided helpers to convert between ServiceMetadata and structpb.Struct.
// - This pattern ensures robust, type-safe, and future-proof handling of service-specific metadata.
//
// Reference: docs/amadeus/amadeus_context.md#cross-service-standards-integration-path

package user

// ServiceMetadata holds all user service-specific metadata fields.
type Metadata struct {
	BadActor      *BadActorMetadata      `json:"bad_actor,omitempty"`
	Accessibility *AccessibilityMetadata `json:"accessibility,omitempty"`
	Compliance    *ComplianceMetadata    `json:"compliance,omitempty"`
	Audit         *AuditMetadata         `json:"audit,omitempty"`
	// Guest/anonymous user support
	Guest          bool   `json:"guest,omitempty"`            // True if this is a guest/anonymous user
	DeviceID       string `json:"device_id,omitempty"`        // Device hash or fingerprint for device-based auth
	GuestCreatedAt string `json:"guest_created_at,omitempty"` // Timestamp for guest user creation
	// Localization integration (see Amadeus context: Localization Service)
	Locale       string                `json:"locale,omitempty"`       // User's preferred locale (e.g., en-US)
	Language     string                `json:"language,omitempty"`     // User's preferred language (e.g., en)
	Region       string                `json:"region,omitempty"`       // User's region (e.g., US, FR)
	Timezone     string                `json:"timezone,omitempty"`     // User's timezone (e.g., Europe/Paris)
	Localization *LocalizationMetadata `json:"localization,omitempty"` // Nested localization metadata

	// --- New Authentication Channels (2024) ---
	// Email Verification & Password Reset (see: http://dvignesh1496.medium.com/email-verification-and-password-reset-flow-using-golang-c8bd037101e8)
	EmailVerified    bool               `json:"email_verified,omitempty"`    // True if email is verified
	VerificationData *VerificationData  `json:"verification_data,omitempty"` // Email verification code and expiry
	PasswordReset    *PasswordResetData `json:"password_reset,omitempty"`    // Password reset code and expiry

	// Passkey/WebAuthn (see: https://dev.to/egregors/passkey-in-go-1efk)
	Passkeys []WebAuthnCredential `json:"passkeys,omitempty"` // Registered passkeys (WebAuthn credentials)

	// Biometric Authentication (see: https://passage.1password.com/post/build-a-go-app-with-biometric-authentication)
	BiometricEnabled  bool   `json:"biometric_enabled,omitempty"`   // True if biometric auth is enabled
	BiometricLastUsed string `json:"biometric_last_used,omitempty"` // Timestamp of last biometric auth
	// -----------------------------------------
	// Add more as standards evolve
}

type BadActorMetadata struct {
	Score          float64            `json:"score,omitempty"`
	Reason         string             `json:"reason,omitempty"`
	DeviceIDs      []string           `json:"device_ids,omitempty"`
	Locations      []LocationMetadata `json:"locations,omitempty"`
	Frequency      *FrequencyMetadata `json:"frequency,omitempty"`
	AccountsLinked []string           `json:"accounts_linked,omitempty"`
	LastFlaggedAt  string             `json:"last_flagged_at,omitempty"`
	History        []EventMetadata    `json:"history,omitempty"`
}

type LocationMetadata struct {
	IP      string `json:"ip,omitempty"`
	City    string `json:"city,omitempty"`
	Country string `json:"country,omitempty"`
}

type FrequencyMetadata struct {
	Window string `json:"window,omitempty"`
	Count  int    `json:"count,omitempty"`
}

type EventMetadata struct {
	Event     string `json:"event,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
}

type AccessibilityMetadata struct {
	Features map[string]bool `json:"features,omitempty"`
	// Add more fields as needed (e.g., alt_text, captions, etc.)
}

type ComplianceMetadata struct {
	Standards []ComplianceStandard `json:"standards,omitempty"`
	CheckedBy string               `json:"checked_by,omitempty"`
	CheckedAt string               `json:"checked_at,omitempty"`
	Method    string               `json:"method,omitempty"`
	Issues    []ComplianceIssue    `json:"issues_found,omitempty"`
}

type ComplianceStandard struct {
	Name      string `json:"name,omitempty"`
	Level     string `json:"level,omitempty"`
	Version   string `json:"version,omitempty"`
	Compliant bool   `json:"compliant,omitempty"`
}

type ComplianceIssue struct {
	Type     string `json:"type,omitempty"`
	Location string `json:"location,omitempty"`
	Resolved bool   `json:"resolved,omitempty"`
}

type AuditMetadata struct {
	CreatedBy    string   `json:"created_by,omitempty"`
	History      []string `json:"history,omitempty"`
	LastModified string   `json:"last_modified,omitempty"`
}

// LocalizationMetadata holds localization and compliance info for the user.
type LocalizationMetadata struct {
	LastLocalizedAt string              `json:"last_localized_at,omitempty"` // Timestamp of last localization
	Compliance      *ComplianceMetadata `json:"compliance,omitempty"`        // Accessibility/compliance info
	// Add more fields as localization standards evolve
}

// VerificationData holds email verification code and expiry.
type VerificationData struct {
	Code      string `json:"code"`
	ExpiresAt string `json:"expires_at"`
}

// PasswordResetData holds password reset code and expiry.
type PasswordResetData struct {
	Code      string `json:"code"`
	ExpiresAt string `json:"expires_at"`
}

// WebAuthnCredential holds a registered passkey credential.
type WebAuthnCredential struct {
	CredentialID string   `json:"credential_id"`
	PublicKey    string   `json:"public_key"`
	Transports   []string `json:"transports"`
	CreatedAt    string   `json:"created_at"`
}

// --- External Provider Placeholders/Mocks ---

// EmailProvider defines the interface for sending emails (verification, password reset, etc.).
type EmailProvider interface {
	SendVerificationEmail(to, code string) error
	SendPasswordResetEmail(to, code string) error
}

// MockEmailProvider is a mock implementation for testing.
type MockEmailProvider struct{}

func (m *MockEmailProvider) SendVerificationEmail(_, _ string) error {
	// Mock: Log or store the email for test assertions
	return nil
}

func (m *MockEmailProvider) SendPasswordResetEmail(_, _ string) error {
	// Mock: Log or store the email for test assertions
	return nil
}

// WebAuthnProvider defines the interface for WebAuthn operations (passkey registration/login).
type WebAuthnProvider interface {
	BeginRegistration(username string) (challenge string, err error)
	FinishRegistration(username, response string) (credential WebAuthnCredential, err error)
	BeginLogin(username string) (challenge string, err error)
	FinishLogin(username, response string) (ok bool, err error)
}

// MockWebAuthnProvider is a mock implementation for testing.
type MockWebAuthnProvider struct{}

func (m *MockWebAuthnProvider) BeginRegistration(_ string) (string, error) {
	return "mock-challenge", nil
}

func (m *MockWebAuthnProvider) FinishRegistration(_, _ string) (WebAuthnCredential, error) {
	return WebAuthnCredential{CredentialID: "mock-id", PublicKey: "mock-key", Transports: []string{"usb"}, CreatedAt: "2025-01-01T00:00:00Z"}, nil
}

func (m *MockWebAuthnProvider) BeginLogin(_ string) (string, error) {
	return "mock-challenge", nil
}

func (m *MockWebAuthnProvider) FinishLogin(_, _ string) (bool, error) {
	return true, nil
}

// BiometricProvider defines the interface for biometric authentication (e.g., Passage).
type BiometricProvider interface {
	IsBiometricEnabled(userID string) (bool, error)
	MarkBiometricUsed(userID string) error
}

// MockBiometricProvider is a mock implementation for testing.
type MockBiometricProvider struct{}

func (m *MockBiometricProvider) IsBiometricEnabled(_ string) (bool, error) {
	return true, nil
}

func (m *MockBiometricProvider) MarkBiometricUsed(_ string) error {
	return nil
}

// [CANONICAL] All metadata must be normalized and calculated via metadata.NormalizeAndCalculate before persistence or emission.
// Ensure required fields (versioning, audit, etc.) are present under the correct namespace.
