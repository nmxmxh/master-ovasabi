package server

import (
	"net/http"
	"sync"
	"time"

	"github.com/nmxmxh/master-ovasabi/internal/service"
	"go.uber.org/zap"
)

// StartHTTPServer sets up and starts the HTTP server in a goroutine.
func StartHTTPServer(log *zap.Logger, provider *service.Provider) {
	mux := http.NewServeMux()
	wsClients := &sync.Map{} // user_id -> *websocket.Conn (shared)
	RegisterMediaUploadHandlers(mux, log, provider, wsClients)
	RegisterWebSocketHandlers(mux, log, provider, wsClients)
	// Add other handlers as needed

	httpPort := ":8090" // Configurable if needed
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
