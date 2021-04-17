package proxy

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"

	"github.com/erkexzcx/stalkerhek/hls"
)

func handleRewriteITV(w http.ResponseWriter, r *http.Request, keyCMD string) {
	hls.PlaylistMux.RLock()
	defer hls.PlaylistMux.RUnlock()

	// Find ITV using CMD
	title, found := hls.PlaylistCMD2Title[keyCMD]
	if !found {
		log.Println("STB requested 'create_link' of type 'itv', but gave invalid CMD string:", keyCMD)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	channel := hls.Playlist[title]

	// Must give full path to IPTV stream
	destinationHost := config.Proxy.RewriteTo
	if config.Proxy.RewriteTo != "" {
		requestHost, _, _ := net.SplitHostPort(r.Host)
		_, portHLS, _ := net.SplitHostPort(config.HLS.Bind)
		destinationHost = requestHost + ":" + portHLS
	}
	destination = "http://" + destinationHost + "/iptv/" + url.PathEscape(title)

	responseText := channel.StalkerStream.GenerateRewrittenResponse(destination)
	fmt.Println(responseText)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(responseText))
}

func handleRewriteTVArchive(w http.ResponseWriter, r *http.Request, keyCMD string) {
	randomHash := generateHash(32)
	hls.NewGeneratedStream(randomHash, &TVArchive{})

}
