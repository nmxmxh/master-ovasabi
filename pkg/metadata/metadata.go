package metadata

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"google.golang.org/protobuf/encoding/protojson"
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

// MetadataHandler is the canonical handler for all metadata operations (creation, chaining, idempotency, calculation, search).
type MetadataHandler struct{}

// DefaultMetadata returns a canonical metadata map with all required fields initialized.
func (MetadataHandler) DefaultMetadata() map[string]interface{} {
	now := time.Now().UTC().Format(time.RFC3339)
	return map[string]interface{}{
		"updated_at":        now,
		"version":           1,
		"prev_state_id":     "",
		"next_state_id":     "",
		"related_state_ids": []string{},
		"calculation":       map[string]interface{}{},
		"scheduling":        map[string]interface{}{},
		"features":          []string{},
		"custom_rules":      map[string]interface{}{},
		"audit":             map[string]interface{}{},
		"tags":              []string{},
		"service_specific":  map[string]interface{}{},
		"knowledge_graph":   map[string]interface{}{},
	}
}

// GenerateIdempotentKey generates a unique, idempotent key for a metadata instance based on its normalized content and context.
func (MetadataHandler) GenerateIdempotentKey(meta map[string]interface{}) string {
	// Normalize: sort keys, flatten, and hash
	var keys []string
	for k := range meta {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, "%s=%v;", k, meta[k])
	}
	h := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(h[:])
}

// SetChainLinks sets prev, next, and related state ids in metadata.
func (MetadataHandler) SetChainLinks(meta map[string]interface{}, prev, next string, related []string) {
	meta["prev_state_id"] = prev
	meta["next_state_id"] = next
	meta["related_state_ids"] = related
}

// GetChainLinks retrieves prev, next, and related state ids from metadata.
func (MetadataHandler) GetChainLinks(meta map[string]interface{}) (prev, next string, related []string) {
	prev, _ = meta["prev_state_id"].(string)
	next, _ = meta["next_state_id"].(string)
	if rel, ok := meta["related_state_ids"].([]string); ok {
		related = rel
	} else if rel, ok := meta["related_state_ids"].([]interface{}); ok {
		for _, v := range rel {
			if s, ok := v.(string); ok {
				related = append(related, s)
			}
		}
	}
	return
}

// GrepMetadata searches for a field or value in metadata and returns matching keys/values.
func (MetadataHandler) GrepMetadata(meta map[string]interface{}, query string) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range meta {
		if strings.Contains(k, query) || strings.Contains(fmt.Sprintf("%v", v), query) {
			result[k] = v
		}
	}
	return result
}

// UpdateCalculation updates the calculation field in metadata (e.g., score, tax, etc.).
func (MetadataHandler) UpdateCalculation(meta map[string]interface{}, calc map[string]interface{}) {
	meta["calculation"] = calc
	meta["updated_at"] = time.Now().UTC().Format(time.RFC3339)
}

// AddScore adds or updates a score in the calculation field.
func (MetadataHandler) AddScore(meta map[string]interface{}, score float64) {
	calc, ok := meta["calculation"].(map[string]interface{})
	if !ok {
		calc = map[string]interface{}{}
	}
	calc["score"] = score
	meta["calculation"] = calc
	meta["updated_at"] = time.Now().UTC().Format(time.RFC3339)
}

// AddTax adds or updates a tax value in the calculation field.
func (MetadataHandler) AddTax(meta map[string]interface{}, tax float64) {
	calc, ok := meta["calculation"].(map[string]interface{})
	if !ok {
		calc = map[string]interface{}{}
	}
	calc["tax"] = tax
	meta["calculation"] = calc
	meta["updated_at"] = time.Now().UTC().Format(time.RFC3339)
}

// SetAvailableBalance sets the available balance in the calculation field.
func (MetadataHandler) SetAvailableBalance(meta map[string]interface{}, balance float64) {
	calc, ok := meta["calculation"].(map[string]interface{})
	if !ok {
		calc = map[string]interface{}{}
	}
	calc["available_balance"] = balance
	meta["calculation"] = calc
	meta["updated_at"] = time.Now().UTC().Format(time.RFC3339)
}

// SetPending sets the pending value in the calculation field.
func (MetadataHandler) SetPending(meta map[string]interface{}, pending float64) {
	calc, ok := meta["calculation"].(map[string]interface{})
	if !ok {
		calc = map[string]interface{}{}
	}
	calc["pending"] = pending
	meta["calculation"] = calc
	meta["updated_at"] = time.Now().UTC().Format(time.RFC3339)
}

// TransferOwnership updates the owner, audit, prev_state_id, and updated_at fields, and returns the new idempotent key.
func (h MetadataHandler) TransferOwnership(meta map[string]interface{}, newOwner string, prevMetaID string) string {
	// Update owner
	meta["owner"] = newOwner
	// Update audit/lineage
	h.AppendAudit(meta, map[string]interface{}{
		"action":    "transfer_ownership",
		"to":        newOwner,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
	// Chain prev_state_id
	meta["prev_state_id"] = prevMetaID
	meta["updated_at"] = time.Now().UTC().Format(time.RFC3339)
	return h.GenerateIdempotentKey(meta)
}

// AppendAudit appends an entry to the audit or lineage field in metadata.
func (MetadataHandler) AppendAudit(meta map[string]interface{}, entry map[string]interface{}) {
	audit, ok := meta["audit"].([]interface{})
	if !ok {
		audit = []interface{}{}
	}
	audit = append(audit, entry)
	meta["audit"] = audit
	meta["updated_at"] = time.Now().UTC().Format(time.RFC3339)
}

// Example usage:
// handler := MetadataHandler{}
// meta := handler.DefaultMetadata()
// handler.SetChainLinks(meta, "prev_id", "next_id", []string{"rel1", "rel2"})
// key := handler.GenerateIdempotentKey(meta)
// handler.AddScore(meta, 99.5)
// handler.AddTax(meta, 0.15)
// matches := handler.GrepMetadata(meta, "score")

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
		return NewStructFromMap(nil, nil), nil
	}
	b, err := json.Marshal(meta)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return NewStructFromMap(m, nil), nil
}

// ProtoToMap converts a *commonpb.Metadata proto to a map[string]interface{} for use with MetadataHandler.
func ProtoToMap(meta *commonpb.Metadata) map[string]interface{} {
	if meta == nil {
		return MetadataHandler{}.DefaultMetadata()
	}
	b, err := protojson.Marshal(meta)
	if err != nil {
		return MetadataHandler{}.DefaultMetadata()
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return MetadataHandler{}.DefaultMetadata()
	}
	if _, ok := m["updated_at"]; !ok {
		m["updated_at"] = time.Now().UTC().Format(time.RFC3339)
	}
	return m
}

// MapToProto converts a map[string]interface{} (from MetadataHandler) to a *commonpb.Metadata proto.
func MapToProto(m map[string]interface{}) *commonpb.Metadata {
	if m == nil {
		return &commonpb.Metadata{}
	}
	b, err := json.Marshal(m)
	if err != nil {
		return &commonpb.Metadata{}
	}
	var meta commonpb.Metadata
	if err := protojson.Unmarshal(b, &meta); err != nil {
		return &commonpb.Metadata{}
	}
	return &meta
}

// StructToMap converts a *structpb.Struct to map[string]interface{}.
func StructToMap(s *structpb.Struct) map[string]interface{} {
	if s == nil {
		return map[string]interface{}{}
	}
	return s.AsMap()
}

// MapToStruct converts a map[string]interface{} to *structpb.Struct.
func MapToStruct(m map[string]interface{}) *structpb.Struct {
	s, err := structpb.NewStruct(m)
	if err != nil {
		return &structpb.Struct{Fields: map[string]*structpb.Value{}}
	}
	return s
}

// ProtoToStruct converts a *commonpb.Metadata proto to *structpb.Struct (for storage as jsonb or for gRPC).
func ProtoToStruct(meta *commonpb.Metadata) *structpb.Struct {
	m := ProtoToMap(meta)
	return MapToStruct(m)
}

// StructToProto converts a *structpb.Struct to *commonpb.Metadata proto.
func StructToProto(s *structpb.Struct) *commonpb.Metadata {
	m := StructToMap(s)
	return MapToProto(m)
}

// MapToJSON marshals a map[string]interface{} to JSON bytes.
func MapToJSON(m map[string]interface{}) ([]byte, error) {
	return json.Marshal(m)
}

// JSONToMap unmarshals JSON bytes to map[string]interface{}.
func JSONToMap(b []byte) (map[string]interface{}, error) {
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}
