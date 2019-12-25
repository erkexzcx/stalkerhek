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

func print500(w *http.ResponseWriter, customMessage interface{}) {
	log.Println(customMessage)
	(*w).WriteHeader(http.StatusNotFound)
	(*w).Write([]byte("404 page not found"))
}

func handleIPTVRequest(w http.ResponseWriter, r *http.Request) {
	reqPath := strings.Replace(r.URL.RequestURI(), "/iptv/", "", 1)
	reqPathParts := strings.SplitN(reqPath, "/", 2)

	if len(reqPathParts) == 0 {
		print500(&w, "Unable to properly extract data from request '"+r.URL.Path+"'!")
		return
	}

	channelRequestType := len(reqPathParts) == 1 // true=channel's meta data; false=anything else (like video or deeper links)

	if channelRequestType {
		reqPathParts[0] = strings.Replace(reqPathParts[0], ".m3u8", "", 1) // Remove .m3u8 from channel's name
	}

	encodedTitle := &reqPathParts[0]
	decodedTitle, err := url.PathUnescape(*encodedTitle)
	if err != nil {
		print500(&w, "Unable to decode channel '"+*encodedTitle+"'!")
		return
	}
	channel, ok := tvchannelsMap[decodedTitle]
	if !ok {
		print500(&w, "Unable to find channel '"+decodedTitle+"'!")
		return
	}

	channel.Mux.Lock()
	if channelRequestType && channel.LinkRootStillValid() && channel.LinkCacheValid() {
		log.Println("Loading channel link content from cache...")
		w.Write(channel.LinkCache)
		channel.Mux.Unlock()
		return
	}
	if channelRequestType && !channel.LinkRootStillValid() {
		log.Println("Retrieving new channel link...")
		channel.RefreshLink()
		channel.LastLinkRootAccess = time.Now() // It is generated now
	}
	channel.Mux.Unlock()

	var requiredURL string
	channel.Mux.RLock()
	if channelRequestType {
		requiredURL = channel.Link
	} else {
		requiredURL = channel.LinkRoot + reqPathParts[1]
	}
	channel.Mux.RUnlock()
	if requiredURL == "" {
		print500(&w, "Channel '"+decodedTitle+"' does not have URL assigned!")
		return
	}

	resp, err := http.Get(requiredURL)
	if err != nil {
		print500(&w, err)
		return
	}
	defer resp.Body.Close()

	if channelRequestType {
		channel.Mux.Lock()
		channel.Link = resp.Request.URL.String()
		channel.LinkRoot = deleteAfterLastSlash(resp.Request.URL.String())
		channel.Mux.Unlock()
	}

	channel.Mux.Lock()
	channel.LastLinkRootAccess = time.Now()
	channel.Mux.Unlock()

	log.Println("Quered: ", r.RemoteAddr, resp.Request.URL.String(), resp.StatusCode)

	if strings.HasPrefix(resp.Header.Get("Content-Type"), "video/") || strings.HasPrefix(resp.Header.Get("Content-Type"), "audio/") {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			print500(&w, err)
			return
		}
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
		return
	}

	// octet-stream needs to be reverse-proxified:
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
	w.WriteHeader(resp.StatusCode)

	prefix := "http://" + r.Host + "/iptv/" + *encodedTitle + "/"
	scanner := bufio.NewScanner(resp.Body)
	content := []byte(rewriteLinks(prefix, scanner))
	w.Write(content)
	if channelRequestType {
		channel.Mux.Lock()
		log.Println("Updating cache...")
		channel.LinkCache = content
		channel.LinkCacheCreation = time.Now()
		channel.Mux.Unlock()
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
		channel, exists := tvchannelsMap[v.Name]
		if exists {
			channel.Mux.Lock()
			channel.Cmd = v.Cmd
			channel.Logo = v.Logo
			channel.Mux.Unlock()
		} else {
			tvchannelsMap[v.Name] = &tvchannel{
				Cmd:  v.Cmd,
				Logo: v.Logo,
			}
		}
	}
}
