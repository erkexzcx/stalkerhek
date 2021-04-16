package hls

import (
	"sync"
	"time"

	"github.com/erkexzcx/stalkerhek/stalker"
)

const (
	contentTypeUnknown = 0 // default
	contentTypeHLS     = 1
	contentTypeMedia   = 2
)

// Logo stores TV channel logo details.
type Logo struct {
	Mux              *sync.Mutex
	Link             string // Link to channel's URL
	Cache            []byte // Actual logo
	CacheContentType string // Logo type
}

// Channel stores TV channel details.
type Channel struct {
	StalkerChannel *stalker.Channel // Reference to Stalker channel

	Mux *sync.Mutex // Mux for channel.

	Link        string // Original link, retrieved from Stalkerhek middleware
	ContentType int    // Original link's content type, e.g. HLS, MP4...

	HLSLink     string // Updated HLS TV channel's link
	HLSLinkRoot string // Used for HLS relative paths

	lastAccess time.Time // Last access time of this channel, so we know when to request new channel from Stalker middleware

	Logo *Logo // Reference to channel's logo

	Genre string // TV channel genre. This field does not require synchronization
}

func (c *Channel) validate() error {
	if !c.isValid() {
		newLink, err := c.StalkerChannel.NewLink()
		if err != nil {
			return err
		}

		c.Link = newLink
		c.ContentType = 0
	}

	c.lastAccess = time.Now()
	return nil
}

func (c *Channel) isValid() bool {
	// If channel has never been accessed
	if c.lastAccess.IsZero() {
		return false
	}

	// 30 seconds timout for HLS content
	if c.ContentType == contentTypeHLS {
		return time.Since(c.lastAccess).Seconds() <= 30
	}

	// 5 seconds for everything else
	return time.Since(c.lastAccess).Seconds() <= 5
}
