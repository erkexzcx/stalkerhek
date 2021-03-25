package hls

import (
	"net/http"
)

func handleContentUnknown(w http.ResponseWriter, r *http.Request, cr *ContentRequest) {
	resp, err := response(cr.Channel.LinkURL)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	cr.Channel.LinkType = getLinkType(resp.Header.Get("Content-Type"))
	switch cr.Channel.LinkType {
	case linkTypeMedia:
		handleEstablishedContentMedia(w, r, cr, resp)
	case linkTypeM3U8:
		// Create new M3u8 type channel
		cr.Channel.LinkM3u8Ref = &M3U8Channel{Channel: cr.Channel}
		cr.Channel.LinkM3u8Ref.link = resp.Request.URL.String()
		cr.Channel.LinkM3u8Ref.linkRoot = deleteAfterLastSlash(cr.Channel.LinkM3u8Ref.link)
		handleEstablishedContentM3U8(w, r, cr, resp, cr.Channel.LinkURL)
	default:
		http.Error(w, "invalid media type", http.StatusInternalServerError)
	}
}
