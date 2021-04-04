package proxy

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"

	"github.com/erkexzcx/stalkerhek/stalker"
)

var (
	destination string

	config *stalker.Config

	channels map[string]*stalker.Channel
)

// Start starts main routine.
func Start(c *stalker.Config, chs map[string]*stalker.Channel) {
	config = c

	// Channels will be matched by CMD field, not by title
	newChannels := make(map[string]*stalker.Channel)
	for _, v := range chs {
		newChannels[v.CMD] = v
	}
	channels = newChannels

	// extract scheme://hostname:port from given URL, so we don't have to do it later
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

	query := r.URL.Query()

	var tagAction string
	if tmp, found := query["action"]; found {
		tagAction = tmp[0]
	}

	var tagType string
	if tmp, found := query["type"]; found {
		tagType = tmp[0]
	}

	var tagCMD string
	if tmp, found := query["cmd"]; found {
		tagCMD = tmp[0]
	}

	// ################################################
	// Ignore/fake some requests

	// Handshake
	if tagAction == "handshake" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"js":{"token":"` + config.Portal.Token + `","random":"b8c4ef93de04e675350605eb0086bffe51507b88e6a1662e71fe9372"},"text":"generated in: 0.01s; query counter: 1; cache hits: 0; cache miss: 0; php errors: 0; sql errors: 0;"}`))
		return
	}

	// Watchdog
	if tagAction == "get_events" && tagType == "watchdog" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"js":{"data":{"msgs":0,"additional_services_on":"1"}},"text":"generated in: 0.01s; query counter: 4; cache hits: 0; cache miss: 0; php errors: 0; sql errors: 0;"}`))
		return
	}

	// Log
	if tagAction == "get_events" && tagType == "log" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"js":1,"text":"generated in: 0.001s; query counter: 7; cache hits: 0; cache miss: 0; php errors: 0; sql errors: 0;"}`))
		return
	}

	// Authentication
	if tagAction == "do_auth" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"js":true,"text":"array(2) {\n  [\"status\"]=>\n  string(2) \"OK\"\n  [\"results\"]=>\n  bool(true)\n}\ngenerated in: 1.033s; query counter: 7; cache hits: 0; cache miss: 0; php errors: 0; sql errors: 0;"}`))
		return
	}

	// Logout
	if tagAction == "logout" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"js":true,"text":"generated in: 0.011s; query counter: 4; cache hits: 0; cache miss: 0; php errors: 0; sql errors: 0;"}`))
		return
	}

	// Rewrite links
	if config.Proxy.Rewrite && tagAction == "create_link" {
		if tagCMD == "" {
			log.Println("STB requested 'create_link', but did not give 'cmd' key in URL query...")
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		// Find Stalker channel
		channel, found := channels[tagCMD]
		if !found {
			log.Println("STB requested 'create_link', but gave invalid CMD:", tagCMD)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		// We must give full path to IPTV stream.
		requestHost, _, _ := net.SplitHostPort(r.Host)
		_, portHLS, _ := net.SplitHostPort(config.HLS.Bind)
		destination = "http://" + requestHost + ":" + portHLS + "/iptv/" + url.PathEscape(channel.Title)

		w.WriteHeader(http.StatusOK)

		responseText := generateNewChannelLink(destination, channel.CMD_ID, channel.CMD_CH_ID)
		w.Write([]byte(responseText))

		fmt.Println(responseText)

		return
	}

	// ################################################
	// Rewrite some URL query values

	// Serial number
	if _, exists := query["sn"]; exists {
		query["sn"] = []string{config.Portal.SerialNumber}
	}

	// Device ID
	if _, exists := query["device_id"]; exists {
		query["device_id"] = []string{config.Portal.DeviceID}
	}

	// Device ID2
	if _, exists := query["device_id2"]; exists {
		query["device_id2"] = []string{config.Portal.DeviceID2}
	}

	// Signature
	if _, exists := query["signature"]; exists {
		query["signature"] = []string{config.Portal.Signature}
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
	addHeaders(resp.Header, w.Header())
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
