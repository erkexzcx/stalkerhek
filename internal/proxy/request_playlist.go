package proxy

import (
	"fmt"
	"net/http"
	"net/url"
)

func playlistHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "audio/x-mpegurl; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	fmt.Fprintln(w, "#EXTM3U")
	for title := range playlist {
		link := "http://" + r.Host + "/iptv/" + url.PathEscape(title)
		logo := "http://" + r.Host + "/logo/" + url.PathEscape(title)

		fmt.Fprintf(w, "#EXTINF:-1 tvg-logo=\"%s\" group-title=\"%s\", %s\n%s\n", logo, playlist[title].Genre, title, link)
	}
}
