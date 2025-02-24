package dns

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/vivek-pk/goadblock/internal/blocker"
)

// findAvailablePort finds an available UDP port
func findAvailablePort() (int, error) {
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenUDP("udp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()

	return l.LocalAddr().(*net.UDPAddr).Port, nil
}

func setupTestServer(t *testing.T) (*Server, string, func()) {
	port, err := findAvailablePort()
	if err != nil {
		t.Fatalf("Failed to find available port: %v", err)
	}

	adblocker := blocker.New()
	_ = adblocker.LoadFromURL("https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts")

	server := NewServer(adblocker)
	addr := fmt.Sprintf(":%d", port)
	errChan := make(chan error, 1)

	go func() {
		if err := server.Start(addr); err != nil {
			errChan <- err
		}
	}()

	// Wait for server to start
	startTimeout := time.After(5 * time.Second)
	for {
		select {
		case err := <-errChan:
			t.Fatalf("Server failed to start: %v", err)
		case <-startTimeout:
			t.Fatal("Server startup timed out")
		case <-time.After(100 * time.Millisecond):
			if isServerReady(port) {
				return server, fmt.Sprintf("127.0.0.1:%d", port), func() {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					server.Shutdown(ctx)
				}
			}
		}
	}
}

func isServerReady(port int) bool {
	c := &dns.Client{
		Timeout: 500 * time.Millisecond,
	}
	m := new(dns.Msg)
	m.SetQuestion("google.com.", dns.TypeA)

	_, _, err := c.Exchange(m, fmt.Sprintf("127.0.0.1:%d", port))
	return err == nil
}

func TestDNSServer(t *testing.T) {
	_, addr, cleanup := setupTestServer(t)
	defer cleanup()

	// Configure DNS client with timeout
	c := &dns.Client{
		Timeout: 2 * time.Second,
	}

	tests := []struct {
		name        string
		domain      string
		qtype       uint16
		shouldBlock bool
	}{
		{"Known ad domain A", "doubleclick.net.", dns.TypeA, true},
		{"Known ad domain AAAA", "doubleclick.net.", dns.TypeAAAA, true},
		{"Google ads domain", "googleadservices.com.", dns.TypeA, true},
		{"Regular domain", "google.com.", dns.TypeA, false},
		{"Another regular domain", "github.com.", dns.TypeA, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(dns.Msg)
			m.SetQuestion(tt.domain, tt.qtype)

			// Retry logic for DNS queries
			var resp *dns.Msg
			var err error
			for retries := 3; retries > 0; retries-- {
				resp, _, err = c.Exchange(m, addr)
				if err == nil {
					break
				}
				time.Sleep(100 * time.Millisecond)
			}

			if err != nil {
				t.Fatalf("Query failed: %v", err)
			}

			if len(resp.Answer) == 0 {
				t.Fatal("Expected answer section in response")
			}

			switch tt.qtype {
			case dns.TypeA:
				if a, ok := resp.Answer[0].(*dns.A); ok {
					isZeroIP := a.A.Equal(net.IPv4(0, 0, 0, 0))
					if tt.shouldBlock != isZeroIP {
						t.Errorf("Expected blocked=%v for %s, got IP=%v",
							tt.shouldBlock, tt.domain, a.A)
					}
				}
			case dns.TypeAAAA:
				if aaaa, ok := resp.Answer[0].(*dns.AAAA); ok {
					isZeroIP := aaaa.AAAA.Equal(net.IPv6zero)
					if tt.shouldBlock != isZeroIP {
						t.Errorf("Expected blocked=%v for %s, got IP=%v",
							tt.shouldBlock, tt.domain, aaaa.AAAA)
					}
				}
			}
		})
	}
}

func TestCaching(t *testing.T) {
	server, addr, cleanup := setupTestServer(t)
	defer cleanup()

	domain := "example.com."
	metrics := server.GetMetrics()
	initialMisses := metrics.CacheMisses

	// Make first query
	m := new(dns.Msg)
	m.SetQuestion(domain, dns.TypeA)
	c := new(dns.Client)

	// First query - should miss cache
	_, _, err := c.Exchange(m, addr)
	if err != nil {
		t.Fatalf("First query failed: %v", err)
	}

	// Second query - should hit cache
	_, _, err = c.Exchange(m, addr)
	if err != nil {
		t.Fatalf("Second query failed: %v", err)
	}

	if metrics.CacheHits != 1 {
		t.Errorf("Expected 1 cache hit, got %d", metrics.CacheHits)
	}
	if metrics.CacheMisses != initialMisses+1 {
		t.Errorf("Expected %d cache misses, got %d", initialMisses+1, metrics.CacheMisses)
	}
}
