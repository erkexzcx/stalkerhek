package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
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

	updateM3U8Playlist()
	// go func() {
	// 	for {
	// 		time.Sleep(24 * time.Hour)
	// 		updateM3U8Playlist()
	// 	}
	// }()

	log.Println("Started!")

	// For stalker clients:
	http.HandleFunc("/", handleStalkerRequest)

	// For simple M3U IPTV clients:
	http.HandleFunc("/iptv", handleIPTVPlaylistRequest)
	http.HandleFunc("/iptv/", handleIPTVRequest)

	// Start listening for requests
	log.Fatal(http.ListenAndServe(":8987", nil))
}

func deleteAfterLastSlash(str string) string {
	return str[0 : strings.LastIndex(str, "/")+1]
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
		log.Println(string(outputDoAuth))
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
