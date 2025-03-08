package api

import (
	"encoding/json"
	"net/http"
)

type DomainRequest struct {
	Domain string `json:"domain"`
	List   string `json:"list"`
}

type RegexRequest struct {
	Pattern string `json:"pattern"`
}

// HandleGetBlocklists returns all blocklists
func (s *APIServer) handleGetBlocklists(w http.ResponseWriter, r *http.Request) {
	stats := s.dnsServer.GetBlocker().GetBlocklistStats()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// HandleAddDomainToBlocklist adds a domain to a blocklist
func (s *APIServer) handleAddDomainToBlocklist(w http.ResponseWriter, r *http.Request) {
	var req DomainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Domain == "" || req.List == "" {
		http.Error(w, "Domain and list name are required", http.StatusBadRequest)
		return
	}

	s.dnsServer.GetBlocker().AddDomainToBlocklist(req.Domain, req.List)

	w.WriteHeader(http.StatusCreated)
}

// HandleRemoveDomainFromBlocklist removes a domain from a blocklist
func (s *APIServer) handleRemoveDomainFromBlocklist(w http.ResponseWriter, r *http.Request) {
	var req DomainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Domain == "" || req.List == "" {
		http.Error(w, "Domain and list name are required", http.StatusBadRequest)
		return
	}

	if !s.dnsServer.GetBlocker().RemoveDomainFromBlocklist(req.Domain, req.List) {
		http.Error(w, "Domain not found in blocklist", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// HandleGetWhitelist returns the current whitelist
func (s *APIServer) handleGetWhitelist(w http.ResponseWriter, r *http.Request) {
	whitelist := s.dnsServer.GetBlocker().GetWhitelist()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(whitelist)
}

// HandleAddToWhitelist adds a domain to the whitelist
func (s *APIServer) handleAddToWhitelist(w http.ResponseWriter, r *http.Request) {
	var req DomainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Domain == "" {
		http.Error(w, "Domain is required", http.StatusBadRequest)
		return
	}

	s.dnsServer.GetBlocker().AddToWhitelist(req.Domain)

	w.WriteHeader(http.StatusCreated)
}

// HandleRemoveFromWhitelist removes a domain from the whitelist
func (s *APIServer) handleRemoveFromWhitelist(w http.ResponseWriter, r *http.Request) {
	var req DomainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Domain == "" {
		http.Error(w, "Domain is required", http.StatusBadRequest)
		return
	}

	s.dnsServer.GetBlocker().RemoveFromWhitelist(req.Domain)

	w.WriteHeader(http.StatusOK)
}

// HandleGetRegexPatterns returns all regex blocking patterns
func (s *APIServer) handleGetRegexPatterns(w http.ResponseWriter, r *http.Request) {
	patterns := s.dnsServer.GetBlocker().GetRegexPatterns()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(patterns)
}

// HandleAddRegexPattern adds a regex blocking pattern
func (s *APIServer) handleAddRegexPattern(w http.ResponseWriter, r *http.Request) {
	var req RegexRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Pattern == "" {
		http.Error(w, "Pattern is required", http.StatusBadRequest)
		return
	}

	if err := s.dnsServer.GetBlocker().AddBlockRegex(req.Pattern); err != nil {
		http.Error(w, "Invalid regex pattern: "+err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// HandleRemoveRegexPattern removes a regex blocking pattern
func (s *APIServer) handleRemoveRegexPattern(w http.ResponseWriter, r *http.Request) {
	var req RegexRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Pattern == "" {
		http.Error(w, "Pattern is required", http.StatusBadRequest)
		return
	}

	s.dnsServer.GetBlocker().RemoveBlockRegex(req.Pattern)

	w.WriteHeader(http.StatusOK)
}
