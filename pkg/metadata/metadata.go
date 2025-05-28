package metadata

import (
	"encoding/json"

	"google.golang.org/protobuf/types/known/structpb"
)

type ServiceMetadata struct {
	MFAChallenge      *MFAChallengeData    `json:"mfa_challenge,omitempty"`
	Guest             bool                 `json:"guest,omitempty"`
	GuestCreatedAt    string               `json:"guest_created_at,omitempty"`
	DeviceID          string               `json:"device_id,omitempty"`
	Audit             *AuditMetadata       `json:"audit,omitempty"`
	VerificationData  *VerificationData    `json:"verification_data,omitempty"`
	EmailVerified     bool                 `json:"email_verified,omitempty"`
	PasswordReset     *PasswordResetData   `json:"password_reset,omitempty"`
	Passkeys          []WebAuthnCredential `json:"passkeys,omitempty"`
	BiometricLastUsed string               `json:"biometric_last_used,omitempty"`
	WebAuthnChallenge string               `json:"web_authn_challenge,omitempty"`
}

type MFAChallengeData struct {
	Code        string `json:"code"`
	ChallengeID string `json:"challenge_id"`
	ExpiresAt   string `json:"expires_at"`
}

type AuditMetadata struct {
	LastModified string   `json:"last_modified,omitempty"`
	History      []string `json:"history,omitempty"`
}

type VerificationData struct {
	Code      string `json:"code,omitempty"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

type PasswordResetData struct {
	Code      string `json:"code,omitempty"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

type WebAuthnCredential struct {
	ID        string `json:"id,omitempty"`
	PublicKey string `json:"public_key,omitempty"`
	// Add other fields as needed
}

// ServiceMetadataFromStruct converts a structpb.Struct to *ServiceMetadata.
func ServiceMetadataFromStruct(s *structpb.Struct) (*ServiceMetadata, error) {
	if s == nil {
		return &ServiceMetadata{}, nil
	}
	b, err := s.MarshalJSON()
	if err != nil {
		return nil, err
	}
	var meta ServiceMetadata
	if err := json.Unmarshal(b, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

// ServiceMetadataToStruct converts a *ServiceMetadata to structpb.Struct.
func ServiceMetadataToStruct(meta *ServiceMetadata) (*structpb.Struct, error) {
	if meta == nil {
		return NewStructFromMap(nil), nil
	}
	b, err := json.Marshal(meta)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return NewStructFromMap(m), nil
}
