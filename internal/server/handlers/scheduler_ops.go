package handlers

import (
	"encoding/json"
	"net/http"

	schedulerpb "github.com/nmxmxh/master-ovasabi/api/protos/scheduler/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/contextx"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"go.uber.org/zap"
)

// SchedulerOpsHandler handles scheduler-related actions via the "action" field.
//
// @Summary Scheduler Operations
// @Description Handles scheduler-related actions using the "action" field in the request body. Each action (e.g., create_job, update_job, etc.) has its own required/optional fields. All requests must include a 'metadata' field following the robust metadata pattern (see docs/services/metadata.md).
// @Tags scheduler
// @Accept json
// @Produce json
// @Param request body object true "Composable request with 'action', required fields for the action, and 'metadata' (see docs/services/metadata.md for schema)"
// @Success 200 {object} object "Response depends on action"
// @Failure 400 {object} ErrorResponse
// @Router /api/scheduler_ops [post]

func SchedulerOpsHandler(container *di.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Inject logger into context
		log := contextx.Logger(r.Context())
		ctx := contextx.WithLogger(r.Context(), log)
		var schedulerSvc schedulerpb.SchedulerServiceServer
		if err := container.Resolve(&schedulerSvc); err != nil {
			log.Error("SchedulerServiceServer not found in container", zap.Error(err))
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode scheduler request JSON", zap.Error(err))
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		action, ok := req["action"].(string)
		if !ok || action == "" {
			log.Error("Missing or invalid action in scheduler request", zap.Any("value", req["action"]))
			http.Error(w, "missing or invalid action", http.StatusBadRequest)
			return
		}
		authCtx := contextx.Auth(ctx)
		roles := authCtx.Roles
		isSystem := false
		for _, r := range roles {
			if r == "system" || r == "admin" {
				isSystem = true
				break
			}
		}
		var campaignID int64
		if v, ok := req["campaign_id"]; ok {
			switch vv := v.(type) {
			case float64:
				campaignID = int64(vv)
			case int64:
				campaignID = vv
			}
		}
		switch action {
		case "create_job":
			if !isSystem {
				http.Error(w, "forbidden: system/admin role required", http.StatusForbidden)
				return
			}
			var job schedulerpb.Job
			if jobMap, ok := req["job"].(map[string]interface{}); ok {
				jobBytes, err := json.Marshal(jobMap)
				if err != nil {
					log.Error("Failed to marshal jobMap", zap.Error(err))
					http.Error(w, "invalid job field", http.StatusBadRequest)
					return
				}
				if err := json.Unmarshal(jobBytes, &job); err != nil {
					log.Error("Failed to unmarshal job JSON", zap.Error(err))
					http.Error(w, "invalid job field", http.StatusBadRequest)
					return
				}
			}
			// Enrich metadata with calendar structure for UI rendering
			if m, ok := req["metadata"].(map[string]interface{}); ok {
				if ss, ok := m["service_specific"].(map[string]interface{}); ok {
					if sched, ok := ss["scheduler"].(map[string]interface{}); ok {
						calendar := map[string]interface{}{
							"event_type":   sched["event_type"],
							"start_time":   sched["start_time"],
							"end_time":     sched["end_time"],
							"recurrence":   sched["recurrence"],
							"timezone":     sched["timezone"],
							"participants": sched["participants"],
							"location":     sched["location"],
							"notes":        sched["notes"],
							"color":        sched["color"],    // for UI coloring
							"conflict":     sched["conflict"], // for UI conflict indication
						}
						sched["calendar"] = calendar
						ss["scheduler"] = sched
						m["service_specific"] = ss
						req["metadata"] = m
					}
				}
			}
			protoReq := &schedulerpb.CreateJobRequest{Job: &job, CampaignId: campaignID}
			resp, err := schedulerSvc.CreateJob(ctx, protoReq)
			if err != nil {
				log.Error("Failed to create job", zap.Error(err))
				http.Error(w, "failed to create job", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("Failed to write JSON response", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "update_job":
			if !isSystem {
				http.Error(w, "forbidden: system/admin role required", http.StatusForbidden)
				return
			}
			var job schedulerpb.Job
			if jobMap, ok := req["job"].(map[string]interface{}); ok {
				jobBytes, err := json.Marshal(jobMap)
				if err != nil {
					log.Error("Failed to marshal jobMap", zap.Error(err))
					http.Error(w, "invalid job field", http.StatusBadRequest)
					return
				}
				if err := json.Unmarshal(jobBytes, &job); err != nil {
					log.Error("Failed to unmarshal job JSON", zap.Error(err))
					http.Error(w, "invalid job field", http.StatusBadRequest)
					return
				}
			}
			// Enrich metadata with calendar structure for UI rendering
			if m, ok := req["metadata"].(map[string]interface{}); ok {
				if ss, ok := m["service_specific"].(map[string]interface{}); ok {
					if sched, ok := ss["scheduler"].(map[string]interface{}); ok {
						calendar := map[string]interface{}{
							"event_type":   sched["event_type"],
							"start_time":   sched["start_time"],
							"end_time":     sched["end_time"],
							"recurrence":   sched["recurrence"],
							"timezone":     sched["timezone"],
							"participants": sched["participants"],
							"location":     sched["location"],
							"notes":        sched["notes"],
							"color":        sched["color"],    // for UI coloring
							"conflict":     sched["conflict"], // for UI conflict indication
						}
						sched["calendar"] = calendar
						ss["scheduler"] = sched
						m["service_specific"] = ss
						req["metadata"] = m
					}
				}
			}
			protoReq := &schedulerpb.UpdateJobRequest{Job: &job, CampaignId: campaignID}
			resp, err := schedulerSvc.UpdateJob(ctx, protoReq)
			if err != nil {
				log.Error("Failed to update job", zap.Error(err))
				http.Error(w, "failed to update job", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("Failed to write JSON response", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "delete_job":
			if !isSystem {
				http.Error(w, "forbidden: system/admin role required", http.StatusForbidden)
				return
			}
			jobID, ok := req["job_id"].(string)
			if !ok || jobID == "" {
				log.Error("Missing or invalid job_id in delete_job", zap.Any("value", req["job_id"]))
				http.Error(w, "missing or invalid job_id", http.StatusBadRequest)
				return
			}
			protoReq := &schedulerpb.DeleteJobRequest{JobId: jobID, CampaignId: campaignID}
			resp, err := schedulerSvc.DeleteJob(ctx, protoReq)
			if err != nil {
				log.Error("Failed to delete job", zap.Error(err))
				http.Error(w, "failed to delete job", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("Failed to write JSON response", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "get_job":
			jobID, ok := req["job_id"].(string)
			if !ok || jobID == "" {
				log.Error("Missing or invalid job_id in get_job", zap.Any("value", req["job_id"]))
				http.Error(w, "missing or invalid job_id", http.StatusBadRequest)
				return
			}
			protoReq := &schedulerpb.GetJobRequest{JobId: jobID, CampaignId: campaignID}
			resp, err := schedulerSvc.GetJob(ctx, protoReq)
			if err != nil {
				log.Error("Failed to get job", zap.Error(err))
				http.Error(w, "failed to get job", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("Failed to write JSON response", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "list_jobs":
			page := int32(1)
			if v, ok := req["page"].(float64); ok {
				page = int32(v)
			}
			pageSize := int32(20)
			if v, ok := req["page_size"].(float64); ok {
				pageSize = int32(v)
			}
			var status string
			if v, ok := req["status"].(string); ok {
				status = v
			} else {
				status = ""
			}
			protoReq := &schedulerpb.ListJobsRequest{Page: page, PageSize: pageSize, Status: status, CampaignId: campaignID}
			resp, err := schedulerSvc.ListJobs(ctx, protoReq)
			if err != nil {
				log.Error("Failed to list jobs", zap.Error(err))
				http.Error(w, "failed to list jobs", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("Failed to write JSON response", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "run_job":
			jobID, ok := req["job_id"].(string)
			if !ok || jobID == "" {
				log.Error("Missing or invalid job_id in run_job", zap.Any("value", req["job_id"]))
				http.Error(w, "missing or invalid job_id", http.StatusBadRequest)
				return
			}
			protoReq := &schedulerpb.RunJobRequest{JobId: jobID, CampaignId: campaignID}
			resp, err := schedulerSvc.RunJob(ctx, protoReq)
			if err != nil {
				log.Error("Failed to run job", zap.Error(err))
				http.Error(w, "failed to run job", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("Failed to write JSON response", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		case "list_job_runs":
			jobID, ok := req["job_id"].(string)
			if !ok || jobID == "" {
				log.Error("Missing or invalid job_id in list_job_runs", zap.Any("value", req["job_id"]))
				http.Error(w, "missing or invalid job_id", http.StatusBadRequest)
				return
			}
			page := int32(1)
			if v, ok := req["page"].(float64); ok {
				page = int32(v)
			}
			pageSize := int32(20)
			if v, ok := req["page_size"].(float64); ok {
				pageSize = int32(v)
			}
			protoReq := &schedulerpb.ListJobRunsRequest{JobId: jobID, Page: page, PageSize: pageSize, CampaignId: campaignID}
			resp, err := schedulerSvc.ListJobRuns(ctx, protoReq)
			if err != nil {
				log.Error("Failed to list job runs", zap.Error(err))
				http.Error(w, "failed to list job runs", http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				log.Error("Failed to write JSON response", zap.Error(err))
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
		default:
			log.Error("Unknown action in scheduler_ops", zap.String("action", action))
			http.Error(w, "unknown action", http.StatusBadRequest)
			return
		}
	}
}
