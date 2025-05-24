package server

import (
	"net/http"
	"os"
	"time"

	"go.uber.org/zap"

	"github.com/nmxmxh/master-ovasabi/internal/server/handlers"
	ws "github.com/nmxmxh/master-ovasabi/internal/server/ws"
	"github.com/nmxmxh/master-ovasabi/pkg/di"
)

// StartHTTPServer sets up and starts the HTTP server in a goroutine.
func StartHTTPServer(log *zap.Logger, container *di.Container) {
	mux := http.NewServeMux()
	ws.RegisterWebSocketHandlers(mux, log, container, nil)

	mux.HandleFunc("/api/media/upload", handlers.MediaOpsHandler(log, container))
	mux.HandleFunc("/api/campaign", handlers.CampaignHandler(log, container))
	mux.HandleFunc("/api/notification", handlers.NotificationHandler(log, container))
	mux.HandleFunc("/api/referral", handlers.ReferralOpsHandler(log, container))
	mux.HandleFunc("/api/content", handlers.ContentOpsHandler(log, container))
	mux.HandleFunc("/api/analytics", handlers.AnalyticsOpsHandler(log, container))
	mux.HandleFunc("/api/product", handlers.ProductOpsHandler(log, container))
	mux.HandleFunc("/api/commerce", handlers.CommerceOpsHandler(log, container))
	mux.HandleFunc("/api/user/auth", handlers.UserOpsHandler(log, container))
	mux.HandleFunc("/api/localization", handlers.LocalizationOpsHandler(log, container))
	mux.HandleFunc("/api/talent", handlers.TalentOpsHandler(log, container))
	mux.HandleFunc("/api/admin", handlers.AdminOpsHandler(log, container))
	mux.HandleFunc("/api/search", handlers.SearchOpsHandler(log, container))

	// Register the NexusOpsHandler for /api/nexus
	mux.Handle("/api/nexus", handlers.NewNexusOpsHandler(container, log))

	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = ":8090"
	}
	server := &http.Server{
		Addr:              httpPort,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second, // Mitigate Slowloris
	}
	go func() {
		log.Info("Starting HTTP server for REST/WebSocket", zap.String("address", httpPort))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP server failed", zap.Error(err))
		}
	}()
}
