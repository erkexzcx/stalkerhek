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
	Mux         *sync.Mutex
	Link        string // Link to channel's URL
	Content     []byte // Logo contents
	ContentType string // Logo's "Content-type" header value
}

type Stream struct {
	Mux           *sync.Mutex
	StalkerStream stalker.Stream

	Link        string    // Temporary link retrieved from Stalker portal
	ContentType int       // Temporary link's content type (e.g. HLS, MP4 etc...)
	accessTime  time.Time // Used for last access tracking, so we know when to consider this stream to be expired

	HLSLink     string // For HLS content types - stores HLS link
	HLSLinkRoot string // For HLS content types - stores link's upper path

	Title string // For ITV only - title
}

func (s *Stream) validate() error {
	if !s.isValid() {
		newLink, err := s.StalkerStream.GenerateLink()
		if err != nil {
			return err
		}

		s.Link = newLink
		s.ContentType = 0
	}

	s.accessTime = time.Now()
	return nil
}

func (c *Stream) isValid() bool {
	// If channel has never been accessed
	if c.accessTime.IsZero() {
		return false
	}

	// 30 seconds timout for HLS content
	if c.ContentType == contentTypeHLS {
		return time.Since(c.accessTime).Seconds() <= 30
	}

	// 5 seconds for everything else
	return time.Since(c.accessTime).Seconds() <= 5
}
