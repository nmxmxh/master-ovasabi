package orchestration

import (
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
)

// ScheduledJob represents a job that is scheduled for execution.
type ScheduledJob struct {
	ID        string
	EntityID  string
	JobType   string
	ExecuteAt string // RFC3339 timestamp
	Metadata  *commonpb.Metadata
}
