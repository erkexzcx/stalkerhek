package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
)

var tvchannelsMap = make(map[string]*tvchannel)

func handlePlaylistRequest(w http.ResponseWriter, r *http.Request) {
	// Sort map
	titles := make([]string, 0, len(tvchannelsMap))
	for tvch := range tvchannelsMap {
		titles = append(titles, tvch)
	}
	sort.Strings(titles)

	//w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.WriteHeader(200)

	fmt.Fprintln(w, "#EXTM3U")
	for _, title := range titles {
		tvchannelsMap[title].Mux.RLock()
		channelLink := "http://" + r.Host + "/iptv/" + url.PathEscape(title) + ".m3u8"
		channelLogo := conf.Portal + "misc/logos/320/" + tvchannelsMap[title].Logo
		fmt.Fprintf(w, "#EXTINF:-1 tvg-logo=\"%s\", %s\n%s\n", channelLogo, title, channelLink)
		tvchannelsMap[title].Mux.RUnlock()
	}
}

func write500(w *http.ResponseWriter, customMessage ...interface{}) {
	log.Println(customMessage...)
	(*w).WriteHeader(http.StatusInternalServerError)
	(*w).Write([]byte("Internal server error"))
}

func quickWrite(w *http.ResponseWriter, content []byte, contentType *string, httpStatus int) {
	(*w).Header().Set("Content-Type", *contentType)
	(*w).WriteHeader(httpStatus)
	(*w).Write(content)
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	reqPath := strings.Replace(r.URL.RequestURI(), "/iptv/", "", 1)
	reqPathParts := strings.SplitN(reqPath, "/", 2)
	if len(reqPathParts) == 0 {
		write500(&w, "Invalid request")
	}
	reqPathParts[0] = strings.TrimSuffix(reqPathParts[0], ".m3u8")

	// Decode extracted tv channel name and find tv channel obj
	unescapedTitle, err := url.PathUnescape(reqPathParts[0])
	if err != nil {
		write500(&w, err)
		return
	}
	c, ok := tvchannelsMap[unescapedTitle]
	if !ok {
		write500(&w, "TV channel '"+unescapedTitle+"' does not exist")
		return
	}

	if len(reqPathParts) == 1 {
		log.Println("Received request [1]:", reqPathParts[0])
	} else {
		log.Println("Received request [2]:", reqPathParts[0], reqPathParts[1])
	}

	if !c.SessionValid() {
		log.Println("Getting new URL from Stalker API...")
		c.RefreshLink()
	}

	// Build destination URL
	var requiredURL string
	c.Mux.RLock()
	if len(reqPathParts) == 1 {
		requiredURL = c.Link
	} else {
		requiredURL = c.LinkRoot + reqPathParts[1]
	}
	c.Mux.RUnlock()

	// Convert URL to URL object
	myURL, err := url.Parse(requiredURL)
	if err != nil {
		write500(&w, "Invalid request")
		return
	}

	if len(reqPathParts) == 1 {
		// Channel-only request
		handleChannelRequest(&w, r, c, &reqPathParts[0], myURL)
	} else {
		// Channel with relative path request
		handleContentRequest(&w, r, c, &reqPathParts[0], myURL)
	}
}

func handleChannelRequest(w *http.ResponseWriter, r *http.Request, c *tvchannel, title *string, u *url.URL) {
	// Lock mutex so no other request is requesting cache
	// If it's locked - someone is already working on it, so just wait for cache
	c.CacheMux.Lock()
	defer c.CacheMux.Unlock()
	if c.LinkCacheValid() {
		log.Println("Serving channel's cache...")
		quickWrite(w, c.Cache, &(c.CacheContentType), 200)
		return
	}

	// Retrieve data
	resp, err := http.Get(u.String())
	if err != nil {
		write500(w, err)
		return
	}
	defer resp.Body.Close()

	// In case we got redirect - update channel's links
	if u.String() != resp.Request.URL.String() {
		c.Mux.Lock()
		c.Link = resp.Request.URL.String()
		c.LinkRoot = deleteAfterLastSlash(c.Link)
		u = resp.Request.URL
		c.Mux.Unlock()
	}

	contentType := strings.ToLower(resp.Header.Get("Content-Type"))

	if resp.StatusCode != 200 {
		quickWrite(w, []byte("HTTP Code not 200"), &contentType, resp.StatusCode)
		return
	}

	log.Println("Channel:", resp.StatusCode, contentType)

	c.Mux.Lock()
	c.SessionUpdateTime = time.Now()
	c.LinkAccessTime = time.Now()
	c.Mux.Unlock()

	log.Println("Final url:", u.String(), resp.StatusCode, contentType)

	if strings.HasPrefix(contentType, "video/") || strings.HasPrefix(contentType, "audio/") {
		content, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			write500(w, err)
			return
		}
		saveCache(u.String(), &content, &contentType)
		(*w).Header().Set("Content-Type", contentType)
		(*w).WriteHeader(resp.StatusCode)
		(*w).Write(content)
		return
	} else if contentType == "application/octet-stream" {
		for k, v := range resp.Header {
			(*w).Header().Set(k, v[0])
		}
		(*w).WriteHeader(resp.StatusCode)
		io.Copy(*w, resp.Body)
		return
	} else if contentType == "application/vnd.apple.mpegurl" || contentType == "application/x-mpegurl" {
		// If M3U/M3U8 content - rewrite links
		c.Mux.RLock()
		linkRoot := c.LinkRoot
		c.Mux.RUnlock()
		prefix := "http://" + r.Host + "/iptv/" + *title + "/"
		scanner := bufio.NewScanner(resp.Body)
		content := []byte(rewriteLinks(&prefix, &linkRoot, scanner))

		// Cache mux is already locked
		c.Cache = content
		c.CacheContentType = contentType

		(*w).Header().Set("Content-Type", contentType)
		(*w).WriteHeader(200)
		(*w).Write(content)
		return
	} else {
		write500(w, "Unsupported contentType:", contentType)
		return
	}
}

func handleContentRequest(w *http.ResponseWriter, r *http.Request, c *tvchannel, title *string, u *url.URL) {
	if content, contentType, ok := loadCache(u.String()); ok {
		log.Println("Serving link cache...")
		quickWrite(w, content, &(contentType), 200)
		return
	}

	// Retrieve data
	resp, err := http.Get(u.String())
	if err != nil {
		write500(w, err)
		return
	}
	defer resp.Body.Close()

	c.Mux.Lock()
	c.SessionUpdateTime = time.Now()
	c.Mux.Unlock()

	contentType := strings.ToLower(resp.Header.Get("Content-Type"))

	if resp.StatusCode != 200 {
		quickWrite(w, []byte("HTTP Code not 200"), &contentType, resp.StatusCode)
		return
	}

	// Sometime's we need to follow more links, and eventually they become root URLs...
	if (contentType == "application/vnd.apple.mpegurl" || contentType == "application/x-mpegurl") && strings.HasSuffix(strings.ToLower(u.RequestURI()), ".m3u8") {
		c.Mux.Lock()
		c.Link = resp.Request.URL.String()
		c.LinkRoot = deleteAfterLastSlash(c.Link)
		u = resp.Request.URL
		c.Mux.Unlock()
	}

	log.Println("Final url:", u.String(), resp.StatusCode, contentType)

	if strings.HasPrefix(contentType, "video/") || strings.HasPrefix(contentType, "audio/") {
		content, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			write500(w, err)
			return
		}
		saveCache(u.String(), &content, &contentType)
		(*w).Header().Set("Content-Type", contentType)
		(*w).WriteHeader(resp.StatusCode)
		(*w).Write(content)
		return
	} else if contentType == "application/octet-stream" {
		for k, v := range resp.Header {
			(*w).Header().Set(k, v[0])
		}
		(*w).WriteHeader(resp.StatusCode)
		io.Copy(*w, resp.Body)
		return
	} else if contentType == "application/vnd.apple.mpegurl" || contentType == "application/x-mpegurl" {
		// If M3U/M3U8 content - rewrite links
		c.Mux.RLock()
		linkRoot := c.LinkRoot
		c.Mux.RUnlock()

		prefix := "http://" + r.Host + "/iptv/" + *title + "/"
		scanner := bufio.NewScanner(resp.Body)
		content := []byte(rewriteLinks(&prefix, &linkRoot, scanner))

		(*w).Header().Set("Content-Type", contentType)
		(*w).WriteHeader(resp.StatusCode)
		(*w).Write(content)
		return
	} else {
		write500(w, "Unsupported contentType:", contentType)
		return
	}
}

var reURILinkExtract = regexp.MustCompile(`URI="([^"]*)"`)

func rewriteLinks(prefix *string, linkRoot *string, scanner *bufio.Scanner) string {
	var sb strings.Builder

	linkRootURL, _ := url.Parse(*linkRoot) // It will act as a base URL for full URLs

	modifyLink := func(link string) string {
		switch {
		case strings.HasPrefix(link, "//"):
			tmpURL, _ := url.Parse(link)
			tmp2URL, _ := url.Parse(tmpURL.RequestURI())
			link = (linkRootURL.ResolveReference(tmp2URL)).String()
			return *prefix + strings.ReplaceAll(link, *linkRoot, "")
		case strings.HasPrefix(link, "/"):
			tmp2URL, _ := url.Parse(link)
			link = (linkRootURL.ResolveReference(tmp2URL)).String()
			return *prefix + strings.ReplaceAll(link, *linkRoot, "")
		default:
			return *prefix + link
		}
	}

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "#") {
			line = modifyLink(line)
		} else if strings.Contains(line, "URI=\"") && !strings.Contains(line, "URI=\"\"") {
			link := reURILinkExtract.FindStringSubmatch(line)[1]
			line = reURILinkExtract.ReplaceAllString(line, `URI="`+modifyLink(link)+`"`)
		}
		sb.WriteString(line)
		sb.WriteByte('\n')
	}

	return sb.String()
}
