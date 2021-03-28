package main

import (
	"flag"
	"log"
	"sync"

	"github.com/erkexzcx/stalkerhek/hls"
	"github.com/erkexzcx/stalkerhek/proxy"
	"github.com/erkexzcx/stalkerhek/stalker"
)

var flagConfig = flag.String("config", "stalkerhek.yml", "path to the config file")

func main() {
	// Change flags on the default logger, so it print's line numbers as well.
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	flag.Parse()

	// Load configuration from file into Portal struct
	c, err := stalker.ReadConfig(flagConfig)
	if err != nil {
		log.Fatalln(err)
	}

	// Authenticate (connect) to Stalker portal and keep-alive it's connection.
	log.Println("Connecting to Stalker middleware...")
	if err = c.Portal.Start(); err != nil {
		log.Fatalln(err)
	}

	// Retrieve channels list.
	log.Println("Retrieving channels list from Stalker middleware...")
	channels, err := c.Portal.RetrieveChannels()
	if err != nil {
		log.Fatalln(err)
	}
	if len(channels) == 0 {
		log.Fatalln("no IPTV channels retrieved from Stalker middleware. quitting...")
	}

	var wg sync.WaitGroup

	if c.HLS.Enabled {
		wg.Add(1)
		go func() {
			log.Println("Starting HLS service...")
			hls.Start(channels, c.HLS.Bind)
			wg.Done()
		}()
	}

	if c.Proxy.Enabled {
		wg.Add(1)
		go func() {
			log.Println("Starting proxy service...")
			proxy.Start(c, channels)
			wg.Done()
		}()
	}

	wg.Wait()
}
