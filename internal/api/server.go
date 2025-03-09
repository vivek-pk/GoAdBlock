package api

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
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
	router        *mux.Router
	hourlyStats   [24]HourlyStats
	hourlyStatsMu sync.RWMutex
	lastHourIndex int
	clientStats   map[string]*ClientStats
	clientStatsMu sync.RWMutex
}

func NewAPIServer(dnsServer *dns.Server, port int) (*APIServer, error) {
	tmpl := InitTemplates()

	server := &APIServer{
		dnsServer:     dnsServer,
		port:          port,
		startTime:     time.Now(),
		recentQueries: make([]Query, 0, 100),
		templates:     tmpl,
		router:        mux.NewRouter(), // Initialize the router
		clientStats:   make(map[string]*ClientStats),
	}

	// Call setupRoutes to register all routes
	server.setupRoutes()

	// Set up static file serving from embedded files
	ServeStaticFiles(server.router)

	return server, nil
}

// In your API server code
func InitTemplates() *template.Template {
	tmpl := template.New("")

	// Parse sidebar template first to make it available to other templates
	template.Must(tmpl.ParseFS(embeddedFiles, "templates/sidebar.html"))

	// Then parse all remaining templates
	template.Must(tmpl.ParseFS(embeddedFiles, "templates/*.html"))

	return tmpl
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
	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      s.router, // Use the router
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return s.server.ListenAndServe()
}

func (s *APIServer) setupRoutes() {
	// Add this debug handler first to log incoming requests
	s.router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("Request: %s %s", r.Method, r.URL.Path)
			next.ServeHTTP(w, r)
		})
	})

	// IMPORTANT: Register static file handler BEFORE other routes
	staticFS, err := fs.Sub(embeddedFiles, "static")
	if err != nil {
		log.Fatalf("Failed to create sub-filesystem for static files: %v", err)
	}
	fileServer := http.FileServer(http.FS(staticFS))
	s.router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fileServer))

	// Then add your API and other routes
	s.router.HandleFunc("/", s.handleDashboard).Methods("GET")
	s.router.HandleFunc("/blocklists", s.handleBlocklistsPage).Methods("GET")
	s.router.HandleFunc("/settings", s.handleSettingsPage).Methods("GET")
	s.router.HandleFunc("/about", s.handleAboutPage).Methods("GET")

	// API endpoints
	s.router.HandleFunc("/api/v1/metrics", s.handleMetrics).Methods("GET")
	s.router.HandleFunc("/api/v1/status", s.handleStatus).Methods("GET")
	s.router.HandleFunc("/api/v1/queries", s.handleQueries).Methods("GET")
	s.router.HandleFunc("/api/v1/stats/hourly", s.handleHourlyStats).Methods("GET")
	s.router.HandleFunc("/api/v1/clients", s.handleClients).Methods("GET")

	// Blocklist management routes
	s.router.HandleFunc("/api/v1/blocklists", s.handleGetBlocklists).Methods("GET")
	s.router.HandleFunc("/api/v1/blocklist/domain", s.handleAddDomainToBlocklist).Methods("POST")
	s.router.HandleFunc("/api/v1/blocklist/domain", s.handleRemoveDomainFromBlocklist).Methods("DELETE")

	// Whitelist management routes
	s.router.HandleFunc("/api/v1/whitelist", s.handleGetWhitelist).Methods("GET")
	s.router.HandleFunc("/api/v1/whitelist", s.handleAddToWhitelist).Methods("POST")
	s.router.HandleFunc("/api/v1/whitelist", s.handleRemoveFromWhitelist).Methods("DELETE")

	// Regex pattern routes
	s.router.HandleFunc("/api/v1/regex", s.handleGetRegexPatterns).Methods("GET")
	s.router.HandleFunc("/api/v1/regex", s.handleAddRegexPattern).Methods("POST")
	s.router.HandleFunc("/api/v1/regex", s.handleRemoveRegexPattern).Methods("DELETE")

	// Add static file serving
	fs := http.FileServer(http.Dir("./internal/api/static"))
	s.router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))
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

// Add handler functions for each page
func (s *APIServer) handleBlocklistsPage(w http.ResponseWriter, r *http.Request) {
	s.templates.ExecuteTemplate(w, "blocklists.html", nil)
}

func (s *APIServer) handleSettingsPage(w http.ResponseWriter, r *http.Request) {
	s.templates.ExecuteTemplate(w, "settings.html", nil)
}

func (s *APIServer) handleAboutPage(w http.ResponseWriter, r *http.Request) {
	s.templates.ExecuteTemplate(w, "about.html", nil)
}
