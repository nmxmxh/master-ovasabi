package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"go.uber.org/zap"

	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"google.golang.org/protobuf/encoding/protojson"
)

// NexusOpsHandler handles nexus-related actions via the "action" field.
//
// @Summary Nexus Operations
// @Description Handles nexus-related actions using the "action" field in the request body. Each action (e.g., orchestrate, introspect, etc.) has its own required/optional fields. All requests must include a 'metadata' field following the robust metadata pattern (see docs/services/metadata.md).
// @Tags nexus
// @Accept json
// @Produce json
// @Param request body object true "Composable request with 'action', required fields for the action, and 'metadata' (see docs/services/metadata.md for schema)"
// @Success 200 {object} object "Response depends on action"
// @Failure 400 {object} ErrorResponse
// @Router /api/nexus_ops [post]

// NexusOpsHandler handles /api/nexus/ops requests.
type NexusOpsHandler struct {
	Container *di.Container
	Log       *zap.Logger
}

func NewNexusOpsHandler(container *di.Container, log *zap.Logger) *NexusOpsHandler {
	return &NexusOpsHandler{Container: container, Log: log}
}

func (h *NexusOpsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var nexusClient nexusv1.NexusServiceClient
	if err := h.Container.Resolve(&nexusClient); err != nil {
		h.Log.Error("Failed to resolve NexusServiceClient", zap.Error(err))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	var req nexusv1.HandleOpsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Log.Warn("Failed to decode HandleOpsRequest", zap.Error(err))
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Before calling the gRPC client, ensure req.CampaignId is set from the incoming HTTP request body (if present).
	// For each action, set campaign_id in the proto request.
	// Example:
	if v, ok := req.Params["campaign_id"]; ok {
		if cid, err := strconv.ParseInt(v, 10, 64); err == nil {
			req.CampaignId = cid
		}
	}

	resp, err := nexusClient.HandleOps(ctx, &req)
	if err != nil {
		h.Log.Error("HandleOps gRPC call failed", zap.Error(err))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	enc := protojson.MarshalOptions{EmitUnpopulated: true}
	data, err := enc.Marshal(resp)
	if err != nil {
		h.Log.Error("Failed to marshal HandleOpsResponse", zap.Error(err))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if _, err := w.Write(data); err != nil {
		h.Log.Error("Failed to write response", zap.Error(err))
	}
}
