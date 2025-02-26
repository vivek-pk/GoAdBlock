package api

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"sort"
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

type HourlyStats struct {
	Requests int
	Blocks   int
}

type ClientStats struct {
	IP             string    `json:"ip"`
	TotalQueries   int64     `json:"totalQueries"`
	BlockedQueries int64     `json:"blockedQueries"`
	LastSeen       time.Time `json:"lastSeen"`
}

type APIServer struct {
	dnsServer     *dns.Server
	port          int
	startTime     time.Time
	recentQueries []Query
	queriesLock   sync.RWMutex
	templates     *template.Template
	server        *http.Server
	hourlyStats   [24]HourlyStats
	hourlyStatsMu sync.RWMutex
	lastHourIndex int
	clientStats   map[string]*ClientStats
	clientStatsMu sync.RWMutex
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
		clientStats:   make(map[string]*ClientStats),
	}, nil
}

// Add method to track queries
func (s *APIServer) AddQuery(domain string, clientIP string, blocked bool) {
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

	s.trackQuery(blocked)
	s.trackClientQuery(clientIP, blocked)
}

func (s *APIServer) trackQuery(blocked bool) {
	s.hourlyStatsMu.Lock()
	defer s.hourlyStatsMu.Unlock()

	currentHour := time.Now().Hour()
	if currentHour != s.lastHourIndex {
		// Roll over to new hour
		s.hourlyStats[currentHour] = HourlyStats{}
		s.lastHourIndex = currentHour
	}

	s.hourlyStats[currentHour].Requests++
	if blocked {
		s.hourlyStats[currentHour].Blocks++
	}
}

func (s *APIServer) trackClientQuery(ip string, blocked bool) {
	s.clientStatsMu.Lock()
	defer s.clientStatsMu.Unlock()

	stats, exists := s.clientStats[ip]
	if !exists {
		stats = &ClientStats{
			IP: ip,
		}
		s.clientStats[ip] = stats
	}

	stats.TotalQueries++
	if blocked {
		stats.BlockedQueries++
	}
	stats.LastSeen = time.Now()
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

func (s *APIServer) handleHourlyStats(w http.ResponseWriter, r *http.Request) {
	s.hourlyStatsMu.RLock()
	defer s.hourlyStatsMu.RUnlock()

	currentHour := time.Now().Hour()
	hours := make([]string, 24)
	requests := make([]int, 24)
	blocks := make([]int, 24)

	for i := 0; i < 24; i++ {
		hour := (currentHour - 23 + i + 24) % 24
		hours[i] = fmt.Sprintf("%02d:00", hour)
		stats := s.hourlyStats[hour]
		requests[i] = stats.Requests
		blocks[i] = stats.Blocks
	}

	response := map[string]interface{}{
		"hours":    hours,
		"requests": requests,
		"blocks":   blocks,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *APIServer) handleClients(w http.ResponseWriter, r *http.Request) {
	s.clientStatsMu.RLock()
	defer s.clientStatsMu.RUnlock()

	clients := make([]*ClientStats, 0, len(s.clientStats))
	for _, stats := range s.clientStats {
		clients = append(clients, stats)
	}

	// Sort by last seen, most recent first
	sort.Slice(clients, func(i, j int) bool {
		return clients[i].LastSeen.After(clients[j].LastSeen)
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"clients": clients,
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
	mux.HandleFunc("/api/v1/stats/hourly", s.handleHourlyStats)
	mux.HandleFunc("/api/v1/clients", s.handleClients)

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
	response := map[string]interface{}{
		"totalQueries":   metrics.TotalQueries,
		"blockedQueries": metrics.BlockedQueries,
		"cacheHits":      metrics.CacheHits,
		"cacheMisses":    metrics.CacheMisses,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode metrics", http.StatusInternalServerError)
		return
	}
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
