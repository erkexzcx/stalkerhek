package proxy

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

func write500(w http.ResponseWriter, customMessage ...interface{}) {
	log.Println(customMessage...)
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("Internal server error"))
}

func quickWrite(w http.ResponseWriter, content []byte, contentType string, httpStatus int) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(httpStatus)
	w.Write(content)
}

func playlistHandler(w http.ResponseWriter, r *http.Request) {
	// Sort map
	titles := make([]string, 0, len(channels))
	for tvch := range channels {
		titles = append(titles, tvch)
	}
	sort.Strings(titles)

	// Write HTTP headers
	w.WriteHeader(200)

	// Write HTTP body
	fmt.Fprintln(w, "#EXTM3U")
	for _, title := range titles {
		channelLink := "http://" + r.Host + "/iptv/" + url.QueryEscape(title)
		fmt.Fprintf(w, "#EXTINF:-1 tvg-logo=\"%s\" group-title=\"%s\", %s\n%s\n", (channels[title]).Stalker.Logo(), (channels[title]).Stalker.Genre(), title, channelLink)
	}
}

func channelHandler(w http.ResponseWriter, r *http.Request) {
	reqPath := strings.Replace(r.URL.RequestURI(), "/iptv/", "", 1)
	reqPathParts := strings.SplitN(reqPath, "/", 2)
	if len(reqPathParts) == 0 {
		write500(w, "Invalid request")
	}

	// Decode extracted tv channel name and find tv channel obj
	unescapedTitle, err := url.QueryUnescape(reqPathParts[0])
	if err != nil {
		write500(w, err)
		return
	}

	// Find channel in the list
	c, ok := channels[unescapedTitle]
	if !ok {
		write500(w, "TV channel '"+unescapedTitle+"' does not exist")
		return
	}
	c.chTypeMux.RLock()
	chType := c.chType
	c.chTypeMux.RUnlock()

	// Debug
	if len(reqPathParts) == 1 {
		log.Println("Received request [1]:", reqPathParts[0])
	} else {
		log.Println("Received request [2]:", reqPathParts[0], reqPathParts[1])
	}

	// Error if channel type is unknown and request URL contains additional path
	if chType == channelTypeUnknown && len(reqPathParts) == 2 {
		write500(w, "Channel needs to be opened first")
		return
	}

	// Lock mutex if channel's type is unknown, so no other routine tries to identify it
	if chType == channelTypeUnknown {
		c.chTypeMux.Lock()
		defer c.chTypeMux.Unlock()
		chType = c.chType // In case previous routine changed it
	}

	switch chType {
	case channelTypeUnknown:
		unknownChannelHandle(w, r, c, unescapedTitle)
	case channelTypeStream:
		// s := streams[unescapedTitle]
		// s.mux.Lock()

		// // Check if stream already exists
		// if s.stream == nil {
		// 	link, err := c.Stalker.NewLink()
		// 	if err != nil {
		// 		write500(w, err)
		// 		return
		// 	}
		// 	resp, err := http.Get(link)
		// 	if err != nil {
		// 		write500(w, err)
		// 		return
		// 	}
		// 	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// 		write500(w, errors.New("site says: "+resp.Status))
		// 		return
		// 	}
		// 	s.stream = resp.Body // do not close it
		// }

		// pipeReader, pipeWriter := io.Pipe()
		// s.AddWriter(pipeWriter)

		// w.Header().Set("Content-Type", "application/octet-stream")
		// w.WriteHeader(200)
		// defer pipeReader.Close()
		// io.Copy(w, pipeReader)
		// --------------------------------
		link, err := c.Stalker.NewLink()
		if err != nil {
			write500(w, err)
			return
		}
		resp, err := http.Get(link)
		if err != nil {
			write500(w, err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			write500(w, errors.New("site says: "+resp.Status))
			return
		}
		w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
		w.WriteHeader(200)
		defer resp.Body.Close()
		io.Copy(w, resp.Body)
	case channelTypeMedia:
		link, err := c.Stalker.NewLink()
		if err != nil {
			write500(w, err)
			return
		}
		resp, err := http.Get(link)
		if err != nil {
			write500(w, err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			write500(w, errors.New("site says: "+resp.Status))
			return
		}
		w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
		w.WriteHeader(200)
		defer resp.Body.Close()
		io.Copy(w, resp.Body)
	case channelTypeM3U8:
		m3u8c := m3u8channels[unescapedTitle]

		if !m3u8c.SessionValid() {
			log.Println("Session invalid, updating...")
			if err := m3u8c.UpdateLink(); err != nil {
				write500(w, "Failed to retrieve channel "+unescapedTitle+" from Stalker portal.")
				return
			}
		}

		// Build destination URL
		var requiredURL string
		if len(reqPathParts) == 1 {
			requiredURL = m3u8c.Link()
		} else {
			requiredURL = m3u8c.LinkRoot() + reqPathParts[1]
		}

		// Convert URL to URL object
		myURL, err := url.Parse(requiredURL)
		if err != nil {
			write500(w, "Invalid request")
			return
		}

		// Channel-only request
		if len(reqPathParts) == 1 {
			m3u8ChannelHandle(w, r, m3u8c, reqPathParts[0], myURL)
			return
		}
		// Channel with path request
		m3u8ChannelPathHandle(w, r, m3u8c, reqPathParts[0], myURL)
	}
}

func m3u8ChannelHandle(w http.ResponseWriter, r *http.Request, c *M3U8Channel, title string, u *url.URL) {
	// Lock mutex so no other request is requesting cache
	c.linkOneAtOnceMux.Lock()
	defer c.linkOneAtOnceMux.Unlock()

	if c.LinkCacheValid() {
		log.Println("Serving link cache...")
		quickWrite(w, c.LinkCache(), "application/vnd.apple.mpegurl", 200)
		return
	}

	// Retrieve data
	resp, err := m3u8HTTPClient.Get(u.String())
	if err != nil {
		write500(w, err)
		return
	}
	defer resp.Body.Close()

	// In case we got redirect - update channel's links
	if u.String() != resp.Request.URL.String() {
		c.newRedirectedLink(resp.Request.URL.String())
		u = resp.Request.URL
	}

	contentType := strings.ToLower(resp.Header.Get("Content-Type"))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		quickWrite(w, []byte("Site says: "+resp.Status), contentType, resp.StatusCode)
		log.Println("Channel", u.String()+":", resp.Status)
		return
	}

	c.SetSessionUpdatedNow()

	log.Println("Final url:", u.String(), resp.StatusCode, contentType)

	linkRoot := c.LinkRoot()
	prefix := "http://" + r.Host + "/iptv/" + title + "/"
	scanner := bufio.NewScanner(resp.Body)
	content := []byte(rewriteLinks(prefix, linkRoot, scanner))

	// Cache mux is already locked
	c.SetLinkCache(content)
	c.SetLinkCacheCreatedNow()

	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(200)
	w.Write(content)
}

func m3u8ChannelPathHandle(w http.ResponseWriter, r *http.Request, c *M3U8Channel, title string, u *url.URL) {

	// Try to load from cache first
	if contents, ok := m3u8TSCache.Get(u.String()); ok {
		log.Println("Serving media cache...")
		quickWrite(w, contents.([]byte), "application/vnd.apple.mpegurl", 200)
		return
	}

	// Retrieve data
	resp, err := m3u8HTTPClient.Get(u.String())
	if err != nil {
		write500(w, err)
		return
	}
	defer resp.Body.Close()

	contentType := strings.ToLower(resp.Header.Get("Content-Type"))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		quickWrite(w, []byte("Site says: "+resp.Status), contentType, resp.StatusCode)
		log.Println("Media", u.String()+":", resp.Status)
		return
	}

	c.SetSessionUpdatedNow()

	if (contentType == "application/vnd.apple.mpegurl" || contentType == "application/x-mpegurl") && strings.HasSuffix(strings.ToLower(u.RequestURI()), ".m3u8") {
		// When there is redirect URL inside M3U8 file. 2 lines content...
		u = resp.Request.URL
		c.newRedirectedLink(resp.Request.URL.String())

		linkRoot := c.LinkRoot()
		prefix := "http://" + r.Host + "/iptv/" + title + "/"
		scanner := bufio.NewScanner(resp.Body)
		content := []byte(rewriteLinks(prefix, linkRoot, scanner))

		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(resp.StatusCode)
		w.Write(content)
	} else if strings.HasPrefix(contentType, "video/") || strings.HasPrefix(contentType, "audio/") {
		// TS files
		content, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			write500(w, err)
		}
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(200)
		w.Write(content)

		m3u8TSCache.SetDefault(u.String(), content) // Save to cache
	}
}

func unknownChannelHandle(w http.ResponseWriter, r *http.Request, c *Channel, t string) {
	// If we don't know the channel type, it means we have never opened it yet
	link, err := c.Stalker.NewLink()
	if err != nil {
		write500(w, "Failed to get channel link from Stalker: ", err)
		return
	}

	// Get response from that link
	resp, err := http.Get(link)
	if err != nil {
		write500(w, "Failed to get response from the server for '"+link+"': ", err)
		return
	}
	// Note that resp.Body closure is not deferred from here

	// Check for bad HTTP status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		write500(w, "'"+link+"' returned: ", resp.Status)
		return
	}

	contentType := resp.Header.Get("Content-Type")

	// Get channel's type
	chType, err := getChannelType(contentType)
	if err != nil {
		write500(w, "Failed to identify content type of '"+link+"': ", err)
		return
	}

	c.chType = chType

	// This function can only be reached when user requests channel without additional path

	switch chType {
	case channelTypeM3U8:
		defer resp.Body.Close()

		m3u8c := &M3U8Channel{Stalker: c.Stalker}
		m3u8channels[t] = m3u8c

		m3u8c.link = resp.Request.URL.String()
		m3u8c.linkRoot = deleteAfterLastSlash(m3u8c.link)
		m3u8c.sessionUpdated = time.Now()

		prefix := "http://" + r.Host + "/iptv/" + url.QueryEscape(t) + "/" // We got plain channel titiel, so need to query escape it
		scanner := bufio.NewScanner(resp.Body)
		content := []byte(rewriteLinks(prefix, m3u8c.linkRoot, scanner))

		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(200)
		w.Write(content)

		m3u8c.linkCache = content
		m3u8c.linkCacheCreated = time.Now()
	case channelTypeStream:
		// s := &Stream{stream: nil}
		// streams[t] = s
		// s.stream = resp.Body

		// pipeReader, pipeWriter := io.Pipe()
		// s.AddWriter(pipeWriter)

		// w.Header().Set("Content-Type", contentType)
		// w.WriteHeader(200)

		// s.Start() // Starts pooler/buffer reader that writes to given writers
		// io.Copy(w, pipeReader)
		// --------------------------------
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(200)
		defer resp.Body.Close()
		io.Copy(w, resp.Body)
	case channelTypeMedia:
		// Nothing really to do for media type - just copy/paste
		defer resp.Body.Close()
		io.Copy(w, resp.Body)
	}
}

func getChannelType(contentType string) (channelType int, err error) {
	contentType = strings.ToLower(contentType)
	switch {
	case contentType == "application/vnd.apple.mpegurl" || contentType == "application/x-mpegurl":
		return channelTypeM3U8, nil
	case strings.HasPrefix(contentType, "video/") || strings.HasPrefix(contentType, "audio/"):
		return channelTypeMedia, nil
	case contentType == "application/octet-stream":
		return channelTypeStream, nil
	default:
		return -1, errors.New("unrecognized Content-Type '" + contentType + "'")
	}
}
