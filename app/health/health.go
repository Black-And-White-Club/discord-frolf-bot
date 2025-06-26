package health

import (
	"encoding/json"
	"net/http"
	"time"
)

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
	Uptime    string    `json:"uptime"`
}

// Handler provides health check endpoints
type Handler struct {
	startTime time.Time
	version   string
}

// NewHandler creates a new health check handler
func NewHandler(version string) *Handler {
	return &Handler{
		startTime: time.Now(),
		version:   version,
	}
}

// Health returns the health status of the application
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   h.version,
		Uptime:    time.Since(h.startTime).String(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// Ready returns the readiness status of the application
func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	// For now, we're always ready if we can respond
	// In the future, this could check dependencies like database, NATS, etc.
	response := map[string]string{
		"status": "ready",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// StartServer starts the health check HTTP server
func (h *Handler) StartServer(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", h.Health)
	mux.HandleFunc("/ready", h.Ready)

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	return server.ListenAndServe()
}
