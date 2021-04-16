package hls

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func handleContent(cr *ContentRequest) {
	linkType := cr.ChannelRef.ContentType

	if linkType == contentTypeUnknown {
		handleContentUnknown(cr)
		return
	}

	// At this point we will no longer modify channel details, so we get a copy of 'ChannelRef'
	// value and set to 'Channel' so we can avoid synchronization
	cr.Channel = *cr.ChannelRef
	cr.ChannelRef.Mux.Unlock()

	switch linkType {
	case contentTypeHLS:
		handleContentHLS(cr)
	case contentTypeMedia:
		handleContentMedia(cr)
	default:
		http.Error(cr.ResponseWriter, "invalid media type", http.StatusInternalServerError)
	}
}

// ####################################################

func handleContentUnknown(cr *ContentRequest) {
	resp, err := response(cr.ChannelRef.Link)
	if err != nil {
		cr.ChannelRef.Mux.Unlock()
		http.Error(cr.ResponseWriter, "internal server error", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	cr.ChannelRef.ContentType = getLinkType(resp.Header.Get("Content-Type"))

	if cr.ChannelRef.ContentType == contentTypeHLS {
		// Initiate new HLS channel
		cr.ChannelRef.HLSLink = resp.Request.URL.String()
		cr.ChannelRef.HLSLinkRoot = deleteAfterLastSlash(cr.ChannelRef.HLSLink)
	}

	handleContent(cr)
}

// ####################################################

func handleContentHLS(cr *ContentRequest) {
	var link string
	if cr.Suffix == "" {
		link = cr.Channel.HLSLink
	} else {
		link = cr.Channel.HLSLinkRoot + cr.Suffix
	}

	resp, err := response(link)
	if err != nil {
		http.Error(cr.ResponseWriter, "internal server error", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	handleEstablishedContentHLS(cr, resp, link)
}

func handleEstablishedContentHLS(cr *ContentRequest, resp *http.Response, link string) {
	prefix := "http://" + cr.Request.Host + "/iptv/" + url.PathEscape(cr.Title) + "/"

	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	switch {
	case contentType == "application/vnd.apple.mpegurl" || contentType == "application/x-mpegurl": // HLS metadata
		content := rewriteLinks(&resp.Body, prefix, cr.Channel.HLSLinkRoot)
		addHeaders(resp.Header, cr.ResponseWriter.Header(), false)
		cr.ResponseWriter.WriteHeader(http.StatusOK)
		fmt.Fprint(cr.ResponseWriter, content)
	default: // media (or anything else)
		handleEstablishedContentMedia(cr, resp)
	}
}

// ####################################################

func handleContentMedia(cr *ContentRequest) {
	resp, err := response(cr.Channel.Link)
	if err != nil {
		http.Error(cr.ResponseWriter, "internal server error", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	handleEstablishedContentMedia(cr, resp)
}

func handleEstablishedContentMedia(cr *ContentRequest, resp *http.Response) {
	addHeaders(resp.Header, cr.ResponseWriter.Header(), true)
	cr.ResponseWriter.WriteHeader(resp.StatusCode)
	io.Copy(cr.ResponseWriter, resp.Body)
}
