// Metadata Standard Reference
// --------------------------
// All service-specific metadata must include the `versioning` field as described in:
//   - docs/services/versioning.md
//   - docs/amadeus/amadeus_context.md
// For all available metadata actions, patterns, and service-specific extensions, see:
//   - docs/services/metadata.md (general metadata documentation)
//   - docs/services/versioning.md (versioning/environment standard)
//
// This file implements media service-specific metadata patterns. See above for required fields and integration points.
//
// Service-Specific Metadata Pattern for Media Service
// --------------------------------------------------
//
// This file defines the canonical Go struct for all media service-specific metadata fields,
// covering all platform standards (captions, translations, accessibility, compliance, etc.).
//
// Usage:
// - Use MediaMetadata to read/update all service-specific metadata fields in Go.
// - Use the provided helpers to convert between MediaMetadata and structpb.Struct.
// - This pattern ensures robust, type-safe, and future-proof handling of media metadata.
//
// Reference: docs/amadeus/amadeus_context.md#cross-service-standards-integration-path

package media

import (
	"encoding/json"

	structpb "google.golang.org/protobuf/types/known/structpb"
)

type Metadata struct {
	Versioning    map[string]interface{} `json:"versioning"` // Required versioning field
	Captions      []CaptionTrack         `json:"captions,omitempty"`
	Translations  []TranslationTrack     `json:"translations,omitempty"`
	Accessibility *AccessibilityMetadata `json:"accessibility,omitempty"`
	Compliance    *ComplianceMetadata    `json:"compliance,omitempty"`
	Thumbnails    []ThumbnailInfo        `json:"thumbnails,omitempty"`
	PlaybackURLs  map[string]string      `json:"playback_urls,omitempty"` // e.g., {"hls": ..., "dash": ...}
	Duration      float64                `json:"duration,omitempty"`      // seconds
	Bitrate       int64                  `json:"bitrate,omitempty"`
	Resolution    string                 `json:"resolution,omitempty"` // e.g., "1920x1080"
	FrameRate     float64                `json:"frame_rate,omitempty"`
	AspectRatio   string                 `json:"aspect_ratio,omitempty"`
	Codec         string                 `json:"codec,omitempty"`
	Container     string                 `json:"container,omitempty"`
	Optimizations []string               `json:"optimizations,omitempty"` // e.g., ["compressed", "normalized"]
	Custom        map[string]interface{} `json:"custom,omitempty"`        // For future extensibility
}

type CaptionTrack struct {
	Language string `json:"language"`
	URL      string `json:"url"`
	Format   string `json:"format"`         // e.g., "vtt", "srt"
	Kind     string `json:"kind,omitempty"` // e.g., "subtitles", "captions"
	Label    string `json:"label,omitempty"`
	Default  bool   `json:"default,omitempty"`
}

type TranslationTrack struct {
	Language   string                 `json:"language"`
	Type       string                 `json:"type"`       // "audio", "subtitle"
	Provenance map[string]interface{} `json:"provenance"` // See docs/services/metadata.md
	URL        string                 `json:"url"`
}

type AccessibilityMetadata struct {
	AltText         string   `json:"alt_text,omitempty"`
	AudioDescURL    string   `json:"audio_description_url,omitempty"`
	Features        []string `json:"features,omitempty"`         // e.g., ["captions", "sign_language"]
	PlatformSupport []string `json:"platform_support,omitempty"` // e.g., ["desktop", "mobile", "screen_reader"]
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

type ThumbnailInfo struct {
	URL         string  `json:"url"`
	Width       int     `json:"width,omitempty"`
	Height      int     `json:"height,omitempty"`
	TimeOffset  float64 `json:"time_offset,omitempty"` // seconds
	Description string  `json:"description,omitempty"`
}

// MediaMetadataFromStruct converts a structpb.Struct to MediaMetadata.
func MetadataFromStruct(s *structpb.Struct) (*Metadata, error) {
	if s == nil {
		return &Metadata{}, nil
	}
	b, err := json.Marshal(s.AsMap())
	if err != nil {
		return nil, err
	}
	var meta Metadata
	err = json.Unmarshal(b, &meta)
	if err != nil {
		return nil, err
	}
	return &meta, nil
}

// MediaMetadataToStruct converts MediaMetadata to structpb.Struct.
func MetadataToStruct(meta *Metadata) (*structpb.Struct, error) {
	if meta == nil {
		return structpb.NewStruct(map[string]interface{}{})
	}
	b, err := json.Marshal(meta)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	err = json.Unmarshal(b, &m)
	if err != nil {
		return nil, err
	}
	return structpb.NewStruct(m)
}
