package hls

import (
	"log"
	"net/http"
	"sort"
	"sync"

	"github.com/erkexzcx/stalkerhek/stalker"
)

var userAgent string

// Start starts web server and serves playlist
func Start(chs map[string]*stalker.Channel, bind string) {

	// Initialize playlist
	playlist = make(map[string]*Channel)
	sortedChannels = make([]string, 0, len(chs))
	for k, v := range chs {
		playlist[k] = &Channel{
			StalkerChannel: v,
			Mux:            sync.Mutex{},
			Logo:           v.Logo(),
			Genre:          v.Genre(),
		}
		sortedChannels = append(sortedChannels, k)
	}
	sort.Strings(sortedChannels)

	http.HandleFunc("/iptv", playlistHandler)
	http.HandleFunc("/iptv/", channelHandler)
	http.HandleFunc("/logo/", logoHandler)

	log.Println("HLS service should be started!")
	panic(http.ListenAndServe(bind, nil))
}
