package hls

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
)

// Handles '/iptv' requests
func playlistHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "audio/x-mpegurl; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	fmt.Fprintln(w, "#EXTM3U")
	for _, title := range sortedChannels {
		link := "/iptv/" + url.PathEscape(title)
		logo := "/logo/" + url.PathEscape(title)

		fmt.Fprintf(w, "#EXTINF:-1 tvg-logo=\"%s\" group-title=\"%s\", %s\n%s\n", logo, playlist[title].Genre, title, link)
	}
}

// Handles '/iptv/' requests
func channelHandler(w http.ResponseWriter, r *http.Request) {
	cr, err := getContentRequest(w, r, "/iptv/")
	if err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Lock channel's mux
	cr.ChannelRef.Mux.Lock()

	// Keep track on channel access time
	if err = cr.ChannelRef.validate(); err != nil {
		cr.ChannelRef.Mux.Unlock()
		http.Error(w, "internal server error", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	// Handle content
	handleContent(cr)
}

// Handles '/logo/' requests
func logoHandler(w http.ResponseWriter, r *http.Request) {
	cr, err := getContentRequest(w, r, "/logo/")
	if err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Lock
	cr.ChannelRef.Logo.Mux.Lock()

	// Retrieve from Stalker middleware if no cache is present
	if len(cr.ChannelRef.Logo.Cache) == 0 {
		img, contentType, err := download(cr.ChannelRef.Logo.Link)
		if err != nil {
			cr.ChannelRef.Logo.Mux.Unlock()
			http.Error(w, "internal server error", http.StatusInternalServerError)
			log.Println(err)
			return
		}
		cr.ChannelRef.Logo.Cache = img
		cr.ChannelRef.Logo.CacheContentType = contentType
	}

	// Create local copy so we don't need thread syncrhonization
	logo := *cr.ChannelRef.Logo

	// Unlock
	cr.ChannelRef.Logo.Mux.Unlock()

	w.Header().Set("Content-Type", logo.CacheContentType)
	w.Write(logo.Cache)
}
