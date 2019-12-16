package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
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
//
// ===================================================================================

var token string

var httpClient = &http.Client{}

func handleRequest(w http.ResponseWriter, r *http.Request) {

	// This is everything what goes after hostname & port. Starts with slash:
	URI := r.URL.RequestURI()

	if strings.Contains(URI, "/load.php") {
		if strings.Contains(URI, "action=handshake") {
			log.Println("Ignoring request: " + portalURLDomain + r.URL.RequestURI())
			w.Write([]byte(`{"js":{"token":"3A235D33B2A92809F90E3023B341FBF2","random":"76554c2d20175568a0fef0ab6639fb3f8e1ef9b0","not_valid":0},"text":"generated in: 0.011s; query counter: 1; cache hits: 0; cache miss: 0; php errors: 0; sql errors: 0;"}`))
			return
		} else if strings.Contains(URI, "action=logout") {
			log.Println("Ignoring request: " + portalURLDomain + r.URL.RequestURI())
			w.Write([]byte(``)) // Write nothing?
			return
		}
	}

	// Rewrite any identifying parameter, such as serial number, username, device ID etc...
	q := r.URL.Query()
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

	// Forward request to stalker portal and return it's output and the same HTTP code
	log.Println(portalURLDomain + r.URL.RequestURI())
	resp, err := getRequest(portalURLDomain + r.URL.RequestURI())
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}

func main() {
	authenticate() // Authenticate with stalker portal

	// TODO - I think we need to implement keep-alive mechanism as well. Every stalker client sends "watchdog" requests to the server
	// to keep authentication alive (sends every X minutes), so would be great to do it from this client too and ignore all clients
	// keep-alive requests

	// Start listening for requests
	http.HandleFunc("/", handleRequest)
	log.Fatal(http.ListenAndServe(":80", nil)) // Note that to open/start listening on 0-1024 ports you need root access
}

// Returns performed request to given link. This also sets some header values, such as correct cookies and authorization token if it exists
func getRequest(link string) (*http.Response, error) {

	req, err := http.NewRequest("GET", link, nil)
	if err != nil {
		log.Fatalln(err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (QtEmbedded; U; Linux; C) AppleWebKit/533.3 (KHTML, like Gecko) MAG200 stbapp ver: 4 rev: 2116 Mobile Safari/533.3")
	req.Header.Set("X-User-Agent", "Model: MAG254; Link: Ethernet")
	if len(token) > 0 {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Cookie", "PHPSESSID=null; sn="+sn+"; mac="+macEncoded+"; stb_lang=en; timezone="+timeZone)

	return httpClient.Do(req)
}

var tokenRegex = regexp.MustCompile(`"token"\s*:\s*"([A-Z0-9]+)"`)

// Authenticate with stalker portal
func authenticate() {
	// Server identifies us by token. This is a string and will be sent with each HTTP request.
	//
	// How it works: You send a request to the server (see below URL) with a suggested token ('token' parameter).
	// Server checks if such token is in use. If in use - server will return randomly generated token in JSON and
	// below code will extract it and use in each HTTP request. If token is not in use - server don't return new
	// randomly generated token and authenticates using suggested token instead.
	resp, err := getRequest(portalURLDomain + "/stalker_portal/server/load.php?type=stb&action=handshake&prehash=0&token=&JsHttpRequest=1-xml")
	if err != nil {
		log.Fatalln(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	parts := tokenRegex.FindStringSubmatch(string(body))
	if len(parts) > 0 {
		token = parts[1]
	} else {
		log.Fatalln("Unable to authenticate at step 1. Just ignore and try again please!")
	}
	resp.Body.Close()

	// Since we have a token, we need to authorize it (associate it with your credentials)
	resp, err = getRequest(portalURLDomain + "/stalker_portal/server/load.php?type=stb&action=do_auth&login=" + login + "&password=" + password + "&device_id=" + deviceID + "&device_id2=" + deviceID2 + "&JsHttpRequest=1-xml")
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	if !strings.Contains(string(body), "bool(true)") {
		log.Fatalln("Unable to authenticate at step 2. Check your credentials and try again")
	}
}
