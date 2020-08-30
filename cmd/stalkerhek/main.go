package main

import (
	"flag"
	"log"

	"github.com/erkexzcx/stalkerhek/internal/proxy"

	"github.com/erkexzcx/stalkerhek/internal/config"

	"github.com/erkexzcx/stalkerhek/pkg/stalker"
)

var flagBind = flag.String("bind", "0.0.0.0:8987", "bind IP and port")
var flagConfig = flag.String("config", "stalkerhek.yml", "path to the config file")

func main() {
	// Change flags on the default logger, so it print's line numbers as well.
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	flag.Parse()

	// Load configuration from file.
	c, err := config.Load(flagConfig)
	if err != nil {
		log.Fatalln(err)
	}

	// Create new portal object out of configuration file.
	portal := stalker.Portal{
		Model:        c.Model,
		SerialNumber: c.SerialNumber,
		DeviceID:     c.DeviceID,
		DeviceID2:    c.DeviceID2,
		Signature:    c.Signature,
		MAC:          c.MAC,
		Username:     c.Username,
		Password:     c.Password,
		Location:     c.Location,
		TimeZone:     c.TimeZone,
		Token:        c.Token,
	}

	// Authenticate (connect) to Stalker portal and keep-alive it's connection.
	if err = portal.Start(); err != nil {
		log.Fatalln(err)
	}

	// Retrieve channels list.
	channels, err := portal.RetrieveChannels()
	if err != nil {
		log.Fatalln(err)
	}

	// Start web server
	proxy.Start(channels, flagBind)
}
