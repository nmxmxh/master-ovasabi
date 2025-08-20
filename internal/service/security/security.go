package security

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	securitypb "github.com/nmxmxh/master-ovasabi/api/protos/security/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

// --- Per-minute aggregation for audit events ---.
var (
	auditEventCountsMu sync.Mutex
	auditEventCounts   = make(map[string]map[string]map[string]map[string]int) // minute -> eventType -> action -> principal -> count
)

func recordAuditEventAggregate(eventType, action, principal string) {
	minute := time.Now().Truncate(time.Minute).Format(time.RFC3339)
	auditEventCountsMu.Lock()
	defer auditEventCountsMu.Unlock()
	if _, ok := auditEventCounts[minute]; !ok {
		auditEventCounts[minute] = make(map[string]map[string]map[string]int)
	}
	if _, ok := auditEventCounts[minute][eventType]; !ok {
		auditEventCounts[minute][eventType] = make(map[string]map[string]int)
	}
	if _, ok := auditEventCounts[minute][eventType][action]; !ok {
		auditEventCounts[minute][eventType][action] = make(map[string]int)
	}
	auditEventCounts[minute][eventType][action][principal]++
}

func init() {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			<-ticker.C
			auditEventCountsMu.Lock()
			for minute, eventTypes := range auditEventCounts {
				for eventType, actions := range eventTypes {
					for action, principals := range actions {
						for principal, count := range principals {
							zap.L().Info("Security audit aggregate", zap.String("minute", minute), zap.String("event_type", eventType), zap.String("action", action), zap.String("principal", principal), zap.Int("count", count))
						}
					}
				}
				delete(auditEventCounts, minute)
			}
			auditEventCountsMu.Unlock() // Ensure unlock even if map is empty
		}
	}()
}

// Service implements the SecurityServiceServer interface with rich metadata handling and repository integration.
type Service struct {
	securitypb.UnimplementedSecurityServiceServer
	log          *zap.Logger
	cache        *redis.Cache // optional, can be nil
	repo         *Repository
	eventEnabled bool
	handler      *graceful.Handler // Canonical handler for orchestration
}

func NewService(_ context.Context, log *zap.Logger, cache *redis.Cache, repo *Repository, eventEnabled bool) *Service {
	handler := graceful.NewHandler(log, nil, cache, "security", "v1", eventEnabled)
	s := &Service{
		log:          log,
		cache:        cache,
		repo:         repo,
		eventEnabled: eventEnabled,
		handler:      handler,
	}
	return s
}

// Authenticate verifies user identity and returns a session token.
func (s *Service) Authenticate(ctx context.Context, req *securitypb.AuthenticateRequest) (*securitypb.AuthenticateResponse, error) {
	meta := s.extractSecurityMetadata(req.GetMetadata())
	meta.LastAudit = time.Now().Format(time.RFC3339)
	principal := req.GetPrincipalId()
	cred := req.GetCredential()

	// 1. Validate credentials (simulate: non-empty, not "banned")
	valid := cred != "" && cred != "banned"
	if !valid {
		meta.EscalationLevel = "block"
		meta.RiskScore = 1.0
		meta.BadActor = &BadActorMetadata{
			Score:         1.0,
			Reason:        "Banned or invalid credential",
			DeviceIDs:     meta.DeviceIDs,
			LastFlaggedAt: meta.LastAudit,
		}
		meta.AuditHistory = append(meta.AuditHistory, AuditEntry{
			Timestamp: meta.LastAudit,
			Action:    "authenticate",
			Actor:     principal,
			Result:    "fail",
			Details:   "Invalid or banned credential",
		})
		metaMap := map[string]interface{}{"security": meta}
		fullMap := map[string]interface{}{"service_specific": metaMap}
		normMeta := metadata.MapToProto(fullMap)
		s.handler.Error(ctx, "authenticate", codes.PermissionDenied, "authentication failed", nil, normMeta, principal)
		return &securitypb.AuthenticateResponse{
			SessionToken: "",
			Metadata:     normMeta,
		}, nil
	}

	// 2. Check for existing identity, create if not exists
	identity, err := s.repo.GetIdentity(ctx, "user", principal)
	if err != nil || identity == nil {
		normMeta := metadata.MapToProto(map[string]interface{}{"service_specific": map[string]interface{}{"security": meta}})
		metaBytes, err := json.Marshal(metadata.ProtoToMap(normMeta))
		if err != nil {
			s.log.Error("failed to marshal metadata", zap.Error(err))
			s.handler.Error(ctx, "marshal_metadata", codes.Internal, "failed to marshal metadata", err, nil, principal)
			return nil, graceful.ToStatusError(err)
		}
		master := &Master{
			Type:     "user",
			Status:   "active",
			Metadata: metaBytes,
		}
		_, masterUUID, err := s.repo.CreateMaster(ctx, master)
		if err != nil {
			s.handler.Error(ctx, "create_master", codes.Internal, "failed to create master", err, nil, principal)
			return nil, graceful.ToStatusError(err)
		}
		var masterBigintID int64
		row := s.repo.GetDB().QueryRowContext(ctx, `SELECT master_id FROM service_security_master WHERE uuid = $1`, masterUUID)
		if err := row.Scan(&masterBigintID); err != nil {
			s.handler.Error(ctx, "fetch_master_id", codes.Internal, "failed to fetch master_id for new security master", err, nil, principal)
			return nil, graceful.ToStatusError(err)
		}
		normCred := metadata.MapToProto(map[string]interface{}{"service_specific": map[string]interface{}{"security": cred}})
		credBytes, err := json.Marshal(metadata.ProtoToMap(normCred))
		if err != nil {
			s.log.Error("failed to marshal metadata", zap.Error(err))
			s.handler.Error(ctx, "marshal_metadata", codes.Internal, "failed to marshal metadata", err, nil, principal)
			return nil, graceful.ToStatusError(err)
		}
		identity = &Identity{
			MasterID:     masterBigintID,
			MasterUUID:   masterUUID,
			IdentityType: "user",
			Identifier:   principal,
			Credentials:  credBytes,
			RiskScore:    meta.RiskScore,
		}
		_, err = s.repo.CreateIdentity(ctx, identity)
		if err != nil {
			s.handler.Error(ctx, "create_identity", codes.Internal, "failed to create identity", err, nil, principal)
			return nil, graceful.ToStatusError(err)
		}
	}

	// 3. Risk scoring: escalate if repeated failed attempts (simulate)
	if meta.BadActor != nil && meta.BadActor.Score > 0.8 {
		meta.EscalationLevel = "review"
	}

	// 4. Audit
	meta.AuditHistory = append(meta.AuditHistory, AuditEntry{
		Timestamp: meta.LastAudit,
		Action:    "authenticate",
		Actor:     principal,
		Result:    "success",
	})

	metaMap := map[string]interface{}{"security": meta}
	fullMap := map[string]interface{}{"service_specific": metaMap}
	normMeta := metadata.MapToProto(fullMap)

	// Canonical orchestration: use graceful.WrapSuccess and StandardOrchestrate for all post-success flows
	s.handler.Success(ctx, "authenticate", codes.OK, "authentication succeeded", nil, normMeta, principal, nil)
	return &securitypb.AuthenticateResponse{
		SessionToken: "session-token-stub",
		Metadata:     normMeta,
	}, nil
}

// Authorize checks if a session token is allowed to perform an action on a resource.
func (s *Service) Authorize(ctx context.Context, req *securitypb.AuthorizeRequest) (*securitypb.AuthorizeResponse, error) {
	meta := s.extractSecurityMetadata(req.GetMetadata())
	meta.LastAudit = time.Now().Format(time.RFC3339)
	principal := req.GetPrincipalId()
	action := req.GetAction()

	// 1. Check permission in repository (simulate RBAC: allow if not "restricted")
	allowed := action != "restricted"
	reason := "allowed"
	if !allowed {
		reason = "action restricted"
		meta.EscalationLevel = "warn"
		meta.RiskScore += 0.2
	}

	// 2. Audit
	meta.AuditHistory = append(meta.AuditHistory, AuditEntry{
		Timestamp: meta.LastAudit,
		Action:    "authorize:" + action,
		Actor:     principal,
		Result:    reason,
	})

	metaMap := map[string]interface{}{"security": meta}
	fullMap := map[string]interface{}{"service_specific": metaMap}
	normMeta := metadata.MapToProto(fullMap)

	// If you want to log to repository, implement LogAuthorization in Repository.
	// For now, just log and use handler for error reporting if needed.

	s.handler.Success(ctx, "authorize", codes.OK, "authorization checked", nil, normMeta, principal, nil)
	return &securitypb.AuthorizeResponse{
		Allowed:  allowed,
		Reason:   reason,
		Metadata: normMeta,
	}, nil
}

// ValidateCredential checks if a credential is valid and returns its status.
func (s *Service) ValidateCredential(ctx context.Context, req *securitypb.ValidateCredentialRequest) (*securitypb.ValidateCredentialResponse, error) {
	meta := s.extractSecurityMetadata(req.GetMetadata())
	meta.LastAudit = time.Now().Format(time.RFC3339)
	cred := req.GetCredential()
	valid := cred != "" && cred != "expired"
	if !valid {
		meta.RiskScore = 1.0
		meta.EscalationLevel = "block"
		meta.BadActor = &BadActorMetadata{
			Score:         1.0,
			Reason:        "Expired or invalid credential",
			LastFlaggedAt: meta.LastAudit,
		}
	}
	meta.AuditHistory = append(meta.AuditHistory, AuditEntry{
		Timestamp: meta.LastAudit,
		Action:    "validate_credential",
		Actor:     "",
		Result:    boolToResult(valid),
	})
	metaMap := map[string]interface{}{"security": meta}
	fullMap := map[string]interface{}{"service_specific": metaMap}
	normMeta := metadata.MapToProto(fullMap)

	// If you want to log to repository, implement LogCredentialValidation in Repository.
	// For now, just log and use handler for error reporting if needed.

	s.handler.Success(ctx, "validate_credential", codes.OK, "credential validated", nil, normMeta, cred, nil)
	return &securitypb.ValidateCredentialResponse{
		Valid:    valid,
		Metadata: normMeta,
	}, nil
}

// DetectThreats analyzes a request for potential threats.
func (s *Service) DetectThreats(ctx context.Context, req *securitypb.DetectThreatsRequest) (*securitypb.DetectThreatsResponse, error) {
	meta := s.extractSecurityMetadata(req.GetMetadata())
	meta.LastAudit = time.Now().Format(time.RFC3339)
	principal := req.GetPrincipalId()

	// 1. Analyze for bad actor signals (simulate: escalate if >0.7 risk or repeated device)
	threats := []*securitypb.ThreatSignal{}
	if meta.RiskScore > 0.7 || (meta.BadActor != nil && meta.BadActor.Score > 0.7) {
		threats = append(threats, &securitypb.ThreatSignal{
			Type:        "bad_actor",
			Description: "High risk score or flagged device",
			Score:       meta.RiskScore,
			Metadata:    req.GetMetadata(),
		})
		meta.EscalationLevel = "review"
		meta.BadActor = &BadActorMetadata{
			Score:         meta.RiskScore,
			Reason:        "High risk or flagged device",
			LastFlaggedAt: meta.LastAudit,
		}
	} else {
		threats = append(threats, &securitypb.ThreatSignal{
			Type:        "anomaly",
			Description: "No significant threat",
			Score:       0.1,
			Metadata:    req.GetMetadata(),
		})
	}

	// 2. Compliance: add compliance check (simulate: always WCAG AA)
	meta.Compliance = &ComplianceMetadata{
		Standards: []ComplianceStandard{{
			Name:      "WCAG",
			Level:     "AA",
			Version:   "2.1",
			Compliant: true,
		}},
		CheckedBy: "security-service",
		CheckedAt: meta.LastAudit,
		Method:    "automated",
	}

	meta.AuditHistory = append(meta.AuditHistory, AuditEntry{
		Timestamp: meta.LastAudit,
		Action:    "detect_threats",
		Actor:     principal,
		Result:    "analyzed",
	})

	metaMap := map[string]interface{}{"security": meta}
	fullMap := map[string]interface{}{"service_specific": metaMap}
	normMeta := metadata.MapToProto(fullMap)

	s.handler.Success(ctx, "detect_threats", codes.OK, "threats detected", nil, normMeta, principal, nil)
	return &securitypb.DetectThreatsResponse{
		Threats:  threats,
		Metadata: normMeta,
	}, nil
}

// AuditEvent logs a security-related event.
func (s *Service) AuditEvent(ctx context.Context, req *securitypb.AuditEventRequest) (*securitypb.AuditEventResponse, error) {
	meta := s.extractSecurityMetadata(req.GetMetadata())
	meta.LastAudit = time.Now().Format(time.RFC3339)
	principal := req.GetPrincipalId()

	// Look up the real system/root master record
	var systemMasterID string
	var systemMasterUUID string
	var systemMasterBigintID int64
	masterRow := s.repo.GetDB().QueryRowContext(ctx, `SELECT id, uuid, master_id FROM service_security_master WHERE type = 'system' LIMIT 1`)
	err := masterRow.Scan(&systemMasterID, &systemMasterUUID, &systemMasterBigintID)
	if err != nil {
		s.log.Error("failed to find system/root master record", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "system/root master record not found: %v", err)
	}

	// 1. Fetch the previous event's hash from the most recent event (if any)
	var prevHash string
	row := s.repo.GetDB().QueryRowContext(ctx, `SELECT metadata FROM service_security_event ORDER BY occurred_at DESC LIMIT 1`)
	var prevMetaBytes []byte
	if err := row.Scan(&prevMetaBytes); err == nil && len(prevMetaBytes) > 0 {
		var prevMetaMap map[string]interface{}
		err := json.Unmarshal(prevMetaBytes, &prevMetaMap)
		if err == nil {
			if audit, ok := prevMetaMap["audit"].(map[string]interface{}); ok {
				if h, ok := audit["entry_hash"].(string); ok {
					prevHash = h
				}
			}
		}
	}

	// 2. Serialize current event data for hashing
	hashInput := map[string]interface{}{
		"principal":   principal,
		"event_type":  req.GetEventType(),
		"resource":    req.GetResource(),
		"action":      req.GetAction(),
		"occurred_at": time.Now().Format(time.RFC3339),
	}
	hashInputBytes, err := json.Marshal(hashInput)
	if err != nil {
		s.log.Error("failed to marshal hash input", zap.Error(err))
		return nil, graceful.ToStatusError(graceful.WrapErr(ctx, codes.Internal, "failed to marshal hash input", err))
	}

	// 3. Compute entry_hash = SHA256(eventData || prevHash)
	h := sha256.New()
	h.Write(hashInputBytes)
	h.Write([]byte(prevHash))
	entryHash := hex.EncodeToString(h.Sum(nil))

	// 4. Store prev_hash and entry_hash in the Details field of the new AuditEntry
	meta.AuditHistory = append(meta.AuditHistory, AuditEntry{
		Timestamp: meta.LastAudit,
		Action:    req.GetAction(),
		Actor:     principal,
		Result:    "recorded",
		Details:   "prev_hash:" + prevHash + ";entry_hash:" + entryHash,
	})

	// 1. Store event in repository
	normDetails := metadata.MapToProto(map[string]interface{}{
		"resource": req.GetResource(),
		"action":   req.GetAction(),
	})
	detailsBytes, err := json.Marshal(metadata.ProtoToMap(normDetails))
	if err != nil {
		s.log.Error("failed to marshal details", zap.Error(err))
		return nil, graceful.ToStatusError(graceful.WrapErr(ctx, codes.Internal, "failed to marshal details", err))
	}
	normMeta := metadata.MapToProto(map[string]interface{}{"service_specific": map[string]interface{}{"security": meta}})
	metaBytes, err := json.Marshal(metadata.ProtoToMap(normMeta))
	if err != nil {
		s.log.Error("failed to marshal metadata", zap.Error(err))
		return nil, graceful.ToStatusError(graceful.WrapErr(ctx, codes.Internal, "failed to marshal metadata", err))
	}
	event := &Event{
		MasterID:   systemMasterBigintID,
		EventType:  req.GetEventType(),
		Principal:  principal,
		Details:    detailsBytes,
		OccurredAt: time.Now(),
		Metadata:   metaBytes,
		PrevHash:   prevHash,
		EntryHash:  entryHash,
	}
	_, err = s.repo.RecordEvent(ctx, event)
	if err != nil {
		s.log.Error("failed to record audit event", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "internal error: %v", err)
	}

	metaMap := map[string]interface{}{"security": meta}
	fullMap := map[string]interface{}{"service_specific": metaMap}
	normMeta = metadata.MapToProto(fullMap)
	recordAuditEventAggregate(req.GetEventType(), req.GetAction(), req.GetPrincipalId())

	s.handler.Success(ctx, "audit_event", codes.OK, "audit event recorded", nil, normMeta, principal, nil)
	return &securitypb.AuditEventResponse{
		Success:  true,
		Metadata: normMeta,
	}, nil
}

// QueryEvents streams security events (audit log entries).
func (s *Service) QueryEvents(ctx context.Context, req *securitypb.QueryEventsRequest) (*securitypb.QueryEventsResponse, error) {
	meta := s.extractSecurityMetadata(req.GetMetadata())
	meta.LastAudit = time.Now().Format(time.RFC3339)
	filter := map[string]interface{}{}
	if req.GetPrincipalId() != "" {
		filter["principal"] = req.GetPrincipalId()
	}
	if req.GetEventType() != "" {
		filter["event_type"] = req.GetEventType()
	}
	securityEvents, err := s.repo.GetEvents(ctx, filter)
	if err != nil {
		s.log.Error("failed to query events", zap.Error(err))
	}
	protoEvents := make([]*securitypb.SecurityEvent, 0, len(securityEvents))
	for _, e := range securityEvents {
		protoEvents = append(protoEvents, &securitypb.SecurityEvent{
			Id:          e.ID,
			PrincipalId: e.Principal,
			EventType:   e.EventType,
			// Resource and Action fields are no longer directly available; can be extracted from e.Details if needed
			// Resource:    ...
			// Action:     ...
			// Timestamp, Details, etc. as needed
		})
	}
	metaMap := map[string]interface{}{"security": meta}
	fullMap := map[string]interface{}{"service_specific": metaMap}
	normMeta := metadata.MapToProto(fullMap)

	s.handler.Success(ctx, "query_events", codes.OK, "events queried", nil, normMeta, "security-query", nil)
	return &securitypb.QueryEventsResponse{
		Events:   protoEvents,
		Metadata: normMeta,
	}, nil
}

// boolToResult returns "success" or "fail" for a boolean.
func boolToResult(b bool) string {
	if b {
		return "success"
	}
	return "fail"
}

// extractSecurityMetadata parses the security service-specific metadata from a common.Metadata proto.
func (s *Service) extractSecurityMetadata(meta *commonpb.Metadata) *ServiceMetadata {
	if meta == nil || meta.ServiceSpecific == nil {
		return NewSecurityMetadata()
	}
	fields := meta.ServiceSpecific.GetFields()
	secField, ok := fields["security"]
	if !ok || secField == nil || secField.GetStructValue() == nil {
		return NewSecurityMetadata()
	}
	m := metadata.ProtoToMap(&commonpb.Metadata{ServiceSpecific: &structpb.Struct{Fields: map[string]*structpb.Value{"security": secField}}})
	if ss, ok := m["service_specific"].(map[string]interface{}); ok {
		if _, ok := ss["security"].(map[string]interface{}); ok {
			// For now, return NewSecurityMetadata() as a placeholder
			return NewSecurityMetadata()
		}
	}
	return NewSecurityMetadata()
}

// NewSecurityMetadata creates a canonical security metadata struct with reasonable defaults.
func NewSecurityMetadata() *ServiceMetadata {
	return &ServiceMetadata{
		RiskScore:       0.0,
		RiskFactors:     []string{},
		LastAudit:       "",
		AuditHistory:    []AuditEntry{},
		Compliance:      nil,
		BadActor:        nil,
		LinkedAccounts:  []string{},
		DeviceIDs:       []string{},
		Locations:       []LocationMetadata{},
		EscalationLevel: "info",
		LastEscalatedAt: "",
		UserID:          "",
		ContentID:       "",
		LocalizationID:  "",
	}
}
