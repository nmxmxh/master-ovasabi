package events

import commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"

// EventEnvelope is the canonical, extensible wrapper for all event-driven messages in the system.
type EventEnvelope struct {
	ID        string             `json:"id"`
	Type      string             `json:"type"`
	Payload   *commonpb.Payload  `json:"payload"`
	Metadata  *commonpb.Metadata `json:"metadata"`
	Timestamp int64              `json:"timestamp,omitempty"`
}
