package blocker

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
)

type AdBlocker struct {
	domains   map[string]bool
	domainsRW sync.RWMutex
}

func New() *AdBlocker {
	return &AdBlocker{
		domains: make(map[string]bool),
	}
}

func (ab *AdBlocker) LoadFromURL(url string) error {
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
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 2 && (fields[0] == "0.0.0.0" || fields[0] == "127.0.0.1") {
			domain := strings.ToLower(fields[1])
			if domain == "localhost" || strings.HasSuffix(domain, ".local") {
				continue
			}

			ab.domainsRW.Lock()
			ab.domains[domain] = true
			ab.domainsRW.Unlock()
			domainCount++
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading hosts file: %v", err)
	}

	log.Printf("Successfully loaded %d ad domains", domainCount)
	return nil
}

func (ab *AdBlocker) IsBlocked(domain string) bool {
	domain = strings.ToLower(strings.TrimSuffix(domain, "."))

	ab.domainsRW.RLock()
	defer ab.domainsRW.RUnlock()

	if ab.domains[domain] {
		return true
	}

	parts := strings.Split(domain, ".")
	for i := 0; i < len(parts)-1; i++ {
		parentDomain := strings.Join(parts[i:], ".")
		if ab.domains[parentDomain] {
			return true
		}
	}

	return false
}
