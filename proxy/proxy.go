package proxy

import (
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/erkexzcx/stalkerhek/stalker"
)

var (
	destination string
	portal      *stalker.Portal
)

func Start(p *stalker.Portal, bind string) {
	portal = p

	// extract scheme://hostname:port from given URL
	link, err := url.Parse(p.Location)
	if err != nil {
		log.Fatalln(err)
	}
	destination = link.Scheme + "://" + link.Host

	http.HandleFunc("/", requestHandler)

	log.Println("Proxy service should be started!")
	panic(http.ListenAndServe(bind, nil))
}

func requestHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	var tagAction string
	if tmp, found := query["action"]; found {
		tagAction = tmp[0]
	}

	var tagType string
	if tmp, found := query["type"]; found {
		tagType = tmp[0]
	}

	// ################################################
	// Ignore/fake some requests

	// Handshake
	if tagAction == "handshake" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"js":{"token":"123456789012345678901234567890AF","random":"123456789012345678901234567890AF7890AFAA"},"text":"generated in: 0.001s; query counter: 1; cache hits: 0; cache miss: 0; php errors: 0; sql errors: 0;"}`))
		// TODO - when given token is accepted by the server, message is different (shorter) and would be more applicable here.
		return
	}

	// Watchdog
	if tagAction == "get_events" && tagType == "watchdog" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"js":{"data":{"msgs":0,"additional_services_on":"1"}},"text":"generated in: 0.01s; query counter: 4; cache hits: 0; cache miss: 0; php errors: 0; sql errors: 0;"}`))
		return
	}

	// Authentication
	if tagAction == "do_auth" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"js":true,"text":"array(2) {\n  [\"status\"]=>\n  string(2) \"OK\"\n  [\"results\"]=>\n  bool(true)\n}\ngenerated in: 1.033s; query counter: 7; cache hits: 0; cache miss: 0; php errors: 0; sql errors: 0;"}`))
		return
	}

	// Logoff
	if tagAction == "???" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`???`))
		return
	}

	// ################################################
	// Rewrite some URL query values

	// Serial number
	if _, exists := query["sn"]; exists {
		query["sn"] = []string{portal.SerialNumber}
	}

	// Device ID
	if _, exists := query["device_id"]; exists {
		query["device_id"] = []string{portal.DeviceID}
	}

	// Device ID2
	if _, exists := query["device_id2"]; exists {
		query["device_id2"] = []string{portal.DeviceID2}
	}

	// Signature
	if _, exists := query["signature"]; exists {
		query["signature"] = []string{portal.Signature}
	}

	// ################################################
	// Proxy modified request to real Stalker portal and return the response

	// Build (modified) URL
	finalLink := destination + r.URL.Path
	if len(r.URL.RawQuery) != 0 {
		finalLink += "?" + query.Encode()
	}

	// Perform request
	resp, err := getRequest(finalLink, r)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Send response
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
