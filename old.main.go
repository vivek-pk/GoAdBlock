package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/miekg/dns"
)

var (
	adDomains   = make(map[string]bool)
	adDomainsRW sync.RWMutex
)

func loadAdDomains(url string) error {
	// Download domains from StevenBlack's hosts file
	log.Printf("Downloading hosts file from %s", url)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download hosts file: %v", err)
	}
	defer resp.Body.Close()

	var domainCount int
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse hosts file format (0.0.0.0 domain.com or 127.0.0.1 domain.com)
		fields := strings.Fields(line)
		if len(fields) >= 2 && (fields[0] == "0.0.0.0" || fields[0] == "127.0.0.1") {
			domain := strings.ToLower(fields[1])

			// Skip localhost entries
			if domain == "localhost" || strings.HasSuffix(domain, ".local") {
				continue
			}

			adDomainsRW.Lock()
			adDomains[domain] = true
			adDomainsRW.Unlock()
			domainCount++
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading hosts file: %v", err)
	}

	log.Printf("Successfully loaded %d ad domains", domainCount)
	return nil
}

func isAdDomain(domain string) bool {
	// Remove the trailing dot from DNS queries and convert to lowercase
	domain = strings.ToLower(strings.TrimSuffix(domain, "."))

	adDomainsRW.RLock()
	defer adDomainsRW.RUnlock()

	// Check exact match
	if adDomains[domain] {
		return true
	}

	// Check if domain is a subdomain of any blocked domain
	parts := strings.Split(domain, ".")
	for i := 0; i < len(parts)-1; i++ {
		parentDomain := strings.Join(parts[i:], ".")
		if adDomains[parentDomain] {
			return true
		}
	}

	return false
}

func logDNSQuery(domain string, isBlocked bool, clientIP net.IP) {
	status := "allowed"
	if isBlocked {
		status = "blocked"
	}
	log.Printf("DNS Query from %s: %s - %s", clientIP, domain, status)
}

func handleDNSRequest(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = false

	switch r.Opcode {
	case dns.OpcodeQuery:
		for _, q := range m.Question {
			switch q.Qtype {
			case dns.TypeA:
				clientIP, _, _ := net.SplitHostPort(w.RemoteAddr().String())
				isBlocked := isAdDomain(q.Name)
				logDNSQuery(q.Name, isBlocked, net.ParseIP(clientIP))

				if isBlocked {
					// Return 0.0.0.0 for blocked domains
					m.Answer = append(m.Answer, &dns.A{
						Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 60},
						A:   net.IPv4(0, 0, 0, 0),
					})
				} else {
					// Forward non-blocked queries to Google's DNS
					resp, err := dns.Exchange(r, "8.8.8.8:53")
					if err == nil && resp != nil {
						m.Answer = resp.Answer
					}
				}
			}
		}
	}

	w.WriteMsg(m)
}

func main() {
	// Load ad domains from StevenBlack's hosts file
	hostsURL := "https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts"
	err := loadAdDomains(hostsURL)
	if err != nil {
		log.Fatalf("Failed to load ad domains: %v", err)
	}

	// Create DNS server
	server := &dns.Server{Addr: ":53", Net: "udp"}
	dns.HandleFunc(".", handleDNSRequest)

	log.Println("Starting GoAdBlock DNS server on :53")
	log.Println("Using StevenBlack's hosts file for ad blocking")
	err = server.ListenAndServe()
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
