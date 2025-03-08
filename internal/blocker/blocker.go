package blocker

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
)

// BlockList represents a named collection of blocked domains
type BlockList struct {
	Name    string
	Domains map[string]struct{}
	Count   int
}

// Blocker holds domain blocking information
type Blocker struct {
	blocklists     map[string]*BlockList
	whitelist      map[string]struct{}
	blockRegexes   []*regexp.Regexp
	mu             sync.RWMutex
	blocklistStats map[string]int // Track blocks per blocklist
}

// New creates a new Blocker
func New() *Blocker {
	return &Blocker{
		blocklists:     make(map[string]*BlockList),
		whitelist:      make(map[string]struct{}),
		blockRegexes:   make([]*regexp.Regexp, 0),
		blocklistStats: make(map[string]int),
	}
}

// Update the IsBlocked method to return both a boolean and a reason string
func (b *Blocker) IsBlocked(domain string) (bool, string) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	domain = strings.ToLower(domain)
	domain = strings.TrimSuffix(domain, ".") // Remove trailing dot which DNS queries often have

	// Check whitelist first
	if _, ok := b.whitelist[domain]; ok {
		log.Printf("Domain %s is whitelisted, allowing", domain)
		return false, ""
	}

	// Check exact domain match in blocklists
	for listName, list := range b.blocklists {
		if _, ok := list.Domains[domain]; ok {
			log.Printf("Domain %s found in blocklist %s", domain, listName)
			b.blocklistStats[listName]++
			return true, listName
		}

		// Check parent domains (subdomains)
		parts := strings.Split(domain, ".")
		for i := 1; i < len(parts); i++ {
			parentDomain := strings.Join(parts[i:], ".")
			if _, ok := list.Domains[parentDomain]; ok {
				log.Printf("Domain %s matched parent domain %s in blocklist %s",
					domain, parentDomain, listName)
				b.blocklistStats[listName]++
				return true, listName
			}
		}
	}

	// Check regex patterns
	for _, regex := range b.blockRegexes {
		if regex.MatchString(domain) {
			log.Printf("Domain %s matched regex pattern: %s", domain, regex.String())
			return true, "regex:" + regex.String()
		}
	}

	log.Printf("Domain %s not found in any blocklist, allowing", domain)
	return false, ""
}

// LoadFromURL loads blocked domains from a URL
func (b *Blocker) LoadFromURL(url string, name string) error {
	if name == "" {
		name = url // Use URL as name if not provided
	}

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return b.loadFromReader(resp.Body, name)
}

func (b *Blocker) loadFromReader(reader io.Reader, listName string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Create new blocklist or get existing one
	list, exists := b.blocklists[listName]
	if !exists {
		list = &BlockList{
			Name:    listName,
			Domains: make(map[string]struct{}),
		}
		b.blocklists[listName] = list
		b.blocklistStats[listName] = 0
	}

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse hosts file format (0.0.0.0 example.com or 127.0.0.1 example.com)
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			domain := strings.ToLower(fields[1])
			list.Domains[domain] = struct{}{}
		}
	}

	// Update count
	list.Count = len(list.Domains)

	return scanner.Err()
}

// LoadMultipleLists loads multiple blocklists
func (b *Blocker) LoadMultipleLists(sources map[string]string) error {
	for name, url := range sources {
		if err := b.LoadFromURL(url, name); err != nil {
			return fmt.Errorf("failed to load blocklist %s: %w", name, err)
		}
	}
	return nil
}

// AddToWhitelist adds a domain to the whitelist
func (b *Blocker) AddToWhitelist(domain string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	domain = strings.ToLower(domain)
	b.whitelist[domain] = struct{}{}
}

// RemoveFromWhitelist removes a domain from the whitelist
func (b *Blocker) RemoveFromWhitelist(domain string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	domain = strings.ToLower(domain)
	delete(b.whitelist, domain)
}

// IsWhitelisted checks if a domain is whitelisted
func (b *Blocker) IsWhitelisted(domain string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	domain = strings.ToLower(domain)
	_, ok := b.whitelist[domain]
	return ok
}

// AddBlockRegex adds a regex pattern for blocking
func (b *Blocker) AddBlockRegex(pattern string) error {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.blockRegexes = append(b.blockRegexes, regex)
	return nil
}

// RemoveBlockRegex removes a regex pattern by its string representation
func (b *Blocker) RemoveBlockRegex(pattern string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Find and remove the regex pattern
	for i, regex := range b.blockRegexes {
		if regex.String() == pattern {
			b.blockRegexes = append(b.blockRegexes[:i], b.blockRegexes[i+1:]...)
			break
		}
	}
}

// GetBlocklistStats returns statistics about blocklists
func (b *Blocker) GetBlocklistStats() map[string]map[string]int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	stats := make(map[string]map[string]int)

	for name, list := range b.blocklists {
		stats[name] = map[string]int{
			"domains": list.Count,
			"blocks":  b.blocklistStats[name],
		}
	}

	return stats
}

// GetWhitelist returns the current whitelist
func (b *Blocker) GetWhitelist() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	whitelist := make([]string, 0, len(b.whitelist))
	for domain := range b.whitelist {
		whitelist = append(whitelist, domain)
	}

	return whitelist
}

// GetRegexPatterns returns the current regex patterns
func (b *Blocker) GetRegexPatterns() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	patterns := make([]string, len(b.blockRegexes))
	for i, regex := range b.blockRegexes {
		patterns[i] = regex.String()
	}

	return patterns
}

// AddDomainToBlocklist adds a domain to a specific blocklist
func (b *Blocker) AddDomainToBlocklist(domain, listName string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	domain = strings.ToLower(domain)

	// Create blocklist if it doesn't exist
	if _, exists := b.blocklists[listName]; !exists {
		b.blocklists[listName] = &BlockList{
			Name:    listName,
			Domains: make(map[string]struct{}),
		}
		b.blocklistStats[listName] = 0
	}

	b.blocklists[listName].Domains[domain] = struct{}{}
	b.blocklists[listName].Count = len(b.blocklists[listName].Domains)
}

// RemoveDomainFromBlocklist removes a domain from a specific blocklist
func (b *Blocker) RemoveDomainFromBlocklist(domain, listName string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	domain = strings.ToLower(domain)

	// Check if blocklist exists
	list, exists := b.blocklists[listName]
	if !exists {
		return false
	}

	// Check if domain exists in blocklist
	if _, ok := list.Domains[domain]; !ok {
		return false
	}

	// Remove domain
	delete(list.Domains, domain)
	list.Count = len(list.Domains)

	return true
}
