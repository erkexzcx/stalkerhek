package hls

import (
	"log"
	"net/http"
	"sort"
	"sync"

	"github.com/erkexzcx/stalkerhek/stalker"
)

var playlist map[string]*Channel
var sortedChannels []string

type generatableChannel struct {
}

// Start starts main routine.
func Start(c *stalker.Config, chs map[string]*stalker.Channel) {
	// Initialize playlist
	playlist = make(map[string]*Channel)
	sortedChannels = make([]string, 0, len(chs))
	for k, v := range chs {
		playlist[k] = &Channel{
			StalkerChannel: v,
			Mux:            &sync.Mutex{},
			Logo: &Logo{
				Mux:  &sync.Mutex{},
				Link: v.Logo(),
			},
			Genre: v.Genre(),
		}
		sortedChannels = append(sortedChannels, k)
	}
	sort.Strings(sortedChannels)

	mux := http.NewServeMux()
	mux.HandleFunc("/iptv", playlistHandler)
	mux.HandleFunc("/iptv/", channelHandler)
	mux.HandleFunc("/logo/", logoHandler)

	mux.HandleFunc("/generated/", generatedHandler)

	log.Println("HLS service should be started!")
	panic(http.ListenAndServe(c.HLS.Bind, mux))
}
