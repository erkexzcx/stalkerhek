package hls

import (
	"log"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/erkexzcx/stalkerhek/stalker"
)

var stalkerConfig *stalker.Config

var Playlist = make(map[string]*Channel)
var PlaylistSortedChannels = make([]string, 0)
var PlaylistMux = sync.RWMutex{} // Used to freeze all clients until playlist is updated

// Start starts main routine.
func Start(c *stalker.Config) {
	stalkerConfig = c

	// Generate once
	updateITVPlaylist()

	// Refresh ITV channels every 24 hours
	go func(){
		for {
			time.Sleep(24 * time.Hour)
			PlaylistMux.Lock()
			updateITVPlaylist()
			PlaylistMux.Unlock()
		}
	}()

	// // Every 1 hour clear old and no longer usable generated "channels" by Proxy service
	// go func(){
	// 	for {
	// 		time.Sleep(time.Hour)
	// 		generatedPlaylistMux.Lock()
	// 		cleanupGeneratedPlaylist()
	// 		generatedPlaylistMux.Unlock()
	// 	}
	// }()

	mux := http.NewServeMux()
	mux.HandleFunc("/iptv", playlistHandler)
	mux.HandleFunc("/iptv/", channelHandler)
	mux.HandleFunc("/logo/", logoHandler)
	mux.HandleFunc("/generated/", generatedHandler)

	log.Println("HLS service should be started!")
	panic(http.ListenAndServe(c.HLS.Bind, mux))
}

// This function initiates playlist
func updateITVPlaylist() {
	var channels map[string]*stalker.Channel
	var err error

	var allGood bool
	for i := 0; i < 3; i++ {
		channels, err = stalkerConfig.Portal.RetrieveChannels()
		if err != nil {
			log.Printf("Attempt %d/3: failed to retrieve channels list (%s)\n", i+1, err.Error())
			time.Sleep(time.Second)
			continue
		}
		if len(channels) == 0 {
			log.Printf("Attempt %d/3: failed to retrieve channels list (no channels returned)\n", i+1)\
			time.Sleep(time.Second)
			continue
		}
		allGood = true
		break
	}

	if !allGood {
		// TODO - is there anything else we can do?
		log.Fatalln("failed to retrieve channels list from Stalker portal")
	}

	for k, v := range channels {
		Playlist[k] = &Channel{
			StalkerChannel: v,
			Mux:            &sync.Mutex{},
			Logo: &Logo{
				Mux:  &sync.Mutex{},
				Link: v.Logo(),
			},
			Genre: v.Genre(),
		}
		PlaylistSortedChannels = append(PlaylistSortedChannels, k)
	}
	sort.Strings(PlaylistSortedChannels)
}
