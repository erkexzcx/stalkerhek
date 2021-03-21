package main

import (
	"flag"
	"log"

	"github.com/erkexzcx/stalkerhek/proxy"
	"github.com/erkexzcx/stalkerhek/stalker"
)

var flagBind = flag.String("bind", "0.0.0.0:8987", "bind IP and port")
var flagConfig = flag.String("config", "stalkerhek.yml", "path to the config file")

func main() {
	// Change flags on the default logger, so it print's line numbers as well.
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	flag.Parse()

	// Load configuration from file into Portal struct
	p, err := stalker.ReadConfig(flagConfig)
	if err != nil {
		log.Fatalln(err)
	}

	// Authenticate (connect) to Stalker portal and keep-alive it's connection.
	if err = p.Start(); err != nil {
		log.Fatalln(err)
	}

	// Retrieve channels list.
	channels, err := p.RetrieveChannels()
	if err != nil {
		log.Fatalln(err)
	}

	// Start web server
	proxy.Start(channels, flagBind)
}
