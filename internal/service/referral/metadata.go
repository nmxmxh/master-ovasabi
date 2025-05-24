package referral

import (
	"errors"
	"fmt"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// Canonical keys for referral metadata (for onboarding, extensibility, and analytics).
const (
	ReferralMetaFraudSignals = "fraud_signals" // map[string]interface{}: device, location, frequency, etc.
	ReferralMetaRewards      = "rewards"       // map[string]interface{}: points, status, payout, etc.
	ReferralMetaAudit        = "audit"         // map[string]interface{}: created_by, timestamps, etc.
	ReferralMetaCampaign     = "campaign"      // map[string]interface{}: campaign-specific info
	ReferralMetaDevice       = "device"        // map[string]interface{}: device_hash, fingerprint, etc.
	ReferralMetaService      = "referral"      // service_specific.referral namespace
)

// BuildReferralMetadata builds a canonical referral metadata struct for storage and analytics.
func BuildReferralMetadata(fraudSignals, rewards, audit, campaign, device map[string]interface{}) (*commonpb.Metadata, error) {
	referralMap := map[string]interface{}{}
	if fraudSignals != nil {
		referralMap[ReferralMetaFraudSignals] = fraudSignals
	}
	if rewards != nil {
		referralMap[ReferralMetaRewards] = rewards
	}
	if audit != nil {
		referralMap[ReferralMetaAudit] = audit
	}
	if campaign != nil {
		referralMap[ReferralMetaCampaign] = campaign
	}
	if device != nil {
		referralMap[ReferralMetaDevice] = device
	}
	ss := map[string]interface{}{ReferralMetaService: referralMap}
	ssStruct, err := structpb.NewStruct(ss)
	if err != nil {
		return nil, fmt.Errorf("failed to build service_specific struct: %w", err)
	}
	return &commonpb.Metadata{ServiceSpecific: ssStruct}, nil
}

// ValidateReferralMetadata ensures required fields are present and well-formed.
func ValidateReferralMetadata(meta *commonpb.Metadata) error {
	if meta == nil || meta.ServiceSpecific == nil {
		return errors.New("missing service_specific metadata")
	}
	fields := meta.ServiceSpecific.Fields
	referralVal, ok := fields[ReferralMetaService]
	if !ok || referralVal.GetStructValue() == nil {
		return errors.New("missing referral namespace in service_specific")
	}
	referralMap := referralVal.GetStructValue().AsMap()
	// Example: require at least fraud_signals and audit for compliance
	if _, ok := referralMap[ReferralMetaFraudSignals]; !ok {
		return errors.New("missing fraud_signals in referral metadata")
	}
	if _, ok := referralMap[ReferralMetaAudit]; !ok {
		return errors.New("missing audit in referral metadata")
	}
	return nil
}

// Example: Extract fraud signals from referral metadata.
func GetFraudSignals(meta *commonpb.Metadata) (map[string]interface{}, error) {
	if meta == nil || meta.ServiceSpecific == nil {
		return nil, errors.New("missing service_specific metadata")
	}
	fields := meta.ServiceSpecific.Fields
	referralVal, ok := fields[ReferralMetaService]
	if !ok || referralVal.GetStructValue() == nil {
		return nil, errors.New("missing referral namespace in service_specific")
	}
	referralMap := referralVal.GetStructValue().AsMap()
	if fs, ok := referralMap[ReferralMetaFraudSignals].(map[string]interface{}); ok {
		return fs, nil
	}
	return nil, errors.New("fraud_signals not found or invalid")
}

// Add more helpers as needed for rewards, audit, campaign, etc.
