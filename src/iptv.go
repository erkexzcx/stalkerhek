package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sort"
	"strings"
	"time"
)

var tvchannelsMap = make(map[string]*tvchannel)

func handleIPTVPlaylistRequest(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "#EXTM3U")

	titles := make([]string, 0, len(tvchannelsMap))
	for tvch := range tvchannelsMap {
		titles = append(titles, tvch)
	}
	sort.Strings(titles)
	for _, title := range titles {
		tvchannelsMap[title].Mux.RLock()
		fmt.Fprintf(w, "#EXTINF:-1 tvg-logo=\"%s\", %s\n%s\n", portalURLDomain+"/stalker_portal/misc/logos/320/"+tvchannelsMap[title].Logo, title, "http://"+r.Host+"/iptv/"+url.PathEscape(title)+".m3u8")
		tvchannelsMap[title].Mux.RUnlock()
	}
}

func print500(w *http.ResponseWriter, customMessage ...interface{}) {
	log.Println(customMessage...)
	(*w).WriteHeader(http.StatusNotFound)
	(*w).Write([]byte("404 page not found"))
}

func handleIPTVRequest(w http.ResponseWriter, r *http.Request) {
	reqPath := strings.Replace(r.URL.RequestURI(), "/iptv/", "", 1)
	reqPathParts := strings.SplitN(reqPath, "/", 2)

	// Requested TV channel's title is reqPathParts[0]
	// Requested data (relative path) is reqPathParts[1]

	// Invalid request or if TV channel name length is lower than len(title)+len(".m3u8")
	if len(reqPathParts) == 0 {
		print500(&w, "Invalid request:", r.URL.Path)
	}

	// Remove ".m3u8" suffix
	reqPathParts[0] = strings.TrimSuffix(reqPathParts[0], ".m3u8")

	// Path decode channel name and attempt to find it in the map
	decodedTitle, _ := url.PathUnescape(reqPathParts[0])
	c, ok := tvchannelsMap[decodedTitle]
	if !ok {
		print500(&w, "Unable to find channel", decodedTitle)
		return
	}

	if len(reqPathParts) == 1 {
		log.Println("Query:", reqPathParts[0])
		handleIPTVChannelRequest(&w, r, c, &reqPathParts[0])
	} else {
		log.Println("Query:", reqPathParts[0], reqPathParts[1])
		handleIPTVDataRequest(&w, r, c, &reqPathParts[0], &reqPathParts[1])
	}

}

func handleIPTVChannelRequest(w *http.ResponseWriter, r *http.Request, c *tvchannel, t *string) {

	// Server cache if still valid
	c.CacheMux.Lock()
	defer c.CacheMux.Unlock()
	if c.LinkCacheValid() {
		log.Println("Loading channel link content from cache...")
		c.Mux.Lock()
		(*w).Header().Set("Content-Type", c.Cache.ContentType)
		(*w).WriteHeader(http.StatusOK)
		(*w).Write(c.Cache.Content)
		c.Mux.Unlock()
		return
	}

	// Update links from stalker if links are no longer valid
	var updateLinks bool
	if !c.LinkStillValid() {
		log.Println("Retrieving new channel link...")
		c.RefreshLink()
		updateLinks = true
	}

	// Get link:
	c.Mux.RLock()
	requiredLink := c.Link
	c.Mux.RUnlock()

	// Retrieve data
	resp, err := http.Get(requiredLink)
	if err != nil {
		print500(w, err)
		return
	}
	defer resp.Body.Close()

	// In case we got redirect - update channel's links
	if updateLinks {
		c.Mux.Lock()
		c.Link = resp.Request.URL.String()
		c.LinkRoot = deleteAfterLastSlash(c.Link)
		c.Mux.Unlock()
	}

	returnOutput(w, r, c, t, resp, requiredLink, true)
}

func handleIPTVDataRequest(w *http.ResponseWriter, r *http.Request, c *tvchannel, t, d *string) {
	// Get link:
	c.Mux.RLock()
	requiredLink := c.LinkRoot + *d
	c.Mux.RUnlock()

	// Attempt to retrieve it from cache
	c.LinksCacheMux.Lock()
	defer c.LinksCacheMux.Unlock()
	linkCache, exists := c.LinksCache[requiredLink]
	if exists {
		log.Println("Loading media from cache...")
		(*w).Header().Set("Content-Type", linkCache.ContentType)
		(*w).WriteHeader(http.StatusOK)
		(*w).Write(linkCache.Content)
		return
	}

	// Retrieve data
	resp, err := http.Get(requiredLink)
	if err != nil {
		print500(w, err)
		return
	}
	defer resp.Body.Close()

	returnOutput(w, r, c, t, resp, requiredLink, false)
}

func returnOutput(w *http.ResponseWriter, r *http.Request, c *tvchannel, t *string, resp *http.Response, u string, cr bool) {
	contentType := resp.Header.Get("Content-Type")
	log.Println(u, resp.StatusCode, contentType)

	if resp.StatusCode != 200 {
		(*w).Header().Set("Content-Type", contentType)
		(*w).WriteHeader(resp.StatusCode)
		return
	}

	if strings.HasPrefix(contentType, "video/") || strings.HasPrefix(contentType, "audio/") {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			print500(w, err)
			return
		}
		// Update cache
		if !cr {
			c.LinksCache[u] = &cache{
				ContentType: contentType,
				Content:     body,
			}
			// Weird approach, but gotta get it work first
			go func() {
				time.Sleep(10 * time.Second)
				c.LinksCacheMux.Lock()
				delete(c.LinksCache, u)
				c.LinksCacheMux.Unlock()
			}()
		}
		// Write output
		(*w).Header().Set("Content-Type", contentType)
		(*w).WriteHeader(resp.StatusCode)
		(*w).Write(body)
		return
	}

	if contentType == "application/octet-stream" {
		myurl, _ := url.Parse(u)
		proxy := httputil.NewSingleHostReverseProxy(myurl)

		// Update the headers to a	log.Println(resp.StatusCode)	log.Println(resp.StatusCode)llow for SSL redirection
		r.URL.Host = myurl.Host
		r.URL.Scheme = myurl.Scheme
		r.Host = myurl.Host

		// Note that ServeHttp is non blocking and uses a go routine under the hood
		proxy.ServeHTTP(*w, r)
		return
	}

	(*w).Header().Set("Content-Type", contentType)
	(*w).WriteHeader(resp.StatusCode)

	prefix := "http://" + r.Host + "/iptv/" + *t + "/"
	scanner := bufio.NewScanner(resp.Body)
	content := []byte(rewriteLinks(prefix, scanner))
	(*w).Write(content)

	if cr {
		log.Println("Updating cache...")
		c.Cache.Content = content
		c.CacheCreationTime = time.Now()
	} else {
		c.LinksCache[u] = &cache{
			ContentType: contentType,
			Content:     content,
		}
		//Weird approach, but gotta get it work first
		go func() {
			time.Sleep(10 * time.Second)
			c.LinksCacheMux.Lock()
			defer c.LinksCacheMux.Unlock()
			delete(c.LinksCache, u)
		}()
	}
}

func rewriteLinks(prefix string, scanner *bufio.Scanner) string {
	var sb strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "#") {
			line = prefix + line
		} else if strings.Contains(line, "URI=\"") && !strings.Contains(line, "URI=\"\"") {
			line = strings.ReplaceAll(line, "URI=\"", "URI=\""+prefix)
		}
		sb.WriteString(line)
		sb.WriteByte('\n')
	}
	return sb.String()
}

func updateM3U8Playlist() {

	type cstruct struct {
		Js struct {
			Data []struct {
				Name string `json:"name"`
				Cmd  string `json:"cmd"`
				Logo string `json:"logo"`
			} `json:"data"`
		} `json:"js"`
	}
	var cs cstruct

	// content, err := ioutil.ReadFile("/tmp/channelsCache")
	// if err != nil {
	// 	panic(err)
	// }

	req, err := getRequest(portalURLDomain + "/stalker_portal/server/load.php?type=itv&action=get_all_channels&force_ch_link_check=&JsHttpRequest=1-xml")
	if err != nil {
		panic(err)
	}
	defer req.Body.Close()
	content, err := ioutil.ReadAll(req.Body)
	if err != nil {
		panic(err)
	}

	// err = ioutil.WriteFile("/tmp/channelsCache", content, 0644)
	// if err != nil {
	// 	panic(err)
	// }

	if err := json.Unmarshal(content, &cs); err != nil {
		panic(err)
	}

	for _, v := range cs.Js.Data {
		tvchannelsMap[v.Name] = &tvchannel{
			Cmd:        v.Cmd,
			Logo:       v.Logo,
			LinksCache: make(map[string]*cache),
		}
	}
}
