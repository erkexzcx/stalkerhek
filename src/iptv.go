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
	"sync"
	"time"
)

type tvchannel struct {
	CMD            string
	Logo           string
	Link           string
	LinkRoot       string
	LastAccessTime time.Time
	Mux            sync.RWMutex
	Cache          map[string]*[]byte
	CacheMux       sync.RWMutex
}

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
		fmt.Fprintf(w, "#EXTINF:-1 tvg-logo=\"%s\", %s\n%s\n\n", portalURLDomain+"/stalker_portal/misc/logos/320/"+tvchannelsMap[title].Logo, title, "http://"+r.Host+"/iptv/"+url.PathEscape(title)+".m3u8")
		tvchannelsMap[title].Mux.RUnlock()
	}
}

func print500(w *http.ResponseWriter, customMessage interface{}) {
	log.Println(customMessage)
	(*w).WriteHeader(http.StatusNotFound)
	(*w).Write([]byte("404 page not found"))
}

func handleIPTVRequest(w http.ResponseWriter, r *http.Request) {
	reqPath := strings.Replace(r.URL.RequestURI(), "/iptv/", "", 1)
	reqPathParts := strings.SplitN(reqPath, "/", 2)
	reqPathPartsLen := len(reqPathParts)

	log.Println("Received:", r.URL.String())

	// Exit if no channel and/or no path provided:
	if reqPathPartsLen == 0 {
		print500(&w, "Unable to properly extract data from request '"+r.URL.Path+"'!")
		return
	}

	// Remove ".m3u8" from channel name
	if reqPathPartsLen == 1 {
		reqPathParts[0] = strings.Replace(reqPathParts[0], ".m3u8", "", 1)
	}

	// Extract channel name:
	encodedChannelName := &reqPathParts[0]
	decodedChannelName, err := url.PathUnescape(*encodedChannelName)
	if err != nil {
		print500(&w, "Unable to decode channel '"+*encodedChannelName+"'!")
		return
	}

	// Retrieve channel from channels map:
	channel, ok := tvchannelsMap[decodedChannelName]
	if !ok {
		print500(&w, "Unable to find channel '"+decodedChannelName+"'!")
		return
	}

	// For channel we need URL. For anything else we need URL root:
	var requiredURL string
	channel.Access() // Update URL if needed
	channel.Mux.RLock()
	if reqPathPartsLen == 1 {
		requiredURL = channel.Link
	} else {
		requiredURL = channel.LinkRoot + reqPathParts[1]
	}
	channel.Mux.RUnlock()

	if requiredURL == "" {
		print500(&w, "Channel '"+decodedChannelName+"' does not have URL assigned!")
		return
	}

	// Retrieve contents
	resp, err := http.Get(requiredURL)
	if err != nil {
		print500(&w, err)
		return
	}
	defer resp.Body.Close()

	// Rewrite URI (just in case we got a redirect)
	if reqPathPartsLen == 1 {
		channel.Mux.Lock()
		channel.Link = resp.Request.URL.String()
		channel.LinkRoot = deleteAfterLastSlash(resp.Request.URL.String())
		channel.Mux.Unlock()
	}

	log.Println("Queried: ", resp.Request.URL.String(), resp.StatusCode)

	// If content is stream
	if resp.Header.Get("Content-Type") == "application/octet-stream" {
		myurl, _ := url.Parse(requiredURL)
		proxy := httputil.NewSingleHostReverseProxy(myurl)

		// Update the headers to allow for SSL redirection
		r.URL.Host = myurl.Host
		r.URL.Scheme = myurl.Scheme
		r.Host = myurl.Host

		// Note that ServeHttp is non blocking and uses a go routine under the hood
		proxy.ServeHTTP(w, r)
		return
	}

	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))

	// If content is video
	if strings.HasPrefix(resp.Header.Get("Content-Type"), "video") || strings.HasPrefix(resp.Header.Get("Content-Type"), "audio") {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			print500(&w, err)
			return
		}
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
		return
	}

	// Write everything, but rewrite links to itself
	w.WriteHeader(resp.StatusCode)
	prefix := "http://" + r.Host + "/iptv/" + *encodedChannelName + "/"
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "#") {
			line = prefix + line
		} else if strings.Contains(line, "URI=\"") && !strings.Contains(line, "URI=\"\"") {
			line = strings.ReplaceAll(line, "URI=\"", "URI=\""+prefix)
		}
		w.Write([]byte(line + "\n"))
	}
}

func initializeM3U8Playlist() {

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
			CMD:            v.Cmd,
			Logo:           v.Logo,
			LastAccessTime: time.Now(),
		}
	}
}

func (channel *tvchannel) Access() {

	elapsed := time.Since(channel.LastAccessTime)

	channel.LastAccessTime = time.Now()

	if elapsed.Seconds() <= 30 && channel.Link != "" {
		return
	}

	log.Println("Resolving channel!!!")

	type tmpstruct struct {
		Js struct {
			Cmd string `json:"cmd"`
		} `json:"js"`
	}
	var tmp tmpstruct

	// Mutex is not needed, since parent codeblock is already RLocked
	resp, err := getRequest(portalURLDomain + "/stalker_portal/server/load.php?action=create_link&type=itv&cmd=" + url.PathEscape(channel.CMD) + "&JsHttpRequest=1-xml")
	if err != nil {
		log.Println("Failed to resolve channel")
	}
	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(content, &tmp); err != nil {
		panic(err)
	}

	strs := strings.Split(tmp.Js.Cmd, " ")
	if len(strs) == 2 {
		channel.Link = strs[1]
		channel.LinkRoot = deleteAfterLastSlash(strs[1])
	}

}
