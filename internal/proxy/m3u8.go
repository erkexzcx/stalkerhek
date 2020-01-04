package proxy

import (
	"bufio"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

func write500(w *http.ResponseWriter, customMessage ...interface{}) {
	log.Println(customMessage...)
	(*w).WriteHeader(http.StatusInternalServerError)
	(*w).Write([]byte("Internal server error"))
}

func quickWrite(w *http.ResponseWriter, content []byte, contentType string, httpStatus int) {
	(*w).Header().Set("Content-Type", contentType)
	(*w).WriteHeader(httpStatus)
	(*w).Write(content)
}

func m3u8Handler(w http.ResponseWriter, r *http.Request) {
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
	c, ok := getChannel(unescapedTitle)
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
		log.Println("Session invalid, updating...")
		if err := c.UpdateLink(); err != nil {
			write500(&w, "Failed to retrieve channel "+unescapedTitle+" from Stalker portal.")
			return
		}
	}

	// Build destination URL
	var requiredURL string
	if len(reqPathParts) == 1 {
		requiredURL = c.Link()
	} else {
		requiredURL = c.LinkRoot() + reqPathParts[1]
	}

	// Convert URL to URL object
	myURL, err := url.Parse(requiredURL)
	if err != nil {
		write500(&w, "Invalid request")
		return
	}

	// Channel-only request
	if len(reqPathParts) == 1 {
		handleChannelRequest(&w, r, c, &reqPathParts[0], myURL)
		return
	}
	// Channel with relative path request
	handleContentRequest(&w, r, c, &reqPathParts[0], myURL)
}

func handleChannelRequest(w *http.ResponseWriter, r *http.Request, c *m3u8Channel, title *string, u *url.URL) {
	// Lock mutex so no other request is requesting cache
	c.linkOneAtOnceMux.Lock()
	defer c.linkOneAtOnceMux.Unlock()

	if c.LinkCacheValid() {
		log.Println("Serving link cache...")
		quickWrite(w, c.LinkCache(), "application/vnd.apple.mpegurl", 200)
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

	if strings.HasPrefix(contentType, "video/") || strings.HasPrefix(contentType, "audio/") || contentType == "application/octet-stream" {
		hj, ok := (*w).(http.Hijacker)
		if !ok {
			http.Error(*w, "webserver doesn't support hijacking", http.StatusInternalServerError)
			return
		}
		conn, bufrw, err := hj.Hijack()
		if err != nil {
			http.Error(*w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Don't forget to close the connection:
		defer conn.Close()
		resp.Write(bufrw)
		bufrw.Flush()
		return
	} else if contentType == "application/vnd.apple.mpegurl" || contentType == "application/x-mpegurl" {
		// If M3U/M3U8 content - rewrite links
		linkRoot := c.LinkRoot()
		prefix := "http://" + r.Host + "/iptv/" + *title + "/"
		scanner := bufio.NewScanner(resp.Body)
		content := []byte(rewriteLinks(&prefix, &linkRoot, scanner))

		// Cache mux is already locked
		c.SetLinkCache(content)
		c.SetLinkCacheCreatedNow()

		(*w).Header().Set("Content-Type", contentType)
		(*w).WriteHeader(200)
		(*w).Write(content)
		return
	} else {
		write500(w, "Unsupported Content-Type:", contentType)
		return
	}
}

func handleContentRequest(w *http.ResponseWriter, r *http.Request, c *m3u8Channel, title *string, u *url.URL) {
	// Retrieve data
	resp, err := http.Get(u.String())
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

	// Sometime's we need to follow more links, and eventually they become root URLs...
	if (contentType == "application/vnd.apple.mpegurl" || contentType == "application/x-mpegurl") && strings.HasSuffix(strings.ToLower(u.RequestURI()), ".m3u8") {
		u = resp.Request.URL
		c.newRedirectedLink(resp.Request.URL.String())
	}

	log.Println("Final url:", u.String(), resp.StatusCode, contentType)

	if strings.HasPrefix(contentType, "video/") || strings.HasPrefix(contentType, "audio/") || contentType == "application/octet-stream" {
		hj, ok := (*w).(http.Hijacker)
		if !ok {
			http.Error(*w, "webserver doesn't support hijacking", http.StatusInternalServerError)
			return
		}
		conn, bufrw, err := hj.Hijack()
		if err != nil {
			http.Error(*w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Don't forget to close the connection:
		defer conn.Close()
		resp.Write(bufrw)
		bufrw.Flush()
		return
	} else if contentType == "application/vnd.apple.mpegurl" || contentType == "application/x-mpegurl" {
		// If M3U/M3U8 content - rewrite links
		linkRoot := c.LinkRoot()
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
