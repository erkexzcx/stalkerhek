package proxy

import (
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/erkexzcx/stalkerhek/stalker"
)

var destination string
var config *stalker.Config

// Start starts main routine.
func Start(c *stalker.Config) {
	config = c

	// extract scheme://hostname:port from given URL, so we don't have to do that later
	link, err := url.Parse(config.Portal.Location)
	if err != nil {
		log.Fatalln(err)
	}
	destination = link.Scheme + "://" + link.Host

	mux := http.NewServeMux()
	mux.HandleFunc("/", requestHandler)

	log.Println("Proxy service should be started!")
	panic(http.ListenAndServe(config.Proxy.Bind, mux))
}

func requestHandler(w http.ResponseWriter, r *http.Request) {
	log.Println(r.RequestURI)

	// Create key=value map, but make value as string (not as []string) leaving only first value.
	originalQuery := r.URL.Query()
	simplifiedQuery := make(map[string]string, len(originalQuery))
	for k, v := range originalQuery {
		simplifiedQuery[k] = v[0]
	}

	// ################################################

	// We are going to constantly use "action" key's value, so dedicate variable for it:
	keyAction := simplifiedQuery["action"]

	// Handshake
	if keyAction == "handshake" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"js":{"token":"` + config.Portal.Token + `","random":"b8c4ef93de04e675350605eb0086bffe51507b88e6a1662e71fe9372"},"text":"generated in: 0.01s; query counter: 1; cache hits: 0; cache miss: 0; php errors: 0; sql errors: 0;"}`))
		return
	}

	if keyAction == "get_events" {
		keyType := simplifiedQuery["type"]

		// Watchdog
		if keyType == "watchdog" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"js":{"data":{"msgs":0,"additional_services_on":"1"}},"text":"generated in: 0.01s; query counter: 4; cache hits: 0; cache miss: 0; php errors: 0; sql errors: 0;"}`))
			return
		}

		// Log
		if keyType == "log" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"js":1,"text":"generated in: 0.001s; query counter: 7; cache hits: 0; cache miss: 0; php errors: 0; sql errors: 0;"}`))
			return
		}

	}

	// Authentication
	if keyAction == "do_auth" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"js":true,"text":"array(2) {\n  [\"status\"]=>\n  string(2) \"OK\"\n  [\"results\"]=>\n  bool(true)\n}\ngenerated in: 1.033s; query counter: 7; cache hits: 0; cache miss: 0; php errors: 0; sql errors: 0;"}`))
		return
	}

	// Logout
	if keyAction == "logout" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"js":true,"text":"generated in: 0.011s; query counter: 4; cache hits: 0; cache miss: 0; php errors: 0; sql errors: 0;"}`))
		return
	}

	// ################################################

	// If 'rewrite' option is enabled
	if config.Proxy.Rewrite && keyAction == "create_link" {
		keyType := simplifiedQuery["type"]

		if keyType == "itv" {
			handleRewriteITV(w, r, simplifiedQuery["cmd"])
			return
		}

		// if keyType == "tv_archive" {
		// 	handleRewriteITV(w, r, simplifiedQuery["cmd"])
		// 	return
		// }

		// if keyType == "vod" {
		// 	handleRewriteITV(w, r, simplifiedQuery["cmd"])
		// 	return
		// }
	}

	// ################################################

	// Serial number
	if _, exists := simplifiedQuery["sn"]; exists {
		originalQuery["sn"] = []string{config.Portal.SerialNumber}
	}

	// Device ID
	if _, exists := simplifiedQuery["device_id"]; exists {
		originalQuery["device_id"] = []string{config.Portal.DeviceID}
	}

	// Device ID2
	if _, exists := simplifiedQuery["device_id2"]; exists {
		originalQuery["device_id2"] = []string{config.Portal.DeviceID2}
	}

	// Signature
	if _, exists := simplifiedQuery["signature"]; exists {
		originalQuery["signature"] = []string{config.Portal.Signature}
	}

	// ################################################

	// Build (modified) URL
	finalLink := destination + r.URL.Path
	if len(r.URL.RawQuery) != 0 {
		finalLink += "?" + originalQuery.Encode()
	}

	// Perform request
	resp, err := getRequest(finalLink, r)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Send response
	addHeaders(resp.Header, w.Header())
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
