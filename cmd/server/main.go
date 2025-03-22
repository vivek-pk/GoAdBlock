package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vivek-pk/goadblock/internal/api"
	"github.com/vivek-pk/goadblock/internal/blocker"
	"github.com/vivek-pk/goadblock/internal/config"
	"github.com/vivek-pk/goadblock/internal/dns"
)

func main() {
	configErr := config.InitConfig()
	if configErr != nil {
		log.Fatalf("Failed to load configs : %v", configErr)
	}

	log.Printf("Configuration loaded - Using DNS port: %d, HTTP port: %d",
		config.GetDnsPort(), config.GetHttpPort())

	// Initialize ad blocker
	adblocker := blocker.New()

	// Load blocklists with debug info
	log.Println("Loading blocklists...")
	blocklists := map[string]string{
		"stevenblack": "https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts",
		"adaway":      "https://adaway.org/hosts.txt",
	}

	err := adblocker.LoadMultipleLists(blocklists)
	if err != nil {
		log.Fatalf("Failed to load blocklists: %v", err)
	}

	// Print stats after loading
	stats := adblocker.GetBlocklistStats()
	log.Printf("Loaded %d blocklists", len(stats))
	for name, stat := range stats {
		log.Printf("Blocklist %s: %d domains", name, stat["domains"])
	}

	// Add regex pattern for blocking
	err = adblocker.AddBlockRegex(`^ad[0-9]+\.example\.com$`)
	if err != nil {
		log.Fatalf("Failed to add block regex: %v", err)
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create API server first
	apiServer, err := api.NewAPIServer(nil, config.GetHttpPort())
	if err != nil {
		log.Fatalf("Failed to create API server: %v", err)
	}

	// Create DNS server with API notifier and config
	dnsConfig := dns.ServerConfig{
		UpstreamServers: []string{"8.8.8.8:53", "1.1.1.1:53"},
		BlockingMode:    "zero_ip",
		BlockingIP:      "0.0.0.0",
		CacheSize:       10000,
	}
	dnsServer := dns.NewServer(adblocker, apiServer, dnsConfig)

	// Update API server's DNS server reference
	apiServer.SetDNSServer(dnsServer)

	// Start servers one by one
	log.Printf("Starting DNS server on :%d", config.GetDnsPort())
	dnsErrChan := make(chan error, 1)
	go func() {
		if err := dnsServer.Start(fmt.Sprintf(":%d", config.GetDnsPort())); err != nil {
			dnsErrChan <- err
		}
	}()

	// Wait for DNS server to be ready
	select {
	case <-dnsServer.Ready:
		log.Println("DNS server started successfully")
	case err := <-dnsErrChan:
		log.Fatalf("Failed to start DNS server: %v", err)
	case <-time.After(5 * time.Second):
		log.Fatalf("DNS server startup timed out")
	}

	// Now start the API server
	log.Printf("Starting API server on :%d", config.GetHttpPort())
	apiErrChan := make(chan error, 1)
	go func() {
		if err := apiServer.Start(); err != nil {
			apiErrChan <- err
		}
	}()

	// Give API server time to initialize
	time.Sleep(500 * time.Millisecond)
	log.Println("API server started successfully")
	log.Println("GoAdBlock is running. Press Ctrl+C to stop.")

	// Wait for shutdown signal or error
	select {
	case <-sigChan:
		log.Println("Received shutdown signal")
		// Give services 5 seconds to shutdown gracefully
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		log.Println("Shutting down API server...")
		if err := apiServer.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down API server: %v", err)
		}

		log.Println("Shutting down DNS server...")
		if err := dnsServer.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down DNS server: %v", err)
		}

	case err := <-dnsErrChan:
		log.Fatalf("DNS server error: %v", err)
	case err := <-apiErrChan:
		log.Fatalf("API server error: %v", err)
	}

	log.Println("Servers shutdown complete")
}
