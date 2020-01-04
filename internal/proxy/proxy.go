package proxy

import (
	"log"
	"net/http"
	"strings"

	"github.com/erkexzcx/stalkerhek/pkg/stalker"
)

var stalkerChannels map[string]*stalker.Channel

var channels map[string]*m3u8Channel

// Start starts listening for requests. Eventually it starts a proxy server.
func Start(chs map[string]*stalker.Channel) {
	stalkerChannels = chs
	channels = make(map[string]*m3u8Channel, len(chs))

	http.HandleFunc("/iptv", playlistHandler)
	http.HandleFunc("/iptv/", m3u8Handler)

	log.Println("Started!")
	log.Fatal(http.ListenAndServe(":8987", nil))
}

func deleteAfterLastSlash(str string) string {
	return str[0 : strings.LastIndex(str, "/")+1]
}
