package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	schedulerpb "github.com/nmxmxh/master-ovasabi/api/protos/scheduler/v1"
	"github.com/nmxmxh/master-ovasabi/internal/server/httputil"
	"github.com/nmxmxh/master-ovasabi/pkg/contextx"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"go.uber.org/zap"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
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
			httputil.WriteJSONError(w, log, http.StatusInternalServerError, "internal server error", err)
			return
		}
		if r.Method != http.MethodPost {
			httputil.WriteJSONError(w, log, http.StatusMethodNotAllowed, "method not allowed", nil)
			return
		}
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode scheduler request JSON", zap.Error(err))
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "invalid JSON", err)
			return
		}
		action, ok := req["action"].(string)
		if !ok || action == "" {
			log.Error("Missing or invalid action in scheduler request", zap.Any("value", req["action"]))
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "missing or invalid action", nil)
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

		actionHandlers := map[string]func(){
			"create_job": func() {
				if !isSystem {
					httputil.WriteJSONError(w, log, http.StatusForbidden, "forbidden: system/admin role required", nil)
					return
				}
				enrichSchedulerMetadata(req)
				handleSchedulerAction(w, ctx, log, req, &schedulerpb.CreateJobRequest{}, schedulerSvc.CreateJob)
			},
			"update_job": func() {
				if !isSystem {
					httputil.WriteJSONError(w, log, http.StatusForbidden, "forbidden: system/admin role required", nil)
					return
				}
				enrichSchedulerMetadata(req)
				handleSchedulerAction(w, ctx, log, req, &schedulerpb.UpdateJobRequest{}, schedulerSvc.UpdateJob)
			},
			"delete_job": func() {
				if !isSystem {
					httputil.WriteJSONError(w, log, http.StatusForbidden, "forbidden: system/admin role required", nil)
					return
				}
				handleSchedulerAction(w, ctx, log, req, &schedulerpb.DeleteJobRequest{}, schedulerSvc.DeleteJob)
			},
			"get_job": func() {
				handleSchedulerAction(w, ctx, log, req, &schedulerpb.GetJobRequest{}, schedulerSvc.GetJob)
			},
			"list_jobs": func() {
				handleSchedulerAction(w, ctx, log, req, &schedulerpb.ListJobsRequest{}, schedulerSvc.ListJobs)
			},
			"run_job": func() {
				handleSchedulerAction(w, ctx, log, req, &schedulerpb.RunJobRequest{}, schedulerSvc.RunJob)
			},
			"list_job_runs": func() {
				handleSchedulerAction(w, ctx, log, req, &schedulerpb.ListJobRunsRequest{}, schedulerSvc.ListJobRuns)
			},
		}

		if handler, found := actionHandlers[action]; found {
			handler()
		} else {
			log.Error("Unknown action in scheduler_ops", zap.String("action", action))
			httputil.WriteJSONError(w, log, http.StatusBadRequest, "unknown action", nil)
		}
	}
}

// enrichSchedulerMetadata adds a calendar structure to the metadata for UI rendering.
func enrichSchedulerMetadata(req map[string]interface{}) {
	if m, ok := req["metadata"].(map[string]interface{}); ok {
		if ss, ok := m["service_specific"].(map[string]interface{}); ok {
			if sched, ok := ss["scheduler"].(map[string]interface{}); ok {
				// Helper to safely get values from the map
				getVal := func(key string) interface{} {
					if val, exists := sched[key]; exists {
						return val
					}
					return nil
				}
				calendar := map[string]interface{}{
					"event_type":   getVal("event_type"),
					"start_time":   getVal("start_time"),
					"end_time":     getVal("end_time"),
					"recurrence":   getVal("recurrence"),
					"timezone":     getVal("timezone"),
					"participants": getVal("participants"),
					"location":     getVal("location"),
					"notes":        getVal("notes"),
					"color":        getVal("color"),    // for UI coloring
					"conflict":     getVal("conflict"), // for UI conflict indication
				}
				sched["calendar"] = calendar
				ss["scheduler"] = sched
				m["service_specific"] = ss
				req["metadata"] = m
			}
		}
	}
}

// handleSchedulerAction is a generic helper to reduce boilerplate in SchedulerOpsHandler.
func handleSchedulerAction[T proto.Message, U proto.Message](
	w http.ResponseWriter,
	ctx context.Context,
	log *zap.Logger,
	reqMap map[string]interface{},
	req T,
	svcFunc func(context.Context, T) (U, error),
) {
	if err := mapToProtoScheduler(reqMap, req); err != nil {
		httputil.WriteJSONError(w, log, http.StatusBadRequest, "invalid request body", err)
		return
	}

	resp, err := svcFunc(ctx, req)
	if err != nil {
		st, _ := status.FromError(err)
		httpStatus := httputil.GRPCStatusToHTTPStatus(st.Code())
		log.Error("scheduler service call failed", zap.Error(err), zap.String("grpc_code", st.Code().String()))
		httputil.WriteJSONError(w, log, httpStatus, st.Message(), nil)
		return
	}

	httputil.WriteJSONResponse(w, log, resp)
}

// mapToProtoScheduler converts a map[string]interface{} to a proto.Message using JSON as an intermediate.
func mapToProtoScheduler(data map[string]interface{}, v proto.Message) error {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return protojson.Unmarshal(jsonBytes, v)
}
