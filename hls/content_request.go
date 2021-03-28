package hls

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
)

// ContentRequest represents HTTP request that is received from the user
type ContentRequest struct {
	ResponseWriter http.ResponseWriter
	Request        *http.Request

	Title      string
	Suffix     string
	ChannelRef *Channel

	Channel Channel
}

// Returns ContentRequest objected that contains HTTP request, its responseWriter and TV channel reference.
func getContentRequest(w http.ResponseWriter, r *http.Request, expectedPrefix string) (*ContentRequest, error) {
	reqPath := strings.Replace(r.URL.RequestURI(), expectedPrefix, "", 1)
	reqPathParts := strings.SplitN(reqPath, "/", 2)
	if len(reqPathParts) == 0 {
		return nil, errors.New("bad request")
	}

	// Unescape channel title
	var err error
	reqPathParts[0], err = url.PathUnescape(reqPathParts[0])
	if err != nil {
		return nil, err
	}

	// Find channel reference
	channelRef, ok := playlist[reqPathParts[0]]
	if !ok {
		return nil, errors.New("bad request")
	}

	// /iptv/<channel>
	if len(reqPathParts) == 1 {
		return &ContentRequest{
			ResponseWriter: w,
			Request:        r,
			Title:          reqPathParts[0],
			Suffix:         "",
			ChannelRef:     channelRef,
		}, nil
	}

	// /iptv/<channel>/<something_more>
	return &ContentRequest{
		ResponseWriter: w,
		Request:        r,
		Title:          reqPathParts[0],
		Suffix:         reqPathParts[1],
		ChannelRef:     channelRef,
	}, nil
}
