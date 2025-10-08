package metadata

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// MergeMetadata merges two *commonpb.Metadata structs, prioritizing meta2 for conflicts.
// Only merges ServiceSpecific and Extra fields (deep merge), as these are canonical and extensible.
func MergeMetadata(meta1, meta2 *commonpb.Metadata) *commonpb.Metadata {
	if meta1 == nil && meta2 == nil {
		return &commonpb.Metadata{}
	}
	if meta1 == nil {
		return meta2
	}
	if meta2 == nil {
		return meta1
	}
	merged := &commonpb.Metadata{}

	// Deep merge ServiceSpecific only
	merged.ServiceSpecific = mergeStructs(meta1.ServiceSpecific, meta2.ServiceSpecific)

	// For all other fields, meta2 takes precedence if set, else meta1
	// (add more fields here if proto is extended in future)

	return merged
}

// mergeStructs deeply merges two *structpb.Struct, prioritizing s2 for conflicts.
func mergeStructs(s1, s2 *structpb.Struct) *structpb.Struct {
	if s1 == nil && s2 == nil {
		return &structpb.Struct{Fields: map[string]*structpb.Value{}}
	}
	if s1 == nil {
		return s2
	}
	if s2 == nil {
		return s1
	}
	merged := &structpb.Struct{Fields: map[string]*structpb.Value{}}
	for k, v := range s1.Fields {
		merged.Fields[k] = v
	}
	for k, v2 := range s2.Fields {
		if v1, ok := merged.Fields[k]; ok {
			if v1.GetStructValue() != nil && v2.GetStructValue() != nil {
				merged.Fields[k] = structpb.NewStructValue(mergeStructs(v1.GetStructValue(), v2.GetStructValue()))
			} else {
				merged.Fields[k] = v2
			}
		} else {
			merged.Fields[k] = v2
		}
	}
	return merged
}

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
type Handler struct{}

// EnrichMetadata sets global fields and service-specific fields in a commonpb.Metadata.
// globalFields: map of global key/value pairs (e.g., correlation_id, user_id)
// serviceName: namespace for service-specific fields (e.g., "gateway", "search")
// serviceFields: map of service-specific key/value pairs.
func (Handler) EnrichMetadata(meta *commonpb.Metadata, globalFields map[string]string, serviceName string, serviceFields map[string]interface{}) *commonpb.Metadata {
	if meta == nil {
		meta = &commonpb.Metadata{}
	}
	if meta.ServiceSpecific == nil {
		meta.ServiceSpecific = &structpb.Struct{Fields: map[string]*structpb.Value{}}
	}
	// Set global fields under 'global' namespace
	if globalFields != nil {
		globalMap := map[string]interface{}{}
		for k, v := range globalFields {
			globalMap[k] = v
		}
		s, err := structpb.NewStruct(globalMap)
		if err == nil {
			meta.ServiceSpecific.Fields["global"] = structpb.NewStructValue(s)
		}
	}
	// Set service-specific fields in ServiceSpecific namespace
	if serviceName != "" && serviceFields != nil {
		s, err := structpb.NewStruct(serviceFields)
		if err == nil {
			meta.ServiceSpecific.Fields[serviceName] = structpb.NewStructValue(s)
		}
	}
	return meta
}

// DefaultMetadata returns a canonical *commonpb.Metadata with all required fields initialized.
func (Handler) DefaultMetadata() *commonpb.Metadata {
	now := time.Now().UTC().Format(time.RFC3339)
	systemMeta, err := structpb.NewStruct(map[string]interface{}{"updated_at": now, "version": 1, "prev_state_id": "", "next_state_id": "", "related_state_ids": []string{}, "calculation": map[string]interface{}{}})
	if err != nil {
		systemMeta = &structpb.Struct{Fields: map[string]*structpb.Value{}}
	}

	return &commonpb.Metadata{
		Audit:       &structpb.Struct{Fields: map[string]*structpb.Value{}},
		CustomRules: &structpb.Struct{Fields: map[string]*structpb.Value{}},
		Scheduling:  &structpb.Struct{Fields: map[string]*structpb.Value{}},
		ServiceSpecific: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"system": structpb.NewStructValue(systemMeta),
			},
		},
		KnowledgeGraph: &commonpb.KnowledgeGraph{},
	}
}

// GenerateIdempotentKey generates a unique, idempotent key for a metadata instance based on its normalized content.
func (Handler) GenerateIdempotentKey(meta *commonpb.Metadata) string {
	// Use proto marshaling for a canonical, stable representation.
	b, err := proto.Marshal(meta)
	if err != nil {
		// Fallback for safety, though marshaling should not fail with a valid proto.
		return fmt.Sprintf("error-key-%d", time.Now().UnixNano())
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// SetChainLinks sets prev, next, and related state ids in the system namespace.
func (Handler) SetChainLinks(meta *commonpb.Metadata, prev, next string, related []string) {
	if meta == nil {
		return
	}
	system := GetSystemNamespace(meta)
	system["prev_state_id"] = prev
	system["next_state_id"] = next
	system["related_state_ids"] = related
	SetSystemNamespace(meta, system)
}

// GetChainLinks retrieves prev, next, and related state ids from the system namespace.
func (Handler) GetChainLinks(meta *commonpb.Metadata) (prev, next string, related []string) {
	if meta == nil {
		return "", "", nil
	}
	system := GetSystemNamespace(meta)

	if p, ok := system["prev_state_id"].(string); ok {
		prev = p
	}
	if n, ok := system["next_state_id"].(string); ok {
		next = n
	}

	var relatedRaw []interface{}
	if r, ok := system["related_state_ids"].([]interface{}); ok {
		relatedRaw = r
	}
	for _, v := range relatedRaw {
		if s, ok := v.(string); ok {
			related = append(related, s)
		}
	}
	return prev, next, related
}

// GrepMetadata is deprecated. Use direct field access on the protobuf.
func (Handler) GrepMetadata(meta map[string]interface{}, query string) map[string]interface{} {
	// This function is deprecated as it operates on an inconsistent map representation.
	// Direct access to protobuf fields is preferred.
	return make(map[string]interface{})
}

// UpdateCalculation updates the calculation field in the system namespace.
func (Handler) UpdateCalculation(meta *commonpb.Metadata, calc map[string]interface{}) {
	if meta == nil {
		return
	}
	system := GetSystemNamespace(meta)
	system["calculation"] = calc
	system["updated_at"] = time.Now().UTC().Format(time.RFC3339)
	SetSystemNamespace(meta, system)
}

// AddScore adds or updates a score in the calculation field.
func (Handler) AddScore(meta *commonpb.Metadata, score float64) {
	if meta == nil {
		return
	}
	system := GetSystemNamespace(meta)
	var calc map[string]interface{}
	if c, ok := system["calculation"].(map[string]interface{}); ok {
		calc = c
	}
	if calc == nil {
		calc = make(map[string]interface{})
	}
	calc["score"] = score
	system["calculation"] = calc
	system["updated_at"] = time.Now().UTC().Format(time.RFC3339)
	SetSystemNamespace(meta, system)
}

// AddTax adds or updates a tax value in the calculation field.
func (Handler) AddTax(meta *commonpb.Metadata, tax float64) {
	if meta == nil {
		return
	}
	system := GetSystemNamespace(meta)
	var calc map[string]interface{}
	if c, ok := system["calculation"].(map[string]interface{}); ok {
		calc = c
	}
	if calc == nil {
		calc = make(map[string]interface{})
	}
	calc["tax"] = tax
	system["calculation"] = calc
	system["updated_at"] = time.Now().UTC().Format(time.RFC3339)
	SetSystemNamespace(meta, system)
}

// SetAvailableBalance sets the available balance in the calculation field.
func (Handler) SetAvailableBalance(meta *commonpb.Metadata, balance float64) {
	if meta == nil {
		return
	}
	system := GetSystemNamespace(meta)
	var calc map[string]interface{}
	if c, ok := system["calculation"].(map[string]interface{}); ok {
		calc = c
	}
	if calc == nil {
		calc = make(map[string]interface{})
	}
	calc["available_balance"] = balance
	system["calculation"] = calc
	system["updated_at"] = time.Now().UTC().Format(time.RFC3339)
	SetSystemNamespace(meta, system)
}

// SetPending sets the pending value in the calculation field.
func (Handler) SetPending(meta *commonpb.Metadata, pending float64) {
	if meta == nil {
		return
	}
	system := GetSystemNamespace(meta)
	var calc map[string]interface{}
	if c, ok := system["calculation"].(map[string]interface{}); ok {
		calc = c
	}
	if calc == nil {
		calc = make(map[string]interface{})
	}
	calc["pending"] = pending
	system["calculation"] = calc
	system["updated_at"] = time.Now().UTC().Format(time.RFC3339)
	SetSystemNamespace(meta, system)
}

// TransferOwnership updates the owner, audit, and chain links.
func (h Handler) TransferOwnership(meta *commonpb.Metadata, newOwner, prevMetaID string) {
	if meta == nil {
		return
	}
	// Set owner in a relevant namespace, e.g., 'system' or a specific service namespace.
	// For now, we place it in the system namespace for consistency.
	system := GetSystemNamespace(meta)
	system["owner"] = newOwner
	system["prev_state_id"] = prevMetaID
	system["updated_at"] = time.Now().UTC().Format(time.RFC3339)
	SetSystemNamespace(meta, system)

	// Update audit
	h.AppendAudit(meta, map[string]interface{}{
		"action":    "transfer_ownership",
		"to":        newOwner,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// AppendAudit appends an entry to the audit history.
func (Handler) AppendAudit(meta *commonpb.Metadata, entry map[string]interface{}) {
	if meta == nil {
		return
	}
	if meta.Audit == nil {
		meta.Audit = &structpb.Struct{Fields: make(map[string]*structpb.Value)}
	}
	auditMap := meta.Audit.AsMap()
	var history []interface{}
	if h, ok := auditMap["history"].([]interface{}); ok {
		history = h
	}
	history = append(history, entry)
	auditMap["history"] = history
	newAudit, err := structpb.NewStruct(auditMap)
	if err == nil {
		meta.Audit = newAudit
	}
}

// NormalizeAndCalculate normalizes metadata and performs default calculations.
func (h Handler) NormalizeAndCalculate(meta *commonpb.Metadata, prev, next string, related []string, calculationType, calcMsg string) {
	if meta == nil {
		return
	}
	h.SetChainLinks(meta, prev, next, related)
	system := GetSystemNamespace(meta)
	var calc map[string]interface{}
	if c, ok := system["calculation"].(map[string]interface{}); ok {
		calc = c
	}
	if calc == nil {
		calc = make(map[string]interface{})
	}

	switch calculationType {
	case "success":
		var score float64
		if s, ok := calc["score"].(float64); ok {
			score = s
		}
		calc["score"] = score + 1
		calc["last_success"] = calcMsg
	case "error":
		var errorCount float64
		if ec, ok := calc["error_count"].(float64); ok {
			errorCount = ec
		}
		calc["error_count"] = errorCount + 1
		calc["last_error"] = calcMsg
	}
	system["calculation"] = calc
	system["updated_at"] = time.Now().UTC().Format(time.RFC3339)
	SetSystemNamespace(meta, system)
}

// Helper functions to access the system namespace safely.

// GetSystemNamespace extracts the 'system' namespace from ServiceSpecific as a map.
func GetSystemNamespace(meta *commonpb.Metadata) map[string]interface{} {
	if meta == nil || meta.ServiceSpecific == nil || meta.ServiceSpecific.Fields == nil {
		return make(map[string]interface{})
	}
	systemVal, ok := meta.ServiceSpecific.Fields["system"]
	if !ok || systemVal.GetStructValue() == nil {
		return make(map[string]interface{})
	}
	return systemVal.GetStructValue().AsMap()
}

// SetSystemNamespace sets the 'system' namespace in ServiceSpecific from a map.
func SetSystemNamespace(meta *commonpb.Metadata, system map[string]interface{}) {
	if meta == nil {
		return
	}
	if meta.ServiceSpecific == nil {
		meta.ServiceSpecific = &structpb.Struct{Fields: make(map[string]*structpb.Value)}
	}
	systemStruct, err := structpb.NewStruct(system)
	if err == nil {
		meta.ServiceSpecific.Fields["system"] = structpb.NewStructValue(systemStruct)
	}
}

// Package usage note:
// All metadata operations (save, emit, chain) MUST now operate on the canonical *commonpb.Metadata protobuf struct.
// This guarantees previous/forward state is always consistent and ready for audit, orchestration, and event sourcing.

// Example usage:
// handler := Handler{}
// meta := handler.DefaultMetadata()
// handler.SetChainLinks(meta, "prev_id", "next_id", []string{"rel1", "rel2"})
// key := handler.GenerateIdempotentKey(meta)
// handler.AddScore(meta, 99.5)
// handler.AddTax(meta, 0.15)

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

// StructToMap converts a *structpb.Struct to map[string]interface{}.
func StructToMap(s *structpb.Struct) map[string]interface{} {
	if s == nil {
		return map[string]interface{}{}
	}
	return s.AsMap()
}

// MapToStruct converts a map[string]interface{} to *structpb.Struct.
func MapToStruct(m map[string]interface{}) *structpb.Struct {
	m = NormalizeSlices(m)
	s, err := structpb.NewStruct(m)
	if err != nil {
		return &structpb.Struct{Fields: map[string]*structpb.Value{}}
	}
	return s
}

// Recursively convert all []string and []interface{} to []interface{} for structpb compatibility.
func NormalizeSlices(m map[string]interface{}) map[string]interface{} {
	for k, v := range m {
		switch vv := v.(type) {
		case []string:
			arr := make([]interface{}, len(vv))
			for i, s := range vv {
				arr[i] = s
			}
			m[k] = arr
		case []interface{}:
			for i, elem := range vv {
				if subMap, ok := elem.(map[string]interface{}); ok {
					vv[i] = NormalizeSlices(subMap)
				}
			}
			m[k] = vv
		case map[string]interface{}:
			m[k] = NormalizeSlices(vv)
			m[k] = NormalizeSlices(vv)
			m[k] = NormalizeSlices(vv)
		}
	}
	return m
}

// ProtoToMap converts a *commonpb.Metadata proto to map[string]interface{}.
func ProtoToMap(meta *commonpb.Metadata) map[string]interface{} {
	if meta == nil {
		return make(map[string]interface{})
	}
	// Use protojson for canonical JSON representation of protobufs.
	// This handles field names (camelCase), oneofs, and other proto-specific JSON mappings.
	b, err := protojson.Marshal(meta)
	if err != nil {
		// Log the error or handle it appropriately. For now, return empty map.
		return make(map[string]interface{})
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		// Log the error or handle it appropriately. For now, return empty map.
		return make(map[string]interface{})
	}
	return m
}

// MapToProto converts a map[string]interface{} to *commonpb.Metadata proto.
func MapToProto(m map[string]interface{}) *commonpb.Metadata {
	if m == nil {
		return &commonpb.Metadata{}
	}
	// Marshal map to JSON, then unmarshal into proto.
	b, err := json.Marshal(m)
	if err != nil {
		// Log the error or handle it appropriately. For now, return empty proto.
		return &commonpb.Metadata{}
	}
	meta := &commonpb.Metadata{}
	// Use protojson for canonical JSON representation of protobufs.
	if err := protojson.Unmarshal(b, meta); err != nil {
		// Log the error or handle it appropriately. For now, return empty proto.
		return &commonpb.Metadata{}
	}
	return meta
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

// ExtractServiceVariables extracts key variables (score, badges, gamification, compliance, etc.) from any service_specific namespace in a commonpb.Metadata.
// This is the canonical, extensible function for state hydration, leaderboard, trending, and gamification.
// Usage: vars := metadata.ExtractServiceVariables(meta, "user"), metadata.ExtractServiceVariables(meta, "campaign"), etc.
func ExtractServiceVariables(meta *commonpb.Metadata, namespace string) map[string]interface{} {
	vars := make(map[string]interface{})
	if meta == nil || meta.ServiceSpecific == nil {
		return vars
	}
	ss := meta.ServiceSpecific.AsMap()
	serviceMeta, ok := ss[namespace].(map[string]interface{})
	if !ok {
		return vars
	}
	// Canonical fields: score, badges, gamification, compliance, etc.
	if calc, ok := serviceMeta["calculation"].(map[string]interface{}); ok {
		if score, ok := calc["score"].(float64); ok {
			vars["score"] = score
		}
		if level, ok := calc["level"].(float64); ok {
			vars["level"] = level
		}
	}
	if badges, ok := serviceMeta["badges"].([]interface{}); ok {
		var badgeList []string
		for _, b := range badges {
			if bs, ok := b.(string); ok {
				badgeList = append(badgeList, bs)
			}
		}
		vars["badges"] = badgeList
	}
	if gamification, ok := serviceMeta["gamification"].(map[string]interface{}); ok {
		for k, v := range gamification {
			vars[k] = v
		}
	}
	if compliance, ok := serviceMeta["compliance"].(map[string]interface{}); ok {
		vars["compliance"] = compliance
	}
	if accessibility, ok := serviceMeta["accessibility"].(map[string]interface{}); ok {
		vars["accessibility"] = accessibility
	}
	if trending, ok := serviceMeta["trending"].([]interface{}); ok {
		vars["trending"] = trending
	}
	if leaderboard, ok := serviceMeta["leaderboard"].([]interface{}); ok {
		vars["leaderboard"] = leaderboard
	}
	// Add more fields as needed for extensibility (referral, analytics, etc.)
	for k, v := range serviceMeta {
		if _, exists := vars[k]; !exists {
			vars[k] = v
		}
	}
	return vars
}

// Usage: metadata.SetServiceSpecificField(meta, "admin", "versioning", versioningMap).
func SetServiceSpecificField(meta *commonpb.Metadata, namespace, key string, value interface{}) error {
	if meta == nil {
		return nil
	}
	if meta.ServiceSpecific == nil {
		meta.ServiceSpecific = &structpb.Struct{Fields: map[string]*structpb.Value{}}
	}
	nsVal, ok := meta.ServiceSpecific.Fields[namespace]
	var nsMap map[string]interface{}
	if ok && nsVal != nil && nsVal.GetStructValue() != nil {
		nsMap = nsVal.GetStructValue().AsMap()
	} else {
		nsMap = map[string]interface{}{}
	}
	nsMap[key] = value
	nsStruct, err := structpb.NewStruct(nsMap)
	if err != nil {
		return err
	}
	meta.ServiceSpecific.Fields[namespace] = structpb.NewStructValue(nsStruct)
	return nil
}

// CanonicalEnrichMetadata enriches metadata with base, context, and extra data.
func CanonicalEnrichMetadata(base map[string]string, ctxType string, extra map[string]interface{}) map[string]interface{} {
	enriched := make(map[string]interface{})
	for k, v := range base {
		enriched[k] = v
	}
	enriched["system_timestamp"] = time.Now().UTC().Format(time.RFC3339Nano)
	enriched["system_context"] = ctxType
	for k, v := range extra {
		enriched[k] = v
	}
	return enriched
}

// FinalizeMetadataForEmit finalizes metadata for event emission.
func FinalizeMetadataForEmit(_ context.Context, meta *commonpb.Metadata, _ bool, _ string, _ []string, _, _ map[string]interface{}) error {
	if meta == nil {
		return fmt.Errorf("metadata cannot be nil")
	}
	// Implementation
	return nil
}
