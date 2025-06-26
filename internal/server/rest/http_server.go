package server

import (
	"net/http"
	"strings"
	"time"

	gozap "go.uber.org/zap"

	"github.com/nmxmxh/master-ovasabi/internal/server/handlers"
	"github.com/nmxmxh/master-ovasabi/internal/server/httputil"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/metaversion"
)

// StartHTTPServer sets up and returns the HTTP server with the specified address. The caller is responsible for starting and stopping it.
func StartHTTPServer(log *gozap.Logger, container *di.Container, httpAddr string) *http.Server {
	mux := http.NewServeMux()
	// WebSocket endpoints are now handled by the standalone ws-gateway service.
	// If you need to interact with WebSockets, use the ws-gateway at /ws and /ws/{campaign_id}/{user_id}.
	// ws.RegisterWebSocketHandlers(mux, log, container, nil)

	mux.HandleFunc("/api/media/upload", handlers.MediaOpsHandler(container))
	mux.HandleFunc("/api/campaigns/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/state") && r.Method == http.MethodGet {
			if strings.Contains(r.URL.Path, "/user/") {
				handlers.CampaignUserStateHandler(container)(w, r)
				return
			}
			handlers.CampaignStateHandler(container)(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/leaderboard") && r.Method == http.MethodGet {
			handlers.CampaignLeaderboardHandler(container)(w, r)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("/api/campaign", handlers.CampaignOpsHandler(container))
	mux.HandleFunc("/api/notification", handlers.NotificationHandler(container)) // Still NotificationHandler, not NotificationOpsHandler
	mux.HandleFunc("/api/referral", handlers.ReferralOpsHandler(container))
	mux.HandleFunc("/api/content", handlers.ContentOpsHandler(container))
	mux.HandleFunc("/api/analytics", handlers.AnalyticsOpsHandler(container))
	mux.HandleFunc("/api/product", handlers.ProductOpsHandler(container))
	mux.HandleFunc("/api/commerce", handlers.CommerceOpsHandler(container))
	mux.HandleFunc("/api/user/auth", handlers.UserOpsHandler(container))
	mux.HandleFunc("/api/localization", handlers.LocalizationOpsHandler(container))
	mux.HandleFunc("/api/talent", handlers.TalentOpsHandler(container))
	mux.HandleFunc("/api/admin", handlers.AdminOpsHandler(container))
	mux.HandleFunc("/api/search", handlers.SearchOpsHandler(container))

	// Register the NexusOpsHandler for /api/nexus
	mux.Handle("/api/nexus", handlers.NewNexusOpsHandler(container, log))

	// --- INJECT METAVERSION MIDDLEWARE HERE ---
	// In production, pass evaluator from main server setup.
	// For now, use a default evaluator with no flags for demonstration.
	evaluator := metaversion.NewOpenFeatureEvaluator([]string{"new_ui", "beta_api"})
	wrappedMux := httputil.MetaversionMiddleware(evaluator, log)(mux)

	server := &http.Server{
		Addr:              httpAddr,
		Handler:           wrappedMux,
		ReadHeaderTimeout: 10 * time.Second, // Mitigate Slowloris
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	return server
}
