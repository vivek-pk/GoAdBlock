package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vivek-pk/goadblock/internal/api"
	"github.com/vivek-pk/goadblock/internal/blocker"
	"github.com/vivek-pk/goadblock/internal/dns"
)

func main() {
	// Initialize ad blocker
	adblocker := blocker.New()
	err := adblocker.LoadFromURL("https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts")
	if err != nil {
		log.Fatalf("Failed to load ad domains: %v", err)
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start DNS server
	dnsServer := dns.NewServer(adblocker)
	dnsErrChan := make(chan error, 1)
	go func() {
		log.Println("Starting GoAdBlock DNS server on :53")
		if err := dnsServer.Start(":53"); err != nil {
			dnsErrChan <- err
		}
	}()

	// Start API server
	apiServer := api.NewAPIServer(dnsServer, 8080)
	apiErrChan := make(chan error, 1)
	go func() {
		log.Println("Starting API server on :8080")
		if err := apiServer.Start(); err != nil {
			apiErrChan <- err
		}
	}()

	// Wait for shutdown signal or error
	select {
	case <-sigChan:
		log.Println("Received shutdown signal")
		// Give services 5 seconds to shutdown gracefully
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		if err := apiServer.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down API server: %v", err)
		}
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
