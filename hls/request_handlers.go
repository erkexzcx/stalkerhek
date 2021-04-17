package hls

import (
	"fmt"
	"net/http"
	"net/url"
)

// Handles '/iptv' requests
func playlistHandler(w http.ResponseWriter, r *http.Request) {
	PlaylistMux.RLock()
	defer PlaylistMux.RUnlock()

	w.Header().Set("Content-Type", "audio/x-mpegurl; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	fmt.Fprintln(w, "#EXTM3U")
	for _, title := range playlistSorted {
		link := "/iptv/" + url.PathEscape(title)
		fmt.Fprintf(w, "#EXTINF:-1 tvg-logo=\"\" group-title=\"\", %s\n%s\n", title, link)
	}
}

// Handles '/iptv/' requests
func channelHandler(w http.ResponseWriter, r *http.Request) {
	PlaylistMux.RLock()
	defer PlaylistMux.RUnlock()

	cr, err := getContentRequest(w, r, "/iptv/", true)
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
		return
	}

	// Handle content
	handleContent(cr)
}

// Handles '/generated/' requests
func generatedHandler(w http.ResponseWriter, r *http.Request) {
	cr, err := getContentRequest(w, r, "/generated/", false)
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
		return
	}

	// Handle content
	handleContent(cr)
}
