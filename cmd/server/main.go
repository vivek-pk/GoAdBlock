package main

import (
	"log"

	"github.com/vivek-pk/goadblock/internal/blocker"
	"github.com/vivek-pk/goadblock/internal/dns"
)

func main() {
	adblocker := blocker.New()
	err := adblocker.LoadFromURL("https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts")
	if err != nil {
		log.Fatalf("Failed to load ad domains: %v", err)
	}

	server := dns.NewServer(adblocker)
	log.Println("Starting GoAdBlock DNS server on :53")
	log.Println("Using StevenBlack's hosts file for ad blocking")

	if err := server.Start(":53"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
