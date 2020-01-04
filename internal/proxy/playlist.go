package proxy

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
)

func playlistHandler(w http.ResponseWriter, r *http.Request) {
	// Sort map
	titles := make([]string, 0, len(stalkerChannels))
	for tvch := range stalkerChannels {
		titles = append(titles, tvch)
	}
	sort.Strings(titles)

	// Write HTTP headers
	//w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.WriteHeader(200)

	// Write HTTP body
	fmt.Fprintln(w, "#EXTM3U")
	for _, title := range titles {
		channel, _ := stalkerChannels[title]
		channelLink := "http://" + r.Host + "/iptv/" + url.PathEscape(title) + ".m3u8"
		fmt.Fprintf(w, "#EXTINF:-1 tvg-logo=\"%s\", %s\n%s\n", channel.Logo(), title, channelLink)
	}
}
