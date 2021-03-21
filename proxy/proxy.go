package proxy

import (
	"log"
	"net/http"
	"sync"

	"github.com/erkexzcx/stalkerhek/stalker"
)

var userAgent string

// Start starts web server and serves playlist
func Start(chs map[string]*stalker.Channel, flagBind *string) {

	playlist = make(map[string]*Channel)
	for k, v := range chs {
		playlist[k] = &Channel{
			StalkerChannel: v,
			Mux:            sync.Mutex{},
			Logo:           v.Logo(),
			Genre:          v.Genre(),
		}
	}

	// Some global vars
	m3u8channels = make(map[string]*M3U8Channel, len(chs))

	http.HandleFunc("/iptv", playlistHandler)
	http.HandleFunc("/iptv/", channelHandler)
	http.HandleFunc("/logo/", logoHandler)

	log.Println("Web server should be started!")

	panic(http.ListenAndServe(*flagBind, nil))
}
