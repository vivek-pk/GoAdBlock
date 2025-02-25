package api

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vivek-pk/goadblock/internal/dns"
)

//go:embed templates/*
var templateFS embed.FS

type Query struct {
	ID        string    `json:"id"`
	Domain    string    `json:"domain"`
	Blocked   bool      `json:"blocked"`
	Timestamp time.Time `json:"timestamp"`
}

type APIServer struct {
	dnsServer     *dns.Server
	port          int
	startTime     time.Time
	recentQueries []Query
	queriesLock   sync.RWMutex
	templates     *template.Template
	server        *http.Server
}

func NewAPIServer(dnsServer *dns.Server, port int) (*APIServer, error) {
	tmpl, err := template.ParseFS(templateFS, "templates/*.html")
	if err != nil {
		return nil, err
	}

	return &APIServer{
		dnsServer:     dnsServer,
		port:          port,
		startTime:     time.Now(),
		recentQueries: make([]Query, 0, 100),
		templates:     tmpl,
	}, nil
}

// Add method to track queries
func (s *APIServer) AddQuery(domain string, blocked bool) {
	s.queriesLock.Lock()
	defer s.queriesLock.Unlock()

	query := Query{
		ID:        uuid.New().String(),
		Domain:    domain,
		Blocked:   blocked,
		Timestamp: time.Now(),
	}

	// Add to front of slice
	s.recentQueries = append([]Query{query}, s.recentQueries...)

	// Keep only last 100 queries
	if len(s.recentQueries) > 100 {
		s.recentQueries = s.recentQueries[:100]
	}
}

// Add new handler for queries
func (s *APIServer) handleQueries(w http.ResponseWriter, r *http.Request) {
	s.queriesLock.RLock()
	defer s.queriesLock.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"queries": s.recentQueries,
	})
}

func (s *APIServer) Start() error {
	mux := http.NewServeMux()

	// Serve dashboard
	mux.HandleFunc("/", s.handleDashboard)

	// API endpoints
	mux.HandleFunc("/api/v1/metrics", s.handleMetrics)
	mux.HandleFunc("/api/v1/status", s.handleStatus)
	mux.HandleFunc("/api/v1/queries", s.handleQueries)

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

func (s *APIServer) handleDashboard(w http.ResponseWriter, r *http.Request) {
	s.templates.ExecuteTemplate(w, "dashboard.html", nil)
}

func (s *APIServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := s.dnsServer.GetMetrics()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

func (s *APIServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"status": "running",
		"uptime": time.Since(s.startTime).String(),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// Add SetDNSServer method
func (s *APIServer) SetDNSServer(server *dns.Server) {
	s.dnsServer = server
}
