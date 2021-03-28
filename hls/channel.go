package hls

import (
	"sync"
	"time"

	"github.com/erkexzcx/stalkerhek/stalker"
)

const (
	linkTypeUnknown = 0 // default
	linkTypeHLS     = 1
	linkTypeMedia   = 2
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

	Link     string // Original link, retrieved from Stalkerhek middleware
	LinkType int    // Original link's type

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
		c.LinkType = 0
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
	if c.LinkType == linkTypeHLS {
		return time.Since(c.lastAccess).Seconds() <= 30
	}

	// 5 seconds for everything else
	return time.Since(c.lastAccess).Seconds() <= 5
}
