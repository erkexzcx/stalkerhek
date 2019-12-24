package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// ===================================================================================
//
// Update only these values. Everything MUST BE URL ENCODED (as seen in wireshark URLs and cookies - unedited)
//
const sn = "0000000000000"                                                           // Set Serial Number
const deviceID = "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"  // Set device ID
const deviceID2 = "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff" // Set device ID2
const signature = "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff" // Set signature here
const macEncoded = "00%3A00%3A00%3A00%3A00%3A00"                                     // Set MAC address
const login = ""                                                                     // Set username
const password = ""                                                                  // Set password
const portalURLDomain = "http://domain.example.com"                                  // IMPORTANT! Must end without slash at the end
const timeZone = "Europe%2FVilnius"                                                  // Set local timezone
var token = "XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"                                       // Set token. It might be updated automatically if in use

// ===================================================================================

type tvchannel struct {
	CMD              string
	ResolvedLink     string
	ResolvedLinkRoot string
	Logo             string
}

var tvchannelsMap = make(map[string]*tvchannel)
var tvChannelsMux sync.RWMutex

// Store some outputs for re-use:
var outputHandshake []byte
var outputDoAuth []byte
var outputGetProfile []byte

var cacheMap = make(map[string]*[]byte)
var cacheMutex sync.RWMutex

var httpClient = &http.Client{}

func main() {

	log.Println("Starting...")

	authenticate() // Authenticate with stalker portal

	performWatchdogUpdate()
	go func() {
		for {
			time.Sleep(2 * time.Minute)
			performWatchdogUpdate()
		}
	}()

	performEPGUpdate()
	go func() {
		for {
			time.Sleep(2 * time.Hour)
			performEPGUpdate()
		}
	}()

	go func() {
		for {
			time.Sleep(15 * time.Second)
			performResolvedM3U8KeepAlive()
		}
	}()

	// Parse channels for M3U playlist
	initializeM3U8Playlist()

	log.Println("Started!")

	// For stalker clients:
	http.HandleFunc("/", handleStalkerRequest)

	// For simple M3U IPTV clients:
	http.HandleFunc("/iptv", handleIPTVPlaylistRequest)
	http.HandleFunc("/iptv/", handleIPTVRequest)

	// Start listening for requests
	log.Fatal(http.ListenAndServe(":8987", nil))
}

func handleIPTVPlaylistRequest(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "#EXTM3U")

	tvChannelsMux.RLock()
	titles := make([]string, 0, len(tvchannelsMap))
	for tvch := range tvchannelsMap {
		titles = append(titles, tvch)
	}
	sort.Strings(titles)
	for _, title := range titles {
		fmt.Fprintf(w, "#EXTINF:-1 tvg-logo=\"%s\", %s\n%s\n\n", portalURLDomain+"/stalker_portal/misc/logos/320/"+tvchannelsMap[title].Logo, title, "http://"+r.Host+"/iptv/"+url.PathEscape(title)+".m3u8")
	}
	tvChannelsMux.RUnlock()
}

func print404(w *http.ResponseWriter, customMessage interface{}) {
	log.Println(customMessage)
	(*w).WriteHeader(http.StatusNotFound)
	(*w).Write([]byte("404 page not found"))
}

func handleIPTVRequest(w http.ResponseWriter, r *http.Request) {
	reqPath := strings.Replace(r.URL.RequestURI(), "/iptv/", "", 1)
	reqPathParts := strings.SplitN(reqPath, "/", 2)
	reqPathPartsLen := len(reqPathParts)

	// Exit if no channel and/or no path provided:
	if reqPathPartsLen == 0 {
		print404(&w, "Unable to properly extract data from request '"+r.URL.Path+"'!")
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
		print404(&w, "Unable to decode channel '"+*encodedChannelName+"'!")
		return
	}

	// Retrieve channel from channels map:
	tvChannelsMux.RLock()
	channel, ok := tvchannelsMap[decodedChannelName]
	tvChannelsMux.RUnlock()
	if !ok {
		print404(&w, "Unable to find channel '"+decodedChannelName+"'!")
		return
	}

	// For channel we need URL. For anything else we need URL root:
	var requiredURL string
	tvChannelsMux.RLock()
	if reqPathPartsLen == 1 {
		if channel.ResolvedLink == "" {
			resolveChannel(channel)
		}
		requiredURL = channel.ResolvedLink
	} else {
		requiredURL = channel.ResolvedLinkRoot + reqPathParts[1]
	}
	tvChannelsMux.RUnlock()

	if requiredURL == "" {
		print404(&w, "Channel '"+decodedChannelName+"' does not have URL assigned!")
		return
	}

	// Retrieve contents
	resp, err := http.Get(requiredURL)
	if err != nil {
		print404(&w, err)
		return
	}
	defer resp.Body.Close()

	// Rewrite URI (just in case we got a redirect)
	if reqPathPartsLen == 1 {
		tvChannelsMux.RLock()
		channel.ResolvedLink = resp.Request.URL.String()
		channel.ResolvedLinkRoot = deleteAfterLastSlash(channel.ResolvedLink)
		tvChannelsMux.RUnlock()
	}

	// If path ends with ".ts" - return raw fetched bytes
	if strings.HasSuffix(r.URL.Path, ".ts") {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			print404(&w, err)
			return
		}
		w.WriteHeader(http.StatusOK)
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

func deleteAfterLastSlash(str string) string {
	return str[0 : strings.LastIndex(str, "/")+1]
}

func handleStalkerRequest(w http.ResponseWriter, r *http.Request) {

	q := r.URL.Query()

	_, hasAction := q["action"]

	if r.URL.Path == "/stalker_portal/server/load.php" {
		if hasAction {
			switch q["action"][0] {
			case "logout":
				log.Println("Ignoring request: " + portalURLDomain + r.URL.String())
				return
			case "handshake":
				log.Println("Ignoring request: " + portalURLDomain + r.URL.String())
				w.Write(outputHandshake)
				return
			case "do_auth":
				log.Println("Ignoring request: " + portalURLDomain + r.URL.String())
				w.Write(outputDoAuth)
				return
			case "get_profile":
				log.Println("Ignoring request: " + portalURLDomain + r.URL.String())
				w.Write(outputGetProfile)
				return
			case "get_events":
				_, ok := q["type"]
				if ok && q["type"][0] == "watchdog" {
					cacheMutex.RLock()
					w.Write(*(cacheMap["watchdog"]))
					cacheMutex.RUnlock()
					return
				}
				log.Println("Ignoring request: " + portalURLDomain + r.URL.String())
				w.Write(outputGetProfile)
				return
			case "get_epg_info":
				log.Println("Ignoring request: " + portalURLDomain + r.URL.String())
				cacheMutex.RLock()
				w.Write(*(cacheMap["epg"]))
				cacheMutex.RUnlock()
				return
			}
		}
	}

	// Rewrite any identifying parameter, such as serial number, username, device ID etc...

	_, ok := q["login"]
	if ok {
		q.Set("login", login)
	}
	_, ok = q["password"]
	if ok {
		q.Set("password", password)
	}
	_, ok = q["sn"]
	if ok {
		q.Set("sn", sn)
	}
	_, ok = q["device_id"]
	if ok {
		q.Set("device_id", deviceID)
	}
	_, ok = q["device_id2"]
	if ok {
		q.Set("device_id2", deviceID2)
	}
	_, ok = q["signature"]
	if ok {
		q.Set("signature", signature)
	}
	r.URL.RawQuery = q.Encode()

	requestURI := r.URL.RequestURI()

	if hasAction || !strings.HasSuffix(r.URL.Path, ".php") {
		cacheMutex.RLock()
		content, ok := cacheMap[requestURI]
		cacheMutex.RUnlock()
		if ok {
			w.Write(*content)
			log.Println("Loaded from cache", portalURLDomain+requestURI)
			return
		}
	}

	// Forward request to stalker portal and return it's output and the same HTTP code
	log.Println(portalURLDomain + requestURI)
	resp, err := getRequest(portalURLDomain + requestURI)
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	// Save cache:
	if hasAction || !strings.HasSuffix(r.URL.Path, ".php") {
		cacheMutex.Lock()
		cacheMap[requestURI] = &body
		cacheMutex.Unlock()
	}

	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}

// Returns performed request to given link. This also sets some header values, such as correct cookies and authorization token if it exists
func getRequest(link string) (*http.Response, error) {
	req, err := http.NewRequest("GET", link, nil)
	if err != nil {
		log.Fatalln(err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (QtEmbedded; U; Linux; C) AppleWebKit/533.3 (KHTML, like Gecko) MAG200 stbapp ver: 4 rev: 2116 Mobile Safari/533.3")
	req.Header.Set("X-User-Agent", "Model: MAG254; Link: Ethernet")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Cookie", "PHPSESSID=null; sn="+sn+"; mac="+macEncoded+"; stb_lang=en; timezone="+timeZone)

	return httpClient.Do(req)
}

// Authenticate with stalker portal
func authenticate() {
	var tokenRegex = regexp.MustCompile(`"token"\s*:\s*"([A-Z0-9]+)"`)
	// Server identifies us by token. This is a string and will be sent with each HTTP request.
	//
	// How it works: You send a request to the server (see below URL) with a suggested token ('token' parameter).
	// Server checks if such token is in use. If in use - server will return randomly generated token in JSON and
	// below code will extract it and use in each HTTP request. If token is not in use - server don't return new
	// randomly generated token and authenticates using suggested token instead.
	resp, err := getRequest(portalURLDomain + "/stalker_portal/server/load.php?type=stb&action=handshake&prehash=0&token=" + token + "&JsHttpRequest=1-xml")
	if err != nil {
		log.Fatalln(err)
	}
	outputHandshake, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	parts := tokenRegex.FindStringSubmatch(string(outputHandshake))
	if len(parts) > 0 {
		token = parts[1]
	}
	resp.Body.Close()

	// Since we have a token, we need to authorize it (associate it with your credentials)
	resp, err = getRequest(portalURLDomain + "/stalker_portal/server/load.php?type=stb&action=do_auth&login=" + login + "&password=" + password + "&device_id=" + deviceID + "&device_id2=" + deviceID2 + "&JsHttpRequest=1-xml")
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()
	outputDoAuth, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	if !strings.Contains(string(outputDoAuth), "bool(true)") {
		log.Fatalln("Unable to authenticate at step 2. Check your credentials and try again")
	}

	// Get profile and store output:
	resp, err = getRequest(portalURLDomain + "/stalker_portal/server/load.php?action=get_profile&sn=" + sn + "&device_id=" + deviceID + "&device_id2=" + deviceID2 + "&signature=" + signature + "&stb_type=MAG254&hd=1&ver=ImageDescription:%200.2.18-r17-254;%20ImageDate:%20Mon%20Feb%2020%2015:19:12%20EET%202017;%20PORTAL%20version:%205.2.0;%20API%20Version:%20JS%20API%20version:%20340;%20STB%20API%20version:%20146;%20Player%20Engine%20version:%200x57d&type=stb&auth_second_step=0&image_version=218&not_valid_token=0&num_banks=2&hw_version=1.11-BD-00&JsHttpRequest=1-xml")
	if err != nil {
		log.Fatalln(err)
	}
	outputGetProfile, err = ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		log.Fatalln(err)
	}
}

func performWatchdogUpdate() {
	req, err := getRequest(portalURLDomain + "/stalker_portal/server/load.php?action=get_events&event_active_id=0&init=0&type=watchdog&cur_play_type=1&JsHttpRequest=1-xml")
	if err != nil {
		log.Println("Failed to perform watchdog check")
	} else {
		log.Println("Watchdog request done")
	}
	content, err := ioutil.ReadAll(req.Body)
	req.Body.Close()
	if err != nil {
		log.Println("Failed to fetch watchdog request content")
	}
	cacheMutex.Lock()
	cacheMap["watchdog"] = &content
	cacheMutex.Unlock()
}

func performEPGUpdate() {
	req, err := getRequest(portalURLDomain + "/stalker_portal/server/load.php?action=get_epg_info&period=5&type=itv&JsHttpRequest=1-xml")
	if err != nil {
		log.Println("Failed to retrieve EPG")
	} else {
		log.Println("EPG request done")
	}
	content, err := ioutil.ReadAll(req.Body)
	req.Body.Close()
	if err != nil {
		log.Println("Failed to fetch epg request content")
	}
	cacheMutex.Lock()
	cacheMap["epg"] = &content
	cacheMutex.Unlock()
}

func performResolvedM3U8KeepAlive() {
	tvChannelsMux.RLock()
	for k, v := range tvchannelsMap {
		if v.ResolvedLink == "" {
			return
		}
		go func(k string, v *tvchannel) {
			resp, err := http.Get(v.ResolvedLink)
			if err != nil {
				log.Println("Failed to keep-alive channel", k, "with url", *v)
				return
			}
			log.Println("Keep-alive", *v)
			resp.Body.Close()
		}(k, v)
	}
	tvChannelsMux.RUnlock()
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

	req, err := getRequest(portalURLDomain + "/stalker_portal/server/load.php?type=itv&action=get_all_channels&force_ch_link_check=&JsHttpRequest=1-xml")
	if err != nil {
		panic(err)
	}
	defer req.Body.Close()
	content, err := ioutil.ReadAll(req.Body)
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(content, &cs); err != nil {
		panic(err)
	}

	for _, v := range cs.Js.Data {
		tvchannelsMap[v.Name] = &tvchannel{
			CMD:  v.Cmd,
			Logo: v.Logo,
		}
	}
}

func resolveChannel(channel *tvchannel) {

	type tmpstruct struct {
		Js struct {
			Cmd string `json:"cmd"`
		} `json:"js"`
	}
	var tmp tmpstruct

	// Mutex is not needed, since parent codeblock is already RLocked
	resp, err := getRequest(portalURLDomain + "/stalker_portal/server/load.php?action=create_link&type=itv&cmd=" + url.PathEscape(channel.CMD) + "&JsHttpRequest=1-xml")
	if err != nil {
		log.Println("Failed to resolve channel", channel.ResolvedLink)
	}
	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(content, &tmp); err != nil {
		panic(err)
	}
	channel.ResolvedLink = strings.Split(tmp.Js.Cmd, " ")[1]
	channel.ResolvedLinkRoot = deleteAfterLastSlash(channel.ResolvedLink)
}
