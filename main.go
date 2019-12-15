package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
)

const sn = "0000000000000"                                                           // Set your Serial Number
const deviceID = "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"  // Set your device ID
const deviceID2 = "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff" // Set your device ID2
const signature = "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff" // Set your signature here
const macEncoded = "00%3A00%3A00%3A00%3A00%3A00"                                     // Set your URL encoded MAC address
const login = ""                                                                     // Set your URL encoded username
const password = ""                                                                  // Set your URL encoded password
const portalURLDomain = "http://domain.example.com"                                  // Must end withOUT slash at the end!

const timeZone = "Europe%2FVilnius" // Update to your local timezone (URL encoded)

var token string

var httpClient = &http.Client{}

func handleRequest(w http.ResponseWriter, r *http.Request) {

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

	// Rewrite any identifying parameter's value, deviceID or sn:
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

	// Perform request to real server and return output
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

	// Start listening for requests
	http.HandleFunc("/", handleRequest)
	log.Fatal(http.ListenAndServe(":80", nil))
}

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

func authenticate() {
	// Generate token (server will provide this):
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
		log.Fatalln("Unable to authenticate (step 1)")
	}
	resp.Body.Close()

	// Authorize this token (assosiate username & password with this token):
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
		log.Fatalln("Unable to authenticate (step 2)")
	}
}
