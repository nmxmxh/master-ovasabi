package server

import (
	"net/http"
	"os"
	"strings"
	"time"

	gozap "go.uber.org/zap"

	"github.com/nmxmxh/master-ovasabi/internal/server/handlers"
	"github.com/nmxmxh/master-ovasabi/internal/server/httputil"
	ws "github.com/nmxmxh/master-ovasabi/internal/server/ws"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
	"github.com/nmxmxh/master-ovasabi/pkg/metaversion"
)

// StartHTTPServer sets up and starts the HTTP server in a goroutine.
// evaluator and logger should be provided from main server setup.
func StartHTTPServer(log *gozap.Logger, container *di.Container) {
	mux := http.NewServeMux()
	ws.RegisterWebSocketHandlers(mux, log, container, nil)

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
	mux.HandleFunc("/api/campaign", handlers.CampaignHandler(container))
	mux.HandleFunc("/api/notification", handlers.NotificationHandler(container))
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

	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = ":8090"
	}

	// --- INJECT METAVERSION MIDDLEWARE HERE ---
	// In production, pass evaluator from main server setup.
	// For now, use a default evaluator with no flags for demonstration.
	evaluator := metaversion.NewOpenFeatureEvaluator([]string{"new_ui", "beta_api"})
	wrappedMux := httputil.MetaversionMiddleware(evaluator, log)(mux)

	server := &http.Server{
		Addr:              httpPort,
		Handler:           wrappedMux,
		ReadHeaderTimeout: 10 * time.Second, // Mitigate Slowloris
	}
	go func() {
		log.Info("Starting HTTP server for REST/WebSocket", gozap.String("address", httpPort))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP server failed", gozap.Error(err))
		}
	}()
}
