package security

import (
	"context"
	"encoding/json"
	"strconv"
	"sync"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	securitypb "github.com/nmxmxh/master-ovasabi/api/protos/security/v1"
	wsPkg "github.com/nmxmxh/master-ovasabi/internal/server/ws" // for systemAggMu and systemAggStats
	pattern "github.com/nmxmxh/master-ovasabi/internal/service/pattern"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	structpb "google.golang.org/protobuf/types/known/structpb"
	// for systemAggMu and systemAggStats.
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

	// Unified system aggregation
	wsPkg.SystemAggMu.Lock()
	if eventType == "security.audit_event" {
		wsPkg.SystemAggStats.Audit.Events++
	} else {
		wsPkg.SystemAggStats.Security.Events++
	}
	wsPkg.SystemAggMu.Unlock()
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
			auditEventCountsMu.Unlock()
		}
	}()
}

// Service implements the SecurityServiceServer interface with rich metadata handling and repository integration.
type Service struct {
	securitypb.UnimplementedSecurityServiceServer
	log          *zap.Logger
	cache        *redis.Cache // optional, can be nil
	repo         *Repository
	eventEmitter EventEmitter
	eventEnabled bool
}

func NewService(ctx context.Context, log *zap.Logger, cache *redis.Cache, repo *Repository, eventEmitter EventEmitter, eventEnabled bool) *Service {
	s := &Service{
		log:          log,
		cache:        cache,
		repo:         repo,
		eventEmitter: eventEmitter,
		eventEnabled: eventEnabled,
	}
	// Register the service in the knowledge graph at startup
	if err := pattern.RegisterWithNexus(ctx, log, "security", nil); err != nil {
		log.Error("RegisterWithNexus failed in NewService (security)", zap.Error(err))
	}
	return s
}

// Authenticate verifies user identity and returns a session token.
func (s *Service) Authenticate(ctx context.Context, req *securitypb.AuthenticateRequest) (*securitypb.AuthenticateResponse, error) {
	meta, err := s.extractSecurityMetadata(req.GetMetadata())
	if err != nil {
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{"error": err.Error()})
			if err != nil {
				s.log.Error("Failed to create structpb.Struct for security.authentication_failed event", zap.Error(err))
				return nil, status.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "security.authentication_failed", req.GetPrincipalId(), errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit security.authentication_failed event", zap.Error(errEmit))
			}
		}
		return nil, status.Errorf(codes.InvalidArgument, "invalid metadata: %v", err)
	}
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
		updatedStruct, err := ServiceMetadataToStruct(meta)
		if err != nil {
			s.log.Error("failed to convert ServiceMetadata to struct", zap.Error(err))
			return nil, status.Errorf(codes.Internal, "metadata conversion failed: %v", err)
		}
		s.orchestrateMetadata(ctx, "security_auth", principal, req.GetMetadata())
		if s.eventEnabled && s.eventEmitter != nil {
			errEmit := s.eventEmitter.EmitEvent(ctx, "security.authentication_failed", principal, req.GetMetadata())
			if errEmit != nil {
				s.log.Warn("Failed to emit security.authentication_failed event", zap.Error(errEmit))
			}
		}
		return &securitypb.AuthenticateResponse{
			SessionToken: "",
			Metadata: &commonpb.Metadata{
				ServiceSpecific: &structpb.Struct{Fields: map[string]*structpb.Value{"security": structpb.NewStructValue(updatedStruct)}},
			},
		}, nil
	}

	// 2. Check for existing identity, create if not exists
	identity, err := s.repo.GetIdentity(ctx, "user", principal)
	if err != nil || identity == nil {
		master := &Master{
			Type:     "user",
			Status:   "active",
			Metadata: mustMarshal(meta),
		}
		masterID, masterUUID, err := s.repo.CreateMaster(ctx, master)
		if err != nil {
			s.log.Error("failed to create master", zap.Error(err))
		}
		identity = &Identity{
			MasterID:     masterID,
			MasterUUID:   masterUUID,
			IdentityType: "user",
			Identifier:   principal,
			Credentials:  mustMarshal(cred),
			RiskScore:    meta.RiskScore,
		}
		_, err = s.repo.CreateIdentity(ctx, identity)
		if err != nil {
			s.log.Error("failed to create identity", zap.Error(err))
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

	updatedStruct, err := ServiceMetadataToStruct(meta)
	if err != nil {
		s.log.Error("failed to convert ServiceMetadata to struct", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "metadata conversion failed: %v", err)
	}
	s.orchestrateMetadata(ctx, "security_auth", principal, req.GetMetadata())
	if s.eventEnabled && s.eventEmitter != nil {
		req.Metadata, _ = events.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "security.authenticated", principal, req.GetMetadata())
	}
	return &securitypb.AuthenticateResponse{
		SessionToken: "session-token-stub",
		Metadata: &commonpb.Metadata{
			ServiceSpecific: &structpb.Struct{Fields: map[string]*structpb.Value{"security": structpb.NewStructValue(updatedStruct)}},
		},
	}, nil
}

// Authorize checks if a session token is allowed to perform an action on a resource.
func (s *Service) Authorize(ctx context.Context, req *securitypb.AuthorizeRequest) (*securitypb.AuthorizeResponse, error) {
	meta, err := s.extractSecurityMetadata(req.GetMetadata())
	if err != nil {
		s.log.Error("invalid metadata", zap.Error(err))
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{"error": err.Error()})
			if err != nil {
				s.log.Error("Failed to create structpb.Struct for security.authorization_failed event", zap.Error(err))
				return nil, status.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "security.authorization_failed", req.GetPrincipalId(), errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit security.authorization_failed event", zap.Error(errEmit))
			}
		}
		return nil, status.Errorf(codes.InvalidArgument, "invalid metadata: %v", err)
	}
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

	updatedStruct, err := ServiceMetadataToStruct(meta)
	if err != nil {
		s.log.Error("failed to convert ServiceMetadata to struct", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "metadata conversion failed: %v", err)
	}
	s.orchestrateMetadata(ctx, "security_authorize", principal, req.GetMetadata())
	if s.eventEnabled && s.eventEmitter != nil {
		req.Metadata, _ = events.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "security.authorized", principal, req.GetMetadata())
	}
	return &securitypb.AuthorizeResponse{
		Allowed: allowed,
		Reason:  reason,
		Metadata: &commonpb.Metadata{
			ServiceSpecific: &structpb.Struct{Fields: map[string]*structpb.Value{"security": structpb.NewStructValue(updatedStruct)}},
		},
	}, nil
}

// ValidateCredential checks if a credential is valid and returns its status.
func (s *Service) ValidateCredential(ctx context.Context, req *securitypb.ValidateCredentialRequest) (*securitypb.ValidateCredentialResponse, error) {
	meta, err := s.extractSecurityMetadata(req.GetMetadata())
	if err != nil {
		s.log.Error("invalid metadata", zap.Error(err))
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{"error": err.Error()})
			if err != nil {
				s.log.Error("Failed to create structpb.Struct for security.credential_validation_failed event", zap.Error(err))
				return nil, status.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "security.credential_validation_failed", "", errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit security.credential_validation_failed event", zap.Error(errEmit))
			}
		}
		return nil, status.Errorf(codes.InvalidArgument, "invalid metadata: %v", err)
	}
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
	updatedStruct, err := ServiceMetadataToStruct(meta)
	if err != nil {
		s.log.Error("failed to convert ServiceMetadata to struct", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "metadata conversion failed: %v", err)
	}
	s.orchestrateMetadata(ctx, "security_validate_credential", "", req.GetMetadata())
	if s.eventEnabled && s.eventEmitter != nil {
		req.Metadata, _ = events.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "security.credential_validated", "", req.GetMetadata())
	}
	return &securitypb.ValidateCredentialResponse{
		Valid: valid,
		Metadata: &commonpb.Metadata{
			ServiceSpecific: &structpb.Struct{Fields: map[string]*structpb.Value{"security": structpb.NewStructValue(updatedStruct)}},
		},
	}, nil
}

// DetectThreats analyzes a request for potential threats.
func (s *Service) DetectThreats(ctx context.Context, req *securitypb.DetectThreatsRequest) (*securitypb.DetectThreatsResponse, error) {
	meta, err := s.extractSecurityMetadata(req.GetMetadata())
	if err != nil {
		s.log.Error("invalid metadata", zap.Error(err))
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{"error": err.Error()})
			if err != nil {
				s.log.Error("Failed to create structpb.Struct for security.threat_detection_failed event", zap.Error(err))
				return nil, status.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "security.threat_detection_failed", req.GetPrincipalId(), errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit security.threat_detection_failed event", zap.Error(errEmit))
			}
		}
		return nil, status.Errorf(codes.InvalidArgument, "invalid metadata: %v", err)
	}
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

	updatedStruct, err := ServiceMetadataToStruct(meta)
	if err != nil {
		s.log.Error("failed to convert ServiceMetadata to struct", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "metadata conversion failed: %v", err)
	}
	s.orchestrateMetadata(ctx, "security_detect_threats", principal, req.GetMetadata())
	if s.eventEnabled && s.eventEmitter != nil {
		req.Metadata, _ = events.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "security.threat_detected", req.GetPrincipalId(), req.GetMetadata())
	}
	return &securitypb.DetectThreatsResponse{
		Threats:  threats,
		Metadata: &commonpb.Metadata{ServiceSpecific: &structpb.Struct{Fields: map[string]*structpb.Value{"security": structpb.NewStructValue(updatedStruct)}}},
	}, nil
}

// AuditEvent logs a security-related event.
func (s *Service) AuditEvent(ctx context.Context, req *securitypb.AuditEventRequest) (*securitypb.AuditEventResponse, error) {
	meta, err := s.extractSecurityMetadata(req.GetMetadata())
	if err != nil {
		s.log.Error("invalid metadata", zap.Error(err))
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{"error": err.Error()})
			if err != nil {
				s.log.Error("Failed to create structpb.Struct for security.audit_event_failed event", zap.Error(err))
				return nil, status.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "security.audit_event_failed", req.GetPrincipalId(), errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit security.audit_event_failed event", zap.Error(errEmit))
			}
		}
		return nil, status.Errorf(codes.InvalidArgument, "invalid metadata: %v", err)
	}
	meta.LastAudit = time.Now().Format(time.RFC3339)
	principal := req.GetPrincipalId()

	// Look up the real system/root master record
	var systemMasterID int64
	var systemMasterUUID string
	masterRow := s.repo.db.QueryRowContext(ctx, `SELECT id, uuid FROM service_security_master WHERE type = 'system' LIMIT 1`)
	err = masterRow.Scan(&systemMasterID, &systemMasterUUID)
	if err != nil {
		s.log.Error("failed to find system/root master record", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "system/root master record not found: %v", err)
	}

	// 1. Store event in repository
	event := &Event{
		MasterID:   systemMasterID,
		MasterUUID: systemMasterUUID,
		EventType:  req.GetEventType(),
		Principal:  principal,
		Resource:   req.GetResource(),
		Action:     req.GetAction(),
		OccurredAt: time.Now(),
		Metadata:   mustMarshal(meta),
	}
	_, err = s.repo.RecordEvent(ctx, event)
	if err != nil {
		s.log.Error("failed to record audit event", zap.Error(err))
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{"error": err.Error()})
			if err != nil {
				s.log.Error("Failed to create structpb.Struct for security.audit_event_failed event", zap.Error(err))
				return nil, status.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "security.audit_event_failed", req.GetPrincipalId(), errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit security.audit_event_failed event", zap.Error(errEmit))
			}
		}
	}
	if s.eventEnabled && s.eventEmitter != nil {
		req.Metadata, _ = events.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "security.audit_event", req.GetPrincipalId(), req.GetMetadata())
	}

	meta.AuditHistory = append(meta.AuditHistory, AuditEntry{
		Timestamp: meta.LastAudit,
		Action:    req.GetAction(),
		Actor:     principal,
		Result:    "recorded",
	})

	updatedStruct, err := ServiceMetadataToStruct(meta)
	if err != nil {
		s.log.Error("failed to convert ServiceMetadata to struct", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "metadata conversion failed: %v", err)
	}
	s.orchestrateMetadata(ctx, "security_audit_event", principal, req.GetMetadata())
	recordAuditEventAggregate(req.GetEventType(), req.GetAction(), req.GetPrincipalId())
	return &securitypb.AuditEventResponse{
		Success: true,
		Metadata: &commonpb.Metadata{
			ServiceSpecific: &structpb.Struct{Fields: map[string]*structpb.Value{"security": structpb.NewStructValue(updatedStruct)}},
		},
	}, nil
}

// QueryEvents streams security events (audit log entries).
func (s *Service) QueryEvents(ctx context.Context, req *securitypb.QueryEventsRequest) (*securitypb.QueryEventsResponse, error) {
	meta, err := s.extractSecurityMetadata(req.GetMetadata())
	if err != nil {
		s.log.Error("invalid metadata", zap.Error(err))
		return nil, status.Errorf(codes.InvalidArgument, "invalid metadata: %v", err)
	}
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
			Id:          strconv.FormatInt(e.ID, 10),
			PrincipalId: e.Principal,
			EventType:   e.EventType,
			Resource:    e.Resource,
			Action:      e.Action,
			// Timestamp, Details, etc. as needed
		})
	}
	updatedStruct, err := ServiceMetadataToStruct(meta)
	if err != nil {
		s.log.Error("failed to convert ServiceMetadata to struct", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "metadata conversion failed: %v", err)
	}
	return &securitypb.QueryEventsResponse{
		Events:   protoEvents,
		Metadata: &commonpb.Metadata{ServiceSpecific: &structpb.Struct{Fields: map[string]*structpb.Value{"security": structpb.NewStructValue(updatedStruct)}}},
	}, nil
}

// --- Helper methods for metadata extraction and orchestration ---

func mustMarshal(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		// Handle error: log and return nil or empty slice
		// (or panic if this should never fail)
		return nil
	}
	return b
}

func boolToResult(b bool) string {
	if b {
		return "success"
	}
	return "fail"
}

// extractSecurityMetadata parses the security service-specific metadata from a common.Metadata proto.
func (s *Service) extractSecurityMetadata(meta *commonpb.Metadata) (*ServiceMetadata, error) {
	if meta == nil || meta.ServiceSpecific == nil {
		return NewSecurityMetadata(), nil
	}
	fields := meta.ServiceSpecific.GetFields()
	secField, ok := fields["security"]
	if !ok || secField == nil || secField.GetStructValue() == nil {
		return NewSecurityMetadata(), nil
	}
	return ServiceMetadataFromStruct(secField.GetStructValue())
}

// orchestrateMetadata runs all orchestration/caching/knowledge graph hooks for a given entity.
func (s *Service) orchestrateMetadata(ctx context.Context, entityType, entityID string, meta *commonpb.Metadata) {
	if s.cache != nil && meta != nil {
		if err := pattern.CacheMetadata(ctx, s.log, s.cache, entityType, entityID, meta, 10*time.Minute); err != nil {
			s.log.Error("failed to cache metadata", zap.Error(err))
		}
	}
	if err := pattern.RegisterSchedule(ctx, s.log, entityType, entityID, meta); err != nil {
		s.log.Error("failed to register schedule", zap.Error(err))
	}
	if err := pattern.EnrichKnowledgeGraph(ctx, s.log, entityType, entityID, meta); err != nil {
		s.log.Error("failed to enrich knowledge graph", zap.Error(err))
	}
	if err := pattern.RegisterWithNexus(ctx, s.log, entityType, meta); err != nil {
		s.log.Error("failed to register with Nexus", zap.Error(err))
	}
}

// --- End of Security Service Template ---
