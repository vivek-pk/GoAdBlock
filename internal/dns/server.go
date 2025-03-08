package dns

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/miekg/dns"
	"github.com/vivek-pk/goadblock/internal/blocker"
)

// Server represents a DNS server
type Server struct {
	blocker         *blocker.Blocker
	notifier        BlockNotifier
	server          *dns.Server
	cache           *DNSCache
	upstreamAddrs   []string
	currentUpstream int
	metrics         *Metrics
	shutdown        chan struct{}
	apiNotifier     APINotifier
	Ready           chan struct{}
	blockingMode    string
	blockingIP      net.IP
}

type ServerConfig struct {
	UpstreamServers []string
	BlockingMode    string
	BlockingIP      string
	CacheSize       int
}

type DNSCache struct {
	entries map[string]*CacheEntry
	mu      sync.RWMutex
}

type CacheEntry struct {
	Answer    []dns.RR
	ExpiresAt time.Time
}

type Metrics struct {
	TotalQueries   int64
	BlockedQueries int64
	CacheHits      int64
	CacheMisses    int64
	mu             sync.RWMutex
}

type APINotifier interface {
	AddQuery(domain string, clientIP string, blocked bool)
}

// BlockNotifier is an interface for components that need to be notified of blocked domains
type BlockNotifier interface {
	OnDomainBlocked(domain string, clientIP string, reason string)
}

// Update NewServer function to accept config
func NewServer(blocker *blocker.Blocker, apiNotifier APINotifier, config ServerConfig) *Server {
	// Create default config if needed
	if len(config.UpstreamServers) == 0 {
		config.UpstreamServers = []string{
			"8.8.8.8:53", // Google
			"1.1.1.1:53", // Cloudflare
		}
	}
	if config.BlockingMode == "" {
		config.BlockingMode = "zero_ip"
	}
	if config.BlockingIP == "" {
		config.BlockingIP = "0.0.0.0"
	}
	if config.CacheSize <= 0 {
		config.CacheSize = 10000
	}

	return &Server{
		blocker:     blocker,
		apiNotifier: apiNotifier,
		cache: &DNSCache{
			entries: make(map[string]*CacheEntry, config.CacheSize),
		},
		upstreamAddrs: config.UpstreamServers,
		metrics:       &Metrics{},
		shutdown:      make(chan struct{}),
		Ready:         make(chan struct{}),
		blockingMode:  config.BlockingMode,
		blockingIP:    net.ParseIP(config.BlockingIP),
	}
}

// Backward compatibility wrapper
func NewServerSimple(blocker *blocker.Blocker, apiNotifier APINotifier) *Server {
	return NewServer(blocker, apiNotifier, ServerConfig{
		UpstreamServers: []string{
			"8.8.8.8:53", // Google
			"1.1.1.1:53", // Cloudflare
		},
		BlockingMode: "zero_ip",
		BlockingIP:   "0.0.0.0",
		CacheSize:    10000,
	})
}

func (s *Server) handleRequest(w dns.ResponseWriter, r *dns.Msg) {
	s.metrics.incrementTotal()

	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = false

	switch r.Opcode {
	case dns.OpcodeQuery:
		for _, q := range m.Question {
			switch q.Qtype {
			case dns.TypeA, dns.TypeAAAA:
				clientIP, _, _ := net.SplitHostPort(w.RemoteAddr().String())
				isBlocked, reason := s.blocker.IsBlocked(q.Name)
				log.Printf("DNS query: %s, blocked: %v, reason: %s", q.Name, isBlocked, reason)

				// Notify API server of query
				if s.apiNotifier != nil {
					s.apiNotifier.AddQuery(q.Name, clientIP, isBlocked)
				}

				if isBlocked {
					// Notify block listeners
					if s.notifier != nil {
						s.notifier.OnDomainBlocked(q.Name, clientIP, reason)
					}

					s.metrics.incrementBlocked()
					if q.Qtype == dns.TypeA {
						m.Answer = append(m.Answer, &dns.A{
							Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 60},
							A:   net.IPv4(0, 0, 0, 0), // Block by returning 0.0.0.0
						})
					} else {
						m.Answer = append(m.Answer, &dns.AAAA{
							Hdr:  dns.RR_Header{Name: q.Name, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: 60},
							AAAA: net.IPv6zero, // Block IPv6 too
						})
					}

					log.Printf("Blocked domain %s, returning null IP", q.Name)
				} else {
					// Check cache first
					if answer := s.checkCache(q.Name, q.Qtype); answer != nil {
						m.Answer = answer
						s.metrics.incrementCacheHit()
					} else {
						s.metrics.incrementCacheMiss()
						resp, err := s.queryUpstream(r)
						if err == nil && resp != nil {
							m.Answer = resp.Answer
							s.updateCache(q.Name, q.Qtype, resp.Answer)
						}
					}
				}
			}
		}
	}

	w.WriteMsg(m)
}

func (s *Server) queryUpstream(r *dns.Msg) (*dns.Msg, error) {
	// Round-robin through upstream servers
	s.currentUpstream = (s.currentUpstream + 1) % len(s.upstreamAddrs)
	return dns.Exchange(r, s.upstreamAddrs[s.currentUpstream])
}

func (s *Server) checkCache(name string, qtype uint16) []dns.RR {
	s.cache.mu.RLock()
	defer s.cache.mu.RUnlock()

	key := getCacheKey(name, qtype)
	if entry, exists := s.cache.entries[key]; exists && time.Now().Before(entry.ExpiresAt) {
		return entry.Answer
	}
	return nil
}

func (s *Server) updateCache(name string, qtype uint16, answer []dns.RR) {
	if len(answer) == 0 {
		return
	}

	s.cache.mu.Lock()
	defer s.cache.mu.Unlock()

	// Cache for 5 minutes
	s.cache.entries[getCacheKey(name, qtype)] = &CacheEntry{
		Answer:    answer,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
}

func getCacheKey(name string, qtype uint16) string {
	return fmt.Sprintf("%s:%d", name, qtype)
}

// Metrics methods
func (m *Metrics) incrementTotal() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TotalQueries++
	log.Printf("Total queries: %d", m.TotalQueries) // Debug log
}

func (m *Metrics) incrementBlocked() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.BlockedQueries++
	log.Printf("Blocked queries: %d", m.BlockedQueries) // Debug log
}

func (m *Metrics) incrementCacheHit() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CacheHits++
	log.Printf("Cache hits: %d", m.CacheHits) // Debug log
}

func (m *Metrics) incrementCacheMiss() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CacheMisses++
	log.Printf("Cache misses: %d", m.CacheMisses) // Debug log
}

func (s *Server) GetMetrics() *Metrics {
	return s.metrics
}

func (s *Server) Start(addr string) error {
	s.server = &dns.Server{Addr: addr, Net: "udp"}
	dns.HandleFunc(".", s.handleRequest)

	errChan := make(chan error, 1)
	go func() {
		errChan <- s.server.ListenAndServe()
	}()

	// Signal ready after successful bind
	close(s.Ready)

	// Wait for either shutdown signal or error
	select {
	case <-s.shutdown:
		return nil
	case err := <-errChan:
		return err
	}
}

func (s *Server) Shutdown(ctx context.Context) error {
	// Signal shutdown
	close(s.shutdown)

	// Shutdown the DNS server
	if s.server != nil {
		return s.server.Shutdown()
	}
	return nil
}

func logQuery(domain string, isBlocked bool, clientIP net.IP) {
	status := "allowed"
	if isBlocked {
		status = "blocked"
	}
	log.Printf("DNS Query from %s: %s - %s", clientIP, domain, status)
}

// Add this method to your DNS Server struct
func (s *Server) GetBlocker() *blocker.Blocker {
	return s.blocker
}
