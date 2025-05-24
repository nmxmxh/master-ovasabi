// OVASABI Scheduler Service Metadata Pattern
// -----------------------------------------
// This file defines the canonical metadata structure and helpers for the Scheduler service.
// It follows the robust, extensible metadata pattern described in:
//   - docs/services/metadata.md
//   - docs/amadeus/amadeus_context.md
// All scheduler jobs and job runs must use this pattern for orchestration, analytics, and extensibility.
//
// See also: api/protos/common/v1/metadata.proto

package scheduler

import (
	"fmt"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// SchedulerMetadata defines scheduler-specific metadata fields.
type Metadata struct {
	JobPriority  string   `json:"job_priority,omitempty"`
	JobType      string   `json:"job_type,omitempty"`
	TriggerEvent string   `json:"trigger_event,omitempty"`
	Escalation   string   `json:"escalation,omitempty"`
	Notification []string `json:"notification,omitempty"`
}

// SchedulingInfo defines the scheduling fields for jobs.
type SchedulingInfo struct {
	Cron      string `json:"cron,omitempty"`
	Timezone  string `json:"timezone,omitempty"`
	StartTime string `json:"start_time,omitempty"`
	EndTime   string `json:"end_time,omitempty"`
}

// ExtractSchedulerMetadata extracts scheduler-specific metadata from commonpb.Metadata.
func ExtractSchedulerMetadata(meta *commonpb.Metadata) (*Metadata, error) {
	if meta == nil || meta.ServiceSpecific == nil {
		return &Metadata{}, nil
	}
	ss := meta.ServiceSpecific.GetFields()[""]
	if ss == nil || ss.GetStructValue() == nil {
		return &Metadata{}, nil
	}
	fields := ss.GetStructValue().GetFields()
	m := &Metadata{}
	if v, ok := fields["job_priority"]; ok {
		m.JobPriority = v.GetStringValue()
	}
	if v, ok := fields["job_type"]; ok {
		m.JobType = v.GetStringValue()
	}
	if v, ok := fields["trigger_event"]; ok {
		m.TriggerEvent = v.GetStringValue()
	}
	if v, ok := fields["escalation"]; ok {
		m.Escalation = v.GetStringValue()
	}
	if v, ok := fields["notification"]; ok && v.GetListValue() != nil {
		for _, n := range v.GetListValue().GetValues() {
			m.Notification = append(m.Notification, n.GetStringValue())
		}
	}
	return m, nil
}

// ExtractSchedulingInfo extracts scheduling info from commonpb.Metadata.
func ExtractSchedulingInfo(meta *commonpb.Metadata) (*SchedulingInfo, error) {
	if meta == nil || meta.Scheduling == nil {
		return &SchedulingInfo{}, nil
	}
	fields := meta.Scheduling.GetFields()
	info := &SchedulingInfo{}
	if v, ok := fields["cron"]; ok {
		info.Cron = v.GetStringValue()
	}
	if v, ok := fields["timezone"]; ok {
		info.Timezone = v.GetStringValue()
	}
	if v, ok := fields["start_time"]; ok {
		info.StartTime = v.GetStringValue()
	}
	if v, ok := fields["end_time"]; ok {
		info.EndTime = v.GetStringValue()
	}
	return info, nil
}

// EnrichSchedulerMetadata adds/updates scheduler-specific fields in commonpb.Metadata.
func EnrichSchedulerMetadata(meta *commonpb.Metadata, sched *Metadata) error {
	if meta == nil || sched == nil {
		return fmt.Errorf("metadata or scheduler metadata is nil")
	}
	ss := &structpb.Struct{Fields: map[string]*structpb.Value{}}
	if sched.JobPriority != "" {
		ss.Fields["job_priority"] = structpb.NewStringValue(sched.JobPriority)
	}
	if sched.JobType != "" {
		ss.Fields["job_type"] = structpb.NewStringValue(sched.JobType)
	}
	if sched.TriggerEvent != "" {
		ss.Fields["trigger_event"] = structpb.NewStringValue(sched.TriggerEvent)
	}
	if sched.Escalation != "" {
		ss.Fields["escalation"] = structpb.NewStringValue(sched.Escalation)
	}
	if len(sched.Notification) > 0 {
		lv := &structpb.ListValue{}
		for _, n := range sched.Notification {
			lv.Values = append(lv.Values, structpb.NewStringValue(n))
		}
		ss.Fields["notification"] = structpb.NewListValue(lv)
	}
	if meta.ServiceSpecific == nil {
		meta.ServiceSpecific = &structpb.Struct{Fields: map[string]*structpb.Value{}}
	}
	meta.ServiceSpecific.Fields["scheduler"] = structpb.NewStructValue(ss)
	return nil
}

// ValidateSchedulerMetadata validates required scheduler metadata fields.
func ValidateSchedulerMetadata(m *Metadata) error {
	if m.JobType == "" {
		return fmt.Errorf("job_type is required in scheduler metadata")
	}
	return nil
}
