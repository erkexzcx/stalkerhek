package proxy

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"

	"github.com/erkexzcx/stalkerhek/hls"
)

func generateITVResponse(link, id, ch_id string) string {
	return `{"js":{"id":"` + id + `","cmd":"` + specialLinkEscape(link) + `","streamer_id":0,"link_id":` + ch_id + `,"load":0,"error":""},"text":"array(6) {\n  [\"id\"]=>\n  string(4) \"` + id + `\"\n  [\"cmd\"]=>\n  string(99) \"` + specialLinkEscape(link) + `\"\n  [\"streamer_id\"]=>\n  int(0)\n  [\"link_id\"]=>\n  int(` + ch_id + `)\n  [\"load\"]=>\n  int(0)\n  [\"error\"]=>\n  string(0) \"\"\n}\ngenerated in: 0.01s; query counter: 8; cache hits: 0; cache miss: 0; php errors: 0; sql errors: 0;"}`
}

func handleRewriteITV(w http.ResponseWriter, r *http.Request, simplifiedQuery map[string]string) {
	hls.PlaylistMux.RLock()
	defer hls.PlaylistMux.RUnlock()

	keyCMD := simplifiedQuery["cmd"]
	channel, found := hls.PlaylistCMD[keyCMD]
	if !found {
		log.Println("STB requested 'create_link' of type 'itv', but gave invalid CMD string:", keyCMD)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Must give full path to IPTV stream
	destinationHost := config.Proxy.RewriteTo
	if config.Proxy.RewriteTo != "" {
		requestHost, _, _ := net.SplitHostPort(r.Host)
		_, portHLS, _ := net.SplitHostPort(config.HLS.Bind)
		destinationHost = requestHost + ":" + portHLS
	}
	destination = "http://" + destinationHost + "/iptv/" + url.PathEscape(channel.StalkerChannel.Title)

	responseText := generateITVResponse(destination, channel.StalkerChannel.CMD_ID, channel.StalkerChannel.CMD_CH_ID)
	fmt.Println(responseText)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(responseText))
}
