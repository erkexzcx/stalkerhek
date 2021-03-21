package proxy

import (
	"strings"
	"sync"
	"time"

	"github.com/erkexzcx/stalkerhek/stalker"
)

const (
	linkTypeUnknown = 0
	linkTypeM3U8    = 1
	linkTypeMedia   = 2
)

// Channel stores TV channel details.
type Channel struct {
	StalkerChannel *stalker.Channel // Reference to Stalker channel
	Mux            sync.Mutex

	LinkURL     string       // Actual link
	LinkType    int          // Default is 0 (unknown)
	LinkM3u8Ref *M3U8Channel // Reference. For non M3U8 channels it will be empty

	sessionUpdated time.Time

	LogoCache            []byte
	LogoCacheContentType string

	// Synchronization is not required for below fields
	Logo  string
	Genre string
}

var playlist map[string]*Channel

func getLinkType(contentType string) int {
	contentType = strings.ToLower(contentType)
	switch {
	case contentType == "application/vnd.apple.mpegurl" || contentType == "application/x-mpegurl":
		return linkTypeM3U8
	case strings.HasPrefix(contentType, "video/") || strings.HasPrefix(contentType, "audio/") || contentType == "application/octet-stream":
		return linkTypeMedia
	default:
		return linkTypeMedia
	}
}
