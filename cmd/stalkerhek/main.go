package stalkerhek

import (
	"log"

	"github.com/erkexzcx/stalkerhek/internal/proxy"

	"github.com/erkexzcx/stalkerhek/internal/config"

	"github.com/erkexzcx/stalkerhek/pkg/stalker"
)

var portal *stalker.Portal

// Init initiates everything
func Init() {
	log.Println("Starting...")

	// Load config from file
	c, err := config.LoadConfig()
	if err != nil {
		log.Fatalln(err)
	}

	// Validate config file
	if err := c.Validate(); err != nil {
		log.Fatalln("Invalid config:", err)
	}

	// Get stalker portal out of configuration
	portal = c.StalkerPortal()

	// Connect to stalker portal and do what's necesarry
	log.Println("Connecting to Stalker portal...")
	if err = portal.Start(); err != nil {
		log.Fatalln("Failed to reserve token:", err)
	}

	// Retrieve channels list
	log.Println("Retrieving channels list from Stalker portal...")
	channels, err := portal.RetrieveChannels()
	if err != nil {
		log.Fatalln(err)
	}

	// Start web server
	proxy.Start(channels)
}
