package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/vivek-pk/goadblock/internal/dns"
)

type APIServer struct {
	dnsServer *dns.Server
	port      int
	startTime time.Time
	server    *http.Server
}

func NewAPIServer(dnsServer *dns.Server, port int) *APIServer {
	return &APIServer{
		dnsServer: dnsServer,
		port:      port,
		startTime: time.Now(),
	}
}

func (s *APIServer) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/metrics", s.handleMetrics)
	mux.HandleFunc("/api/v1/status", s.handleStatus)

	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return s.server.ListenAndServe()
}

func (s *APIServer) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *APIServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := s.dnsServer.GetMetrics()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

func (s *APIServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"status": "running",
		"uptime": "todo", // We'll add uptime tracking later
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}
