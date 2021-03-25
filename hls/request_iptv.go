package hls

import (
	"log"
	"net/http"
	"time"
)

func channelHandler(w http.ResponseWriter, r *http.Request) {
	cr, err := getContentRequest(w, r, "/iptv/")
	if err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Lock this channel, so no other routine can request upstream servers
	cr.Channel.Mux.Lock()
	defer cr.Channel.Mux.Unlock()

	if cr.validSession() {
		cr.Channel.sessionUpdated = time.Now()
	} else {
		if err := cr.updateChannel(); err != nil {
			log.Println("Failed to update channel:", err)
		}
	}

	switch cr.Channel.LinkType {
	case linkTypeUnknown:
		handleContentUnknown(w, r, cr)
	case linkTypeM3U8:
		handleContentM3U8(w, r, cr)
	case linkTypeMedia:
		handleContentMedia(w, r, cr)
	default:
		http.Error(w, "invalid media type", http.StatusInternalServerError)
	}
}
