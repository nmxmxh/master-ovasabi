// Package metaversion provides canonical versioning and feature flag management for the OVASABI platform.
package metaversion

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"time"

	"go.uber.org/zap"
)

// Versioning holds all versioning and feature flag metadata for a user/session/entity.
type Versioning struct {
	SystemVersion  string    `json:"system_version"`
	ServiceVersion string    `json:"service_version"`
	UserVersion    string    `json:"user_version"`
	Environment    string    `json:"environment"`
	FeatureFlags   []string  `json:"feature_flags"`
	ABTestGroup    string    `json:"ab_test_group"`
	LastMigratedAt time.Time `json:"last_migrated_at"`
}

// InitialVersion is the default version for all fields at package init.
const InitialVersion = "0.0.1"

// NewDefault returns a Versioning struct with all fields set to initial values.
func NewDefault() Versioning {
	return Versioning{
		SystemVersion:  InitialVersion,
		ServiceVersion: InitialVersion,
		UserVersion:    InitialVersion,
		Environment:    "dev",
		FeatureFlags:   []string{},
		ABTestGroup:    "A",
		LastMigratedAt: time.Now().UTC(),
	}
}

// contextKey is unexported to avoid collisions.
type contextKey struct{}

var versioningContextKey = &contextKey{}

// InjectContext returns a new context with the given Versioning.
func InjectContext(ctx context.Context, v Versioning) context.Context {
	return context.WithValue(ctx, versioningContextKey, v)
}

// FromContext extracts Versioning from context, or returns false if not present.
func FromContext(ctx context.Context) (Versioning, bool) {
	v, ok := ctx.Value(versioningContextKey).(Versioning)
	return v, ok
}

// MergeMetadata merges Versioning into a metadata map under service_specific.user.versioning.
func MergeMetadata(metadata map[string]interface{}, v Versioning) map[string]interface{} {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	ss, ok := metadata["service_specific"].(map[string]interface{})
	if !ok {
		ss = make(map[string]interface{})
	}
	user, ok := ss["user"].(map[string]interface{})
	if !ok {
		user = make(map[string]interface{})
	}
	user["versioning"] = v
	ss["user"] = user
	metadata["service_specific"] = ss
	return metadata
}

// ToMap converts Versioning to a map for embedding in metadata or JWT claims.
func (v Versioning) ToMap(log *zap.Logger) map[string]interface{} {
	b, err := json.Marshal(v)
	if err != nil {
		if log != nil {
			log.Error("failed to marshal Versioning", zap.Error(err))
		}
		return map[string]interface{}{}
	}
	m := make(map[string]interface{})
	if err := json.Unmarshal(b, &m); err != nil {
		if log != nil {
			log.Error("failed to unmarshal Versioning", zap.Error(err))
		}
		return map[string]interface{}{}
	}
	return m
}

// FromMap parses Versioning from a map (e.g., from metadata or JWT claims).
func FromMap(m map[string]interface{}, log *zap.Logger) (Versioning, error) {
	b, err := json.Marshal(m)
	if err != nil {
		if log != nil {
			log.Error("failed to marshal map for Versioning", zap.Error(err))
		}
		return NewDefault(), err
	}
	var v Versioning
	if err := json.Unmarshal(b, &v); err != nil {
		if log != nil {
			log.Error("failed to unmarshal map to Versioning", zap.Error(err))
		}
		return NewDefault(), err
	}
	return v, nil
}

// ValidateVersioning checks that all version fields are semantic and required fields are set.
func ValidateVersioning(v Versioning) error {
	semver := regexp.MustCompile(`^\d+\.\d+\.\d+$`)
	if !semver.MatchString(v.SystemVersion) {
		return errors.New("invalid system_version format")
	}
	if !semver.MatchString(v.ServiceVersion) {
		return errors.New("invalid service_version format")
	}
	if !semver.MatchString(v.UserVersion) {
		return errors.New("invalid user_version format")
	}
	if v.Environment == "" {
		return errors.New("environment must be set")
	}
	return nil
}
